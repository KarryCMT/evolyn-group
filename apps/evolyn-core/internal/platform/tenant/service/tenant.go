package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"evolyn/internal/contextx"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	iamrepository "evolyn/internal/platform/iam/repository"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantrepository "evolyn/internal/platform/tenant/repository"
	"evolyn/internal/utils/request"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// TxManager 事务边界抽象（FIX-020）：具体实现在 infrastructure（ctx 传播
// 事务 session），Service 只依赖最小接口，便于单测以快照/恢复模拟回滚
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// SubscriptionSeeder 开通事务内补种初始订阅（版本信息一期）：消费者侧窄
// 接口，由 edition 服务结构性实现，装配期可选注入（nil 时开通不落订阅，
// 读取侧按 tenants.plan 合成兜底视图）。活动订阅是权益事实源，新租户
// 开通必须同事务补齐，避免「开通即缺事实源」
type SubscriptionSeeder interface {
	// SeedInitial 在调用方事务内为新租户补种初始订阅；planCode 取开通
	// 请求的套餐（free/pro 落长期活动订阅，trial 无到期信息落待补录态）
	SeedInitial(ctx context.Context, tenantID uint, planCode string) error
}

// IAMRepositories 开通租户所需的 iam 仓储能力子集：接口化便于测试替身，
// 生产侧由 *iamrepository.Repositories 天然满足
type IAMRepositories interface {
	Account() iamrepository.AccountRepository
	User() iamrepository.UserRepository
	RBAC() iamrepository.RBACRepository
	Group() iamrepository.GroupRepository
	Department() iamrepository.DepartmentRepository
	AdminGroup() iamrepository.AdminGroupRepository
}

// MemberFieldSeeder 开通租户时预置成员字段默认配置的最小能力（成员信息
// 管理一期）：由 iam 字段配置服务实现，经 MemberFieldSeederInjector 注入
// （口径同 QuotaGuardInjector，避免构造参数继续膨胀）
type MemberFieldSeeder interface {
	SeedDefaults(ctx context.Context, tenantID uint) error
}

// MemberFieldSeederInjector 支持事后注入字段配置种子器的租户服务
type MemberFieldSeederInjector interface {
	UseFieldSeeder(seeder MemberFieldSeeder)
}

// ProductConfigSeeder 开通租户时初始化产品配置的最小能力（产品中心一期）：
// 由 tenantproduct 服务实现，经 ProductConfigSeederInjector 注入——为全部
// 当前 active 目录创建 enabled=true、scope=all 的配置行，与初始订阅、
// 管理员成员保持同一开通事务边界（产品中心文档 8.2）
type ProductConfigSeeder interface {
	SeedDefaults(ctx context.Context, tenantID uint) error
}

// ProductConfigSeederInjector 支持事后注入产品配置种子器的租户服务
type ProductConfigSeederInjector interface {
	UseProductSeeder(seeder ProductConfigSeeder)
}

// NotificationSettingSeeder 开通租户时预置通知设置聚合根的最小能力（消息
// 中心 P1）：由 notification 设置服务实现，经 NotificationSettingSeederInjector
// 注入（口径同 ProductConfigSeeder）；读取侧 EnsureSetting 另有幂等兜底，
// 兼容注入缺失的存量路径
type NotificationSettingSeeder interface {
	SeedDefaults(ctx context.Context, tenantID uint) error
}

// NotificationSettingSeederInjector 支持事后注入通知设置种子器的租户服务
type NotificationSettingSeederInjector interface {
	UseNotificationSeeder(seeder NotificationSettingSeeder)
}

// 租户内基线角色名（与默认租户种子、db.sql 口径一致）。
// 名称直接面向组织角色页展示，因此使用中文；租户内权限判定只依赖角色规则，
// 不以名称作为授权依据。
const (
	// TenantAdminRole 常量收敛于 iam 模型层（内置系统管理员组的成员事实源），
	// 租户域 seed 与 iam 域内置组代理共用同一名称
	TenantAdminRole     = iammodel.TenantAdminRoleName
	AuthenticatedRole   = "已认证用户"
	UnAuthenticatedRole = "未认证用户"
)

// DefaultRetentionPeriod 注销数据默认保留期（FIX-012）：保留期内可恢复，
// 到期由 Purge Worker 物理清理。可用配置 tenant.retentionDays 覆盖
const DefaultRetentionPeriod = 30 * 24 * time.Hour

// ErrCodeDuplicated 租户编码冲突（ADR-008：重复冲突 409）：
// 指定编码开通时撞已有租户（含注销保留期内的墓碑），编码细节不出网
var ErrCodeDuplicated = httpx.NewBiz("TENANT_CODE_DUPLICATED", "租户编码已存在", http.StatusConflict)

// ErrTenantNameInvalid 租户侧名称编辑的稳定错误码；不暴露长度等内部校验细节。
var ErrTenantNameInvalid = httpx.NewBiz("TENANT_NAME_INVALID", "租户名称格式不正确", http.StatusBadRequest)

// tenantService 租户域服务：开通/查询/配置/生命周期流转（运营面）。
// 依赖 iam 仓储完成「开通即建 owner 成员/顶级部门 + 租户内系统组/角色种子」；
// 开通全流程经 TxManager 单事务提交（FIX-020）；审计记录关键运营操作
// （FIX-013），配额服务在开建成员前校验（FIX-011）
type tenantService struct {
	tx                 TxManager
	tenantRepo         tenantrepository.TenantRepository
	iam                IAMRepositories
	quota              QuotaService
	audit              auditservice.Recorder
	retention          time.Duration
	subSeeder          SubscriptionSeeder
	fieldSeeder        MemberFieldSeeder
	productSeeder      ProductConfigSeeder
	notificationSeeder NotificationSettingSeeder
}

func NewTenantService(
	tx TxManager,
	tenantRepo tenantrepository.TenantRepository,
	iam IAMRepositories,
	quota QuotaService,
	audit auditservice.Recorder,
	retention time.Duration,
	seeders ...SubscriptionSeeder,
) TenantService {
	if retention <= 0 {
		retention = DefaultRetentionPeriod
	}
	svc := &tenantService{
		tx:         tx,
		tenantRepo: tenantRepo,
		iam:        iam,
		quota:      quota,
		audit:      audit,
		retention:  retention,
	}
	if len(seeders) > 0 {
		svc.subSeeder = seeders[0]
	}
	return svc
}

// UseFieldSeeder 注入成员字段配置种子器（成员信息管理一期）：新租户开通
// 事务内预置 15 个预置字段的默认显示策略；未注入时读取侧幂等兜底补齐
func (s *tenantService) UseFieldSeeder(seeder MemberFieldSeeder) {
	s.fieldSeeder = seeder
}

// UseProductSeeder 注入产品配置种子器（产品中心一期）：新租户开通事务内
// 为全部 active 目录初始化默认产品配置；未注入时产品中心列表侧保守合成
// 停用卡片（写入路径仍按未初始化拒绝）
func (s *tenantService) UseProductSeeder(seeder ProductConfigSeeder) {
	s.productSeeder = seeder
}

// UseNotificationSeeder 注入通知设置种子器（消息中心 P1）：新租户开通事务内
// 预置通知设置聚合根；未注入时读取侧 EnsureSetting 幂等兜底
func (s *tenantService) UseNotificationSeeder(seeder NotificationSettingSeeder) {
	s.notificationSeeder = seeder
}

// OpenTenantRequest 开通租户请求：OwnerAccountID 与 Owner 账号信息二选一
type OpenTenantRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Plan string `json:"plan"`

	OwnerAccountID uint `json:"ownerAccountId"` // 复用既有账号

	// 新建 owner 账号（登录身份）
	OwnerName     string `json:"ownerName"`
	OwnerPhone    string `json:"ownerPhone"`
	OwnerEmail    string `json:"ownerEmail"`
	OwnerPassword string `json:"ownerPassword"`

	// Onboarding 注册向导企业画像：仅自助开通信道填写，平台运营开通不暴露
	// （json:"-" 不出现在 API 契约，经 SelfOpen 内部传递）
	Onboarding tenantmodel.OnboardingConfig `json:"-"`
}

// Validate 开通参数校验
func (r *OpenTenantRequest) Validate() error {
	if r.Code == "" || r.Name == "" {
		return fmt.Errorf("tenant code/name is required")
	}
	if r.Plan == "" {
		r.Plan = tenantmodel.PlanFree
	}
	if !tenantmodel.IsValidPlan(r.Plan) {
		return fmt.Errorf("unknown plan: %s", r.Plan)
	}
	if r.OwnerAccountID == 0 && (r.OwnerName == "" || len(r.OwnerPassword) < 6) {
		return fmt.Errorf("owner account id or owner name/password (>=6 chars) is required")
	}
	return nil
}

// Open 开通租户（P3-1，FIX-020 全事务）：租户 + owner 账号/成员 + 租户内
// 系统组/角色种子一步到位。Account/Tenant/Owner Member/Role/Group/Binding
// 等全部写操作共享同一数据库事务，任一步失败整体回滚，不留半初始化租户
func (s *tenantService) Open(ctx context.Context, req *OpenTenantRequest) (*tenantmodel.Tenant, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var tenant *tenantmodel.Tenant
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		var err error
		tenant, err = s.openInTx(tctx, req)
		return err
	}); err != nil {
		return nil, err
	}

	// 审计在事务提交成功后独立写入（FIX-020 决策：业务回滚不留「未发生的
	// 开通」流水；审计 best-effort，落库失败仅告警不阻断已提交业务）
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "tenant", Action: "create", ResourceType: "tenant",
			ResourceID: strconv.FormatUint(uint64(tenant.ID), 10),
			TenantID:   tenant.ID,
			After:      map[string]any{"code": tenant.Code, "name": tenant.Name, "plan": tenant.Plan},
		})
	}

	return tenant, nil
}

// SelfOpen 自助开通租户（登录态「创建团队」，POST /auth/tenant）：创建者即
// 所有者并绑定 tenant-admin 角色。独立事务提交，成功后记审计（与 Open 同口径）；
// 需加入调用方外层事务的组合场景（注册向导最终提交）请改用 SelfOpenInTx
func (s *tenantService) SelfOpen(ctx context.Context, ownerAccountID uint, name string, onboarding tenantmodel.OnboardingConfig) (*tenantmodel.Tenant, error) {
	if err := validateSelfOpen(ownerAccountID, name); err != nil {
		return nil, err
	}

	var tenant *tenantmodel.Tenant
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		var err error
		tenant, err = s.selfOpenCore(tctx, ownerAccountID, name, onboarding)
		return err
	}); err != nil {
		return nil, err
	}

	// 审计在事务提交成功后独立写入（best-effort，FIX-020 决策）
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "tenant", Action: "create", ResourceType: "tenant",
			ResourceID: strconv.FormatUint(uint64(tenant.ID), 10),
			TenantID:   tenant.ID,
			After:      map[string]any{"code": tenant.Code, "name": tenant.Name, "plan": tenant.Plan},
		})
	}
	return tenant, nil
}

// SelfOpenInTx 事务内自助开通（注册向导最终提交的组合通道）：与 SelfOpen
// 同一核心流程，但不另开事务、不记审计——仓储经 ResolveDB 自动加入调用方
// ctx 携带的外层事务；若在此记审计会先于外层提交落库，外层回滚时留下
// 「未发生的开通」假流水（FIX-020 口径），审计由调用方在提交成功后补记
func (s *tenantService) SelfOpenInTx(ctx context.Context, ownerAccountID uint, name string, onboarding tenantmodel.OnboardingConfig) (*tenantmodel.Tenant, error) {
	if err := validateSelfOpen(ownerAccountID, name); err != nil {
		return nil, err
	}
	return s.selfOpenCore(ctx, ownerAccountID, name, onboarding)
}

// validateSelfOpen 自助开通参数门禁：名称 2-50 字符（去首尾空白）、
// owner 账号必填——在进入事务前拒绝，避免无效请求占用事务连接
func validateSelfOpen(ownerAccountID uint, name string) error {
	name = strings.TrimSpace(name)
	if runeLen := len([]rune(name)); runeLen < 2 || runeLen > 50 {
		return fmt.Errorf("tenant name must be 2-50 characters")
	}
	if ownerAccountID == 0 {
		return fmt.Errorf("owner account is required")
	}
	return nil
}

// selfOpenCore 自助开通核心流程（不含参数门禁/事务/审计）：随机编码撞
// 唯一索引自动换码重试 + openInTx 建租户/owner 成员/基线种子。直接调用
// openInTx（不经 Open 的 Validate），套餐显式取免费版默认
func (s *tenantService) selfOpenCore(ctx context.Context, ownerAccountID uint, name string, onboarding tenantmodel.OnboardingConfig) (*tenantmodel.Tenant, error) {
	const attempts = 3
	var (
		tenant *tenantmodel.Tenant
		err    error
	)
	for range attempts {
		tenant, err = s.openInTx(ctx, &OpenTenantRequest{
			Name:           strings.TrimSpace(name),
			Code:           generateTenantCode(),
			Plan:           tenantmodel.PlanFree,
			OwnerAccountID: ownerAccountID,
			Onboarding:     onboarding,
		})
		if err == nil {
			return tenant, nil
		}
		if !strings.Contains(err.Error(), "already exists") {
			return nil, err
		}
	}
	return nil, fmt.Errorf("failed to generate unique tenant code: %w", err)
}

// generateTenantCode 租户编码：t- 前缀 + 8 位随机 hex（16^32 空间），
// 仅作登录识别用途，不承载业务语义
func generateTenantCode() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand 失败的极端场景：退回时间戳尾数，保留唯一性概率
		return fmt.Sprintf("t-%x", time.Now().UnixNano())
	}
	return fmt.Sprintf("t-%x", buf)
}

// openInTx 事务内的开通主流程：调用方已开启事务，本方法内任一步失败由
// 外层整体回滚。全程使用剥离租户的 ctx（contextx.DetachTenant 保留事务
// session 与操作者/元数据）：运营者会话自带的租户上下文会污染新租户的
// 数据写入与按名查询（组/角色按租户过滤）
func (s *tenantService) openInTx(ctx context.Context, req *OpenTenantRequest) (*tenantmodel.Tenant, error) {
	bctx := contextx.DetachTenant(ctx)

	if _, err := s.tenantRepo.GetByCode(bctx, req.Code); err == nil {
		return nil, httpx.Wrap(ErrCodeDuplicated, fmt.Errorf("tenant code %s already exists", req.Code))
	}

	// owner 账号：复用或新建
	var (
		ownerAccount *iammodel.Account
		err          error
	)
	if req.OwnerAccountID > 0 {
		ownerAccount, err = s.iam.Account().GetByID(bctx, req.OwnerAccountID)
	} else {
		var password []byte
		password, err = bcrypt.GenerateFromPassword([]byte(req.OwnerPassword), bcrypt.DefaultCost)
		if err == nil {
			ownerAccount, err = s.iam.Account().Create(bctx, &iammodel.Account{
				Name:     req.OwnerName,
				Nickname: req.OwnerName,
				Phone:    req.OwnerPhone,
				Email:    req.OwnerEmail,
				Password: string(password),
			})
		}
	}
	if err != nil {
		return nil, err
	}

	tenant := &tenantmodel.Tenant{
		Code:           req.Code,
		Name:           req.Name,
		Plan:           req.Plan,
		Status:         tenantmodel.TenantActive,
		OwnerAccountId: &ownerAccount.ID,
		Config: func() tenantmodel.TenantConfig {
			// 默认配置为底、叠加注册向导采集的企业画像
			config := tenantmodel.DefaultTenantConfig()
			config.Onboarding = req.Onboarding
			return config
		}(),
	}
	tenant, err = s.tenantRepo.Create(bctx, tenant)
	if err != nil {
		return nil, err
	}

	// 版本信息一期：开通事务内补种初始订阅（活动订阅是权益事实源）；
	// 注入失败即整体回滚，不允许出现「有租户无订阅」的半写状态
	if s.subSeeder != nil {
		if err = s.subSeeder.SeedInitial(bctx, tenant.ID, req.Plan); err != nil {
			return nil, err
		}
	}

	// owner 成员：先配额校验（FIX-011），再显式指定租户归属创建
	if s.quota != nil {
		if err = s.quota.Check(bctx, tenant.ID, tenantmodel.QuotaMembers); err != nil {
			return nil, err
		}
	}
	nickname := ownerAccount.Nickname
	if nickname == "" {
		nickname = ownerAccount.Name
	}
	member := &iammodel.User{AccountId: ownerAccount.ID, Nickname: nickname}
	member.TenantID = tenant.ID
	if _, err = s.iam.User().Create(bctx, member); err != nil {
		return nil, err
	}

	// 每个租户以自身名称创建唯一的组织根部门，并将创建者归属到该部门。
	// 这里重新注入新租户上下文，使部门及成员-部门关系与常规租户写入保持
	// 一致；bctx 仍携带外层事务，因此任一步失败会随整次开通一起回滚。
	tenantCtx := contextx.NewTenantContext(bctx, tenant.ID)
	rootDepartment := &iammodel.Department{Name: tenant.Name}
	rootDepartment.TenantID = tenant.ID
	if _, err = s.iam.Department().Create(tenantCtx, rootDepartment); err != nil {
		return nil, err
	}
	if err = s.iam.Department().SetMemberDepartments(tenantCtx, member, []uint{rootDepartment.ID}); err != nil {
		return nil, err
	}

	// 基线种子返回本租户刚创建的角色：owner 绑定直接用内存对象，不再按名
	// 回查——无租户过滤的 GetRoleByName 会按 id 升序命中其他租户的同名
	// tenant-admin 角色，造成跨租户角色绑定污染（FIX-022 攻击面）
	baselineRoles, err := s.seedTenantBaseline(bctx, tenant.ID)
	if err != nil {
		return nil, err
	}

	// 成员信息管理一期：开通事务内预置成员字段默认配置（幂等；注入失败
	// 即整体回滚，不允许出现「有租户无字段配置」的半写状态，读取侧另有兜底）
	if s.fieldSeeder != nil {
		if err = s.fieldSeeder.SeedDefaults(contextx.NewTenantContext(bctx, tenant.ID), tenant.ID); err != nil {
			return nil, err
		}
	}

	// 产品中心一期：开通事务内初始化产品配置（幂等；为全部 active 目录建
	// enabled=true、scope=all 的配置行）；注入失败即整体回滚，保证新租户
	// 与存量回填租户的产品配置口径一致（产品中心文档 8.2）
	if s.productSeeder != nil {
		if err = s.productSeeder.SeedDefaults(contextx.NewTenantContext(bctx, tenant.ID), tenant.ID); err != nil {
			return nil, err
		}
	}

	// 消息中心 P1：开通事务内预置通知设置聚合根（幂等）；注入失败即整体
	// 回滚，读取侧 EnsureSetting 另有幂等兜底兼容存量路径
	if s.notificationSeeder != nil {
		if err = s.notificationSeeder.SeedDefaults(contextx.NewTenantContext(bctx, tenant.ID), tenant.ID); err != nil {
			return nil, err
		}
	}

	// 权限中心-管理员模块：开通事务内预置内置系统管理员组（幂等）；成员由
	// 上方 owner 绑定的 tenant-admin 角色实时推导，不落成员表
	if err = s.seedBuiltinAdminGroup(bctx, tenant.ID); err != nil {
		return nil, err
	}

	for i := range baselineRoles {
		if baselineRoles[i].Name == TenantAdminRole {
			if err = s.iam.User().AddRole(bctx, &baselineRoles[i], member); err != nil {
				return nil, err
			}
			break
		}
	}

	return tenant, nil
}

// seedTenantBaseline 租户内基线种子：系统组×3 + 基础角色×3 + 组角色绑定。
// 显式写 TenantID（bctx 无租户上下文，避免落到列默认值）。
// 返回本租户刚创建的角色切片（带 ID），供调用方直接绑定关系
func (s *tenantService) seedTenantBaseline(bctx context.Context, tenantID uint) ([]iammodel.Role, error) {
	roles := []iammodel.Role{
		{Name: TenantAdminRole, Rules: iammodel.Rules{
			// 租户组织根节点（租户名称）自助维护；套餐、配额仍只允许平台运营面修改。
			{Resource: iammodel.TenantResource, Operation: iammodel.EditOperation},
			// /members 是租户成员管理路由，创建者须具备完整管理权限。
			{Resource: iammodel.MemberResource, Operation: iammodel.AllOperation},
			{Resource: "groups", Operation: iammodel.AllOperation},
			{Resource: "roles", Operation: iammodel.AllOperation},
			{Resource: "departments", Operation: iammodel.AllOperation},
			// 应用管理（M2-A）：租户管理员全量；存量租户由 000014 订正
			{Resource: "applications", Operation: iammodel.AllOperation},
			// 文件管理：私有对象的签名、确认与删除均由租户管理员全量管理；
			// 存量租户由 000017 补齐。
			{Resource: "files", Operation: iammodel.AllOperation},
			// 版本信息只读概览（版本信息一期）；存量租户由 000030 补齐
			{Resource: "editions", Operation: request.GetOperation},
			// 成员信息管理（字段设置/卡片展示）：租户管理员全量；
			// 存量租户由 000031 补齐
			{Resource: iammodel.MemberFieldSettingResource, Operation: iammodel.AllOperation},
			// 权限中心-管理员模块：租户创建者须能读取并维护内置系统管理员组；
			// 存量租户由 000032/000035 补齐，新租户必须在基线中直接具备该权限。
			{Resource: iammodel.AdminGroupResource, Operation: iammodel.AllOperation},
			// 产品中心（内置产品启停/可用范围）：view 展开 get+list 覆盖
			// 卡片列表，update 覆盖启停与范围替换两条 PUT 子资源路径；
			// 存量租户由 000033 补齐
			{Resource: iammodel.TenantProductResource, Operation: iammodel.ViewOperation},
			{Resource: iammodel.TenantProductResource, Operation: request.UpdateOperation},
			// 企业日志（登录/操作日志）：view 覆盖查询接口，create 覆盖导出
			// 任务创建（enterprise-logs:export 语义，下载路径复核 create）；
			// 存量租户由 000036 补齐。该资源不经管理组间接放行。
			{Resource: iammodel.EnterpriseLogResource, Operation: iammodel.ViewOperation},
			{Resource: iammodel.EnterpriseLogResource, Operation: request.CreateOperation},
			// 表单资产（ADR-010）：表单设计/发布/删除全量；存量租户由 000037
			// 按「管理员规则签名」补授（与角色名无关）
			{Resource: iammodel.FormResource, Operation: iammodel.AllOperation},
			// 通知设置（消息中心）：租户级偏好与自定义提醒对象全量管理；
			// 存量租户由 000039 按「管理员规则签名」补授。不经管理组间接放行
			{Resource: iammodel.NotificationSettingResource, Operation: iammodel.AllOperation},
			// 表单菜单按钮动作（ADR-011）：切换类型/复制/隐藏等菜单按钮的
			// 动作授权键，动作不对应 URL 门（各域 Service 复核）；存量租户由
			// 000047 按「管理员规则签名」补授
			{Resource: iammodel.FormMenuActionResource, Operation: iammodel.AllOperation},
		}},
		{Name: AuthenticatedRole, Rules: iammodel.Rules{
			{Resource: "users", Operation: iammodel.AllOperation},
			{Resource: "auth", Operation: iammodel.AllOperation},
			// 账号自助（/accounts/me 仅限本人资料/密码，无全局账号管理面）：
			// 注册向导第 3 步「完善信息」依赖 update，存量租户由 000011 订正
			{Resource: "accounts", Operation: iammodel.AllOperation},
			// 应用只读（M2-A）：工作台「我的应用」对全体成员可见；创建/编辑/
			// 删除由租户管理员按角色另授；存量租户由 000014 订正
			{Resource: "applications", Operation: iammodel.ViewOperation},
			// 成员可创建并读取自己的文件；文件服务还会复核 creator_id，避免
			// 仅凭通用资源 view 读取同租户其他成员的未绑定文件。
			{Resource: "files", Operation: iammodel.EditOperation},
			// 表单记录提交（ADR-010）：全体成员可提交填写（bootstrap 走
			// applications:get）；表单设计权限在 forms 资源，与本规则分离；
			// 存量租户由 000038 补授
			{Resource: iammodel.FormRecordResource, Operation: request.CreateOperation},
			// 消息中心（消息中心 P1）：全体成员读写自己的收件箱（view 覆盖
			// 摘要/列表，update 覆盖已读）；存量租户由 000039 补授
			{Resource: iammodel.NotificationResource, Operation: iammodel.ViewOperation},
			{Resource: iammodel.NotificationResource, Operation: request.UpdateOperation},
			// 菜单节点个人收藏（ADR-011）：凡节点可见即可收藏，个人状态而非
			// 授权对象；存量租户由 000047 按 authenticated 系统分组补授
			{Resource: iammodel.MenuFavoriteResource, Operation: request.CreateOperation},
			{Resource: iammodel.MenuFavoriteResource, Operation: request.DeleteOperation},
		}},
		{Name: UnAuthenticatedRole, Rules: iammodel.Rules{
			{Resource: "auth", Operation: "create"},
		}},
	}
	for i := range roles {
		roles[i].TenantID = tenantID
		if _, err := s.iam.RBAC().Create(bctx, &roles[i]); err != nil {
			return nil, err
		}
	}

	groups := []iammodel.Group{
		{Name: iammodel.RootGroup, Kind: iammodel.SystemGroup, Describe: "tenant root group"},
		{Name: iammodel.AuthenticatedGroup, Kind: iammodel.SystemGroup, Describe: "system group contains all authenticated user"},
		{Name: iammodel.UnAuthenticatedGroup, Kind: iammodel.SystemGroup, Describe: "system group contains all unauthenticated user"},
	}
	for i := range groups {
		groups[i].TenantID = tenantID
	}
	if err := s.iam.Group().CreateGroups(bctx, groups); err != nil {
		return nil, err
	}

	// 组-角色绑定（按名取回带 ID 的对象，bctx 无租户过滤不适用——
	// 组/角色查询仍会按 ctx 过滤，此处直接用刚创建的切片对象）
	bindings := []struct{ groupName, roleName string }{
		{iammodel.RootGroup, TenantAdminRole},
		{iammodel.AuthenticatedGroup, AuthenticatedRole},
		{iammodel.UnAuthenticatedGroup, UnAuthenticatedRole},
	}
	for _, b := range bindings {
		var (
			group *iammodel.Group
			role  *iammodel.Role
		)
		for i := range groups {
			if groups[i].Name == b.groupName {
				group = &groups[i]
				break
			}
		}
		for i := range roles {
			if roles[i].Name == b.roleName {
				role = &roles[i]
				break
			}
		}
		if group == nil || role == nil {
			return nil, fmt.Errorf("baseline seed mismatch: %s/%s", b.groupName, b.roleName)
		}
		if err := s.iam.Group().AddRole(bctx, role, group); err != nil {
			return nil, err
		}
	}

	return roles, nil
}

// seedBuiltinAdminGroup 开通事务内预置内置系统管理员组（幂等，权限中心-
// 管理员模块）：每租户一行 built_in 组，成员不落 admin_group_members——
// 由 owner 刚绑定的 tenant-admin 角色实时推导，保持单一事实源
func (s *tenantService) seedBuiltinAdminGroup(bctx context.Context, tenantID uint) error {
	tctx := contextx.NewTenantContext(bctx, tenantID)
	adminGroups := s.iam.AdminGroup()
	if _, err := adminGroups.GetByName(tctx, iammodel.AdminGroupBuiltinName); err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	group := &iammodel.AdminGroup{
		Name:        iammodel.AdminGroupBuiltinName,
		Scope:       iammodel.AdminGroupScopeSystem,
		BuiltIn:     true,
		ScopeConfig: iammodel.AdminGroupScopeConfig{},
	}
	// bctx 无租户过滤不适用于组查询，TenantID 显式赋值与 Callback 回填口径一致
	group.TenantID = tenantID
	_, err := adminGroups.Create(tctx, group)
	return err
}

func (s *tenantService) List(ctx context.Context) ([]tenantmodel.Tenant, error) {
	return s.tenantRepo.List(ctx)
}

func (s *tenantService) Get(ctx context.Context, id string) (*tenantmodel.Tenant, error) {
	tid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	return s.tenantRepo.GetByID(ctx, uint(tid))
}

func (s *tenantService) Update(ctx context.Context, id string, tenant *tenantmodel.Tenant) (*tenantmodel.Tenant, error) {
	tid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	if tenant.Plan != "" && !tenantmodel.IsValidPlan(tenant.Plan) {
		return nil, fmt.Errorf("unknown plan: %s", tenant.Plan)
	}
	tenant.ID = uint(tid)
	updated, err := s.tenantRepo.Update(ctx, tenant)
	if err == nil && s.audit != nil {
		// 套餐/配额变更属商业化敏感操作，优先记录（FIX-013）；
		// 平台域链路无租户上下文，显式落目标租户归属
		s.audit.Record(ctx, auditservice.Entry{
			Module: "tenant", Action: "update", ResourceType: "tenant",
			ResourceID: id, TenantID: uint(tid),
			After: map[string]any{"name": tenant.Name, "plan": tenant.Plan, "quotas": tenant.Quotas},
		})
	}
	return updated, err
}

// TransferOwner 安全转移创建人：目标账号不存在成员身份时先建成员并校验配额，
// 再确保其具备 tenant-admin，最后以窄更新写 owner_account_id。原创建人的
// 成员与权限保持不变，避免一次归属转移意外剥夺其原有租户访问权。
func (s *tenantService) TransferOwner(ctx context.Context, tenantID, targetAccountID uint) error {
	if tenantID == 0 || targetAccountID == 0 {
		return httpx.NewBiz(httpx.CodeValidation, "租户和目标账号不能为空", http.StatusBadRequest)
	}

	var previousOwner uint
	err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		if err := s.tenantRepo.LockByID(tctx, tenantID); err != nil {
			return err
		}
		tenant, err := s.tenantRepo.GetByID(tctx, tenantID)
		if err != nil {
			return err
		}
		if tenant.OwnerAccountId != nil {
			previousOwner = *tenant.OwnerAccountId
		}
		if previousOwner == targetAccountID {
			return nil
		}

		bctx := contextx.DetachTenant(tctx)
		account, err := s.iam.Account().GetByID(bctx, targetAccountID)
		if err != nil {
			return err
		}
		tenantCtx := contextx.NewTenantContext(bctx, tenantID)
		member, err := s.iam.User().GetByAccountAndTenant(tenantCtx, targetAccountID, tenantID)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if s.quota != nil {
				if err = s.quota.Check(tenantCtx, tenantID, tenantmodel.QuotaMembers); err != nil {
					return err
				}
			}
			nickname := account.Nickname
			if nickname == "" {
				nickname = account.Name
			}
			member = &iammodel.User{AccountId: account.ID, Nickname: nickname}
			member.TenantID = tenantID
			if _, err = s.iam.User().Create(tenantCtx, member); err != nil {
				return err
			}
		}

		adminRole, err := s.iam.RBAC().GetRoleByName(tenantCtx, TenantAdminRole)
		if err != nil {
			return err
		}
		if err = s.iam.User().AddRole(tenantCtx, adminRole, member); err != nil {
			return err
		}
		return s.tenantRepo.UpdateOwner(tctx, tenantID, targetAccountID)
	})
	if err != nil {
		return err
	}
	if s.audit != nil && previousOwner != targetAccountID {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "tenant", Action: "transfer_owner", ResourceType: "tenant",
			ResourceID: strconv.FormatUint(uint64(tenantID), 10), TenantID: tenantID,
			Before: map[string]uint{"ownerAccountId": previousOwner},
			After:  map[string]uint{"ownerAccountId": targetAccountID},
		})
	}
	return nil
}

// UpdateMyName 租户侧自助更新当前租户名称。tenantID 只能由租户中间件从 JWT
// 注入，控制器不接受路径或请求体中的租户 ID，避免越权修改其他租户。
func (s *tenantService) UpdateMyName(ctx context.Context, tenantID uint, name string) (*tenantmodel.Tenant, error) {
	name = strings.TrimSpace(name)
	if tenantID == 0 || len([]rune(name)) < 2 || len([]rune(name)) > 50 {
		return nil, httpx.Wrap(ErrTenantNameInvalid, fmt.Errorf("tenant name length must be 2-50"))
	}

	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	beforeName := tenant.Name
	if err := s.tenantRepo.UpdateName(ctx, tenantID, name); err != nil {
		return nil, err
	}
	tenant.Name = name

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module:       "tenant",
			Action:       "update_name",
			ResourceType: "tenant",
			ResourceID:   strconv.FormatUint(uint64(tenantID), 10),
			TenantID:     tenantID,
			Before:       map[string]any{"name": beforeName},
			After:        map[string]any{"name": name},
		})
	}
	return tenant, nil
}

// SetStatus 生命周期流转（FIX-007/012）：状态变更即时生效（请求链状态拦截
// 依赖 UpdateStatus 同步失效缓存）；deleted 落注销申请与保留截止时间，
// active 恢复时清空注销时间线
func (s *tenantService) SetStatus(ctx context.Context, id string, status string) error {
	switch status {
	case tenantmodel.TenantActive, tenantmodel.TenantFrozen, tenantmodel.TenantDeleted:
	default:
		return fmt.Errorf("invalid tenant status: %s", status)
	}

	tid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	var lifecycle *tenantrepository.LifecycleTimes
	if status == tenantmodel.TenantDeleted {
		now := time.Now()
		until := now.Add(s.retention)
		lifecycle = &tenantrepository.LifecycleTimes{
			DeleteRequestedAt: now,
			RetentionUntil:    until,
		}
	}

	if err := s.tenantRepo.UpdateStatus(ctx, uint(tid), status, lifecycle); err != nil {
		return err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "tenant", Action: "status", ResourceType: "tenant",
			ResourceID: id, TenantID: uint(tid),
			After: map[string]any{"status": status, "lifecycle": lifecycle},
		})
	}
	return nil
}
