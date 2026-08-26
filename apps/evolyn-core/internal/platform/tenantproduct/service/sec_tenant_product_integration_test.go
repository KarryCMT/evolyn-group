package service

import (
	"context"
	"testing"

	"evolyn/internal/contextx"
	"evolyn/internal/infrastructure"
	auditrepository "evolyn/internal/platform/audit/repository"
	auditservice "evolyn/internal/platform/audit/service"
	iammodel "evolyn/internal/platform/iam/model"
	iamrepository "evolyn/internal/platform/iam/repository"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantrepository "evolyn/internal/platform/tenant/repository"
	tenantservice "evolyn/internal/platform/tenant/service"
	apperrors "evolyn/internal/platform/tenantproduct"
	tpmodel "evolyn/internal/platform/tenantproduct/model"
	tprepository "evolyn/internal/platform/tenantproduct/repository"
	"evolyn/internal/testsupport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- 产品中心跨租户/范围语义集成测试（真实 PostgreSQL，TEST_PG_DSN 未设置
// 时自动跳过）。链路覆盖：testsupport 全量迁移（含 000033）→ 租户开通事务
// （UseProductSeeder 种子）→ Service → Repository 显式租户条件 → SQL 约束。

func TestTenantProductIntegration(t *testing.T) {
	db := testsupport.NewPostgres(t)
	rdb := testsupport.DisabledRedis()
	iamRepo := iamrepository.NewRepositories(db, rdb)
	tenantRepo := tenantrepository.NewRepository(db, rdb)
	auditSvc := auditservice.NewService(auditrepository.NewRepository(db))
	quotaSvc := tenantservice.NewQuotaService(tenantRepo, tenantRepo, iamRepo.User(), nil)
	txManager := infrastructure.NewTxManager(db)

	repo := tprepository.NewRepository(db)
	// edition 窄端口不注入：集成焦点在租户/范围语义，版本投影留空
	productSvc := NewTenantProductService(txManager, repo, nil, auditSvc)

	tenantSvc := tenantservice.NewTenantService(txManager, tenantRepo, iamRepo, quotaSvc, auditSvc, 0)
	if injector, ok := tenantSvc.(tenantservice.ProductConfigSeederInjector); ok {
		injector.UseProductSeeder(productSvc)
	}

	ctx := context.Background()
	openTenant := func(code, ownerName string) *tenantmodel.Tenant {
		t.Helper()
		tenant, err := tenantSvc.Open(ctx, &tenantservice.OpenTenantRequest{
			Code: code, Name: code, Plan: tenantmodel.PlanFree,
			OwnerName: ownerName, OwnerPassword: "secret123",
		})
		require.NoError(t, err, "open tenant %s", code)
		return tenant
	}
	ownerMember := func(tenant *tenantmodel.Tenant, ownerName string) *iammodel.User {
		t.Helper()
		account, err := iamRepo.Account().GetByName(ctx, ownerName)
		require.NoError(t, err)
		member, err := iamRepo.User().GetByAccountAndTenant(ctx, account.ID, tenant.ID)
		require.NoError(t, err)
		return member
	}

	// ---- 1. 开通事务种子：默认启用、范围 all（文档 11 用例 1）----
	alpha := openTenant("sec-tp-alpha", "owner-alpha-tp")
	beta := openTenant("sec-tp-beta", "owner-beta-tp")
	alphaMember := ownerMember(alpha, "owner-alpha-tp")
	betaMember := ownerMember(beta, "owner-beta-tp")

	view, err := productSvc.List(ctx, alpha.ID)
	require.NoError(t, err)
	require.Len(t, view.Items, 1)
	card := view.Items[0]
	assert.Equal(t, "lingyanyun", card.Code)
	assert.True(t, card.Enabled)
	assert.Equal(t, tpmodel.ScopeModeAll, card.AccessScope.Mode)
	assert.Equal(t, int64(1), card.Revision)
	assert.Equal(t, int64(1), card.AccessScope.EligibleMemberCount)

	// ---- 2. 跨租户边界：租户 A 不能把租户 B 的部门/成员写入范围（用例 2）----
	betaCtx := contextx.NewTenantContext(ctx, beta.ID)
	betaDepts, err := iamRepo.Department().List(betaCtx)
	require.NoError(t, err)
	require.NotEmpty(t, betaDepts)

	_, err = productSvc.UpdateAccessScope(ctx, alpha.ID, "lingyanyun", &tpmodel.UpdateAccessScopeRequest{
		Mode: tpmodel.ScopeModePartial, DepartmentIds: []uint{betaDepts[0].ID}, Revision: 1,
	})
	assert.ErrorIs(t, err, apperrors.ErrDepartmentInvalid)

	_, err = productSvc.UpdateAccessScope(ctx, alpha.ID, "lingyanyun", &tpmodel.UpdateAccessScopeRequest{
		Mode: tpmodel.ScopeModePartial, MemberIds: []uint{betaMember.ID}, Revision: 1,
	})
	assert.ErrorIs(t, err, apperrors.ErrMemberInvalid)

	// ---- 3. partial 范围：选中部门含子部门命中（用例 7）----
	alphaCtx := contextx.NewTenantContext(ctx, alpha.ID)
	alphaDepts, err := iamRepo.Department().List(alphaCtx)
	require.NoError(t, err)
	require.NotEmpty(t, alphaDepts)
	root := alphaDepts[0]

	// 在根部门下建子部门，并把 owner 成员迁入子部门
	child := &iammodel.Department{Name: "子部门", ParentId: &root.ID}
	child.TenantID = alpha.ID
	_, err = iamRepo.Department().Create(alphaCtx, child)
	require.NoError(t, err)
	require.NoError(t, iamRepo.Department().SetMemberDepartments(alphaCtx, alphaMember, []uint{child.ID}))

	scopeCard, err := productSvc.UpdateAccessScope(ctx, alpha.ID, "lingyanyun", &tpmodel.UpdateAccessScopeRequest{
		Mode: tpmodel.ScopeModePartial, DepartmentIds: []uint{root.ID}, Revision: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, tpmodel.ScopeModePartial, scopeCard.AccessScope.Mode)
	assert.Equal(t, []uint{root.ID}, scopeCard.AccessScope.DepartmentIds)
	assert.Equal(t, int64(1), scopeCard.AccessScope.EligibleMemberCount)

	evaluator := NewTenantProductAccessEvaluator(repo)
	ok, err := evaluator.CanAccess(alphaCtx, "lingyanyun", alphaMember.ID)
	require.NoError(t, err)
	assert.True(t, ok, "子部门成员应命中选中部门的范围")

	// beta 租户 all 范围：有效成员放行（同码产品、各自租户配置互不影响）
	ok, err = evaluator.CanAccess(betaCtx, "lingyanyun", betaMember.ID)
	require.NoError(t, err)
	assert.True(t, ok)

	// ---- 4. 停用后命中范围成员也被拒（用例 6）----
	disabledCard, err := productSvc.SetEnabled(ctx, alpha.ID, "lingyanyun", &tpmodel.UpdateEnabledRequest{Enabled: false, Revision: 2})
	require.NoError(t, err)
	assert.False(t, disabledCard.Enabled)
	ok, err = evaluator.CanAccess(alphaCtx, "lingyanyun", alphaMember.ID)
	require.NoError(t, err)
	assert.False(t, ok)

	// ---- 5. 乐观锁：持旧 revision 的并发更新得到 409（用例 5）----
	_, err = productSvc.SetEnabled(ctx, alpha.ID, "lingyanyun", &tpmodel.UpdateEnabledRequest{Enabled: true, Revision: 2})
	assert.ErrorIs(t, err, apperrors.ErrRevisionConflict)
	// 以最新版本号恢复启用
	_, err = productSvc.SetEnabled(ctx, alpha.ID, "lingyanyun", &tpmodel.UpdateEnabledRequest{Enabled: true, Revision: 3})
	require.NoError(t, err)
	ok, err = evaluator.CanAccess(alphaCtx, "lingyanyun", alphaMember.ID)
	require.NoError(t, err)
	assert.True(t, ok)
}
