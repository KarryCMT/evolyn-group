package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"evolyn/internal/contextx"
	"evolyn/internal/infrastructure"
	kernel "evolyn/internal/model"
	auditmodel "evolyn/internal/platform/audit/model"
	auditrepository "evolyn/internal/platform/audit/repository"
	auditservice "evolyn/internal/platform/audit/service"
	loginlogmodel "evolyn/internal/platform/auth/loginlog/model"
	loginlogrepository "evolyn/internal/platform/auth/loginlog/repository"
	apperrors "evolyn/internal/platform/enterpriselog"
	"evolyn/internal/platform/enterpriselog/model"
	enterpriselogrepository "evolyn/internal/platform/enterpriselog/repository"
	iammodel "evolyn/internal/platform/iam/model"
	iamrepository "evolyn/internal/platform/iam/repository"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantrepository "evolyn/internal/platform/tenant/repository"
	tenantservice "evolyn/internal/platform/tenant/service"
	"evolyn/internal/testsupport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---- 企业日志跨租户隔离/投影/导出集成测试（真实 PostgreSQL，TEST_PG_DSN
// 未设置时自动跳过）。链路：testsupport 全量迁移（含 000036）→ 双租户 +
// 成员 + 登录日志/审计日志写入 → Service 查询/导出 → SQL 约束。

// testMemberDirectory 成员目录窄端口的测试适配（与 server.go 装配同语义：
// 租户上下文包裹后经成员仓储查询，NotFound → ErrMemberInvalid）
type testMemberDirectory struct {
	users iamrepository.UserRepository
}

func (d testMemberDirectory) ValidateMember(ctx context.Context, tenantID, memberID uint) error {
	_, err := d.users.GetUserByID(contextx.NewTenantContext(ctx, tenantID), memberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrMemberInvalid
		}
		return err
	}
	return nil
}

func TestEnterpriseLogIntegration(t *testing.T) {
	db := testsupport.NewPostgres(t)
	rdb := testsupport.DisabledRedis()
	iamRepo := iamrepository.NewRepositories(db, rdb)
	tenantRepo := tenantrepository.NewRepository(db, rdb)
	auditRepo := auditrepository.NewRepository(db)
	auditSvc := auditservice.NewService(auditRepo)
	txManager := infrastructure.NewTxManager(db)

	repo := enterpriselogrepository.NewRepository(db)
	svc := NewEnterpriseLogService(repo, testMemberDirectory{users: iamRepo.User()}, auditSvc)

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

	// ---- 双租户 + 各自 owner 成员 ----
	alpha := openTenant("sec-elog-alpha", "owner-elog-alpha")
	beta := openTenant("sec-elog-beta", "owner-elog-beta")
	alphaMember := ownerMember(alpha, "owner-elog-alpha")
	betaMember := ownerMember(beta, "owner-elog-beta")

	// ---- 登录日志写入：alpha 带显示名快照，beta 依赖 JOIN 兜底 ----
	loginRepo := loginlogrepository.NewRepository(db)
	require.NoError(t, loginRepo.Create(ctx, &loginlogmodel.LoginLog{
		AccountID: alphaMember.AccountId, TenantID: alpha.ID, MemberID: alphaMember.ID,
		Method: "password", Client: "web", IP: "1.1.1.1", Location: "广东省 深圳市",
		ActorNameSnapshot: "甲管理员",
	}))
	require.NoError(t, loginRepo.Create(ctx, &loginlogmodel.LoginLog{
		AccountID: betaMember.AccountId, TenantID: beta.ID, MemberID: betaMember.ID,
		Method: "password", Client: "wap", IP: "2.2.2.2", Location: "未知",
	}))

	// ---- 审计写入：alpha 走服务投影（事件注册表推导），beta 留一条无投影历史行 ----
	alphaCtx := contextx.NewTenantContext(ctx, alpha.ID)
	betaCtx := contextx.NewTenantContext(ctx, beta.ID)
	auditSvc.Record(alphaCtx, auditservice.Entry{
		Module: "iam", Action: "update", ResourceType: "member", ResourceID: "1",
		MemberID: alphaMember.ID,
		Before:   map[string]string{"nickname": "旧名"},
		After:    map[string]string{"nickname": "新名"},
	})
	require.NoError(t, auditRepo.Create(betaCtx, &auditmodel.AuditLog{
		TenantID: beta.ID, MemberID: betaMember.ID,
		Module: "iam", Action: "update", ResourceType: "member",
	}))

	// ---- 1. 跨租户隔离：alpha 只见本租户登录日志/操作日志 ----
	loginPage, err := svc.ListLoginLogs(ctx, alpha.ID, model.LoginLogQuery{})
	require.NoError(t, err)
	require.Len(t, loginPage.Items, 1)
	assert.Equal(t, "甲管理员", loginPage.Items[0].ActorName) // 展示快照优先
	assert.Equal(t, "web", loginPage.Items[0].Client)

	opPage, err := svc.ListOperationLogs(ctx, alpha.ID, model.OperationLogQuery{})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(opPage.Items), 2, "至少含租户开通审计与成员更新审计")
	var memberUpdate *model.OperationLogItem
	for i := range opPage.Items {
		if opPage.Items[i].EventName == "更新成员" {
			memberUpdate = &opPage.Items[i]
		}
	}
	require.NotNil(t, memberUpdate, "应含成员更新审计（事件注册表投影）")
	assert.Equal(t, "更新成员「新名」", memberUpdate.Summary)
	assert.Equal(t, "成员管理", memberUpdate.CategoryName)

	// beta 侧：无快照登录行经 JOIN 兜底显示名；无投影审计行降级展示
	//（租户开通审计同为 beta 流水，历史行按降级文案识别）
	betaLoginPage, err := svc.ListLoginLogs(ctx, beta.ID, model.LoginLogQuery{})
	require.NoError(t, err)
	require.Len(t, betaLoginPage.Items, 1)
	assert.NotEmpty(t, betaLoginPage.Items[0].ActorName, "无快照登录行应回查当前成员昵称")

	betaOpPage, err := svc.ListOperationLogs(ctx, beta.ID, model.OperationLogQuery{})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(betaOpPage.Items), 2)
	var historical *model.OperationLogItem
	for i := range betaOpPage.Items {
		if betaOpPage.Items[i].CategoryName == "历史操作记录" {
			historical = &betaOpPage.Items[i]
		}
	}
	require.NotNil(t, historical, "无投影审计行应降级为历史操作记录")
	assert.Equal(t, "历史操作记录", historical.Summary)

	// ---- 2. 筛选校验：跨租户成员/未知分类/时间倒挂 ----
	_, err = svc.ListLoginLogs(ctx, alpha.ID, model.LoginLogQuery{MemberID: betaMember.ID})
	assert.ErrorIs(t, err, apperrors.ErrMemberInvalid)
	_, err = svc.ListOperationLogs(ctx, alpha.ID, model.OperationLogQuery{CategoryCode: "no_such"})
	assert.ErrorIs(t, err, apperrors.ErrCategoryUnknown)
	_, err = svc.ListLoginLogs(ctx, alpha.ID, model.LoginLogQuery{StartDate: "2026-08-10", EndDate: "2026-08-01"})
	assert.ErrorIs(t, err, apperrors.ErrTimeRangeInvalid)

	// ---- 3. 导出全链路：创建（同步就绪）→ 状态/租户复核 → 下载内容 ----
	exportView, err := svc.CreateExport(alphaCtx, alpha.ID, model.CreateExportRequest{Kind: "login"})
	require.NoError(t, err)
	assert.Equal(t, model.ExportStatusReady, exportView.Status)
	assert.Equal(t, int64(1), exportView.Total)
	assert.Contains(t, exportView.FileName, "企业日志-登录日志-")

	file, err := svc.ExportFile(ctx, alpha.ID, exportView.ID)
	require.NoError(t, err)
	assert.Contains(t, string(file.Data), "甲管理员")
	assert.Contains(t, string(file.Data), "电脑网页版")

	// beta 不可见 alpha 的导出任务
	_, err = svc.GetExport(ctx, beta.ID, exportView.ID)
	assert.ErrorIs(t, err, apperrors.ErrExportNotFound)

	// 导出行为已落操作审计（事件注册表推导 + 显式事件码）
	opAfterExport, err := svc.ListOperationLogs(ctx, alpha.ID, model.OperationLogQuery{CategoryCode: "log_export"})
	require.NoError(t, err)
	require.Len(t, opAfterExport.Items, 1)
	assert.Equal(t, "日志导出", opAfterExport.Items[0].CategoryName)
	assert.Contains(t, opAfterExport.Items[0].Summary, "导出登录日志")

	// ---- 4. 操作日志筛选：事件码命中；日期半开区间 ----
	filtered, err := svc.ListOperationLogs(ctx, alpha.ID, model.OperationLogQuery{EventCode: "iam.member.update"})
	require.NoError(t, err)
	require.Len(t, filtered.Items, 1)

	today := time.Now().In(kernel.CSTLocation()).Format("2006-01-02")
	todayPage, err := svc.ListLoginLogs(ctx, alpha.ID, model.LoginLogQuery{StartDate: today, EndDate: today})
	require.NoError(t, err)
	assert.NotEmpty(t, todayPage.Items, "当天闭区间应命中刚写入的登录行")
	past, err := svc.ListLoginLogs(ctx, alpha.ID, model.LoginLogQuery{StartDate: "2020-01-01", EndDate: "2020-01-02"})
	require.NoError(t, err)
	assert.Empty(t, past.Items, "历史区间不应命中")
}
