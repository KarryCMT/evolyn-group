package service

import (
	"context"
	"testing"
	"time"

	kernel "evolyn/internal/model"
	"evolyn/internal/platform/audit/service"
	apperrors "evolyn/internal/platform/enterpriselog"
	"evolyn/internal/platform/enterpriselog/model"
	"evolyn/internal/platform/enterpriselog/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakeRepo 内存仓储：覆盖查询/扫描/任务存取全接口（离线单测用）
type fakeRepo struct {
	loginRows []model.LoginLogRow
	auditRows []model.AuditLogRow
	tasks     []*model.ExportTask
}

func (f *fakeRepo) ListLoginLogs(_ context.Context, flt repository.LoginLogFilter, offset, limit int) ([]model.LoginLogRow, int64, error) {
	rows := filterLogin(f.loginRows, flt)
	total := int64(len(rows))
	if offset >= len(rows) {
		return []model.LoginLogRow{}, total, nil
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[offset:end], total, nil
}

func (f *fakeRepo) ScanLoginLogs(_ context.Context, flt repository.LoginLogFilter, _ int, fn func([]model.LoginLogRow) error) error {
	return fn(filterLogin(f.loginRows, flt))
}

func (f *fakeRepo) ListAuditLogs(_ context.Context, flt repository.AuditLogFilter, offset, limit int) ([]model.AuditLogRow, int64, error) {
	rows := filterAudit(f.auditRows, flt)
	total := int64(len(rows))
	if offset >= len(rows) {
		return []model.AuditLogRow{}, total, nil
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[offset:end], total, nil
}

func (f *fakeRepo) ScanAuditLogs(_ context.Context, flt repository.AuditLogFilter, _ int, fn func([]model.AuditLogRow) error) error {
	return fn(filterAudit(f.auditRows, flt))
}

func filterLogin(rows []model.LoginLogRow, flt repository.LoginLogFilter) []model.LoginLogRow {
	out := make([]model.LoginLogRow, 0, len(rows))
	for _, r := range rows {
		if flt.MemberID != 0 && r.MemberID != flt.MemberID {
			continue
		}
		out = append(out, r)
	}
	return out
}

func filterAudit(rows []model.AuditLogRow, flt repository.AuditLogFilter) []model.AuditLogRow {
	out := make([]model.AuditLogRow, 0, len(rows))
	for _, r := range rows {
		if flt.CategoryCode != "" && r.CategoryCode != flt.CategoryCode {
			continue
		}
		if flt.EventCode != "" && r.EventCode != flt.EventCode {
			continue
		}
		out = append(out, r)
	}
	return out
}

func (f *fakeRepo) CreateExportTask(_ context.Context, task *model.ExportTask) error {
	task.ID = uint(len(f.tasks) + 1)
	f.tasks = append(f.tasks, task)
	return nil
}

func (f *fakeRepo) GetExportTask(_ context.Context, tenantID, id uint) (*model.ExportTask, error) {
	for _, task := range f.tasks {
		if task.ID == id && task.TenantID == tenantID {
			cp := *task
			return &cp, nil
		}
	}
	// 与仓储 NotFound 语义对齐（服务层按 gorm.ErrRecordNotFound 映射业务错误）
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeRepo) UpdateExportTask(_ context.Context, task *model.ExportTask) error {
	for i := range f.tasks {
		if f.tasks[i].ID == task.ID {
			f.tasks[i] = task
		}
	}
	return nil
}

func (f *fakeRepo) Migrate() error { return nil }

// stubMembers 成员目录桩：仅 11 号成员有效
type stubMembers struct{}

func (stubMembers) ValidateMember(_ context.Context, _, memberID uint) error {
	if memberID == 11 {
		return nil
	}
	return apperrors.ErrMemberInvalid
}

func newTestService(repo *fakeRepo) EnterpriseLogService {
	return NewEnterpriseLogService(repo, stubMembers{}, nil)
}

func TestListLoginLogsValidatesTimeRange(t *testing.T) {
	svc := newTestService(&fakeRepo{})

	_, err := svc.ListLoginLogs(context.Background(), 1, model.LoginLogQuery{StartDate: "2026-08-10", EndDate: "2026-08-01"})
	assert.ErrorIs(t, err, apperrors.ErrTimeRangeInvalid)

	_, err = svc.ListLoginLogs(context.Background(), 1, model.LoginLogQuery{StartDate: "2026/08/10"})
	assert.ErrorIs(t, err, apperrors.ErrDateInvalid)

	_, err = svc.ListLoginLogs(context.Background(), 1, model.LoginLogQuery{MemberID: 99})
	assert.ErrorIs(t, err, apperrors.ErrMemberInvalid)
}

func TestListOperationLogsValidatesCategoryAndEvent(t *testing.T) {
	svc := newTestService(&fakeRepo{})

	_, err := svc.ListOperationLogs(context.Background(), 1, model.OperationLogQuery{CategoryCode: "no_such"})
	assert.ErrorIs(t, err, apperrors.ErrCategoryUnknown)

	_, err = svc.ListOperationLogs(context.Background(), 1, model.OperationLogQuery{EventCode: "no.such.event"})
	assert.ErrorIs(t, err, apperrors.ErrEventUnknown)

	// 合法分类 + 合法事件（不同资源）不报错，交集由 SQL 决定
	_, err = svc.ListOperationLogs(context.Background(), 1, model.OperationLogQuery{
		CategoryCode: "member_management",
		EventCode:    "iam.member.update",
	})
	assert.NoError(t, err)
}

func TestListOperationLogsLegacyDegradeDisplay(t *testing.T) {
	// 存量历史行（无事件码/分类/摘要）降级「历史操作记录」；新行走注册表展示
	repo := &fakeRepo{auditRows: []model.AuditLogRow{
		{ID: 1, ActorNameSnapshot: "张三", Summary: "", EventCode: "", CategoryCode: ""},
		{ID: 2, ActorNameSnapshot: "", DisplayName: "李四（当前昵称）", EventCode: "iam.role.create", CategoryCode: "role_permission", Summary: "添加角色「管理员」"},
	}}
	svc := newTestService(repo)

	page, err := svc.ListOperationLogs(context.Background(), 1, model.OperationLogQuery{})
	require.NoError(t, err)
	require.Len(t, page.Items, 2)

	legacy := page.Items[0]
	assert.Equal(t, "张三", legacy.ActorName)
	assert.Equal(t, "历史操作记录", legacy.CategoryName)
	assert.Equal(t, "历史操作记录", legacy.EventName)
	assert.Equal(t, "历史操作记录", legacy.Summary)

	fresh := page.Items[1]
	assert.Equal(t, "李四（当前昵称）", fresh.ActorName) // 快照缺失回查当前昵称
	assert.Equal(t, "角色权限", fresh.CategoryName)
	assert.Equal(t, "添加角色", fresh.EventName)
	assert.Equal(t, "添加角色「管理员」", fresh.Summary)
}

func TestCreateExportFlow(t *testing.T) {
	repo := &fakeRepo{loginRows: []model.LoginLogRow{
		{ID: 1, ActorNameSnapshot: "张三", Client: "web", IP: "1.1.1.1", Location: "广东省 深圳市", CreatedAt: time.Now()},
		{ID: 2, ActorNameSnapshot: "李四", Client: "wap", IP: "2.2.2.2", Location: "未知", CreatedAt: time.Now()},
	}}
	svc := newTestService(repo)

	// 类型非法 / 成员无效 / 超上限
	_, err := svc.CreateExport(context.Background(), 1, model.CreateExportRequest{Kind: "others"})
	assert.ErrorIs(t, err, apperrors.ErrExportKindInvalid)
	_, err = svc.CreateExport(context.Background(), 1, model.CreateExportRequest{Kind: "login", MemberID: 99})
	assert.ErrorIs(t, err, apperrors.ErrMemberInvalid)

	view, err := svc.CreateExport(context.Background(), 1, model.CreateExportRequest{Kind: "login"})
	require.NoError(t, err)
	assert.Equal(t, model.ExportStatusReady, view.Status)
	assert.Equal(t, int64(2), view.Total)
	assert.Contains(t, view.FileName, "企业日志-登录日志-")

	// 下载内容：BOM + 表头 + 数据行（登录平台映射中文文案）
	file, err := svc.ExportFile(context.Background(), 1, view.ID)
	require.NoError(t, err)
	content := string(file.Data)
	assert.Contains(t, content, "\uFEFF登录人,登录时间,登录地,登录平台,IP")
	assert.Contains(t, content, "张三")
	assert.Contains(t, content, "电脑网页版")
	assert.Contains(t, content, "手机网页版")
}

func TestExportTaskExpiryAndCrossTenant(t *testing.T) {
	repo := &fakeRepo{}
	svc := newTestService(repo)

	view, err := svc.CreateExport(context.Background(), 7, model.CreateExportRequest{Kind: "operation"})
	require.NoError(t, err)

	// 跨租户读取：任务归属租户 7，租户 5 不可见
	_, err = svc.GetExport(context.Background(), 5, view.ID)
	assert.ErrorIs(t, err, apperrors.ErrExportNotFound)
	_, err = svc.ExportFile(context.Background(), 5, view.ID)
	assert.ErrorIs(t, err, apperrors.ErrExportNotFound)

	// 过期投影与下载拒绝：直接改仓储内任务的过期时间
	expired := time.Now().Add(-time.Minute)
	repo.tasks[0].ExpiresAt = toKernelTime(expired)
	view2, err := svc.GetExport(context.Background(), 7, view.ID)
	require.NoError(t, err)
	assert.Equal(t, model.ExportStatusExpired, view2.Status)

	_, err = svc.ExportFile(context.Background(), 7, view.ID)
	assert.ErrorIs(t, err, apperrors.ErrExportExpired)
}

func toKernelTime(t time.Time) *kernel.JSONTime {
	jt := kernel.JSONTime(t)
	return &jt
}

func TestCSVEscape(t *testing.T) {
	assert.Equal(t, `"a,b"`, csvEscape("a,b"))
	assert.Equal(t, `"a""b"`, csvEscape(`a"b`))
	assert.Equal(t, "\"a\nb\"", csvEscape("a\nb"))
	assert.Equal(t, `普通字段`, csvEscape("普通字段"))

	csv := buildCSV([]string{"操作人", "操作详情"}, [][]string{{"张三", "更新成员「王,五」"}})
	assert.Equal(t, "\uFEFF操作人,操作详情\r\n张三,\"更新成员「王,五」\"\r\n", csv)
}

func TestListCategoriesFromCatalog(t *testing.T) {
	svc := newTestService(&fakeRepo{})
	categories := svc.ListCategories()
	require.NotEmpty(t, categories)
	assert.Equal(t, service.CategoryMemberManagement, categories[0].Code)
	assert.Equal(t, "成员管理", categories[0].Name)
}
