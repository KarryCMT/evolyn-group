package service

import (
	"context"
	"testing"
	"time"

	kernel "evolyn/internal/model"
	"evolyn/internal/platform/audit/service"
	apperrors "evolyn/internal/platform/productlog"
	"evolyn/internal/platform/productlog/model"
	"evolyn/internal/platform/productlog/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakeRepo 内存仓储：覆盖查询/扫描/任务存取全接口（离线单测用）。
// 过滤行为镜像仓储口径：分类白名单必过，成员/分类/事件码/应用参与过滤
type fakeRepo struct {
	rows  []model.ProductLogRow
	tasks []*model.ExportTask
}

func filterProduct(rows []model.ProductLogRow, flt repository.ProductLogFilter) []model.ProductLogRow {
	categorySet := map[string]bool{}
	for _, c := range flt.Categories {
		categorySet[c] = true
	}
	out := make([]model.ProductLogRow, 0, len(rows))
	for _, r := range rows {
		if !categorySet[r.CategoryCode] {
			continue
		}
		if flt.CategoryCode != "" && r.CategoryCode != flt.CategoryCode {
			continue
		}
		if flt.EventCode != "" && r.EventCode != flt.EventCode {
			continue
		}
		if flt.MemberID != 0 && r.MemberID != flt.MemberID {
			continue
		}
		out = append(out, r)
	}
	return out
}

func (f *fakeRepo) ListProductLogs(_ context.Context, flt repository.ProductLogFilter, offset, limit int) ([]model.ProductLogRow, int64, error) {
	rows := filterProduct(f.rows, flt)
	total := int64(len(rows))
	if offset >= len(rows) {
		return []model.ProductLogRow{}, total, nil
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[offset:end], total, nil
}

func (f *fakeRepo) ScanProductLogs(_ context.Context, flt repository.ProductLogFilter, _ int, fn func([]model.ProductLogRow) error) error {
	return fn(filterProduct(f.rows, flt))
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

// stubMembers 成员目录桩：仅 11 号成员有效；筛选项返回两个成员
type stubMembers struct{}

func (stubMembers) ValidateMember(_ context.Context, _, memberID uint) error {
	if memberID == 11 {
		return nil
	}
	return apperrors.ErrMemberInvalid
}

func (stubMembers) ListMembers(_ context.Context, _ uint) ([]model.MemberOption, error) {
	return []model.MemberOption{
		{MemberID: 11, Name: "张三"},
		{MemberID: 12, Name: "李四"},
	}, nil
}

// stubApps 应用目录桩：仅 21 号应用有效；筛选项返回两个应用
type stubApps struct{}

func (stubApps) ValidateApplication(_ context.Context, _, applicationID uint) error {
	if applicationID == 21 {
		return nil
	}
	return apperrors.ErrApplicationInvalid
}

func (stubApps) ListApplications(_ context.Context, _ uint) ([]model.ApplicationOption, error) {
	return []model.ApplicationOption{
		{ApplicationID: 21, Code: "app_21", Name: "测试应用"},
		{ApplicationID: 22, Code: "app_22", Name: "项目协作"},
	}, nil
}

func newTestService(repo *fakeRepo) ProductLogService {
	return NewProductLogService(repo, stubMembers{}, stubApps{}, nil)
}

func TestListValidatesFilters(t *testing.T) {
	svc := newTestService(&fakeRepo{})

	// 时间范围 / 日期格式 / 成员 / 应用 / 分类 / 事件码逐项校验
	_, err := svc.List(context.Background(), 1, model.ProductLogQuery{StartDate: "2026-08-10", EndDate: "2026-08-01"})
	assert.ErrorIs(t, err, apperrors.ErrTimeRangeInvalid)
	_, err = svc.List(context.Background(), 1, model.ProductLogQuery{StartDate: "2026/08/10"})
	assert.ErrorIs(t, err, apperrors.ErrDateInvalid)
	_, err = svc.List(context.Background(), 1, model.ProductLogQuery{MemberID: 99})
	assert.ErrorIs(t, err, apperrors.ErrMemberInvalid)
	_, err = svc.List(context.Background(), 1, model.ProductLogQuery{ApplicationID: 99})
	assert.ErrorIs(t, err, apperrors.ErrApplicationInvalid)
	// 企业日志分类不可用于产品日志（两目录互斥）
	_, err = svc.List(context.Background(), 1, model.ProductLogQuery{CategoryCode: service.CategoryMemberManagement})
	assert.ErrorIs(t, err, apperrors.ErrCategoryUnknown)
	_, err = svc.List(context.Background(), 1, model.ProductLogQuery{EventCode: "iam.member.update"})
	assert.ErrorIs(t, err, apperrors.ErrEventUnknown)

	_, err = svc.List(context.Background(), 1, model.ProductLogQuery{
		CategoryCode:  service.CategoryProductForm,
		EventCode:     "form.form.create",
		ApplicationID: 21,
		MemberID:      11,
	})
	assert.NoError(t, err)
}

func TestListScopedToProductCategories(t *testing.T) {
	// 企业治理行（成员管理/日志导出分类）不进产品日志结果；应用快照随行出网
	repo := &fakeRepo{rows: []model.ProductLogRow{
		{ID: 1, ActorNameSnapshot: "张三", EventCode: "form.form.delete", CategoryCode: "form",
			TargetNameSnapshot: "采购申请", Summary: "删除表单「采购申请」", ApplicationNameSnapshot: "测试应用", IP: "1.1.1.1"},
		{ID: 2, ActorNameSnapshot: "李四", EventCode: "iam.member.update", CategoryCode: "member_management",
			Summary: "更新成员「王五」"},
		{ID: 3, ActorNameSnapshot: "", DisplayName: "王五（当前昵称）", EventCode: "", CategoryCode: "", Summary: ""},
	}}
	svc := newTestService(repo)

	page, err := svc.List(context.Background(), 1, model.ProductLogQuery{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)

	item := page.Items[0]
	assert.Equal(t, "张三", item.ActorName)
	assert.Equal(t, "表单管理", item.CategoryName)
	assert.Equal(t, "删除表单", item.EventName)
	assert.Equal(t, "测试应用", item.ApplicationName)
	assert.Equal(t, "采购申请", item.TargetName)
	assert.Equal(t, "删除表单「采购申请」", item.Summary)
}

func TestOptionsAggregation(t *testing.T) {
	svc := newTestService(&fakeRepo{})

	options, err := svc.Options(context.Background(), 1)
	require.NoError(t, err)
	require.NotEmpty(t, options.Categories)
	assert.Equal(t, service.CategoryProductApplication, options.Categories[0].Code)
	assert.Equal(t, "应用管理", options.Categories[0].Name)
	require.Len(t, options.Applications, 2)
	assert.Equal(t, "测试应用", options.Applications[0].Name)
}

func TestCreateExportFlow(t *testing.T) {
	repo := &fakeRepo{rows: []model.ProductLogRow{
		{ID: 1, ActorNameSnapshot: "张三", EventCode: "form.form.create", CategoryCode: "form",
			TargetNameSnapshot: "采购申请", Summary: "创建表单「采购申请」", ApplicationNameSnapshot: "测试应用", IP: "1.1.1.1", CreatedAt: time.Now()},
		{ID: 2, ActorNameSnapshot: "李四", EventCode: "workflow.workflow.publish", CategoryCode: "workflow",
			TargetNameSnapshot: "请假流程", Summary: "发布流程「请假流程」", ApplicationNameSnapshot: "项目协作", IP: "2.2.2.2", CreatedAt: time.Now()},
	}}
	svc := newTestService(repo)

	// 无效成员 / 无效应用 / 企业日志事件码直接拒绝
	_, err := svc.CreateExport(context.Background(), 1, model.CreateExportRequest{MemberID: 99})
	assert.ErrorIs(t, err, apperrors.ErrMemberInvalid)
	_, err = svc.CreateExport(context.Background(), 1, model.CreateExportRequest{ApplicationID: 99})
	assert.ErrorIs(t, err, apperrors.ErrApplicationInvalid)
	_, err = svc.CreateExport(context.Background(), 1, model.CreateExportRequest{EventCode: "enterpriselog.export.create"})
	assert.ErrorIs(t, err, apperrors.ErrEventUnknown)

	view, err := svc.CreateExport(context.Background(), 1, model.CreateExportRequest{})
	require.NoError(t, err)
	assert.Equal(t, model.ExportStatusReady, view.Status)
	assert.Equal(t, int64(2), view.Total)
	assert.Contains(t, view.FileName, "产品日志-")

	// 下载内容：BOM + 表头 + 数据行（空应用/空对象渲染「—」占位）
	file, err := svc.ExportFile(context.Background(), 1, view.ID)
	require.NoError(t, err)
	content := string(file.Data)
	assert.Contains(t, content, "\uFEFF操作人,操作时间,日志范围,操作类型,所属应用,操作对象,操作详情,IP")
	assert.Contains(t, content, "创建表单「采购申请」")
	assert.Contains(t, content, "测试应用")
}

func TestExportTaskExpiryAndCrossTenant(t *testing.T) {
	repo := &fakeRepo{}
	svc := newTestService(repo)

	view, err := svc.CreateExport(context.Background(), 7, model.CreateExportRequest{})
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

	csv := buildCSV([]string{"操作人", "操作详情"}, [][]string{{"张三", "更新表单「采购,申请」"}})
	assert.Equal(t, "\uFEFF操作人,操作详情\r\n张三,\"更新表单「采购,申请」\"\r\n", csv)
}
