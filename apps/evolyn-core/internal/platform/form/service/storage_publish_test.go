// 物理存储发布编排单测（方案 §4.2/§7/§8/§16）：异步 Job 创建、同步发布、
// BUSY 收口、不支持字段/类型变更拒绝、字段身份冻结与表名分配防重。
// 桩为纯内存实现，验证 Service 编排语义；DDL 实际执行由 Worker 集成路径覆盖。
package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/httpx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---- 内存桩 ----

type fakeStorageRepo struct {
	bindings    map[uint]*model.FormStorage // key=formID
	nextID      uint
	conflicts   int  // 前 N 次 Create 返回唯一冲突（表名防重重试用）
	markReady   uint // MarkReady 调用次数
	markPublish uint
}

func newFakeStorageRepo() *fakeStorageRepo {
	return &fakeStorageRepo{bindings: map[uint]*model.FormStorage{}, nextID: 900}
}

func (f *fakeStorageRepo) Create(ctx context.Context, storage *model.FormStorage) (bool, bool, error) {
	if f.conflicts > 0 {
		f.conflicts--
		return false, true, nil
	}
	f.nextID++
	storage.ID = f.nextID
	clone := *storage
	f.bindings[storage.FormID] = &clone
	return true, false, nil
}

func (f *fakeStorageRepo) GetByFormID(ctx context.Context, formID uint) (*model.FormStorage, error) {
	if binding, ok := f.bindings[formID]; ok {
		clone := *binding
		return &clone, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeStorageRepo) LockByForm(ctx context.Context, formID uint) (*model.FormStorage, error) {
	return f.GetByFormID(ctx, formID)
}

func (f *fakeStorageRepo) GetByID(ctx context.Context, id uint) (*model.FormStorage, error) {
	for _, binding := range f.bindings {
		if binding.ID == id {
			clone := *binding
			return &clone, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeStorageRepo) LockByID(ctx context.Context, id uint) (*model.FormStorage, error) {
	return f.GetByID(ctx, id)
}

func (f *fakeStorageRepo) MarkPublishing(ctx context.Context, id uint) error {
	for _, binding := range f.bindings {
		if binding.ID == id {
			binding.State = model.StorageStatePublishing
			f.markPublish++
		}
	}
	return nil
}

func (f *fakeStorageRepo) MarkReady(ctx context.Context, id uint, appliedSchemaVersionID uint) error {
	for _, binding := range f.bindings {
		if binding.ID == id {
			binding.State = model.StorageStateReady
			applied := appliedSchemaVersionID
			binding.AppliedSchemaVersionID = &applied
			f.markReady++
		}
	}
	return nil
}

func (f *fakeStorageRepo) MarkFailed(ctx context.Context, id uint) error {
	for _, binding := range f.bindings {
		if binding.ID == id {
			binding.State = model.StorageStateFailed
		}
	}
	return nil
}

func (f *fakeStorageRepo) Migrate() error { return nil }

type fakeSchemaVersionRepo struct {
	versions []model.FormStorageSchemaVersion
	nextID   uint
}

func newFakeSchemaVersionRepo() *fakeSchemaVersionRepo {
	return &fakeSchemaVersionRepo{nextID: 700}
}

func (f *fakeSchemaVersionRepo) Create(ctx context.Context, version *model.FormStorageSchemaVersion) (*model.FormStorageSchemaVersion, error) {
	f.nextID++
	version.ID = f.nextID
	f.versions = append(f.versions, *version)
	return version, nil
}

func (f *fakeSchemaVersionRepo) GetByID(ctx context.Context, id uint) (*model.FormStorageSchemaVersion, error) {
	for _, version := range f.versions {
		if version.ID == id {
			clone := version
			return &clone, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeSchemaVersionRepo) GetApplied(ctx context.Context, storageID uint) (*model.FormStorageSchemaVersion, error) {
	var latest *model.FormStorageSchemaVersion
	for i := range f.versions {
		version := &f.versions[i]
		if version.StorageID == storageID && version.State == model.SchemaVersionApplied {
			if latest == nil || version.ID > latest.ID {
				latest = version
			}
		}
	}
	if latest == nil {
		return nil, gorm.ErrRecordNotFound
	}
	clone := *latest
	return &clone, nil
}

func (f *fakeSchemaVersionRepo) MarkApplied(ctx context.Context, id uint) error {
	for i := range f.versions {
		if f.versions[i].ID == id {
			f.versions[i].State = model.SchemaVersionApplied
		}
	}
	return nil
}

func (f *fakeSchemaVersionRepo) MarkFailed(ctx context.Context, id uint, errorCode, errorDetail string) error {
	for i := range f.versions {
		if f.versions[i].ID == id {
			f.versions[i].State = model.SchemaVersionFailed
			f.versions[i].ErrorCode = errorCode
		}
	}
	return nil
}

func (f *fakeSchemaVersionRepo) ResetForRetry(ctx context.Context, id uint) error {
	for i := range f.versions {
		if f.versions[i].ID == id {
			f.versions[i].State = model.SchemaVersionPending
		}
	}
	return nil
}

func (f *fakeSchemaVersionRepo) Migrate() error { return nil }

type fakeDDLJobRepo struct {
	jobs      []model.FormDDLJob
	nextID    uint
	hasActive bool
}

func newFakeDDLJobRepo() *fakeDDLJobRepo { return &fakeDDLJobRepo{nextID: 800} }

func (f *fakeDDLJobRepo) Create(ctx context.Context, job *model.FormDDLJob) (*model.FormDDLJob, error) {
	f.nextID++
	job.ID = f.nextID
	f.jobs = append(f.jobs, *job)
	return job, nil
}

func (f *fakeDDLJobRepo) GetByID(ctx context.Context, id uint) (*model.FormDDLJob, error) {
	for _, job := range f.jobs {
		if job.ID == id {
			clone := job
			return &clone, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeDDLJobRepo) HasActiveByStorage(ctx context.Context, storageID uint) (bool, error) {
	return f.hasActive, nil
}

func (f *fakeDDLJobRepo) ClaimNextDue(ctx context.Context, now time.Time, limit int) ([]model.FormDDLJob, error) {
	return nil, nil
}

func (f *fakeDDLJobRepo) SaveJob(ctx context.Context, job *model.FormDDLJob) error { return nil }
func (f *fakeDDLJobRepo) ResetForRetry(ctx context.Context, jobID uint, nextAttemptAt time.Time) error {
	return nil
}
func (f *fakeDDLJobRepo) Migrate() error { return nil }

type fakeChildRepo struct {
	children []model.FormStorageChild
}

func (f *fakeChildRepo) Create(ctx context.Context, child *model.FormStorageChild) (bool, bool, error) {
	for _, existing := range f.children {
		if existing.PhysicalTable == child.PhysicalTable {
			return false, true, nil
		}
	}
	f.children = append(f.children, *child)
	return true, false, nil
}

func (f *fakeChildRepo) ListByStorage(ctx context.Context, storageID uint) ([]model.FormStorageChild, error) {
	out := make([]model.FormStorageChild, 0)
	for _, child := range f.children {
		if child.StorageID == storageID {
			out = append(out, child)
		}
	}
	return out, nil
}

func (f *fakeChildRepo) Migrate() error { return nil }

// ---- 装配 ----

type physicalTestEnv struct {
	svc          FormService
	formRepo     *fakeFormRepo
	formVersions *fakeVersionRepo
	storages     *fakeStorageRepo
	versions     *fakeSchemaVersionRepo
	jobs         *fakeDDLJobRepo
	children     *fakeChildRepo
	recordRepo   *fakeRecordRepo
}

func newPhysicalTestEnv() *physicalTestEnv {
	env := &physicalTestEnv{
		formRepo:     newFakeFormRepo(),
		formVersions: newFakeVersionRepo(),
		storages:     newFakeStorageRepo(),
		versions:     newFakeSchemaVersionRepo(),
		jobs:         newFakeDDLJobRepo(),
		children:     &fakeChildRepo{},
		recordRepo:   &fakeRecordRepo{},
	}
	service := newTestService(&fakeQuota{limit: -1}, env.formRepo, env.formVersions, env.recordRepo, nil)
	if injector, ok := service.(PhysicalStorageInjector); ok {
		injector.UsePhysicalStorage(env.storages, env.versions, env.jobs, nil)
	}
	if injector, ok := service.(StorageChildInjector); ok {
		injector.UseStorageChildren(env.children)
	}
	env.svc = service
	return env
}

// v8PhysicalDraft 构造带 fieldId 的合法草稿（两个标量字段）。
func v8PhysicalDraft(extraWidget string) model.JSONContent {
	items := `[
		{"widget":{"type":"text","widgetName":"_widget_a","fieldId":"aaaaaaaa01","enable":true,"visible":true,"allowBlank":true},"label":"姓名","description":"","labelHidden":false,"lineWidth":6},
		{"widget":{"type":"number","widgetName":"_widget_n","fieldId":"aaaaaaaa02","enable":true,"visible":true,"allowBlank":true},"label":"数量","description":"","labelHidden":false,"lineWidth":6}`
	fieldLayout := `"_widget_a","_widget_n"`
	if extraWidget != "" {
		items += "," + extraWidget
		fieldLayout += `,"_widget_m"`
	}
	items += "]"
	return model.JSONContent(`{"content":{"type":"form","layout":"grid-2","items":` + items + `,"layout_fields":[],"field_layout":[` + fieldLayout + `],"fieldShowRules":[],"submitRule":2,"widget_submit_rules":{},"validators":[],"preSubmitConfirm":{"enable":false,"title":"请确认提交","content":"确认提交当前内容？"},"formEvents":[],"linkages":[]}}`)
}

// draftWithItems 按字段项数组组装完整草稿（field_layout 与 items 同步）。
func draftWithItems(items string, names string) model.JSONContent {
	return model.JSONContent(`{"content":{"type":"form","layout":"grid-2","items":` + items + `,"layout_fields":[],"field_layout":[` + names + `],"fieldShowRules":[],"submitRule":2,"widget_submit_rules":{},"validators":[],"preSubmitConfirm":{"enable":false,"title":"请确认提交","content":"确认提交当前内容？"},"formEvents":[],"linkages":[]}}`)
}

// createPhysicalForm 建表单 + physical 绑定 + 保存草稿，返回表单编码与口令。
func (env *physicalTestEnv) createPhysicalForm(t *testing.T, draft model.JSONContent) (string, int64) {
	t.Helper()
	ctx := tenantCtx(1)
	created, err := env.svc.Create(ctx, memberOfTenant(1), &model.CreateFormRequest{
		AppID: 7, Name: "物理表单", FormType: model.FormTypeStandard,
	})
	require.NoError(t, err)
	// 模拟 Create 事务内的存储绑定（newTestService 构造未走 provision 内挂载）
	created2, conflict, cerr := env.storages.Create(ctx, &model.FormStorage{
		FormID:        env.formRepo.formByCode(created.Code).ID,
		Backend:       model.StorageBackendPhysical,
		PhysicalTable: "tn_fd_01ab_k7q2m8",
		State:         model.StorageStateReady,
	})
	require.NoError(t, cerr)
	require.True(t, created2)
	require.False(t, conflict)
	saved, err := env.svc.SaveDraft(ctx, memberOfTenant(1), created.Code, &model.SaveDraftRequest{
		DraftRevision: 1, ProtocolVersion: model.CurrentProtocolVersion, Content: draft,
	})
	require.NoError(t, err)
	return created.Code, saved.DraftRevision
}

// ---- 用例 ----

// 首次发布：结构变更 → 202 语义（async + jobId）；FormVersion 已建但发布指针
// 不推进（MarkPublished 未执行，方案 §16.2：Job 成功前新版本不成为 latest）。
func TestPublishPhysicalCreatesAsyncDDLJob(t *testing.T) {
	env := newPhysicalTestEnv()
	code, revision := env.createPhysicalForm(t, v8PhysicalDraft(""))
	result, err := env.svc.Publish(tenantCtx(1), memberOfTenant(1), code, &model.PublishRequest{DraftRevision: revision})
	require.NoError(t, err)
	assert.True(t, result.Async)
	require.NotNil(t, result.JobID)
	assert.NotZero(t, *result.JobID)

	// Job 与 SchemaVersion 落库，存储转 PUBLISHING
	require.Len(t, env.jobs.jobs, 1)
	assert.Equal(t, model.DDLJobPending, env.jobs.jobs[0].Status)
	require.Len(t, env.versions.versions, 1)
	assert.Equal(t, model.SchemaVersionPending, env.versions.versions[0].State)
	assert.EqualValues(t, 1, env.storages.markPublish)
	assert.EqualValues(t, 0, env.storages.markReady)

	// 发布指针未推进
	form := env.formRepo.formByCode(code)
	assert.Nil(t, form.LatestVersionID, "latest must not advance before DDL job succeeds")
}

// 无结构变化的再发布：同步完成（async=false），版本直接 APPLIED + 指针推进。
func TestPublishPhysicalNoStructuralChangeIsSync(t *testing.T) {
	env := newPhysicalTestEnv()
	code, revision := env.createPhysicalForm(t, v8PhysicalDraft(""))

	// 首次发布后手动完成 Job（模拟 Worker 成功路径的元数据推进）
	first, err := env.svc.Publish(tenantCtx(1), memberOfTenant(1), code, &model.PublishRequest{DraftRevision: revision})
	require.NoError(t, err)
	require.True(t, first.Async)
	completeFirstDDLJob(t, env, code)

	// 未改字段的第二次发布：同步
	saved, err := env.svc.SaveDraft(tenantCtx(1), memberOfTenant(1), code, &model.SaveDraftRequest{
		DraftRevision: 2, ProtocolVersion: model.CurrentProtocolVersion, Content: v8PhysicalDraft(""),
	})
	require.NoError(t, err)
	second, err := env.svc.Publish(tenantCtx(1), memberOfTenant(1), code, &model.PublishRequest{DraftRevision: saved.DraftRevision})
	require.NoError(t, err)
	assert.False(t, second.Async)
	assert.Nil(t, second.JobID)
	assert.EqualValues(t, 2, env.storages.markReady)
}

// 存在未完成 Job 时新发布拒绝：FORM_STORAGE_BUSY（方案 §16.1）。
func TestPublishPhysicalBusyRejected(t *testing.T) {
	env := newPhysicalTestEnv()
	code, revision := env.createPhysicalForm(t, v8PhysicalDraft(""))
	env.jobs.hasActive = true
	_, err := env.svc.Publish(tenantCtx(1), memberOfTenant(1), code, &model.PublishRequest{DraftRevision: revision})
	assert.ErrorIs(t, err, apperrors.ErrStorageBusy)
}

// 无物理模型的控件（多选）发布拒绝并列出 issues（方案 §4.3/§16.11）。
func TestPublishPhysicalUnsupportedFieldRejected(t *testing.T) {
	env := newPhysicalTestEnv()
	multiSelect := `{"widget":{"type":"checkboxgroup","widgetName":"_widget_m","fieldId":"aaaaaaaa03","enable":true,"visible":true,"allowBlank":true,"options":[{"value":"a","label":"A"}]},"label":"多选","description":"","labelHidden":false,"lineWidth":6}`
	code, revision := env.createPhysicalForm(t, v8PhysicalDraft(multiSelect))
	_, err := env.svc.Publish(tenantCtx(1), memberOfTenant(1), code, &model.PublishRequest{DraftRevision: revision})
	require.ErrorIs(t, err, apperrors.ErrStorageUnsupportedField)
	biz := bizOf(t, err)
	issues, ok := biz.Data.(map[string]any)["issues"].([]SchemaIssue)
	require.True(t, ok, "data carries issues")
	assert.NotEmpty(t, issues)
	assert.Contains(t, issues[0].Message, "不支持物理存储")
}

// 已应用字段的类型变更拒绝：FORM_STORAGE_TYPE_CHANGE_UNSUPPORTED（§16.11）。
func TestPublishPhysicalTypeChangeRejected(t *testing.T) {
	env := newPhysicalTestEnv()
	code, revision := env.createPhysicalForm(t, v8PhysicalDraft(""))
	first, err := env.svc.Publish(tenantCtx(1), memberOfTenant(1), code, &model.PublishRequest{DraftRevision: revision})
	require.NoError(t, err)
	require.True(t, first.Async)
	// 模拟 Worker 成功
	require.NoError(t, env.versions.MarkApplied(context.Background(), env.versions.versions[0].ID))
	binding, err := env.storages.GetByFormID(context.Background(), env.formRepo.formByCode(code).ID)
	require.NoError(t, err)
	env.storages.MarkReady(context.Background(), binding.ID, env.versions.versions[0].ID)

	// 同 fieldId 字段 number → text（改 widget.type）
	changed := `[
		{"widget":{"type":"text","widgetName":"_widget_a","fieldId":"aaaaaaaa01","enable":true,"visible":true,"allowBlank":true},"label":"姓名","description":"","labelHidden":false,"lineWidth":6},
		{"widget":{"type":"text","widgetName":"_widget_n","fieldId":"aaaaaaaa02","enable":true,"visible":true,"allowBlank":true},"label":"数量","description":"","labelHidden":false,"lineWidth":6}]`
	draft := draftWithItems(changed, `"_widget_a","_widget_n"`)
	saved, err := env.svc.SaveDraft(tenantCtx(1), memberOfTenant(1), code, &model.SaveDraftRequest{
		DraftRevision: 2, ProtocolVersion: model.CurrentProtocolVersion, Content: draft,
	})
	require.NoError(t, err)
	_, err = env.svc.Publish(tenantCtx(1), memberOfTenant(1), code, &model.PublishRequest{DraftRevision: saved.DraftRevision})
	assert.ErrorIs(t, err, apperrors.ErrStorageTypeChangeUnsupported)
	biz := bizOf(t, err)
	fields, ok := biz.Data.(map[string]any)["fields"].([]string)
	require.True(t, ok)
	assert.Contains(t, fields[0], "_widget_n")
}

// v8 已发布字段的身份冻结：同 fieldId 换 widgetName / 同 widgetName 换
// fieldId 均拒绝；删除字段与新增字段合法（方案 §4.1/§16.11）。
func TestSaveDraftFieldIdentityFrozen(t *testing.T) {
	env := newPhysicalTestEnv()
	code, revision := env.createPhysicalForm(t, v8PhysicalDraft(""))
	_, err := env.svc.Publish(tenantCtx(1), memberOfTenant(1), code, &model.PublishRequest{DraftRevision: revision})
	require.NoError(t, err)
	// 冻结校验以已生效发布快照为基准：先完成 DDL Job（模拟 Worker 收尾）
	completeFirstDDLJob(t, env, code)

	// 同 fieldId 换 widgetName → 拒绝
	renamed := `[
		{"widget":{"type":"text","widgetName":"_widget_renamed","fieldId":"aaaaaaaa01","enable":true,"visible":true,"allowBlank":true},"label":"姓名","description":"","labelHidden":false,"lineWidth":6},
		{"widget":{"type":"number","widgetName":"_widget_n","fieldId":"aaaaaaaa02","enable":true,"visible":true,"allowBlank":true},"label":"数量","description":"","labelHidden":false,"lineWidth":6}]`
	save := func(items string, names string) error {
		_, err := env.svc.SaveDraft(tenantCtx(1), memberOfTenant(1), code, &model.SaveDraftRequest{
			DraftRevision:   env.formRepo.formByCode(code).DraftRevision,
			ProtocolVersion: model.CurrentProtocolVersion,
			Content:         draftWithItems(items, names),
		})
		return err
	}
	assert.ErrorIs(t, save(renamed, `"_widget_renamed","_widget_n"`), apperrors.ErrFieldIdentityFrozen)

	// 同 widgetName 换 fieldId → 拒绝
	swapped := `[
		{"widget":{"type":"text","widgetName":"_widget_a","fieldId":"aaaaaaaa99","enable":true,"visible":true,"allowBlank":true},"label":"姓名","description":"","labelHidden":false,"lineWidth":6},
		{"widget":{"type":"number","widgetName":"_widget_n","fieldId":"aaaaaaaa02","enable":true,"visible":true,"allowBlank":true},"label":"数量","description":"","labelHidden":false,"lineWidth":6}]`
	assert.ErrorIs(t, save(swapped, `"_widget_a","_widget_n"`), apperrors.ErrFieldIdentityFrozen)

	// 删除字段（弃用）+ 新增字段 → 合法
	dropped := `[
		{"widget":{"type":"text","widgetName":"_widget_a","fieldId":"aaaaaaaa01","enable":true,"visible":true,"allowBlank":true},"label":"姓名","description":"","labelHidden":false,"lineWidth":6},
		{"widget":{"type":"datetime","widgetName":"_widget_new","fieldId":"aaaaaaaa03","enable":true,"visible":true,"allowBlank":true,"format":"date"},"label":"日期","description":"","labelHidden":false,"lineWidth":6}]`
	assert.NoError(t, save(dropped, `"_widget_a","_widget_new"`))
}

// 表名分配：唯一冲突换名重试至成功，最终表名符合 tn_fd_ 规则（方案 §7）。
func TestAttachPhysicalStorageRetriesOnConflict(t *testing.T) {
	env := newPhysicalTestEnv()
	env.storages.conflicts = 2 // 前两次表名冲突
	ctx := tenantCtx(1)
	created, err := env.svc.Create(ctx, memberOfTenant(1), &model.CreateFormRequest{
		AppID: 7, Name: "防重表单", FormType: model.FormTypeStandard,
	})
	require.NoError(t, err)
	form := env.formRepo.formByCode(created.Code)
	svc := env.svc.(*formService)
	require.NoError(t, svc.attachPhysicalStorage(ctx, 1, form, 7))
	require.Len(t, env.storages.bindings, 1)
	for _, binding := range env.storages.bindings {
		assert.True(t, strings.HasPrefix(binding.PhysicalTable, "tn_fd_"), "table name %s", binding.PhysicalTable)
		assert.Len(t, binding.PhysicalTable, 17) // tn_fd_ + app4 + _ + rand6
		assert.Equal(t, model.StorageBackendPhysical, binding.Backend)
		assert.Equal(t, model.StorageStateReady, binding.State)
	}
}

// completeFirstDDLJob 模拟 Worker 成功收尾：SchemaVersion APPLIED →
// Storage READY（applied 指针）→ Form 发布指针推进到首版（latest 生效后
// 字段身份冻结校验才有基准）。
func completeFirstDDLJob(t *testing.T, env *physicalTestEnv, code string) {
	t.Helper()
	form := env.formRepo.formByCode(code)
	require.NotEmpty(t, env.versions.versions)
	schemaVersion := env.versions.versions[len(env.versions.versions)-1]
	require.NoError(t, env.versions.MarkApplied(context.Background(), schemaVersion.ID))
	binding, err := env.storages.GetByFormID(context.Background(), form.ID)
	require.NoError(t, err)
	require.NoError(t, env.storages.MarkReady(context.Background(), binding.ID, schemaVersion.ID))
	// 首个 FormVersion：fakeVersionRepo 的自增 ID 从 501 起
	var published model.FormVersion
	found := false
	for _, version := range env.formVersions.versions {
		if version.FormID == form.ID {
			published = *version
			found = true
		}
	}
	require.True(t, found, "first published version must exist")
	require.NotNil(t, published)
	require.NoError(t, env.formRepo.MarkPublished(context.Background(), form.ID, published.ID, published.VersionNo))
}

// bizOf 从错误链取 BizError（Data 断言助手）。
func bizOf(t *testing.T, err error) *httpx.BizError {
	t.Helper()
	var biz *httpx.BizError
	if errors.As(err, &biz) {
		return biz
	}
	t.Fatalf("expected BizError, got %v", err)
	return nil
}
