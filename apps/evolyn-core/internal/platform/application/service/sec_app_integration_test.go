package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"

	"evolyn/internal/contextx"
	"evolyn/internal/infrastructure"
	apperrors "evolyn/internal/platform/application"
	"evolyn/internal/platform/application/model"
	applicationrepository "evolyn/internal/platform/application/repository"
	auditrepository "evolyn/internal/platform/audit/repository"
	auditservice "evolyn/internal/platform/audit/service"
	iammodel "evolyn/internal/platform/iam/model"
	iamrepository "evolyn/internal/platform/iam/repository"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantrepository "evolyn/internal/platform/tenant/repository"
	tenantservice "evolyn/internal/platform/tenant/service"
	"evolyn/internal/testsupport"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ---- SEC-APP-* / QUOTA-APP-* 真实 PostgreSQL 集成测试矩阵 ----
//
// 验证链路覆盖：Tenant Context → ApplicationService（CheckAndReserve 事务）
// → ApplicationRepository → GORM Callback → PostgreSQL 行锁/部分唯一索引。
// 未配置 TEST_PG_DSN 时自动 Skip（离线套件保持全绿）。

// appEnv 双租户应用域测试环境：alpha/beta 各含 owner 成员与基线角色
type appEnv struct {
	db         *gorm.DB
	iamRepo    *iamrepository.Repositories
	appRepo    applicationrepository.ApplicationRepository
	appSvc     ApplicationService
	quotaSvc   tenantservice.QuotaService
	tenantRepo tenantrepository.TenantRepository

	alpha, beta             *tenantmodel.Tenant
	alphaMember, betaMember *iammodel.User
}

func newAppEnv(t *testing.T) *appEnv {
	t.Helper()

	db := testsupport.NewPostgres(t)
	rdb := testsupport.DisabledRedis()
	iamRepo := iamrepository.NewRepositories(db, rdb)
	tenantRepo := tenantrepository.NewRepository(db, rdb)
	auditSvc := auditservice.NewService(auditrepository.NewRepository(db))
	appRepo := applicationrepository.NewRepository(db)
	quotaSvc := tenantservice.NewQuotaService(tenantRepo, tenantRepo, iamRepo.User(), appRepo)
	txManager := infrastructure.NewTxManager(db)
	tenantSvc := tenantservice.NewTenantService(txManager, tenantRepo, iamRepo, quotaSvc, auditSvc, 0)

	env := &appEnv{
		db:         db,
		iamRepo:    iamRepo,
		appRepo:    appRepo,
		appSvc:     NewApplicationService(txManager, appRepo, quotaSvc, auditSvc, NewRBACAccessEvaluator(iamRepo.User(), iamRepo.Group())),
		quotaSvc:   quotaSvc,
		tenantRepo: tenantRepo,
	}

	env.alpha = env.openTenant(t, tenantSvc, "app-sec-alpha", "app-owner-a")
	env.beta = env.openTenant(t, tenantSvc, "app-sec-beta", "app-owner-b")
	env.alphaMember = env.ownerMember(t, iamRepo, env.alpha, "app-owner-a")
	env.betaMember = env.ownerMember(t, iamRepo, env.beta, "app-owner-b")
	return env
}

func (e *appEnv) openTenant(t *testing.T, svc tenantservice.TenantService, code, ownerName string) *tenantmodel.Tenant {
	t.Helper()
	tenant, err := svc.Open(context.Background(), &tenantservice.OpenTenantRequest{
		Code: code, Name: code, Plan: tenantmodel.PlanFree,
		OwnerName: ownerName, OwnerPassword: "secret123",
	})
	if err != nil {
		t.Fatalf("open tenant %s: %v", code, err)
	}
	return tenant
}

func (e *appEnv) ownerMember(t *testing.T, iamRepo *iamrepository.Repositories, tenant *tenantmodel.Tenant, ownerName string) *iammodel.User {
	t.Helper()
	account, err := iamRepo.Account().GetByName(context.Background(), ownerName)
	if err != nil {
		t.Fatalf("load owner account %s: %v", ownerName, err)
	}
	member, err := iamRepo.User().GetByAccountAndTenant(context.Background(), account.ID, tenant.ID)
	if err != nil {
		t.Fatalf("load owner member of tenant %d: %v", tenant.ID, err)
	}
	return member
}

func appCtx(tenantID uint) context.Context {
	return contextx.NewTenantContext(context.Background(), tenantID)
}

// setAppsLimit 覆盖租户 apps 配额上限（直写 quotas JSONB，绕过服务层）
func (e *appEnv) setAppsLimit(t *testing.T, tenantID uint, limit int64) {
	t.Helper()
	assert.NoError(t, e.db.Exec(
		`UPDATE pf_tenants SET quotas = quotas || ('{"apps": ' || ? || '}')::jsonb WHERE id = ?`,
		strconv.FormatInt(limit, 10), tenantID,
	).Error)
}

func (e *appEnv) rawCount(t *testing.T, sql string, args ...interface{}) int64 {
	t.Helper()
	var count int64
	assert.NoError(t, e.db.Raw(sql, args...).Scan(&count).Error)
	return count
}

func blankReq(name string) *model.CreateBlankRequest {
	return &model.CreateBlankRequest{Name: name}
}

// createPlainMember 落一个无显式角色的普通成员（账号 + 成员，归属指定
// 租户），其权限仅来自 authenticated 系统组基线
func (e *appEnv) createPlainMember(t *testing.T, tenant *tenantmodel.Tenant, name string) *iammodel.User {
	t.Helper()
	account := &iammodel.Account{Name: name, Password: "secret123"}
	account, err := e.iamRepo.Account().Create(context.Background(), account)
	if err != nil {
		t.Fatalf("create plain account %s: %v", name, err)
	}
	member := &iammodel.User{AccountId: account.ID, Nickname: "普通成员"}
	member.TenantID = tenant.ID
	if _, err = e.iamRepo.User().Create(appCtx(tenant.ID), member); err != nil {
		t.Fatalf("create plain member %s: %v", name, err)
	}
	return member
}

// ---- SEC-APP-001~003：租户隔离 ----

// SEC-APP-001：租户 A 上下文读取/更新/删除租户 B 的应用 → 统一 NotFound，
// 且 beta 数据未受影响
func TestSECAPP001CrossTenantAccess(t *testing.T) {
	env := newAppEnv(t)

	created, err := env.appSvc.CreateBlank(appCtx(env.beta.ID), env.betaMember, blankReq("beta 应用"))
	assert.NoError(t, err)

	_, err = env.appSvc.Get(appCtx(env.alpha.ID), env.alphaMember, created.ID)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound))

	_, err = env.appSvc.Update(appCtx(env.alpha.ID), env.alphaMember, created.ID, &model.UpdateApplicationRequest{})
	assert.True(t, errors.Is(err, apperrors.ErrNotFound))

	err = env.appSvc.Delete(appCtx(env.alpha.ID), env.alphaMember, created.ID)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound))

	// 按 code 定位同受租户过滤：alpha 拿 beta 应用的 code 查 → NotFound；
	// beta 本人按 code 查 → 命中同一行
	_, err = env.appSvc.GetByCode(appCtx(env.alpha.ID), env.alphaMember, created.Code)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound))

	byCode, err := env.appSvc.GetByCode(appCtx(env.beta.ID), env.betaMember, created.Code)
	assert.NoError(t, err)
	assert.Equal(t, created.ID, byCode.ID)

	// beta 行仍在库（软删未发生）
	assert.EqualValues(t, 1, env.rawCount(t, "SELECT COUNT(*) FROM tn_applications WHERE id = ?", created.ID))
}

// SEC-APP-002：以租户 A 上下文绑定租户 B 的成员为 owner/creator → 拒绝，
// 不产生任何应用/安装记录
func TestSECAPP002CrossTenantMemberBinding(t *testing.T) {
	env := newAppEnv(t)

	_, err := env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.betaMember, blankReq("伪造应用"))
	assert.True(t, errors.Is(err, apperrors.ErrMemberInvalid))

	assert.Zero(t, env.rawCount(t, "SELECT COUNT(*) FROM tn_applications WHERE tenant_id = ?", env.alpha.ID))
	assert.Zero(t, env.rawCount(t, "SELECT COUNT(*) FROM tn_application_installations WHERE tenant_id = ?", env.alpha.ID))
}

// SEC-APP-003：租户过滤的列表只返回本租户应用
func TestSECAPP003ListTenantScoped(t *testing.T) {
	env := newAppEnv(t)

	_, err := env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.alphaMember, blankReq("alpha 应用"))
	assert.NoError(t, err)
	_, err = env.appSvc.CreateBlank(appCtx(env.beta.ID), env.betaMember, blankReq("beta 应用"))
	assert.NoError(t, err)

	alphaPage, err := env.appSvc.List(appCtx(env.alpha.ID), env.alphaMember, model.ListApplicationsQuery{})
	assert.NoError(t, err)
	assert.Len(t, alphaPage.Items, 1)
	assert.Equal(t, "alpha 应用", alphaPage.Items[0].Name)

	// 软删后从列表消失且不占配额（见 QUOTA-APP-003）
}

// ---- QUOTA-APP-001~003：配额与并发 ----

// SEC-APP-005：跨租户角色不泄漏——beta 租户管理员（tenant-admin 通配
// 规则）以 alpha 上下文调用时权限集为空：List/Get 一律 403，
// 不因 evaluator 合并 alpha 的 authenticated 系统组而获得只读能力
func TestSECAPP005CrossTenantRoleLeak(t *testing.T) {
	env := newAppEnv(t)

	created, err := env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.alphaMember, blankReq("alpha 应用"))
	assert.NoError(t, err)

	_, err = env.appSvc.List(appCtx(env.alpha.ID), env.betaMember, model.ListApplicationsQuery{})
	assert.True(t, errors.Is(err, apperrors.ErrForbidden))

	_, err = env.appSvc.Get(appCtx(env.alpha.ID), env.betaMember, created.ID)
	assert.True(t, errors.Is(err, apperrors.ErrForbidden))

	// beta 管理员在本租户内一切正常（对照：权限集非空）
	selfPage, err := env.appSvc.List(appCtx(env.beta.ID), env.betaMember, model.ListApplicationsQuery{})
	assert.NoError(t, err)
	assert.Empty(t, selfPage.Items)
}

// SEC-APP-006：伪造/陈旧成员对象不生效——evaluator 按 member.ID 在当前
// 租户重载真实角色，调用方在 User 对象上伪造的 TenantID/applications:*
// 角色不影响判定；凭空 ID 一律空集
func TestSECAPP006ForgedMemberObject(t *testing.T) {
	env := newAppEnv(t)

	created, err := env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.alphaMember, blankReq("alpha 应用"))
	assert.NoError(t, err)
	plain := env.createPlainMember(t, env.alpha, "app-forged-victim")

	t.Run("真实 ID + 伪造通配角色：写路径仍按真实角色拒绝", func(t *testing.T) {
		forged := &iammodel.User{Nickname: "伪造管理员"}
		forged.ID = plain.ID
		forged.TenantID = env.alpha.ID
		forged.Roles = []iammodel.Role{{Rules: iammodel.Rules{
			{Resource: "*", Operation: iammodel.AllOperation},
		}}}

		_, err = env.appSvc.Update(appCtx(env.alpha.ID), forged, created.ID, &model.UpdateApplicationRequest{})
		assert.True(t, errors.Is(err, apperrors.ErrForbidden), "伪造角色不得经 evaluator 生效")

		err = env.appSvc.Delete(appCtx(env.alpha.ID), forged, created.ID)
		assert.True(t, errors.Is(err, apperrors.ErrForbidden))

		// capabilities 同口径：重载后普通成员仍只有 view
		detail, err := env.appSvc.Get(appCtx(env.alpha.ID), forged, created.ID)
		assert.NoError(t, err)
		assert.True(t, detail.Capabilities.View)
		assert.False(t, detail.Capabilities.Edit)
		assert.False(t, detail.Capabilities.Delete)
	})

	t.Run("凭空 ID + 匹配租户 + 通配角色：权限集为空", func(t *testing.T) {
		ghost := &iammodel.User{Nickname: "凭空成员"}
		ghost.ID = 99999
		ghost.TenantID = env.alpha.ID
		ghost.Roles = []iammodel.Role{{Rules: iammodel.Rules{
			{Resource: "*", Operation: iammodel.AllOperation},
		}}}

		_, err = env.appSvc.List(appCtx(env.alpha.ID), ghost, model.ListApplicationsQuery{})
		assert.True(t, errors.Is(err, apperrors.ErrForbidden))
	})

	// 数据未受伪造对象影响
	assert.EqualValues(t, 1, env.rawCount(t, "SELECT COUNT(*) FROM tn_applications WHERE id = ?", created.ID))
}

// QUOTA-APP-001：apps 配额上限拦截第 N+1 个应用，安装记录成对出现
func TestQUOTAAPP001LimitEnforced(t *testing.T) {
	env := newAppEnv(t)
	env.setAppsLimit(t, env.alpha.ID, 2)

	_, err := env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.alphaMember, blankReq("应用一"))
	assert.NoError(t, err)
	_, err = env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.alphaMember, blankReq("应用二"))
	assert.NoError(t, err)

	_, err = env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.alphaMember, blankReq("应用三"))
	assert.Error(t, err)
	// 稳定错误码经 BizError 出网（ADR-008），前端按 errCode 分支
	assert.True(t, errors.Is(err, tenantservice.ErrQuotaExceeded), "got: %v", err)

	// 拒绝后不留半写状态：应用数=2、安装记录=2
	assert.EqualValues(t, 2, env.rawCount(t, "SELECT COUNT(*) FROM tn_applications WHERE tenant_id = ?", env.alpha.ID))
	assert.EqualValues(t, 2, env.rawCount(t, "SELECT COUNT(*) FROM tn_application_installations WHERE tenant_id = ?", env.alpha.ID))
}

// QUOTA-APP-002：并发创建不超过上限——CheckAndReserve 的事务内行锁串行化
// 同租户并发路径（10 并发抢 5 个名额，恰有 5 个成功）
func TestQUOTAAPP002ConcurrentCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	env := newAppEnv(t)
	const limit = 5
	env.setAppsLimit(t, env.alpha.ID, limit)

	const total = 10
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		success  int
		rejected int
	)
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.alphaMember,
				blankReq(fmt.Sprintf("并发应用-%d", i)))
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				rejected++
			} else {
				success++
			}
		}(i)
	}
	wg.Wait()

	assert.Equal(t, limit, success, "成功数应恰为上限")
	assert.Equal(t, total-limit, rejected)
	assert.EqualValues(t, limit, env.rawCount(t, "SELECT COUNT(*) FROM tn_applications WHERE tenant_id = ?", env.alpha.ID))
}

// SEC-APP-004：普通成员（仅 authenticated 系统组基线，无显式角色）的
// capabilities 与鉴权中间件同源——AccessEvaluator 合并系统组规则后
// view=true（基线 applications:view）、edit/delete=false
func TestSECAPP004PlainMemberCapabilities(t *testing.T) {
	env := newAppEnv(t)

	created, err := env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.alphaMember, blankReq("alpha 应用"))
	assert.NoError(t, err)

	plain := env.createPlainMember(t, env.alpha, "app-plain-member")

	detail, err := env.appSvc.Get(appCtx(env.alpha.ID), plain, created.ID)
	assert.NoError(t, err)
	assert.True(t, detail.Capabilities.View, "authenticated 基线 applications:view 应并入权限集")
	assert.False(t, detail.Capabilities.Edit)
	assert.False(t, detail.Capabilities.Delete)

	// 写路径复核：普通成员删除被 403 拒绝
	err = env.appSvc.Delete(appCtx(env.alpha.ID), plain, created.ID)
	assert.True(t, errors.Is(err, apperrors.ErrForbidden))
	assert.EqualValues(t, 1, env.rawCount(t, "SELECT COUNT(*) FROM tn_applications WHERE id = ?", created.ID))
}

// QUOTA-APP-003：软删释放配额名额（计数口径排除 deleted_at 非空行）
func TestQUOTAAPP003SoftDeleteReleasesQuota(t *testing.T) {
	env := newAppEnv(t)
	env.setAppsLimit(t, env.alpha.ID, 1)

	first, err := env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.alphaMember, blankReq("独占名额"))
	assert.NoError(t, err)

	_, err = env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.alphaMember, blankReq("第二个"))
	assert.Error(t, err)

	assert.NoError(t, env.appSvc.Delete(appCtx(env.alpha.ID), env.alphaMember, first.ID))

	// 软删后名额释放，可再次创建；安装记录为追加写保留
	second, err := env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.alphaMember, blankReq("第二个"))
	assert.NoError(t, err)
	assert.EqualValues(t, 1, env.rawCount(t, "SELECT COUNT(*) FROM tn_applications WHERE tenant_id = ? AND deleted_at IS NULL", env.alpha.ID))
	assert.EqualValues(t, 2, env.rawCount(t, "SELECT COUNT(*) FROM tn_application_installations WHERE tenant_id = ?", env.alpha.ID))
	_ = second
}
