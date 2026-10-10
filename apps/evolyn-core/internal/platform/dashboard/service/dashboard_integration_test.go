package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"evolyn/internal/contextx"
	enginedashboard "evolyn/internal/engine/dashboard"
	"evolyn/internal/infrastructure"
	apperrors "evolyn/internal/platform/app"
	appmodel "evolyn/internal/platform/app/model"
	apprepository "evolyn/internal/platform/app/repository"
	appservice "evolyn/internal/platform/app/service"
	auditrepository "evolyn/internal/platform/audit/repository"
	auditservice "evolyn/internal/platform/audit/service"
	dashboarderrors "evolyn/internal/platform/dashboard"
	dashboardcontroller "evolyn/internal/platform/dashboard/controller"
	dashboardmodel "evolyn/internal/platform/dashboard/model"
	dashboardrepository "evolyn/internal/platform/dashboard/repository"
	dashboardservice "evolyn/internal/platform/dashboard/service"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	iamrepository "evolyn/internal/platform/iam/repository"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantrepository "evolyn/internal/platform/tenant/repository"
	tenantservice "evolyn/internal/platform/tenant/service"
	"evolyn/internal/testsupport"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 本文件用真实 PostgreSQL 锁定阶段一纵切：DashboardService、GORM 仓储、
// app 菜单窄端口、配额行锁和 HTTP 错误映射。未配置 TEST_PG_DSN 时自动跳过。

type dashboardEnv struct {
	db            *gorm.DB
	iam           *iamrepository.Repositories
	tenantRepo    tenantrepository.TenantRepository
	appRepo       apprepository.AppRepository
	menuRepo      apprepository.MenuRepository
	dashboardRepo dashboardrepository.DashboardRepository
	appSvc        appservice.AppService
	menuSvc       appservice.AppMenuService
	dashboardSvc  dashboardservice.DashboardService

	alpha, beta             *tenantmodel.Tenant
	alphaMember, betaMember *iammodel.User
}

type integrationAppDirectory struct{ apps apprepository.AppRepository }

func (d integrationAppDirectory) AppByID(ctx context.Context, id uint) (dashboardservice.AppView, bool, error) {
	app, err := d.apps.GetByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dashboardservice.AppView{}, true, nil
	}
	if err != nil {
		return dashboardservice.AppView{}, false, err
	}
	return dashboardservice.AppView{ID: app.ID, TenantID: app.TenantID, Code: app.Code, Name: app.Name, Status: app.Status}, false, nil
}

func (d integrationAppDirectory) AppByCode(ctx context.Context, code string) (dashboardservice.AppView, bool, error) {
	app, err := d.apps.GetByCode(ctx, code)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dashboardservice.AppView{}, true, nil
	}
	if err != nil {
		return dashboardservice.AppView{}, false, err
	}
	return dashboardservice.AppView{ID: app.ID, TenantID: app.TenantID, Code: app.Code, Name: app.Name, Status: app.Status}, false, nil
}

func newDashboardEnv(t *testing.T) *dashboardEnv {
	t.Helper()
	db := testsupport.NewPostgres(t)
	rdb := testsupport.DisabledRedis()
	iam := iamrepository.NewRepositories(db, rdb)
	tenantRepo := tenantrepository.NewRepository(db, rdb)
	appRepo := apprepository.NewRepository(db)
	menuRepo := apprepository.NewMenuRepository(db)
	dashboardRepo := dashboardrepository.NewRepository(db)
	auditSvc := auditservice.NewService(auditrepository.NewRepository(db))
	quotaSvc := tenantservice.NewQuotaService(tenantRepo, tenantRepo, iam.User(), appRepo)
	quotaSvc.(tenantservice.QuotaDashboardCounterInjector).UseDashboardCounter(dashboardRepo)
	tx := infrastructure.NewTxManager(db)
	access := appservice.NewRBACAccessEvaluator(iam.User(), iam.Group())
	tenantSvc := tenantservice.NewTenantService(tx, tenantRepo, iam, quotaSvc, auditSvc, 0)
	menuMaintenance := appservice.NewMenuMaintenanceService(menuRepo)

	env := &dashboardEnv{
		db: db, iam: iam, tenantRepo: tenantRepo, appRepo: appRepo, menuRepo: menuRepo,
		dashboardRepo: dashboardRepo,
		appSvc:        appservice.NewAppService(tx, appRepo, quotaSvc, auditSvc, access),
		menuSvc:       appservice.NewMenuService(tx, menuRepo, auditSvc, access),
		dashboardSvc: dashboardservice.NewDashboardService(
			tx, dashboardRepo, quotaSvc, auditSvc, access,
			integrationAppDirectory{apps: appRepo}, menuMaintenance,
		),
	}
	env.alpha = env.openTenant(t, tenantSvc, "dashboard-alpha", "dashboard-owner-a")
	env.beta = env.openTenant(t, tenantSvc, "dashboard-beta", "dashboard-owner-b")
	env.alphaMember = env.owner(t, env.alpha, "dashboard-owner-a")
	env.betaMember = env.owner(t, env.beta, "dashboard-owner-b")
	// 同一测试环境包含多个独立场景，放宽 apps/dashboards 仅用于避免场景间
	// 互相消耗默认配额；并发配额用例会再把 dashboards 精确收窄为 2。
	require.NoError(t, db.Exec(`UPDATE pf_tenants SET quotas = quotas || '{"apps":100,"dashboards":100}'::jsonb WHERE id IN (?, ?)`, env.alpha.ID, env.beta.ID).Error)
	return env
}

func (e *dashboardEnv) openTenant(t *testing.T, svc tenantservice.TenantService, code, owner string) *tenantmodel.Tenant {
	t.Helper()
	tenant, err := svc.Open(context.Background(), &tenantservice.OpenTenantRequest{
		Code: code, Name: code, Plan: tenantmodel.PlanFree, OwnerName: owner, OwnerPassword: "secret123",
	})
	require.NoError(t, err)
	return tenant
}

func (e *dashboardEnv) owner(t *testing.T, tenant *tenantmodel.Tenant, accountName string) *iammodel.User {
	t.Helper()
	account, err := e.iam.Account().GetByName(context.Background(), accountName)
	require.NoError(t, err)
	member, err := e.iam.User().GetByAccountAndTenant(context.Background(), account.ID, tenant.ID)
	require.NoError(t, err)
	return member
}

func (e *dashboardEnv) ctx(tenant *tenantmodel.Tenant, member *iammodel.User) context.Context {
	ctx := contextx.NewTenantContext(context.Background(), tenant.ID)
	return contextx.NewActorContext(ctx, contextx.Actor{AccountID: member.AccountId, MemberID: member.ID})
}

func (e *dashboardEnv) createApp(t *testing.T, tenant *tenantmodel.Tenant, member *iammodel.User, name string) *appmodel.AppDetail {
	t.Helper()
	app, err := e.appSvc.CreateBlank(e.ctx(tenant, member), member, &appmodel.CreateBlankRequest{Name: name})
	require.NoError(t, err)
	return app
}

func (e *dashboardEnv) createPlainMember(t *testing.T, tenant *tenantmodel.Tenant, accountName string) *iammodel.User {
	t.Helper()
	account, err := e.iam.Account().Create(context.Background(), &iammodel.Account{Name: accountName, Password: "secret123"})
	require.NoError(t, err)
	member := &iammodel.User{AccountId: account.ID, Nickname: "普通成员"}
	member.TenantID = tenant.ID
	_, err = e.iam.User().Create(contextx.NewTenantContext(context.Background(), tenant.ID), member)
	require.NoError(t, err)
	return member
}

func (e *dashboardEnv) count(t *testing.T, query string, args ...any) int64 {
	t.Helper()
	var count int64
	require.NoError(t, e.db.Raw(query, args...).Scan(&count).Error)
	return count
}

func precreateRequest(requestID, name string, parent ...string) *dashboardmodel.PrecreateRequest {
	req := &dashboardmodel.PrecreateRequest{RequestID: requestID, Name: name}
	if len(parent) > 0 {
		req.ParentMenuCode = parent[0]
	}
	return req
}

func TestDashboardStage1Integration(t *testing.T) {
	env := newDashboardEnv(t)
	alphaCtx := env.ctx(env.alpha, env.alphaMember)

	t.Run("same tenant create is idempotent and writes one menu revision and audit", func(t *testing.T) {
		app := env.createApp(t, env.alpha, env.alphaMember, "幂等应用")
		first, err := env.dashboardSvc.Precreate(alphaCtx, env.alphaMember, app.Code, precreateRequest("idem-1", "经营总览"))
		require.NoError(t, err)
		second, err := env.dashboardSvc.Precreate(alphaCtx, env.alphaMember, app.Code, precreateRequest("idem-1", "经营总览"))
		require.NoError(t, err)
		assert.Equal(t, first.Code, second.Code)
		assert.EqualValues(t, 1, env.count(t, "SELECT COUNT(*) FROM tn_dashboards WHERE tenant_id = ? AND app_id = ?", env.alpha.ID, app.ID))
		assert.EqualValues(t, 1, env.count(t, "SELECT COUNT(*) FROM tn_app_menu_nodes WHERE tenant_id = ? AND app_id = ? AND menu_type = 'dashboard'", env.alpha.ID, app.ID))
		assert.EqualValues(t, 2, env.count(t, "SELECT menu_revision FROM tn_apps WHERE id = ?", app.ID))
		assert.EqualValues(t, 1, env.count(t, "SELECT COUNT(*) FROM tn_audit_logs WHERE tenant_id = ? AND module = 'dashboard' AND action = 'create' AND resource_id = ? AND member_id = ?", env.alpha.ID, first.Code, env.alphaMember.ID))

		_, err = env.dashboardSvc.Precreate(alphaCtx, env.alphaMember, app.Code, precreateRequest("idem-1", "不同名称"))
		assert.True(t, errors.Is(err, dashboarderrors.ErrIdempotencyConflict))
		assert.EqualValues(t, 1, env.count(t, "SELECT COUNT(*) FROM tn_dashboard_create_bindings WHERE tenant_id = ? AND member_id = ? AND request_id = 'idem-1'", env.alpha.ID, env.alphaMember.ID))
	})

	t.Run("permission and tenant isolation do not leak assets", func(t *testing.T) {
		plain := env.createPlainMember(t, env.alpha, "dashboard-plain")
		alphaApp := env.createApp(t, env.alpha, env.alphaMember, "权限应用")
		_, err := env.dashboardSvc.Precreate(env.ctx(env.alpha, plain), plain, alphaApp.Code, precreateRequest("forbidden-1", "无权创建"))
		assert.True(t, errors.Is(err, dashboarderrors.ErrForbidden))

		betaApp := env.createApp(t, env.beta, env.betaMember, "Beta 应用")
		betaDetail, err := env.dashboardSvc.Precreate(env.ctx(env.beta, env.betaMember), env.betaMember, betaApp.Code, precreateRequest("beta-1", "Beta 仪表盘"))
		require.NoError(t, err)
		_, err = env.dashboardSvc.Get(alphaCtx, env.alphaMember, betaDetail.Code)
		assert.True(t, errors.Is(err, dashboarderrors.ErrNotFound))
	})

	t.Run("invalid or cross app parent rolls back every write", func(t *testing.T) {
		target := env.createApp(t, env.alpha, env.alphaMember, "目标应用")
		other := env.createApp(t, env.alpha, env.alphaMember, "分组来源应用")
		group, err := env.menuSvc.CreateGroup(alphaCtx, env.alphaMember, other.Code, &appmodel.CreateMenuGroupRequest{Name: "其他应用分组", BaseMenuRevision: 1})
		require.NoError(t, err)

		_, err = env.dashboardSvc.Precreate(alphaCtx, env.alphaMember, target.Code, precreateRequest("bad-parent", "不应创建", group.MenuID))
		assert.True(t, errors.Is(err, apperrors.ErrMenuParentInvalid))
		assert.Zero(t, env.count(t, "SELECT COUNT(*) FROM tn_dashboard_create_bindings WHERE tenant_id = ? AND request_id = 'bad-parent'", env.alpha.ID))
		assert.Zero(t, env.count(t, "SELECT COUNT(*) FROM tn_dashboards WHERE tenant_id = ? AND app_id = ?", env.alpha.ID, target.ID))
		assert.Zero(t, env.count(t, "SELECT COUNT(*) FROM tn_app_menu_nodes WHERE tenant_id = ? AND app_id = ? AND menu_type = 'dashboard'", env.alpha.ID, target.ID))
		assert.EqualValues(t, 1, env.count(t, "SELECT menu_revision FROM tn_apps WHERE id = ?", target.ID))
	})

	t.Run("copy preserves draft appearance and source menu group", func(t *testing.T) {
		app := env.createApp(t, env.alpha, env.alphaMember, "复制应用")
		group, err := env.menuSvc.CreateGroup(alphaCtx, env.alphaMember, app.Code, &appmodel.CreateMenuGroupRequest{Name: "经营分析", BaseMenuRevision: 1})
		require.NoError(t, err)
		source, err := env.dashboardSvc.Precreate(alphaCtx, env.alphaMember, app.Code, precreateRequest("copy-source", "经营总览", group.MenuID))
		require.NoError(t, err)
		name, icon, color := source.Name, "bar-chart-box", "purple"
		source, err = env.dashboardSvc.Update(alphaCtx, env.alphaMember, source.Code, &dashboardmodel.UpdateRequest{Name: &name, Icon: &icon, Color: &color})
		require.NoError(t, err)
		source, err = env.dashboardSvc.Get(alphaCtx, env.alphaMember, source.Code)
		require.NoError(t, err)

		copied, err := env.dashboardSvc.Copy(alphaCtx, env.alphaMember, source.Code)
		require.NoError(t, err)
		assert.NotEqual(t, source.Code, copied.Code)
		assert.Equal(t, "经营总览（副本）", copied.Name)
		assert.Equal(t, source.Icon, copied.Icon)
		assert.Equal(t, source.Color, copied.Color)
		assert.JSONEq(t, string(source.Draft), string(copied.Draft))
		assert.EqualValues(t, 1, copied.DraftRevision)
		assert.Zero(t, copied.PublishedVersion)

		type menuPosition struct {
			ParentMenuID *uint
		}
		var sourcePosition, copiedPosition menuPosition
		query := `SELECT n.parent_menu_id FROM tn_app_menu_nodes n JOIN tn_dashboards d ON d.id = n.target_id WHERE n.app_id = ? AND n.menu_type = 'dashboard' AND d.code = ? AND n.deleted_at IS NULL`
		require.NoError(t, env.db.Raw(query, app.ID, source.Code).Scan(&sourcePosition).Error)
		require.NoError(t, env.db.Raw(query, app.ID, copied.Code).Scan(&copiedPosition).Error)
		assert.Equal(t, sourcePosition.ParentMenuID, copiedPosition.ParentMenuID)
		assert.EqualValues(t, 1, env.count(t, "SELECT COUNT(*) FROM tn_audit_logs WHERE tenant_id = ? AND module = 'dashboard' AND action = 'copy' AND resource_id = ?", env.alpha.ID, copied.Code))
	})

	t.Run("failure after asset and menu inserts rolls back binding quota and audit", func(t *testing.T) {
		app := env.createApp(t, env.alpha, env.alphaMember, "中途失败应用")
		// BumpMenuRevision 位于菜单节点 INSERT 之后；把 revision 置为 bigint
		// 上限可稳定触发递增失败，从而验证真实事务对前序写入的回滚。
		require.NoError(t, env.db.Exec("UPDATE tn_apps SET menu_revision = ? WHERE id = ?", int64(^uint64(0)>>1), app.ID).Error)
		_, err := env.dashboardSvc.Precreate(alphaCtx, env.alphaMember, app.Code, precreateRequest("mid-failure", "回滚仪表盘"))
		require.Error(t, err)
		assert.Zero(t, env.count(t, "SELECT COUNT(*) FROM tn_dashboard_create_bindings WHERE tenant_id = ? AND request_id = 'mid-failure'", env.alpha.ID))
		assert.Zero(t, env.count(t, "SELECT COUNT(*) FROM tn_dashboards WHERE tenant_id = ? AND app_id = ?", env.alpha.ID, app.ID))
		assert.Zero(t, env.count(t, "SELECT COUNT(*) FROM tn_app_menu_nodes WHERE tenant_id = ? AND app_id = ? AND menu_type = 'dashboard'", env.alpha.ID, app.ID))
		assert.Zero(t, env.count(t, "SELECT COUNT(*) FROM tn_audit_logs WHERE tenant_id = ? AND module = 'dashboard' AND resource_id <> '' AND after_data->>'name' = '回滚仪表盘'", env.alpha.ID))
	})

	t.Run("combined update and delete each bump menu revision once and clear favorites", func(t *testing.T) {
		app := env.createApp(t, env.alpha, env.alphaMember, "生命周期应用")
		group, err := env.menuSvc.CreateGroup(alphaCtx, env.alphaMember, app.Code, &appmodel.CreateMenuGroupRequest{Name: "运营", BaseMenuRevision: 1})
		require.NoError(t, err)
		detail, err := env.dashboardSvc.Precreate(alphaCtx, env.alphaMember, app.Code, precreateRequest("lifecycle-1", "初始名称"))
		require.NoError(t, err)
		assert.EqualValues(t, 3, env.count(t, "SELECT menu_revision FROM tn_apps WHERE id = ?", app.ID))

		name, icon, color, parent := "更新名称", "chart", "primary", group.MenuID
		_, err = env.dashboardSvc.Update(alphaCtx, env.alphaMember, detail.Code, &dashboardmodel.UpdateRequest{Name: &name, Icon: &icon, Color: &color, ParentMenuCode: &parent})
		require.NoError(t, err)
		assert.EqualValues(t, 4, env.count(t, "SELECT menu_revision FROM tn_apps WHERE id = ?", app.ID))

		var menuID uint
		require.NoError(t, env.db.Raw("SELECT id FROM tn_app_menu_nodes WHERE app_id = ? AND menu_type = 'dashboard' AND deleted_at IS NULL", app.ID).Scan(&menuID).Error)
		require.NoError(t, env.db.Create(&appmodel.MenuFavorite{TenantID: env.alpha.ID, MemberID: env.alphaMember.ID, AppID: app.ID, MenuID: menuID}).Error)
		require.NoError(t, env.dashboardSvc.Delete(alphaCtx, env.alphaMember, detail.Code))
		assert.EqualValues(t, 5, env.count(t, "SELECT menu_revision FROM tn_apps WHERE id = ?", app.ID))
		assert.Zero(t, env.count(t, "SELECT COUNT(*) FROM tn_app_menu_favorites WHERE menu_id = ?", menuID))
		assert.EqualValues(t, 1, env.count(t, "SELECT COUNT(*) FROM tn_audit_logs WHERE tenant_id = ? AND module = 'dashboard' AND action = 'update' AND resource_id = ?", env.alpha.ID, detail.Code))
		assert.EqualValues(t, 1, env.count(t, "SELECT COUNT(*) FROM tn_audit_logs WHERE tenant_id = ? AND module = 'dashboard' AND action = 'delete' AND resource_id = ?", env.alpha.ID, detail.Code))
	})

	t.Run("draft compare and swap maps stale revision to HTTP 409", func(t *testing.T) {
		app := env.createApp(t, env.alpha, env.alphaMember, "草稿应用")
		detail, err := env.dashboardSvc.Precreate(alphaCtx, env.alphaMember, app.Code, precreateRequest("draft-1", "草稿测试"))
		require.NoError(t, err)
		_, err = env.dashboardSvc.SaveDraft(alphaCtx, env.alphaMember, detail.Code, &dashboardmodel.SaveDraftRequest{ExpectedRevision: 1, ProtocolVersion: 1, Document: dashboardmodel.JSONContent(enginedashboard.EmptyContent())})
		require.NoError(t, err)

		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Request = c.Request.WithContext(alphaCtx)
			ginctx.SetUser(c, env.alphaMember)
			c.Next()
		})
		api := router.Group("/api/v1")
		dashboardcontroller.NewDashboardController(env.dashboardSvc).RegisterRoute(api)
		body := bytes.NewBufferString(`{"expectedRevision":1,"protocolVersion":1,"document":{"version":1,"settings":{"desktop":{"columns":12,"rowHeight":80}},"datasets":[],"widgets":[],"filters":[],"interactions":[],"publishScope":{"type":"all"}}}`)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/dashboards/"+detail.Code+"/draft", body)
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		assert.Equal(t, http.StatusConflict, response.Code)
		var envelope httpx.Response
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
		assert.Equal(t, "DASHBOARD_DRAFT_CONFLICT", envelope.ErrCode)
	})
}

func TestDashboardConcurrentQuota(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	env := newDashboardEnv(t)
	ctx := env.ctx(env.alpha, env.alphaMember)
	app := env.createApp(t, env.alpha, env.alphaMember, "并发配额应用")
	require.NoError(t, env.db.Exec(
		`UPDATE pf_tenants SET quotas = quotas || ('{"dashboards": ' || ? || '}')::jsonb WHERE id = ?`,
		strconv.FormatInt(2, 10), env.alpha.ID,
	).Error)

	const total = 8
	var wg sync.WaitGroup
	results := make(chan error, total)
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := env.dashboardSvc.Precreate(ctx, env.alphaMember, app.Code, precreateRequest(fmt.Sprintf("quota-%d", index), fmt.Sprintf("仪表盘-%d", index)))
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)

	success, rejected := 0, 0
	for err := range results {
		if err == nil {
			success++
			continue
		}
		assert.True(t, errors.Is(err, tenantservice.ErrQuotaExceeded), "unexpected error: %v", err)
		rejected++
	}
	assert.Equal(t, 2, success)
	assert.Equal(t, total-2, rejected)
	assert.EqualValues(t, 2, env.count(t, "SELECT COUNT(*) FROM tn_dashboards WHERE tenant_id = ? AND app_id = ? AND deleted_at IS NULL", env.alpha.ID, app.ID))
	assert.EqualValues(t, 2, env.count(t, "SELECT COUNT(*) FROM tn_dashboard_create_bindings WHERE tenant_id = ? AND dashboard_id IS NOT NULL", env.alpha.ID))
}
