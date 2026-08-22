package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"evolyn/internal/contextx"
	kernel "evolyn/internal/model"
	apperrors "evolyn/internal/platform/application"
	"evolyn/internal/platform/application/model"
	"evolyn/internal/platform/application/repository"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantservice "evolyn/internal/platform/tenant/service"
	"evolyn/internal/utils/set"

	"gorm.io/gorm"
)

// 列表分页与外观枚举常量（§8.1/§15）
const (
	defaultListLimit = 20
	maxListLimit     = 100
	maxNameRunes     = 128

	defaultIcon  = "bookmark"
	defaultColor = "primary"
)

// 稳定外观枚举（§15）：图标/颜色键是应用域领域值，服务端白名单校验，
// 空值取默认。图标集对齐前端空白应用弹窗首期选项；扩展键属于服务端
// 演进决策，前端只回传键值不自造
var (
	applicationIcons  = set.NewString("bookmark", "briefcase", "contacts", "chart", "check")
	applicationColors = set.NewString("primary")
)

// TxManager 事务边界抽象（FIX-021）：具体实现在 infrastructure（ctx 传播
// 事务 session），Service 只依赖最小接口
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// applicationService 应用服务实现。配额执行来自租户域（依赖方向
// application→tenant/service 单向），审计提交后 best-effort 写入；
// 访问判定统一经 ApplicationAccessEvaluator（§9.1），capabilities 与
// 写路径复核不依赖 HTTP 中间件
type applicationService struct {
	tx     TxManager
	repo   repository.ApplicationRepository
	quota  tenantservice.QuotaService
	audit  auditservice.Recorder
	access ApplicationAccessEvaluator
}

func NewApplicationService(
	tx TxManager,
	repo repository.ApplicationRepository,
	quota tenantservice.QuotaService,
	audit auditservice.Recorder,
	access ApplicationAccessEvaluator,
) ApplicationService {
	return &applicationService{
		tx:     tx,
		repo:   repo,
		quota:  quota,
		audit:  audit,
		access: access,
	}
}

// ---- 创建（统一实例化：blank 分支） ----

// CreateBlank 创建空白应用（§5.2 实例化时序）：校验成员/外观 → 事务内
// 配额占位+双记录写入 → 提交后审计。M2-A 无跨引擎资产，同步置 ready
func (s *applicationService) CreateBlank(ctx context.Context, member *iammodel.User, req *model.CreateBlankRequest) (*model.ApplicationDetail, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	// owner/creator 绑定前校验操作者确属当前租户（§9.3，禁止裸 ID 写入）
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(apperrors.ErrMemberInvalid,
			fmt.Errorf("member %d (tenant %d) not in tenant %d", memberID(member), memberTenant(member), tenantID))
	}
	// 与路由层 POST /applications（verb=create）同口径的 Service 复核：
	// 内部调用路径同样拦截无 applications:create 的成员（未认证/跨租户
	// 成员的权限集为空，见 rbacAccessEvaluator 归属守卫）
	if !s.access.Permissions(ctx, member)["applications:create"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member %d cannot create application in tenant %d", member.ID, tenantID))
	}

	name, icon, color, err := normalizeAppearance(req.Name, req.Icon, req.Color)
	if err != nil {
		return nil, err
	}

	var app *model.Application
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		var err error
		// 安装记录 M2-A 出网不直接消费（来源摘要由应用行派生），M2-B
		// 模板安装改用其版本/渠道信息
		app, _, err = s.provisionBlank(tctx, tenantID, member, name, icon, color)
		return err
	}); err != nil {
		return nil, err
	}

	// 审计在事务提交成功后独立写入（best-effort，失败不回滚已建应用）
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "application", Action: "create", ResourceType: "application",
			ResourceID: strconv.FormatUint(uint64(app.ID), 10),
			After:      map[string]any{"name": app.Name, "source": app.SourceType},
		})
	}

	return s.detailFor(s.access.Permissions(ctx, member), app), nil
}

// provisionBlank 事务内的空白实例化主流程（§5.1）：CheckAndReserve 内部
// 先锁租户行再判 apps 限额，通过后 fn 于同一事务写入应用与安装记录，
// 任一步失败由外层整体回滚（不留配额占位或半写状态）
func (s *applicationService) provisionBlank(
	ctx context.Context, tenantID uint, member *iammodel.User, name, icon, color string,
) (*model.Application, *model.Installation, error) {
	var (
		app  *model.Application
		inst *model.Installation
	)
	err := s.quota.CheckAndReserve(ctx, tenantID, tenantmodel.QuotaApps, func(tctx context.Context) error {
		code, cerr := newApplicationCode()
		if cerr != nil {
			return cerr
		}
		// 空白蓝图最小实例化：无默认资产需要复制，直接落 ready
		//（§5.3；默认首页壳/应用管理员绑定随 M2-C Provisioner 接入）
		draft := &model.Application{
			Code:            code,
			Name:            name,
			Icon:            icon,
			Color:           color,
			OwnerMemberID:   member.ID,
			CreatorMemberID: member.ID,
			SourceType:      model.SourceTypeBlank,
			Status:          model.ApplicationStatusActive,
			ProvisionStatus: model.ProvisionStatusReady,
		}
		draft.TenantID = tenantID
		created, aerr := s.repo.Create(tctx, draft)
		if aerr != nil {
			return aerr
		}
		app = created

		inst = &model.Installation{
			TenantID:            tenantID,
			ApplicationID:       app.ID,
			SourceType:          model.SourceTypeBlank,
			Channel:             model.InstallChannelSelf,
			InstalledByMemberID: member.ID,
			InstalledAt:         kernel.JSONTime(time.Now()),
		}
		return s.repo.CreateInstallation(tctx, inst)
	})
	if err != nil {
		return nil, nil, err
	}
	return app, inst, nil
}

// newApplicationCode 生成应用编码：app_ + 16 位随机 hex。租户内唯一由
// uk_applications_tenant_code 部分唯一索引兜底，随机空间下冲突可忽略
func newApplicationCode() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "app_" + hex.EncodeToString(buf), nil
}

// ---- 查询 ----

func (s *applicationService) List(ctx context.Context, member *iammodel.User, query model.ListApplicationsQuery) (*model.ApplicationPage, error) {
	// 与路由层 GET /applications（verb=list）同口径的 Service 复核；
	// 权限集同时用于逐条 capabilities 派生，请求内只计算一次
	perms := s.access.Permissions(ctx, member)
	if !perms["applications:list"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member %d cannot list applications", memberID(member)))
	}

	limit := query.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	if query.Status != "" &&
		query.Status != model.ApplicationStatusActive && query.Status != model.ApplicationStatusArchived {
		return nil, httpx.Wrap(apperrors.ErrQueryInvalid, fmt.Errorf("invalid status filter %q", query.Status))
	}
	afterSort, afterID, hasCursor, err := decodeListCursor(query.Cursor)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrCursorInvalid, err)
	}

	apps, hasMore, err := s.repo.List(ctx, repository.ListParams{
		Keyword:     strings.TrimSpace(query.Keyword),
		Status:      query.Status,
		Limit:       limit,
		HasCursor:   hasCursor,
		AfterSortID: afterSort,
		AfterID:     afterID,
	})
	if err != nil {
		return nil, err
	}

	// 权限集已在入口复核时计算，逐条派生 capabilities 复用同一份
	page := &model.ApplicationPage{Items: make([]model.ApplicationDetail, 0, len(apps)), HasMore: hasMore}
	for i := range apps {
		page.Items = append(page.Items, *s.detailFor(perms, &apps[i]))
	}
	if hasMore && len(apps) > 0 {
		last := apps[len(apps)-1]
		page.NextCursor = encodeListCursor(last.SortOrder, last.ID)
	}
	return page, nil
}

func (s *applicationService) Get(ctx context.Context, member *iammodel.User, id uint) (*model.ApplicationDetail, error) {
	app, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	// 与路由层 GET /applications/:id（verb=get）同口径的 Service 复核，
	// 内部调用路径不依赖中间件也能拦住越权读取
	perms := s.access.Permissions(ctx, member)
	if !perms["applications:get"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member %d cannot get application %d", memberID(member), app.ID))
	}
	return s.detailFor(perms, app), nil
}

// ---- 更新与删除 ----

// Update 白名单字段更新：先加载（租户过滤）再校验流转，最后按字段集写
// 回；status 变更承载归档/恢复（active↔archived，§5.4 复用 patch 动词）
func (s *applicationService) Update(ctx context.Context, member *iammodel.User, id uint, req *model.UpdateApplicationRequest) (*model.ApplicationDetail, error) {
	// 与路由层 PATCH（verb=patch）同口径的 Service 复核（§9.1：内部调用
	// 路径同样受控，应用级范围授权落地后在此扩展判定范围）。
	// 只认 patch：本域无 PUT 路由，update 动词不会出现在路由判定中，
	// 若此处等效放行会重新造成「capabilities=true / HTTP 被拒」的口径分裂
	perms := s.access.Permissions(ctx, member)
	if !perms["applications:patch"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member %d cannot update application %d", memberID(member), id))
	}

	app, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if isProvisioning(app) {
		return nil, httpx.Wrap(apperrors.ErrProvisioning,
			fmt.Errorf("application %d provisioning (%s)", app.ID, app.ProvisionStatus))
	}

	fields := map[string]interface{}{}
	action := "update"
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || utf8.RuneCountInString(name) > maxNameRunes {
			return nil, httpx.Wrap(apperrors.ErrNameInvalid, fmt.Errorf("invalid name length"))
		}
		fields["name"] = name
	}
	if req.Icon != nil {
		if !applicationIcons.Has(*req.Icon) {
			return nil, httpx.Wrap(apperrors.ErrIconInvalid, fmt.Errorf("unknown icon %q", *req.Icon))
		}
		fields["icon"] = *req.Icon
	}
	if req.Color != nil {
		if !applicationColors.Has(*req.Color) {
			return nil, httpx.Wrap(apperrors.ErrColorInvalid, fmt.Errorf("unknown color %q", *req.Color))
		}
		fields["color"] = *req.Color
	}
	if req.SortOrder != nil {
		fields["sort_order"] = *req.SortOrder
	}
	if req.Status != nil {
		switch *req.Status {
		case model.ApplicationStatusActive, model.ApplicationStatusArchived:
		default:
			return nil, httpx.Wrap(apperrors.ErrStatusInvalid, fmt.Errorf("unknown status %q", *req.Status))
		}
		if *req.Status != app.Status {
			fields["status"] = *req.Status
			if *req.Status == model.ApplicationStatusArchived {
				action = "archive"
			} else {
				action = "restore"
			}
		}
	}
	if len(fields) == 0 {
		// 未携带任何白名单字段：幂等返回当前状态
		return s.detailFor(perms, app), nil
	}

	before := map[string]any{"name": app.Name, "status": app.Status}
	if err := s.repo.UpdateFields(ctx, id, fields); err != nil {
		return nil, err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "application", Action: action, ResourceType: "application",
			ResourceID: strconv.FormatUint(uint64(app.ID), 10),
			Before:     before,
			After:      fields,
		})
	}

	updated, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.detailFor(perms, updated), nil
}

// Delete 软删：仅写 deleted_at（§7.1 状态列不设 deleted，避免双轨失真）；
// pending/running 中的应用不可删除（实例化任务仍持有它）
func (s *applicationService) Delete(ctx context.Context, member *iammodel.User, id uint) error {
	// 与路由层 DELETE（verb=delete）同口径的 Service 复核
	if perms := s.access.Permissions(ctx, member); !perms["applications:delete"] {
		return httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member %d cannot delete application %d", memberID(member), id))
	}

	app, err := s.load(ctx, id)
	if err != nil {
		return err
	}
	if isProvisioning(app) {
		return httpx.Wrap(apperrors.ErrProvisioning,
			fmt.Errorf("application %d provisioning (%s)", app.ID, app.ProvisionStatus))
	}

	if err := s.repo.SoftDelete(ctx, app); err != nil {
		return err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "application", Action: "delete", ResourceType: "application",
			ResourceID: strconv.FormatUint(uint64(app.ID), 10),
			After:      map[string]any{"name": app.Name, "source": app.SourceType},
		})
	}
	return nil
}

// ---- 出网视图与派生 ----

// detailFor 组装出网视图。capabilities 严格等于「有效权限集（与鉴权
// 中间件同源：显式角色/分组 + authenticated 系统组）∩ 应用状态」的派生
// 结果（§9.2），不落库瞬态布尔——owner 不额外放大能力，否则会与路由层
// RBAC 拒绝结果冲突；owner 专属范围授权随应用级 scope（AccessEvaluator
// 扩展实现）落地后再叠加
func (s *applicationService) detailFor(perms map[string]bool, app *model.Application) *model.ApplicationDetail {
	// 编辑前提：active 且实例化已收敛（ready/failed 可改，失败应用可修后再试）
	editableState := app.Status == model.ApplicationStatusActive && !isProvisioning(app)
	deletableState := !isProvisioning(app)

	return &model.ApplicationDetail{
		ID:              app.ID,
		Code:            app.Code,
		Name:            app.Name,
		Icon:            app.Icon,
		Color:           app.Color,
		Source:          model.ApplicationSource{Type: app.SourceType, Channel: channelForSourceType(app.SourceType)},
		Status:          app.Status,
		ProvisionStatus: app.ProvisionStatus,
		OwnerMemberID:   app.OwnerMemberID,
		CreatorMemberID: app.CreatorMemberID,
		SortOrder:       app.SortOrder,
		CreatedAt:       app.CreatedAt,
		UpdatedAt:       app.UpdatedAt,
		Capabilities: model.ApplicationCapabilities{
			View: perms["applications:get"] || perms["applications:list"],
			// edit 只认 patch（与路由 PATCH 动词一致，本域无 PUT 路由）
			Edit:   perms["applications:patch"] && editableState,
			Delete: perms["applications:delete"] && deletableState,
		},
	}
}

// channelForSourceType 来源默认渠道（M2-A 仅 blank/self；M2-B 模板安装
// 改为从安装记录取真实渠道）
func channelForSourceType(sourceType string) string {
	if sourceType == model.SourceTypeTemplate {
		return "template_center"
	}
	return model.InstallChannelSelf
}

// load 按 ID 加载应用：ctx 携带租户时 GORM Callback 已过滤，跨租户 ID
// 表现为 NotFound。仅「确实无此行」包装 APP_NOT_FOUND（404）；连接中断、
// 超时等基础设施错误原样上抛，由 Controller 统一出口脱敏为 500——
// 避免数据库故障被误报成「应用不存在」（如更新成功后重载失败的场景）
func (s *applicationService) load(ctx context.Context, id uint) (*model.Application, error) {
	app, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(apperrors.ErrNotFound, err)
		}
		return nil, err
	}
	return app, nil
}

// isProvisioning 实例化进行中（§7.1：不可编辑/删除/进设计器）
func isProvisioning(app *model.Application) bool {
	return app.ProvisionStatus == model.ProvisionStatusPending ||
		app.ProvisionStatus == model.ProvisionStatusRunning
}

// normalizeAppearance 创建期外观校验：name 去空格后 1–128 字符（按字符
// 数计，CJK 安全）；icon/color 空值取默认、非空必须命中枚举
func normalizeAppearance(name, icon, color string) (string, string, string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxNameRunes {
		return "", "", "", httpx.Wrap(apperrors.ErrNameInvalid, fmt.Errorf("invalid name length"))
	}
	if icon == "" {
		icon = defaultIcon
	}
	if !applicationIcons.Has(icon) {
		return "", "", "", httpx.Wrap(apperrors.ErrIconInvalid, fmt.Errorf("unknown icon %q", icon))
	}
	if color == "" {
		color = defaultColor
	}
	if !applicationColors.Has(color) {
		return "", "", "", httpx.Wrap(apperrors.ErrColorInvalid, fmt.Errorf("unknown color %q", color))
	}
	return name, icon, color, nil
}

// memberID/memberTenant 安全取值（member 可能为 nil）
func memberID(member *iammodel.User) uint {
	if member == nil {
		return 0
	}
	return member.ID
}

func memberTenant(member *iammodel.User) uint {
	if member == nil {
		return 0
	}
	return member.TenantID
}

// ---- 游标编解码（§8.3） ----

// listCursor 游标内部载荷：列表序 (sort_order, id) 定位值。对客户端不
// 透明（base64url），只允许原样回传
type listCursor struct {
	SortOrder int64 `json:"s"`
	ID        uint  `json:"i"`
}

func encodeListCursor(sortOrder int64, id uint) string {
	b, _ := json.Marshal(listCursor{SortOrder: sortOrder, ID: id})
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeListCursor(cursor string) (sortOrder int64, id uint, has bool, err error) {
	if cursor == "" {
		return 0, 0, false, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, 0, false, fmt.Errorf("invalid list cursor")
	}
	var c listCursor
	if err := json.Unmarshal(raw, &c); err != nil || c.ID == 0 {
		return 0, 0, false, fmt.Errorf("invalid list cursor")
	}
	return c.SortOrder, c.ID, true, nil
}
