package service

import (
	"context"
	"errors"
	"testing"

	"evolyn/internal/contextx"
	"evolyn/internal/infrastructure"
	applicationmodel "evolyn/internal/platform/application/model"
	applicationrepository "evolyn/internal/platform/application/repository"
	auditrepository "evolyn/internal/platform/audit/repository"
	auditservice "evolyn/internal/platform/audit/service"
	iammodel "evolyn/internal/platform/iam/model"
	iamrepository "evolyn/internal/platform/iam/repository"
	apperrors "evolyn/internal/platform/productlog"
	"evolyn/internal/platform/productlog/model"
	productlogrepository "evolyn/internal/platform/productlog/repository"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantrepository "evolyn/internal/platform/tenant/repository"
	tenantservice "evolyn/internal/platform/tenant/service"
	"evolyn/internal/testsupport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---- 产品日志跨租户隔离/目录互斥/应用维度/导出集成测试（真实 PostgreSQL，
// TEST_PG_DSN 未设置时自动跳过）。链路：testsupport 全量迁移（含 000064）→
// 双租户 + 成员 + 应用 + 产品/企业分类审计写入 → Service 查询/导出。

// integrationMemberDirectory 成员目录窄端口的测试适配（与 server.go 装配同语义）
type integrationMemberDirectory struct {
	users iamrepository.UserRepository
}

func (d integrationMemberDirectory) ValidateMember(ctx context.Context, tenantID, memberID uint) error {
	_, err := d.users.GetUserByID(contextx.NewTenantContext(ctx, tenantID), memberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrMemberInvalid
		}
		return err
	}
	return nil
}

func (d integrationMemberDirectory) ListMembers(ctx context.Context, tenantID uint) ([]model.MemberOption, error) {
	users, _, err := d.users.ListPage(
		contextx.NewTenantContext(ctx, tenantID),
		iammodel.MemberListQuery{Page: 1, PageSize: 100},
	)
	if err != nil {
		return nil, err
	}
	options := make([]model.MemberOption, 0, len(users))
	for _, u := range users {
		options = append(options, model.MemberOption{MemberID: u.ID, Name: u.Nickname})
	}
	return options, nil
}

// integrationApplicationDirectory 应用目录窄端口的测试适配（与 server.go 同语义）
type integrationApplicationDirectory struct {
	applications applicationrepository.ApplicationRepository
}

func (d integrationApplicationDirectory) ValidateApplication(ctx context.Context, tenantID, applicationID uint) error {
	_, err := d.applications.GetByID(contextx.NewTenantContext(ctx, tenantID), applicationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrApplicationInvalid
		}
		return err
	}
	return nil
}

func (d integrationApplicationDirectory) ListApplications(ctx context.Context, tenantID uint) ([]model.ApplicationOption, error) {
	apps, _, err := d.applications.List(
		contextx.NewTenantContext(ctx, tenantID),
		applicationrepository.ListParams{Limit: 100},
	)
	if err != nil {
		return nil, err
	}
	options := make([]model.ApplicationOption, 0, len(apps))
	for _, app := range apps {
		options = append(options, model.ApplicationOption{ApplicationID: app.ID, Code: app.Code, Name: app.Name})
	}
	return options, nil
}

func TestProductLogIntegration(t *testing.T) {
	db := testsupport.NewPostgres(t)
	rdb := testsupport.DisabledRedis()
	iamRepo := iamrepository.NewRepositories(db, rdb)
	tenantRepo := tenantrepository.NewRepository(db, rdb)
	auditRepo := auditrepository.NewRepository(db)
	auditSvc := auditservice.NewService(auditRepo)
	txManager := infrastructure.NewTxManager(db)
	applicationRepo := applicationrepository.NewRepository(db)

	repo := productlogrepository.NewRepository(db)
	svc := NewProductLogService(
		repo,
		integrationMemberDirectory{users: iamRepo.User()},
		integrationApplicationDirectory{applications: applicationRepo},
		auditSvc,
	)

	quotaSvc := tenantservice.NewQuotaService(tenantRepo, tenantRepo, iamRepo.User(), nil)
	tenantSvc := tenantservice.NewTenantService(txManager, tenantRepo, iamRepo, quotaSvc, auditSvc, 0)

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
		member, err := iamRepo.User().GetByAccountAndTenant(contextx.NewTenantContext(ctx, tenant.ID), account.ID, tenant.ID)
		require.NoError(t, err)
		return member
	}

	// ---- 双租户 + 各自 owner 成员 + alpha 一个应用 ----
	alpha := openTenant("sec-plog-alpha", "owner-plog-alpha")
	beta := openTenant("sec-plog-beta", "owner-plog-beta")
	alphaMember := ownerMember(alpha, "owner-plog-alpha")
	betaMember := ownerMember(beta, "owner-plog-beta")

	alphaCtx := contextx.NewTenantContext(ctx, alpha.ID)
	alphaApp, err := applicationRepo.Create(alphaCtx, &applicationmodel.Application{
		Code: "app_plog_alpha", Name: "甲测试应用", SourceType: "blank",
	})
	require.NoError(t, err)

	// ---- 产品分类审计写入（alpha：表单删除带应用快照；beta：表单创建无快照）+
	// 企业分类行（alpha：成员更新）验证目录互斥 ----
	auditSvc.Record(alphaCtx, auditservice.Entry{
		Module: "form", Action: "delete", ResourceType: "form", ResourceID: "f_alpha_1",
		MemberID: alphaMember.ID, TargetName: "采购申请",
		ApplicationID: alphaApp.ID, ApplicationCode: alphaApp.Code, ApplicationName: alphaApp.Name,
	})
	auditSvc.Record(alphaCtx, auditservice.Entry{
		Module: "iam", Action: "update", ResourceType: "member", ResourceID: "1",
		MemberID: alphaMember.ID,
		Before:   map[string]string{"nickname": "旧名"},
		After:    map[string]string{"nickname": "新名"},
	})
	auditSvc.Record(contextx.NewTenantContext(ctx, beta.ID), auditservice.Entry{
		Module: "form", Action: "create", ResourceType: "form", ResourceID: "f_beta_1",
		MemberID: betaMember.ID, TargetName: "乙表单",
	})

	// ---- 1. 目录互斥：产品日志只含产品分类行，企业治理行不进结果 ----
	page, err := svc.List(ctx, alpha.ID, model.ProductLogQuery{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1, "成员更新（企业分类）不应出现在产品日志")
	item := page.Items[0]
	assert.Equal(t, "删除表单", item.EventName)
	assert.Equal(t, "表单管理", item.CategoryName)
	assert.Equal(t, "甲测试应用", item.ApplicationName, "应用维度快照应随行出网")
	assert.Equal(t, "删除表单「采购申请」", item.Summary)
	assert.Equal(t, "采购申请", item.TargetName)

	// beta 隔离：beta 只见自己的表单创建行（无应用快照）
	betaPage, err := svc.List(ctx, beta.ID, model.ProductLogQuery{})
	require.NoError(t, err)
	require.Len(t, betaPage.Items, 1)
	assert.Equal(t, "创建表单", betaPage.Items[0].EventName)
	assert.Empty(t, betaPage.Items[0].ApplicationName)

	// ---- 2. 筛选：关键词/事件码/成员；跨租户成员与应用拒绝 ----
	byKeyword, err := svc.List(ctx, alpha.ID, model.ProductLogQuery{Keyword: "采购"})
	require.NoError(t, err)
	require.Len(t, byKeyword.Items, 1)
	noneKeyword, err := svc.List(ctx, alpha.ID, model.ProductLogQuery{Keyword: "不存在关键词"})
	require.NoError(t, err)
	assert.Empty(t, noneKeyword.Items)

	_, err = svc.List(ctx, alpha.ID, model.ProductLogQuery{MemberID: betaMember.ID})
	assert.ErrorIs(t, err, apperrors.ErrMemberInvalid)
	_, err = svc.List(ctx, alpha.ID, model.ProductLogQuery{ApplicationID: 99999})
	assert.ErrorIs(t, err, apperrors.ErrApplicationInvalid)
	_, err = svc.List(ctx, alpha.ID, model.ProductLogQuery{CategoryCode: "member_management"})
	assert.ErrorIs(t, err, apperrors.ErrCategoryUnknown)

	// ---- 3. 筛选项：产品目录 + 成员 + 应用（仅本租户应用） ----
	options, err := svc.Options(ctx, alpha.ID)
	require.NoError(t, err)
	require.Len(t, options.Categories, 6)
	require.Len(t, options.Applications, 1)
	assert.Equal(t, "甲测试应用", options.Applications[0].Name)
	assert.NotEmpty(t, options.Members)

	// ---- 4. 导出全链路：创建（同步就绪）→ 跨租户不可见 → 下载内容 ----
	exportView, err := svc.CreateExport(alphaCtx, alpha.ID, model.CreateExportRequest{})
	require.NoError(t, err)
	assert.Equal(t, model.ExportStatusReady, exportView.Status)
	assert.Equal(t, int64(1), exportView.Total)
	assert.Contains(t, exportView.FileName, "产品日志-")

	file, err := svc.ExportFile(ctx, alpha.ID, exportView.ID)
	require.NoError(t, err)
	assert.Contains(t, string(file.Data), "删除表单「采购申请」")
	assert.Contains(t, string(file.Data), "甲测试应用")

	_, err = svc.GetExport(ctx, beta.ID, exportView.ID)
	assert.ErrorIs(t, err, apperrors.ErrExportNotFound)

	// 导出行为落企业治理类审计（日志导出分类，不进产品日志目录）
	afterExport, err := svc.List(ctx, alpha.ID, model.ProductLogQuery{})
	require.NoError(t, err)
	assert.Len(t, afterExport.Items, 1, "导出行为审计（企业分类）不应回流产品日志")
}
