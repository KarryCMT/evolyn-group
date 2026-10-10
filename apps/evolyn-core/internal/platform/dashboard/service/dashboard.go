package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"evolyn/internal/contextx"
	enginedashboard "evolyn/internal/engine/dashboard"
	kernel "evolyn/internal/model"
	auditservice "evolyn/internal/platform/audit/service"
	dashboarderrors "evolyn/internal/platform/dashboard"
	"evolyn/internal/platform/dashboard/model"
	"evolyn/internal/platform/dashboard/repository"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantservice "evolyn/internal/platform/tenant/service"

	"gorm.io/gorm"
)

const (
	maxNameRunes        = 128
	maxAppearanceRunes  = 32
	appStatusActive     = "active"
	dashboardCopySuffix = "（副本）"
)

type dashboardService struct {
	tx           TxManager
	repo         repository.DashboardRepository
	quota        tenantservice.QuotaService
	audit        auditservice.Recorder
	access       AccessEvaluator
	apps         AppDirectory
	menu         MenuMaintenance
	forms        FormDataCatalog
	queries      DashboardQueryExecutor
	queryTimeout time.Duration
	querySlots   chan struct{}
}

func NewDashboardService(tx TxManager, repo repository.DashboardRepository, quota tenantservice.QuotaService, audit auditservice.Recorder, access AccessEvaluator, apps AppDirectory, menu MenuMaintenance) DashboardService {
	return &dashboardService{tx: tx, repo: repo, quota: quota, audit: audit, access: access, apps: apps, menu: menu, queryTimeout: 10 * time.Second, querySlots: make(chan struct{}, 8)}
}

func (s *dashboardService) UseQueryRuntimeConfig(config QueryRuntimeConfig) {
	if config.Timeout > 0 {
		s.queryTimeout = config.Timeout
	}
	if config.MaxConcurrent > 0 {
		s.querySlots = make(chan struct{}, config.MaxConcurrent)
	}
}

// UseFormDataCatalog 在装配期注入权限感知表单数据目录。未注入时数据源端点
// 明确返回不可用，不允许绕过 form 域权限读取仓储。
func (s *dashboardService) UseFormDataCatalog(catalog FormDataCatalog) { s.forms = catalog }

func (s *dashboardService) UseDashboardQueryExecutor(executor DashboardQueryExecutor) {
	s.queries = executor
}

func (s *dashboardService) Precreate(ctx context.Context, member *iammodel.User, appCode string, req *model.PrecreateRequest) (*model.Detail, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID || !s.access.Permissions(ctx, member)["dashboards:create"] {
		return nil, httpx.Wrap(dashboarderrors.ErrForbidden, fmt.Errorf("member cannot create dashboard"))
	}
	if req == nil {
		return nil, httpx.Wrap(dashboarderrors.ErrRequestInvalid, errors.New("request is nil"))
	}
	requestID := strings.TrimSpace(req.RequestID)
	name := strings.TrimSpace(req.Name)
	parentMenuCode := strings.TrimSpace(req.ParentMenuCode)
	appCode = strings.TrimSpace(appCode)
	if requestID == "" || len(requestID) > 64 || appCode == "" {
		return nil, httpx.Wrap(dashboarderrors.ErrRequestInvalid, fmt.Errorf("invalid request id or app code"))
	}
	if name == "" || utf8.RuneCountInString(name) > maxNameRunes {
		return nil, httpx.Wrap(dashboarderrors.ErrNameInvalid, fmt.Errorf("invalid name"))
	}
	payloadHash := createPayloadHash(appCode, name, parentMenuCode)

	var created *model.Dashboard
	var app AppView
	replayed := false
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		binding := &model.CreateBinding{TenantID: tenantID, MemberID: member.ID, RequestID: requestID, RequestHash: payloadHash}
		claimed, err := s.repo.ClaimCreateBinding(tctx, binding)
		if err != nil {
			return err
		}
		if !claimed {
			replayed = true
			existing, err := s.repo.GetCreateBinding(tctx, tenantID, member.ID, requestID)
			if err != nil {
				return err
			}
			if existing.RequestHash != payloadHash || existing.DashboardID == nil {
				return httpx.Wrap(dashboarderrors.ErrIdempotencyConflict, fmt.Errorf("request id payload mismatch"))
			}
			created, err = s.repo.GetByID(tctx, *existing.DashboardID)
			if err != nil {
				return err
			}
			var notFound bool
			app, notFound, err = s.apps.AppByID(tctx, created.AppID)
			if err != nil {
				return err
			}
			if notFound {
				return httpx.Wrap(dashboarderrors.ErrNotFound, fmt.Errorf("idempotent dashboard app missing"))
			}
			return nil
		}

		var notFound bool
		app, notFound, err = s.apps.AppByCode(tctx, appCode)
		if err != nil {
			return err
		}
		if notFound || app.Status != appStatusActive {
			return httpx.Wrap(dashboarderrors.ErrAppInvalid, fmt.Errorf("app %s unavailable", appCode))
		}
		code, err := newDashboardCode()
		if err != nil {
			return err
		}
		return s.quota.CheckAndReserve(tctx, tenantID, tenantmodel.QuotaDashboards, func(qctx context.Context) error {
			created, err = s.repo.Create(qctx, &model.Dashboard{
				AppID: app.ID, Code: code, Name: name,
				ProtocolVersion: enginedashboard.CurrentVersion,
				DraftContent:    model.JSONContent(enginedashboard.EmptyContent()), DraftRevision: 1,
				CreatorMemberID: member.ID,
				TenantBaseModel: kernel.TenantBaseModel{TenantID: tenantID},
			})
			if err != nil {
				return err
			}
			if s.menu != nil {
				if err := s.menu.AttachDashboardNode(qctx, app.ID, created.ID, name, "", "", parentMenuCode); err != nil {
					return err
				}
			}
			return s.repo.CompleteCreateBinding(qctx, binding.ID, created.ID)
		})
	}); err != nil {
		return nil, err
	}

	if s.audit != nil && !replayed {
		s.audit.Record(ctx, auditservice.Entry{Module: "dashboard", Action: "create", ResourceType: "dashboard", ResourceID: created.Code, TargetName: created.Name, AppID: app.ID, AppCode: app.Code, AppName: app.Name, After: map[string]any{"name": created.Name, "appCode": app.Code}})
	}
	return s.detail(ctx, created, app)
}

func (s *dashboardService) Get(ctx context.Context, member *iammodel.User, code string) (*model.Detail, error) {
	if !s.access.Permissions(ctx, member)["dashboards:get"] {
		return nil, httpx.Wrap(dashboarderrors.ErrForbidden, fmt.Errorf("member cannot read dashboard"))
	}
	dashboard, err := s.load(ctx, code)
	if err != nil {
		return nil, err
	}
	app, notFound, err := s.apps.AppByID(ctx, dashboard.AppID)
	if err != nil {
		return nil, err
	}
	if notFound {
		return nil, httpx.Wrap(dashboarderrors.ErrNotFound, fmt.Errorf("dashboard app missing"))
	}
	return s.detail(ctx, dashboard, app)
}

// GetRuntime 只向拥有运行查看动作的成员返回渲染所需文档，不复用管理态
// dashboards:get 权限。发布链路开放前以当前已保存草稿作为稳定运行源。
func (s *dashboardService) GetRuntime(ctx context.Context, member *iammodel.User, code string) (*model.RuntimeBootstrap, error) {
	if !s.access.Permissions(ctx, member)["dashboard-actions:view"] {
		return nil, httpx.Wrap(dashboarderrors.ErrForbidden, fmt.Errorf("member cannot view dashboard runtime"))
	}
	dashboard, err := s.load(ctx, code)
	if err != nil {
		return nil, err
	}
	normalized := enginedashboard.Normalize(dashboard.DraftContent)
	if len(normalized.Issues) > 0 {
		return nil, httpx.Wrap(dashboarderrors.ErrSchemaInvalid.WithData(map[string]any{"issues": normalized.Issues}), fmt.Errorf("saved dashboard draft invalid"))
	}
	return &model.RuntimeBootstrap{
		Code: dashboard.Code, Name: dashboard.Name, ProtocolVersion: dashboard.ProtocolVersion,
		Version: dashboard.DraftRevision, Document: model.JSONContent(normalized.Content),
	}, nil
}

func (s *dashboardService) Update(ctx context.Context, member *iammodel.User, code string, req *model.UpdateRequest) (*model.Detail, error) {
	if !s.access.Permissions(ctx, member)["dashboards:patch"] {
		return nil, httpx.Wrap(dashboarderrors.ErrForbidden, fmt.Errorf("member cannot update dashboard"))
	}
	if req == nil {
		return nil, httpx.Wrap(dashboarderrors.ErrRequestInvalid, errors.New("request is nil"))
	}
	dashboard, err := s.load(ctx, code)
	if err != nil {
		return nil, err
	}
	fields := map[string]any{}
	before, after := map[string]any{}, map[string]any{}
	var name, icon, color *string
	if req.Name != nil {
		value := strings.TrimSpace(*req.Name)
		if value == "" || utf8.RuneCountInString(value) > maxNameRunes {
			return nil, httpx.Wrap(dashboarderrors.ErrNameInvalid, fmt.Errorf("invalid name"))
		}
		name = &value
		fields["name"] = value
		before["name"], after["name"] = dashboard.Name, value
	}
	for key, input := range map[string]*string{"icon": req.Icon, "color": req.Color} {
		if input == nil {
			continue
		}
		value := strings.TrimSpace(*input)
		if utf8.RuneCountInString(value) > maxAppearanceRunes {
			return nil, httpx.Wrap(dashboarderrors.ErrAppearanceInvalid, fmt.Errorf("invalid %s", key))
		}
		fields[key] = value
		if key == "icon" {
			icon = &value
			before[key] = dashboard.Icon
		} else {
			color = &value
			before[key] = dashboard.Color
		}
		after[key] = value
	}
	if len(fields) > 0 || req.ParentMenuCode != nil {
		if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
			if err := s.repo.UpdateFields(tctx, dashboard.ID, fields); err != nil {
				return err
			}
			if s.menu != nil {
				return s.menu.SyncDashboardNode(tctx, dashboard.AppID, dashboard.ID, name, icon, color, req.ParentMenuCode)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	updated, err := s.load(ctx, code)
	if err != nil {
		return nil, err
	}
	app, notFound, err := s.apps.AppByID(ctx, updated.AppID)
	if err != nil {
		return nil, err
	}
	if notFound {
		return nil, httpx.Wrap(dashboarderrors.ErrNotFound, fmt.Errorf("dashboard app missing"))
	}
	if s.audit != nil && (len(after) > 0 || req.ParentMenuCode != nil) {
		if req.ParentMenuCode != nil {
			after["parentMenuCode"] = strings.TrimSpace(*req.ParentMenuCode)
		}
		s.audit.Record(ctx, auditservice.Entry{Module: "dashboard", Action: "update", ResourceType: "dashboard", ResourceID: updated.Code, TargetName: updated.Name, AppID: app.ID, AppCode: app.Code, AppName: app.Name, Before: before, After: after})
	}
	return s.detail(ctx, updated, app)
}

// Copy 在同一应用内复制仪表盘草稿与展示信息。发布快照不复制，新资产从
// draftRevision=1 开始，并保留在源菜单节点所在分组。
func (s *dashboardService) Copy(ctx context.Context, member *iammodel.User, code string) (*model.Detail, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(dashboarderrors.ErrForbidden, fmt.Errorf("member not in tenant %d", tenantID))
	}
	perms := s.access.Permissions(ctx, member)
	if !perms["dashboards:create"] || !perms["dashboard-actions:copy"] {
		return nil, httpx.Wrap(dashboarderrors.ErrForbidden, fmt.Errorf("member cannot copy dashboard"))
	}
	source, err := s.load(ctx, code)
	if err != nil {
		return nil, err
	}
	app, notFound, err := s.apps.AppByID(ctx, source.AppID)
	if err != nil {
		return nil, err
	}
	if notFound || app.Status != appStatusActive {
		return nil, httpx.Wrap(dashboarderrors.ErrAppInvalid, fmt.Errorf("dashboard app unavailable"))
	}
	newCode, err := newDashboardCode()
	if err != nil {
		return nil, err
	}
	name := dashboardCopyName(source.Name)
	var created *model.Dashboard
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		return s.quota.CheckAndReserve(tctx, tenantID, tenantmodel.QuotaDashboards, func(qctx context.Context) error {
			var createErr error
			created, createErr = s.repo.Create(qctx, &model.Dashboard{
				AppID: source.AppID, Code: newCode, Name: name, Icon: source.Icon, Color: source.Color,
				ProtocolVersion: source.ProtocolVersion,
				DraftContent:    append(model.JSONContent(nil), source.DraftContent...), DraftRevision: 1,
				CreatorMemberID: member.ID,
				TenantBaseModel: kernel.TenantBaseModel{TenantID: tenantID},
			})
			if createErr != nil {
				return createErr
			}
			if s.menu != nil {
				return s.menu.AttachDashboardCopyNode(qctx, source.AppID, source.ID, created.ID, created.Name, created.Icon, created.Color)
			}
			return nil
		})
	}); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "dashboard", Action: "copy", ResourceType: "dashboard", ResourceID: created.Code,
			TargetName: created.Name, AppID: app.ID, AppCode: app.Code, AppName: app.Name,
			After: map[string]any{"sourceCode": source.Code, "appCode": app.Code},
		})
	}
	return s.detail(ctx, created, app)
}

func (s *dashboardService) SaveDraft(ctx context.Context, member *iammodel.User, code string, req *model.SaveDraftRequest) (*model.SaveDraftResult, error) {
	perms := s.access.Permissions(ctx, member)
	if !perms["dashboards:update"] || !perms["dashboard-actions:design"] {
		return nil, httpx.Wrap(dashboarderrors.ErrForbidden, fmt.Errorf("member cannot design dashboard"))
	}
	if req == nil {
		return nil, httpx.Wrap(dashboarderrors.ErrRequestInvalid, errors.New("request is nil"))
	}
	dashboard, err := s.load(ctx, code)
	if err != nil {
		return nil, err
	}
	result := enginedashboard.Normalize(req.Document)
	if req.ProtocolVersion != enginedashboard.CurrentVersion || len(result.Issues) > 0 {
		issues := result.Issues
		if req.ProtocolVersion != enginedashboard.CurrentVersion && len(issues) == 0 {
			issues = []enginedashboard.Issue{{Path: "protocolVersion", Code: "DASHBOARD_VERSION_UNSUPPORTED", Message: "不支持的仪表盘协议版本"}}
		}
		return nil, httpx.Wrap(dashboarderrors.ErrSchemaInvalid.WithData(map[string]any{"issues": issues}), fmt.Errorf("invalid dashboard draft"))
	}
	if dashboard.DraftRevision != req.ExpectedRevision {
		return nil, httpx.Wrap(dashboarderrors.ErrDraftConflict, fmt.Errorf("revision stale"))
	}
	updated, err := s.repo.UpdateDraft(ctx, dashboard.ID, req.ExpectedRevision, req.ProtocolVersion, model.JSONContent(result.Content))
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, httpx.Wrap(dashboarderrors.ErrDraftConflict, fmt.Errorf("revision lost race"))
	}
	return &model.SaveDraftResult{DraftRevision: req.ExpectedRevision + 1, Document: model.JSONContent(result.Content)}, nil
}

func (s *dashboardService) Delete(ctx context.Context, member *iammodel.User, code string) error {
	if !s.access.Permissions(ctx, member)["dashboards:delete"] {
		return httpx.Wrap(dashboarderrors.ErrForbidden, fmt.Errorf("member cannot delete dashboard"))
	}
	dashboard, err := s.load(ctx, code)
	if err != nil {
		return err
	}
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		if err := s.repo.SoftDelete(tctx, dashboard); err != nil {
			return err
		}
		if s.menu != nil {
			return s.menu.DetachDashboardNode(tctx, dashboard.AppID, dashboard.ID)
		}
		return nil
	}); err != nil {
		return err
	}
	app, appNotFound, appErr := s.apps.AppByID(ctx, dashboard.AppID)
	if s.audit != nil && appErr == nil && !appNotFound {
		s.audit.Record(ctx, auditservice.Entry{Module: "dashboard", Action: "delete", ResourceType: "dashboard", ResourceID: dashboard.Code, TargetName: dashboard.Name, AppID: app.ID, AppCode: app.Code, AppName: app.Name, Before: map[string]any{"name": dashboard.Name, "appCode": app.Code}})
	}
	return nil
}

func (s *dashboardService) load(ctx context.Context, code string) (*model.Dashboard, error) {
	dashboard, err := s.repo.GetByCode(ctx, strings.TrimSpace(code))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(dashboarderrors.ErrNotFound, err)
		}
		return nil, err
	}
	return dashboard, nil
}

func (s *dashboardService) detail(ctx context.Context, dashboard *model.Dashboard, app AppView) (*model.Detail, error) {
	summary, err := s.repo.PublishedSummary(ctx, dashboard)
	if err != nil {
		return nil, err
	}
	return &model.Detail{AppID: dashboard.AppID, AppCode: app.Code, Code: dashboard.Code, Name: dashboard.Name, Icon: dashboard.Icon, Color: dashboard.Color, ProtocolVersion: dashboard.ProtocolVersion, DraftRevision: dashboard.DraftRevision, Draft: dashboard.DraftContent, PublishedVersion: dashboard.PublishedVersion, Published: *summary, CreatedAt: dashboard.CreatedAt, UpdatedAt: dashboard.UpdatedAt}, nil
}

func createPayloadHash(appCode, name, parentMenuCode string) string {
	raw, _ := json.Marshal(struct{ AppCode, Name, ParentMenuCode string }{appCode, name, parentMenuCode})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func newDashboardCode() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "dashboard_" + hex.EncodeToString(buf), nil
}

func dashboardCopyName(name string) string {
	base := []rune(strings.TrimSpace(name))
	suffix := []rune(dashboardCopySuffix)
	maxBase := maxNameRunes - len(suffix)
	if len(base) > maxBase {
		base = base[:maxBase]
	}
	return string(base) + dashboardCopySuffix
}
