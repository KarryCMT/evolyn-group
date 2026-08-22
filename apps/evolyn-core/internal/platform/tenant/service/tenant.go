package service

import (
	"context"
	"crypto/rand"
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

	"golang.org/x/crypto/bcrypt"
)

// TxManager 事务边界抽象（FIX-020）：具体实现在 infrastructure（ctx 传播
// 事务 session），Service 只依赖最小接口，便于单测以快照/恢复模拟回滚
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// IAMRepositories 开通租户所需的 iam 仓储能力子集：接口化便于测试替身，
// 生产侧由 *iamrepository.Repositories 天然满足
type IAMRepositories interface {
	Account() iamrepository.AccountRepository
	User() iamrepository.UserRepository
	RBAC() iamrepository.RBACRepository
	Group() iamrepository.GroupRepository
}

// 租户内基线角色名（与默认租户种子、db.sql 口径一致）
const (
	TenantAdminRole     = "tenant-admin"
	AuthenticatedRole   = "authenticated"
	UnAuthenticatedRole = "unauthenticated"
)

// DefaultRetentionPeriod 注销数据默认保留期（FIX-012）：保留期内可恢复，
// 到期由 Purge Worker 物理清理。可用配置 tenant.retentionDays 覆盖
const DefaultRetentionPeriod = 30 * 24 * time.Hour

// ErrCodeDuplicated 租户编码冲突（ADR-008：重复冲突 409）：
// 指定编码开通时撞已有租户（含注销保留期内的墓碑），编码细节不出网
var ErrCodeDuplicated = httpx.NewBiz("TENANT_CODE_DUPLICATED", "租户编码已存在", http.StatusConflict)

// tenantService 租户域服务：开通/查询/配置/生命周期流转（运营面）。
// 依赖 iam 仓储完成「开通即建 owner 成员 + 租户内系统组/角色种子」；
// 开通全流程经 TxManager 单事务提交（FIX-020）；审计记录关键运营操作
// （FIX-013），配额服务在开建成员前校验（FIX-011）
type tenantService struct {
	tx         TxManager
	tenantRepo tenantrepository.TenantRepository
	iam        IAMRepositories
	quota      QuotaService
	audit      auditservice.Recorder
	retention  time.Duration
}

func NewTenantService(
	tx TxManager,
	tenantRepo tenantrepository.TenantRepository,
	iam IAMRepositories,
	quota QuotaService,
	audit auditservice.Recorder,
	retention time.Duration,
) TenantService {
	if retention <= 0 {
		retention = DefaultRetentionPeriod
	}
	return &tenantService{
		tx:         tx,
		tenantRepo: tenantRepo,
		iam:        iam,
		quota:      quota,
		audit:      audit,
		retention:  retention,
	}
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

	// 基线种子返回本租户刚创建的角色：owner 绑定直接用内存对象，不再按名
	// 回查——无租户过滤的 GetRoleByName 会按 id 升序命中其他租户的同名
	// tenant-admin 角色，造成跨租户角色绑定污染（FIX-022 攻击面）
	baselineRoles, err := s.seedTenantBaseline(bctx, tenant.ID)
	if err != nil {
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
			{Resource: "users", Operation: iammodel.AllOperation},
			{Resource: "groups", Operation: iammodel.AllOperation},
			{Resource: "roles", Operation: iammodel.AllOperation},
			{Resource: "departments", Operation: iammodel.AllOperation},
			// 应用管理（M2-A）：租户管理员全量；存量租户由 000014 订正
			{Resource: "applications", Operation: iammodel.AllOperation},
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
