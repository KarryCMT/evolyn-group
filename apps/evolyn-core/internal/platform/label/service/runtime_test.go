package service

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"evolyn/internal/contextx"
	iammodel "evolyn/internal/platform/iam/model"
	labelapp "evolyn/internal/platform/label"
	"evolyn/internal/platform/label/model"
	"evolyn/internal/platform/label/repository"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const runtimeSchema = `{
  "schemaVersion":"1.0","name":"用户标签",
  "page":{"width":90,"height":60,"unit":"mm","dpi":300,"background":"#ffffff"},
  "elements":[{"id":"field-1","type":"field","x":6,"y":6,"width":40,"height":8,"rotation":0,"zIndex":1,"visible":true,"locked":false,"value":{"type":"field","fieldId":"name"},"style":{"fontFamily":"sans-serif","fontSize":4,"fontWeight":400,"color":"#111111","lineHeight":1.2,"textAlign":"left","verticalAlign":"top","overflow":"clip"}}],
  "settings":{"snapToGrid":true,"gridSize":1,"showGrid":false,"outputPresets":[{"id":"large","name":"大尺寸","width":150,"height":100,"unit":"mm","dpi":300},{"id":"small","name":"小尺寸","width":60,"height":40,"unit":"mm","dpi":300}]}
}`

type runtimeVersionRepo struct{ version *model.TemplateVersion }

func (runtimeVersionRepo) MaxVersionNo(context.Context, uint) (int, error) { return 0, nil }
func (runtimeVersionRepo) Create(context.Context, *model.TemplateVersion) (*model.TemplateVersion, error) {
	panic("unexpected Create")
}
func (r runtimeVersionRepo) GetByID(context.Context, uint) (*model.TemplateVersion, error) {
	return r.version, nil
}
func (runtimeVersionRepo) Migrate() error { return nil }

type runtimeFormDirectory struct{ form FormView }

func (d runtimeFormDirectory) FormByCode(context.Context, string) (FormView, bool, error) {
	return d.form, false, nil
}
func (d runtimeFormDirectory) PublishedForm(context.Context, uint) (FormView, bool, error) {
	return d.form, false, nil
}

type runtimeRecordResolver struct {
	allowed   bool
	bulkCalls int
	bulkIDs   []uint
	fields    map[string]any
	errors    map[uint]error
}

func (r *runtimeRecordResolver) record() *RecordView {
	fields := r.fields
	if fields == nil {
		fields = map[string]any{"name": "李四"}
	}
	return &RecordView{FormID: 7, Fields: fields, System: map[string]any{}}
}

func (r *runtimeRecordResolver) GetRecord(context.Context, *iammodel.User, uint, uint) (*RecordView, error) {
	return r.record(), nil
}
func (r *runtimeRecordResolver) GetRecordForOperation(_ context.Context, _ *iammodel.User, _, recordID uint, _ string) (*RecordView, error) {
	if err := r.errors[recordID]; err != nil {
		return nil, err
	}
	return r.record(), nil
}
func (r *runtimeRecordResolver) CanUseOperation(context.Context, *iammodel.User, uint, string) (bool, error) {
	return r.allowed, nil
}
func (r *runtimeRecordResolver) GetRecordsForOperation(_ context.Context, _ *iammodel.User, _ uint, ids []uint, _ string) ([]*RecordView, error) {
	r.bulkCalls++
	r.bulkIDs = append([]uint(nil), ids...)
	result := make([]*RecordView, 0, len(ids))
	for range ids {
		result = append(result, r.record())
	}
	return result, nil
}

type runtimeTaskRepo struct {
	task  *model.RenderTask
	items []model.RenderTaskItem
}

func (r *runtimeTaskRepo) Create(_ context.Context, task *model.RenderTask, items []model.RenderTaskItem) error {
	task.ID = 31
	for index := range items {
		items[index].ID = uint(index + 1)
		items[index].TaskID = task.ID
	}
	r.task, r.items = task, append([]model.RenderTaskItem(nil), items...)
	return nil
}
func (r *runtimeTaskRepo) GetByCode(context.Context, string) (*model.RenderTask, error) {
	return r.task, nil
}
func (r *runtimeTaskRepo) ListItems(context.Context, uint) ([]model.RenderTaskItem, error) {
	return r.items, nil
}
func (r *runtimeTaskRepo) Claim(context.Context, string) (bool, error) {
	r.task.Status = model.RenderTaskRunning
	return true, nil
}
func (r *runtimeTaskRepo) ResetPending(context.Context, uint, string, string) error {
	r.task.Status = model.RenderTaskPending
	return nil
}
func (r *runtimeTaskRepo) MarkItemRunning(_ context.Context, itemID uint) error {
	for index := range r.items {
		if r.items[index].ID == itemID {
			r.items[index].Status = model.RenderItemRunning
		}
	}
	return nil
}
func (r *runtimeTaskRepo) MarkItemFinished(_ context.Context, itemID uint, status, errorCode, errorMessage string) error {
	for index := range r.items {
		if r.items[index].ID == itemID {
			r.items[index].Status, r.items[index].ErrorCode, r.items[index].ErrorMessage = status, errorCode, errorMessage
		}
	}
	return nil
}
func (r *runtimeTaskRepo) UpdateProgress(_ context.Context, _ uint, success, failed, progress int) error {
	r.task.SuccessCount, r.task.FailedCount, r.task.Progress = success, failed, progress
	return nil
}
func (r *runtimeTaskRepo) Finish(_ context.Context, _ uint, status, fileCode, errorCode, errorMessage string, success, failed int) error {
	r.task.Status, r.task.FileCode = status, fileCode
	r.task.ErrorCode, r.task.ErrorMessage = errorCode, errorMessage
	r.task.SuccessCount, r.task.FailedCount, r.task.Progress = success, failed, 100
	return nil
}
func (*runtimeTaskRepo) Migrate() error { return nil }

type runtimeQueue struct{ err error }

func (runtimeQueue) Available() bool { return true }
func (q runtimeQueue) Enqueue(context.Context, uint, string) error {
	return q.err
}

type runtimeArtifactStore struct{}

func (runtimeArtifactStore) Available() bool { return true }
func (runtimeArtifactStore) StorePDF(context.Context, *iammodel.User, string, string, []byte) (string, error) {
	return "file_test", nil
}
func (runtimeArtifactStore) Download(context.Context, *iammodel.User, string) (*ArtifactDownload, error) {
	return &ArtifactDownload{Method: "GET", URL: "https://files.lingyanyun.com/test.pdf"}, nil
}

type capturingArtifactStore struct{ content []byte }

func (*capturingArtifactStore) Available() bool { return true }
func (s *capturingArtifactStore) StorePDF(_ context.Context, _ *iammodel.User, _, _ string, content []byte) (string, error) {
	s.content = append([]byte(nil), content...)
	return "file_pdf", nil
}
func (*capturingArtifactStore) Download(context.Context, *iammodel.User, string) (*ArtifactDownload, error) {
	return &ArtifactDownload{Method: "GET", URL: "https://files.lingyanyun.com/result.pdf"}, nil
}

type runtimeMemberDirectory struct{ member *iammodel.User }

func (d runtimeMemberDirectory) MemberByID(context.Context, uint) (*iammodel.User, error) {
	return d.member, nil
}

func newRuntimeService(records *runtimeRecordResolver, tasks repository.RenderTaskRepository) (TemplateService, *iammodel.User, context.Context) {
	versionID := uint(9)
	template := &model.Template{
		ID: 5, Code: "label_user", Name: "用户标签", AppID: 3, FormID: 7, FormCode: "form_users",
		Status: model.StatusPublished, LatestVersionID: &versionID, PublishedVersion: 4,
	}
	template.TenantID = 1
	member := &iammodel.User{ID: 2, Nickname: "测试成员"}
	member.TenantID = 1
	service := NewTemplateService(
		immediateTx{}, &templateRepoStub{template: template},
		runtimeVersionRepo{version: &model.TemplateVersion{ID: versionID, TemplateID: template.ID, VersionNo: 4, SchemaSnapshot: model.SchemaContent(runtimeSchema)}},
		runtimeFormDirectory{form: FormView{ID: 7, AppID: 3, Code: "form_users", Published: true}}, records,
		accessStub{permissions: map[string]bool{iammodel.LabelTemplateResource + ":update": true}}, nil,
	)
	if tasks != nil {
		ConfigureBatch(service, tasks, runtimeQueue{}, runtimeArtifactStore{}, runtimeMemberDirectory{member: member})
	}
	return service, member, contextx.NewTenantContext(context.Background(), 1)
}

func TestRuntimeProfileAndPreviewUsePublishedPreset(t *testing.T) {
	service, member, ctx := newRuntimeService(&runtimeRecordResolver{allowed: true}, nil)
	profile, err := service.RuntimeProfile(ctx, member, "form_users")
	require.NoError(t, err)
	require.True(t, profile.Available)
	require.Equal(t, 1772, profile.OutputPresets[0].PixelWidth)
	require.Equal(t, 1181, profile.OutputPresets[0].PixelHeight)

	result, err := service.RuntimePreview(ctx, member, "form_users", &model.RuntimePreviewRequest{RecordID: "12", OutputPresetID: "small"})
	require.NoError(t, err)
	require.Contains(t, result.MIMEType, "image/svg+xml")
	require.Contains(t, string(result.Content), `width="60mm"`)
	_, err = service.RuntimePreview(ctx, member, "form_users", &model.RuntimePreviewRequest{RecordID: "12", OutputPresetID: "missing"})
	require.ErrorIs(t, err, labelapp.ErrOutputPresetInvalid)
}

func TestBatchRenderPreflightsInOneBulkCallAndFreezesOutput(t *testing.T) {
	records := &runtimeRecordResolver{allowed: true}
	tasks := &runtimeTaskRepo{}
	service, member, ctx := newRuntimeService(records, tasks)
	created, err := service.BatchRender(ctx, member, &model.BatchRenderRequest{
		FormCode: "form_users", RecordIDs: []string{"12", "7"}, OutputPresetID: "large", Format: "pdf",
	})
	require.NoError(t, err)
	require.Equal(t, model.RenderTaskPending, created.Status)
	require.Equal(t, 1, records.bulkCalls)
	require.Equal(t, []uint{12, 7}, records.bulkIDs)
	require.Equal(t, "large", tasks.task.OutputPresetID)
	require.Equal(t, 150.0, tasks.task.OutputWidth)
	require.Equal(t, 100.0, tasks.task.OutputHeight)
	require.Equal(t, 300, tasks.task.OutputDPI)
	require.Equal(t, []uint{12, 7}, []uint{tasks.items[0].RecordID, tasks.items[1].RecordID})
}

func TestRuntimeProfileRejectsMissingBatchPrintOperation(t *testing.T) {
	service, member, ctx := newRuntimeService(&runtimeRecordResolver{allowed: false}, nil)
	_, err := service.RuntimeProfile(ctx, member, "form_users")
	require.ErrorIs(t, err, labelapp.ErrForbidden)
}

func TestRuntimeProfileDoesNotExposeDraftOrMissingTemplate(t *testing.T) {
	member := &iammodel.User{ID: 2}
	member.TenantID = 1
	ctx := contextx.NewTenantContext(context.Background(), 1)
	records := &runtimeRecordResolver{allowed: true}
	service := NewTemplateService(
		immediateTx{}, &templateRepoStub{byFormErr: gorm.ErrRecordNotFound}, runtimeVersionRepo{},
		runtimeFormDirectory{form: FormView{ID: 7, Code: "form_users", Published: true}}, records,
		accessStub{permissions: map[string]bool{}}, nil,
	)
	profile, err := service.RuntimeProfile(ctx, member, "form_users")
	require.NoError(t, err)
	require.False(t, profile.Available)
	require.Empty(t, profile.TemplateCode)
	require.Empty(t, profile.OutputPresets)

	otherTenant := *member
	otherTenant.TenantID = 2
	_, err = service.RuntimeProfile(ctx, &otherTenant, "form_users")
	require.ErrorIs(t, err, labelapp.ErrForbidden)
}

func TestRuntimePreviewRejectsHiddenTemplateField(t *testing.T) {
	service, member, ctx := newRuntimeService(&runtimeRecordResolver{allowed: true, fields: map[string]any{}}, nil)
	_, err := service.RuntimePreview(ctx, member, "form_users", &model.RuntimePreviewRequest{RecordID: "12", OutputPresetID: "large"})
	require.ErrorIs(t, err, labelapp.ErrFieldNoPermission)

	service, member, ctx = newRuntimeService(&runtimeRecordResolver{allowed: true, errors: map[uint]error{
		12: labelapp.ErrRecordNoPermission,
	}}, nil)
	_, err = service.RuntimePreview(ctx, member, "form_users", &model.RuntimePreviewRequest{RecordID: "12", OutputPresetID: "large"})
	require.ErrorIs(t, err, labelapp.ErrRecordNoPermission)
}

func TestBatchRenderRejectsInvalidRecordSets(t *testing.T) {
	tasks := &runtimeTaskRepo{}
	service, member, ctx := newRuntimeService(&runtimeRecordResolver{allowed: true}, tasks)
	_, err := service.BatchRender(ctx, member, &model.BatchRenderRequest{
		FormCode: "form_users", RecordIDs: []string{"7", "7"}, OutputPresetID: "large",
	})
	require.ErrorIs(t, err, labelapp.ErrBatchInvalid)
	recordIDs := make([]string, maxBatchRecords+1)
	for index := range recordIDs {
		recordIDs[index] = "1"
	}
	_, err = service.BatchRender(ctx, member, &model.BatchRenderRequest{
		FormCode: "form_users", RecordIDs: recordIDs, OutputPresetID: "large",
	})
	require.ErrorIs(t, err, labelapp.ErrBatchInvalid)
}

func TestBatchRenderClosesTaskWhenQueueSubmissionFails(t *testing.T) {
	tasks := &runtimeTaskRepo{}
	service, member, ctx := newRuntimeService(&runtimeRecordResolver{allowed: true}, tasks)
	ConfigureBatch(service, tasks, runtimeQueue{err: errors.New("queue unavailable")}, runtimeArtifactStore{}, runtimeMemberDirectory{member: member})
	_, err := service.BatchRender(ctx, member, &model.BatchRenderRequest{
		FormCode: "form_users", RecordIDs: []string{"7"}, OutputPresetID: "large",
	})
	require.ErrorIs(t, err, labelapp.ErrTaskQueueUnavailable)
	require.Equal(t, model.RenderTaskFailed, tasks.task.Status)
	require.Equal(t, labelapp.ErrTaskQueueUnavailable.Code, tasks.task.ErrorCode)
}

func TestRenderTaskOwnershipAndDownloadReadiness(t *testing.T) {
	tasks := &runtimeTaskRepo{task: &model.RenderTask{
		ID: 31, Code: "lrt_123456789012345678901234", RequestedByMemberID: 2,
		Status: model.RenderTaskPending,
	}}
	service, requester, ctx := newRuntimeService(&runtimeRecordResolver{allowed: true}, tasks)
	other := &iammodel.User{ID: 99}
	other.TenantID = 1
	_, err := service.GetRenderTask(ctx, other, tasks.task.Code)
	require.ErrorIs(t, err, labelapp.ErrTaskNotFound)
	_, err = service.DownloadRenderTask(ctx, requester, tasks.task.Code)
	require.ErrorIs(t, err, labelapp.ErrTaskNotReady)

	target := service.(*templateService)
	target.access = accessStub{permissions: map[string]bool{iammodel.LabelResource + ":get": true}}
	detail, err := service.GetRenderTask(ctx, other, tasks.task.Code)
	require.NoError(t, err)
	require.Equal(t, tasks.task.Code, detail.Code)
	tasks.task.Status = model.RenderTaskPartialSuccess
	tasks.task.FileCode = "file_test"
	download, err := service.DownloadRenderTask(ctx, other, tasks.task.Code)
	require.NoError(t, err)
	require.Equal(t, "GET", download.Method)
}

func TestProcessBatchUsesFrozenPresetAndBuildsMultiPagePDF(t *testing.T) {
	records := &runtimeRecordResolver{allowed: true}
	tasks := &runtimeTaskRepo{}
	service, member, ctx := newRuntimeService(records, tasks)
	_, err := service.BatchRender(ctx, member, &model.BatchRenderRequest{
		FormCode: "form_users", RecordIDs: []string{"12", "7"}, OutputPresetID: "large", Format: "pdf",
	})
	require.NoError(t, err)
	// 即使发布快照读取侧随后返回了同 ID 的不同尺寸，任务仍必须使用创建时
	// 冻结的 150×100mm，不能在 Worker 中重新解释预设。
	changedVersion := &model.TemplateVersion{ID: tasks.task.TemplateVersionID, TemplateID: tasks.task.TemplateID,
		VersionNo: tasks.task.TemplateVersionNo, SchemaSnapshot: model.SchemaContent(strings.Replace(runtimeSchema,
			`"width":150,"height":100`, `"width":300,"height":200`, 1))}
	service.(*templateService).versions = runtimeVersionRepo{version: changedVersion}
	artifacts := &capturingArtifactStore{}
	ConfigureBatch(service, tasks, runtimeQueue{}, artifacts, runtimeMemberDirectory{member: member})
	require.NoError(t, service.ProcessBatch(ctx, tasks.task.Code))
	require.Equal(t, model.RenderTaskSuccess, tasks.task.Status)
	require.Equal(t, 2, tasks.task.SuccessCount)
	require.Zero(t, tasks.task.FailedCount)
	require.Equal(t, "file_pdf", tasks.task.FileCode)
	require.True(t, bytes.HasPrefix(artifacts.content, []byte("%PDF-1.4")))
	require.Equal(t, 2, bytes.Count(artifacts.content, []byte("/Type /Page ")))
	require.Contains(t, string(artifacts.content), "/Count 2")
	// 150×100mm 冻结预设对应约 425.2×283.5pt，不能回退 90×60mm 设计画布。
	require.Contains(t, string(artifacts.content), "/MediaBox [0 0 425.1968503937008 283.46456692913387]")
}

func TestProcessBatchKeepsValidPagesWhenRecordChangesAfterQueue(t *testing.T) {
	records := &runtimeRecordResolver{allowed: true}
	tasks := &runtimeTaskRepo{}
	service, member, ctx := newRuntimeService(records, tasks)
	_, err := service.BatchRender(ctx, member, &model.BatchRenderRequest{
		FormCode: "form_users", RecordIDs: []string{"12", "7"}, OutputPresetID: "small", Format: "pdf",
	})
	require.NoError(t, err)
	// 入队后记录被删除：只失败该页，其余已授权记录仍生成可下载文件。
	records.errors = map[uint]error{7: gorm.ErrRecordNotFound}
	artifacts := &capturingArtifactStore{}
	ConfigureBatch(service, tasks, runtimeQueue{}, artifacts, runtimeMemberDirectory{member: member})
	require.NoError(t, service.ProcessBatch(ctx, tasks.task.Code))
	require.Equal(t, model.RenderTaskPartialSuccess, tasks.task.Status)
	require.Equal(t, 1, tasks.task.SuccessCount)
	require.Equal(t, 1, tasks.task.FailedCount)
	require.Equal(t, 1, bytes.Count(artifacts.content, []byte("/Type /Page ")))
	require.Equal(t, model.RenderItemFailed, tasks.items[1].Status)
	require.Equal(t, labelapp.ErrRecordNotFound.Code, tasks.items[1].ErrorCode)
}

func TestProcessBatchFailsWhenEveryQueuedRecordDisappears(t *testing.T) {
	records := &runtimeRecordResolver{allowed: true}
	tasks := &runtimeTaskRepo{}
	service, member, ctx := newRuntimeService(records, tasks)
	_, err := service.BatchRender(ctx, member, &model.BatchRenderRequest{
		FormCode: "form_users", RecordIDs: []string{"12", "7"}, OutputPresetID: "small", Format: "pdf",
	})
	require.NoError(t, err)
	records.errors = map[uint]error{12: gorm.ErrRecordNotFound, 7: gorm.ErrRecordNotFound}
	require.NoError(t, service.ProcessBatch(ctx, tasks.task.Code))
	require.Equal(t, model.RenderTaskFailed, tasks.task.Status)
	require.Zero(t, tasks.task.SuccessCount)
	require.Equal(t, 2, tasks.task.FailedCount)
	require.Empty(t, tasks.task.FileCode)
}
