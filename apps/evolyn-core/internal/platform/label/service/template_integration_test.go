package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"evolyn/internal/contextx"
	enginelabel "evolyn/internal/engine/label"
	"evolyn/internal/infrastructure"
	formmodel "evolyn/internal/platform/form/model"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	labelapp "evolyn/internal/platform/label"
	labelmodel "evolyn/internal/platform/label/model"
	labelrepository "evolyn/internal/platform/label/repository"
	"evolyn/internal/testsupport"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type staticFormDirectory struct{ form FormView }

func (s staticFormDirectory) FormByCode(_ context.Context, code string) (FormView, bool, error) {
	if code != s.form.Code {
		return FormView{}, true, nil
	}
	return s.form, false, nil
}

func (s staticFormDirectory) PublishedForm(_ context.Context, id uint) (FormView, bool, error) {
	if id != s.form.ID {
		return FormView{}, true, nil
	}
	return s.form, false, nil
}

type mutableRecordResolver struct {
	mu      sync.RWMutex
	record  *RecordView
	err     error
	started chan struct{}
	release chan struct{}
}

type failingPublishRepository struct {
	labelrepository.TemplateRepository
}

func (f failingPublishRepository) MarkPublished(context.Context, uint, uint, int, int64) error {
	return errors.New("injected published-pointer failure")
}

func (r *mutableRecordResolver) GetRecord(context.Context, *iammodel.User, uint, uint) (*RecordView, error) {
	if r.started != nil {
		close(r.started)
		<-r.release
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.record, r.err
}

func (r *mutableRecordResolver) set(record *RecordView, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.record, r.err = record, err
}

// TestTemplateLifecyclePostgres 覆盖生命周期主链与安全边界。测试使用生产迁移、
// 事务管理器和真实仓储；仅表单目录及记录权限裁剪通过窄端口构造确定性输入。
func TestTemplateLifecyclePostgres(t *testing.T) {
	db := testsupport.NewPostgres(t)
	ctx := contextx.NewTenantContext(context.Background(), 301)
	member := &iammodel.User{ID: 41}
	member.TenantID = 301

	form := createLifecycleForm(t, db, ctx)
	formDirectory := staticFormDirectory{form: FormView{
		ID: form.ID, AppID: form.AppID, Code: form.Code, Name: form.Name,
		Published: true, Fields: map[string]bool{"asset_code": true},
	}}
	records := &mutableRecordResolver{record: &RecordView{
		FormID: form.ID, Fields: map[string]any{"asset_code": "A-001"},
		System: map[string]any{"recordId": "1"},
	}}
	permissions := map[string]bool{
		iammodel.LabelTemplateResource + ":create": true,
		iammodel.LabelTemplateResource + ":get":    true,
		iammodel.LabelTemplateResource + ":update": true,
		iammodel.LabelTemplateResource + ":delete": true,
		iammodel.LabelResource + ":create":         true,
	}
	templateRepo := labelrepository.NewTemplateRepository(db)
	versionRepo := labelrepository.NewVersionRepository(db)
	svc := NewTemplateService(
		infrastructure.NewTxManager(db), templateRepo, versionRepo,
		formDirectory, records, accessStub{permissions: permissions}, nil,
	)

	created, err := svc.Create(ctx, member, &labelmodel.CreateTemplateRequest{Name: "资产标签", FormCode: form.Code})
	require.NoError(t, err)
	require.EqualValues(t, 1, created.DraftRevision)

	// 无资源权限与跨租户成员必须在访问仓储结果前收口，不能泄露模板存在性。
	denied := NewTemplateService(
		infrastructure.NewTxManager(db), templateRepo, versionRepo,
		formDirectory, records, accessStub{permissions: map[string]bool{}}, nil,
	)
	_, err = denied.Get(ctx, member, created.Code)
	require.ErrorIs(t, err, labelapp.ErrForbidden)
	crossTenantMember := &iammodel.User{ID: member.ID}
	crossTenantMember.TenantID = 302
	_, err = svc.Get(ctx, crossTenantMember, created.Code)
	require.ErrorIs(t, err, labelapp.ErrForbidden)
	betaCtx := contextx.NewTenantContext(context.Background(), 302)
	betaMember := &iammodel.User{ID: 52}
	betaMember.TenantID = 302
	_, err = svc.Get(betaCtx, betaMember, created.Code)
	require.ErrorIs(t, err, labelapp.ErrTemplateNotFound)

	fieldDraftV2 := lifecycleSchema(t, "资产标签 V1", form.Code, "asset_code")

	// 预览渲染期间并发保存：渲染可以完成，但旧修订不得被标记为已预览。
	blocking := &mutableRecordResolver{
		record: records.record, started: make(chan struct{}), release: make(chan struct{}),
	}
	blockingSvc := NewTemplateService(
		infrastructure.NewTxManager(db), templateRepo, versionRepo,
		formDirectory, blocking, accessStub{permissions: permissions}, nil,
	)
	previewErr := make(chan error, 1)
	go func() {
		_, previewError := blockingSvc.Preview(ctx, member, created.Code, &labelmodel.PreviewRequest{RecordID: "1", Format: "svg"})
		previewErr <- previewError
	}()
	<-blocking.started
	saved, err := svc.SaveDraft(ctx, member, created.Code, &labelmodel.SaveDraftRequest{DraftRevision: 1, Schema: fieldDraftV2})
	require.NoError(t, err)
	require.EqualValues(t, 2, saved.DraftRevision)
	close(blocking.release)
	require.ErrorIs(t, <-previewErr, labelapp.ErrRevisionConflict)

	// 过期写入不能覆盖新草稿，且保存后的页面投影来自完整 Schema。
	_, err = svc.SaveDraft(ctx, member, created.Code, &labelmodel.SaveDraftRequest{DraftRevision: 1, Schema: fieldDraftV2})
	require.ErrorIs(t, err, labelapp.ErrRevisionConflict)
	detail, err := svc.Get(ctx, member, created.Code)
	require.NoError(t, err)
	require.EqualValues(t, 80, detail.Width)
	require.EqualValues(t, 50, detail.Height)

	// 记录权限和字段权限分别映射稳定错误，且不会携带记录值。
	records.set(nil, httpx.NewBiz("FORM_RECORD_FORBIDDEN", "forbidden", 403))
	_, err = svc.Preview(ctx, member, created.Code, &labelmodel.PreviewRequest{RecordID: "1"})
	require.ErrorIs(t, err, labelapp.ErrRecordNoPermission)
	records.set(&RecordView{FormID: form.ID, Fields: map[string]any{}, System: map[string]any{}}, nil)
	_, err = svc.Preview(ctx, member, created.Code, &labelmodel.PreviewRequest{RecordID: "1"})
	require.ErrorIs(t, err, labelapp.ErrFieldNoPermission)
	records.set(&RecordView{FormID: form.ID, Fields: map[string]any{"asset_code": "A-001"}, System: map[string]any{}}, nil)

	_, err = svc.Publish(ctx, member, created.Code, &labelmodel.PublishRequest{DraftRevision: 2})
	require.ErrorIs(t, err, labelapp.ErrRealPreviewRequired)
	_, err = svc.Preview(ctx, member, created.Code, &labelmodel.PreviewRequest{RecordID: "1"})
	require.NoError(t, err)

	// 快照已插入但发布指针推进失败时，统一事务必须回滚两者，不能留下孤儿版本。
	failingSvc := NewTemplateService(
		infrastructure.NewTxManager(db),
		failingPublishRepository{TemplateRepository: templateRepo}, versionRepo,
		formDirectory, records, accessStub{permissions: permissions}, nil,
	)
	_, err = failingSvc.Publish(ctx, member, created.Code, &labelmodel.PublishRequest{DraftRevision: 2})
	require.ErrorContains(t, err, "injected published-pointer failure")
	var versionCount int64
	require.NoError(t, db.WithContext(ctx).Model(&labelmodel.TemplateVersion{}).Count(&versionCount).Error)
	require.Zero(t, versionCount, "发布事务失败不得残留不可变快照")
	rolledBackTemplate, err := templateRepo.GetByCode(ctx, created.Code)
	require.NoError(t, err)
	require.Zero(t, rolledBackTemplate.PublishedVersion)
	require.Nil(t, rolledBackTemplate.LatestVersionID)

	v1, err := svc.Publish(ctx, member, created.Code, &labelmodel.PublishRequest{DraftRevision: 2})
	require.NoError(t, err)
	require.Equal(t, 1, v1.VersionNo)

	// 同修订重试幂等，不新增快照。
	v1Retry, err := svc.Publish(ctx, member, created.Code, &labelmodel.PublishRequest{DraftRevision: 2})
	require.NoError(t, err)
	require.Equal(t, 1, v1Retry.VersionNo)

	fieldDraftV3 := lifecycleSchema(t, "资产标签 V2", form.Code, "asset_code")
	_, err = svc.SaveDraft(ctx, member, created.Code, &labelmodel.SaveDraftRequest{DraftRevision: 2, Schema: fieldDraftV3})
	require.NoError(t, err)
	_, err = svc.Preview(ctx, member, created.Code, &labelmodel.PreviewRequest{RecordID: "1"})
	require.NoError(t, err)
	v2, err := svc.Publish(ctx, member, created.Code, &labelmodel.PublishRequest{DraftRevision: 3})
	require.NoError(t, err)
	require.Equal(t, 2, v2.VersionNo)

	var versions []labelmodel.TemplateVersion
	require.NoError(t, db.WithContext(ctx).Order("version_no ASC").Find(&versions).Error)
	require.Len(t, versions, 2)
	require.JSONEq(t, string(fieldDraftV2), string(versions[0].SchemaSnapshot), "V1 快照必须保持不变")
	require.JSONEq(t, string(fieldDraftV3), string(versions[1].SchemaSnapshot))

	// 同一草稿并发发布由模板行锁串行化，两个调用返回同一业务版本。
	fieldDraftV4 := lifecycleSchema(t, "资产标签 V3", form.Code, "asset_code")
	_, err = svc.SaveDraft(ctx, member, created.Code, &labelmodel.SaveDraftRequest{DraftRevision: 3, Schema: fieldDraftV4})
	require.NoError(t, err)
	_, err = svc.Preview(ctx, member, created.Code, &labelmodel.PreviewRequest{RecordID: "1"})
	require.NoError(t, err)
	results := make(chan *labelmodel.PublishResult, 2)
	errorsCh := make(chan error, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, publishErr := svc.Publish(ctx, member, created.Code, &labelmodel.PublishRequest{DraftRevision: 4})
			results <- result
			errorsCh <- publishErr
		}()
	}
	wait.Wait()
	close(results)
	close(errorsCh)
	for publishErr := range errorsCh {
		require.NoError(t, publishErr)
	}
	for result := range results {
		require.NotNil(t, result)
		require.Equal(t, 3, result.VersionNo)
	}
	versions = nil
	require.NoError(t, db.WithContext(ctx).Order("version_no ASC").Find(&versions).Error)
	require.Len(t, versions, 3)

	require.NoError(t, svc.Delete(ctx, member, created.Code))
	_, err = svc.Get(ctx, member, created.Code)
	require.ErrorIs(t, err, labelapp.ErrTemplateNotFound)
	rebound, err := svc.Create(ctx, member, &labelmodel.CreateTemplateRequest{Name: "重新绑定标签", FormCode: form.Code})
	require.NoError(t, err)
	require.NotEqual(t, created.Code, rebound.Code)
}

func createLifecycleForm(t *testing.T, db *gorm.DB, ctx context.Context) *formmodel.Form {
	t.Helper()
	form := &formmodel.Form{
		AppID: 9, Code: "form_label_lifecycle", Name: "标签生命周期表单",
		FormType: formmodel.FormTypeStandard, DraftContent: formmodel.JSONContent(`{"content":{"items":[]}}`),
		DraftRevision: 1, ProtocolVersion: formmodel.CurrentProtocolVersion, CreatorMemberID: 41,
	}
	require.NoError(t, db.WithContext(ctx).Create(form).Error)
	return form
}

func lifecycleSchema(t *testing.T, name, formCode, fieldID string) json.RawMessage {
	t.Helper()
	schema := enginelabel.Schema{
		SchemaVersion: "1.0", Name: name,
		Page:   enginelabel.Page{Width: 80, Height: 50, Unit: "mm", DPI: 300, Background: "#ffffff"},
		Source: &enginelabel.Source{Type: "form", FormID: formCode},
		Elements: []enginelabel.Element{{
			ID: "field-1", Type: "field", X: 2, Y: 2, Width: 40, Height: 8,
			Visible: true, Value: &enginelabel.ValueSource{Type: "field", FieldID: fieldID},
			Style: &enginelabel.TextStyle{
				FontFamily: "sans-serif", FontSize: 4, FontWeight: 400, Color: "#111111",
				LineHeight: 1.2, TextAlign: "left", VerticalAlign: "top", Overflow: "clip",
			},
		}},
		Settings: enginelabel.Settings{SnapToGrid: true, GridSize: 1},
	}
	require.Empty(t, enginelabel.Validate(&schema))
	encoded, err := json.Marshal(schema)
	require.NoError(t, err)
	return encoded
}
