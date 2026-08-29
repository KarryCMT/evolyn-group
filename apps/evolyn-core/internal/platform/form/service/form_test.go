package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"evolyn/internal/contextx"
	auditservice "evolyn/internal/platform/audit/service"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/form/repository"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	tenantservice "evolyn/internal/platform/tenant/service"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ---- 测试桩 ----

type passThroughTx struct{}

func (passThroughTx) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type fakeAccess struct{ perms map[string]bool }

func (f fakeAccess) Permissions(ctx context.Context, member *iammodel.User) map[string]bool {
	return f.perms
}

var adminPerms = map[string]bool{
	"forms:create": true, "forms:get": true, "forms:list": true,
	"forms:patch": true, "forms:update": true, "forms:delete": true,
	"form-records:create": true,
}

type fakeQuota struct {
	limit int64
	used  int64
}

func (f *fakeQuota) Check(ctx context.Context, tenantID uint, key string) error {
	if f.limit >= 0 && f.used >= f.limit {
		return httpx.Wrap(tenantservice.ErrQuotaExceeded, fmt.Errorf("quota %s exceeded", key))
	}
	return nil
}

func (f *fakeQuota) Usage(ctx context.Context, tenantID uint, key string) (int64, error) {
	return f.used, nil
}

func (f *fakeQuota) CheckAndReserve(ctx context.Context, tenantID uint, key string, fn func(ctx context.Context) error) error {
	if err := f.Check(ctx, tenantID, key); err != nil {
		return err
	}
	return fn(ctx)
}

type fakeRecorder struct{ entries []auditservice.Entry }

func (f *fakeRecorder) Record(ctx context.Context, entry auditservice.Entry) {
	f.entries = append(f.entries, entry)
}

// fakeMenuPort 菜单维护端口桩：记录调用便于断言表单生命周期是否联动菜单
type fakeMenuPort struct {
	attached []struct {
		appID  uint
		formID uint
		name   string
		parent string
	}
	renamed []struct {
		appID  uint
		formID uint
		name   string
	}
	detached []struct {
		appID  uint
		formID uint
	}
}

func (f *fakeMenuPort) AttachFormEntry(ctx context.Context, applicationID, formID uint, name, parentEntryCode string) error {
	f.attached = append(f.attached, struct {
		appID  uint
		formID uint
		name   string
		parent string
	}{applicationID, formID, name, parentEntryCode})
	return nil
}

func (f *fakeMenuPort) SyncFormEntryName(ctx context.Context, applicationID, formID uint, name string) error {
	f.renamed = append(f.renamed, struct {
		appID  uint
		formID uint
		name   string
	}{applicationID, formID, name})
	return nil
}

func (f *fakeMenuPort) SyncFormEntryAppearance(ctx context.Context, applicationID, formID uint, icon, color string) error {
	return nil
}

func (f *fakeMenuPort) DetachFormEntry(ctx context.Context, applicationID, formID uint) error {
	f.detached = append(f.detached, struct {
		appID  uint
		formID uint
	}{applicationID, formID})
	return nil
}

type fakeApps struct {
	apps     map[uint]string // id → status
	codeToID map[string]uint
}

func (f fakeApps) ApplicationByID(ctx context.Context, id uint) (ApplicationView, bool, error) {
	status, ok := f.apps[id]
	if !ok {
		return ApplicationView{}, true, nil
	}
	return ApplicationView{ID: id, Status: status}, false, nil
}

func (f fakeApps) ApplicationByCode(ctx context.Context, code string) (ApplicationView, bool, error) {
	id, ok := f.codeToID[code]
	if !ok {
		return ApplicationView{}, true, nil
	}
	return f.ApplicationByID(ctx, id)
}

type fakeFormRepo struct {
	forms     map[uint]*model.Form
	nextID    uint
	updatedID uint
}

func newFakeFormRepo() *fakeFormRepo {
	return &fakeFormRepo{forms: map[uint]*model.Form{}, nextID: 100}
}

func (f *fakeFormRepo) Create(ctx context.Context, form *model.Form) (*model.Form, error) {
	f.nextID++
	form.ID = f.nextID
	clone := *form
	f.forms[form.ID] = &clone
	return &clone, nil
}

func (f *fakeFormRepo) GetByCode(ctx context.Context, code string) (*model.Form, error) {
	for _, form := range f.forms {
		if form.Code != code {
			continue
		}
		clone := *form
		return &clone, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeFormRepo) formByCode(code string) *model.Form {
	for _, form := range f.forms {
		if form.Code == code {
			return form
		}
	}
	return nil
}

func (f *fakeFormRepo) List(ctx context.Context, params repository.ListParams) ([]model.Form, bool, error) {
	return nil, false, nil
}

func (f *fakeFormRepo) UpdateName(ctx context.Context, id uint, name string) error {
	f.forms[id].Name = name
	return nil
}

func (f *fakeFormRepo) UpdateFormType(ctx context.Context, id uint, formType model.FormType) error {
	f.forms[id].FormType = formType
	return nil
}

func (f *fakeFormRepo) UpdateDraft(ctx context.Context, id uint, fromRevision int64, content model.JSONContent) (bool, error) {
	form, ok := f.forms[id]
	if !ok || form.DraftRevision != fromRevision {
		return false, nil
	}
	form.DraftContent = content
	form.DraftRevision++
	return true, nil
}

func (f *fakeFormRepo) MarkPublished(ctx context.Context, id uint, versionID uint, versionNo int) error {
	form := f.forms[id]
	vid := versionID
	form.LatestVersionID = &vid
	form.PublishedVersion = versionNo
	return nil
}

func (f *fakeFormRepo) SoftDelete(ctx context.Context, form *model.Form) error {
	delete(f.forms, form.ID)
	return nil
}

func (f *fakeFormRepo) CountBillableFormsByTenant(ctx context.Context, tenantID uint) (int64, error) {
	return int64(len(f.forms)), nil
}

func (f *fakeFormRepo) ExistingFormTargets(ctx context.Context, ids []uint) (map[uint]repository.FormMenuTarget, error) {
	existing := map[uint]repository.FormMenuTarget{}
	for _, id := range ids {
		if form, ok := f.forms[id]; ok {
			existing[id] = repository.FormMenuTarget{Code: form.Code, FormType: form.FormType}
		}
	}
	return existing, nil
}

func (f *fakeFormRepo) Migrate() error { return nil }

type fakeVersionRepo struct {
	versions    map[uint]*model.FormVersion
	byFormAndNo map[string]uint
	nextID      uint
}

func newFakeVersionRepo() *fakeVersionRepo {
	return &fakeVersionRepo{versions: map[uint]*model.FormVersion{}, byFormAndNo: map[string]uint{}, nextID: 500}
}

func (f *fakeVersionRepo) Create(ctx context.Context, version *model.FormVersion) (*model.FormVersion, error) {
	f.nextID++
	version.ID = f.nextID
	clone := *version
	f.versions[version.ID] = &clone
	f.byFormAndNo[fmt.Sprintf("%d:%d", version.FormID, version.VersionNo)] = version.ID
	return &clone, nil
}

func (f *fakeVersionRepo) SetSchemaRevision(ctx context.Context, id uint, revision int64) error {
	f.versions[id].SchemaRevision = revision
	return nil
}

func (f *fakeVersionRepo) GetByID(ctx context.Context, id uint) (*model.FormVersion, error) {
	version, ok := f.versions[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	clone := *version
	return &clone, nil
}

func (f *fakeVersionRepo) MaxVersionNo(ctx context.Context, formID uint) (int, error) {
	max := 0
	for _, version := range f.versions {
		if version.FormID == formID && version.VersionNo > max {
			max = version.VersionNo
		}
	}
	return max, nil
}

func (f *fakeVersionRepo) GetByFormAndVersionNo(ctx context.Context, formID uint, versionNo int) (*model.FormVersion, error) {
	id, ok := f.byFormAndNo[fmt.Sprintf("%d:%d", formID, versionNo)]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return f.GetByID(ctx, id)
}

func (f *fakeVersionRepo) Migrate() error { return nil }

type fakeRecordRepo struct{ records []*model.FormRecord }

func (f *fakeRecordRepo) Create(ctx context.Context, record *model.FormRecord) (*model.FormRecord, error) {
	record.ID = uint(len(f.records) + 1)
	clone := *record
	f.records = append(f.records, &clone)
	return &clone, nil
}

func (f *fakeRecordRepo) GetByID(ctx context.Context, id uint) (*model.FormRecord, error) {
	for _, record := range f.records {
		if record.ID == id {
			return record, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeRecordRepo) UpdateValues(ctx context.Context, id uint, values model.JSONContent) error {
	for _, record := range f.records {
		if record.ID == id {
			record.Values = values
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (f *fakeRecordRepo) Migrate() error { return nil }

// ---- 服务工厂 ----

func newTestService(quota *fakeQuota, formRepo *fakeFormRepo, versionRepo *fakeVersionRepo, recordRepo *fakeRecordRepo, menu *fakeMenuPort) FormService {
	if menu == nil {
		menu = &fakeMenuPort{}
	}
	return NewFormService(
		passThroughTx{}, formRepo, versionRepo, recordRepo, quota,
		&fakeRecorder{}, fakeAccess{perms: adminPerms},
		fakeApps{
			apps:     map[uint]string{7: "active", 8: "archived"},
			codeToID: map[string]uint{"app_x": 7, "app_archived": 8},
		},
		menu,
	)
}

func memberOfTenant(tenantID uint) *iammodel.User {
	member := &iammodel.User{ID: 11, Nickname: "tester"}
	member.TenantID = tenantID
	return member
}

func tenantCtx(tenantID uint) context.Context {
	return contextx.NewTenantContext(context.Background(), tenantID)
}

func validDraft() model.JSONContent {
	return model.JSONContent(`{"content":{"type":"form","items":[{"widget":{"type":"text","widgetName":"_widget_a","enable":true,"visible":true,"allowBlank":false},"label":"姓名","description":"","labelHidden":false,"lineWidth":12}]}}`)
}

// ---- 用例 ----

func TestCreateForm(t *testing.T) {
	quota := &fakeQuota{limit: 10}
	formRepo := newFakeFormRepo()
	svc := newTestService(quota, formRepo, newFakeVersionRepo(), &fakeRecordRepo{}, nil)

	detail, err := svc.Create(tenantCtx(1), memberOfTenant(1), &model.CreateFormRequest{
		ApplicationID: 7, Name: "报名表", FormType: model.FormTypeWorkflow,
	})
	assert.NoError(t, err)
	assert.Equal(t, "报名表", detail.Name)
	assert.Regexp(t, `^form_[0-9a-f]{16}$`, detail.Code)
	assert.Equal(t, model.FormTypeWorkflow, detail.FormType)
	assert.EqualValues(t, 1, detail.DraftRevision)
	assert.Equal(t, emptyFormDocument, detail.Draft)
	assert.Equal(t, model.FormTypeWorkflow, formRepo.formByCode(detail.Code).FormType)
	loaded, err := svc.Get(tenantCtx(1), memberOfTenant(1), detail.Code)
	assert.NoError(t, err)
	assert.Equal(t, model.FormTypeWorkflow, loaded.FormType)
	assert.Equal(t, model.FormTypeWorkflow, summaryOf(formRepo.formByCode(detail.Code)).FormType)

	// 表单类型必须由创建请求明确给出，未知值不能进入数据库。
	_, err = svc.Create(tenantCtx(1), memberOfTenant(1), &model.CreateFormRequest{
		ApplicationID: 7, Name: "类型错误", FormType: model.FormType("unknown"),
	})
	assert.ErrorIs(t, err, apperrors.ErrFormTypeInvalid)

	// 归档应用拒绝
	_, err = svc.Create(tenantCtx(1), memberOfTenant(1), &model.CreateFormRequest{
		ApplicationID: 8, Name: "x", FormType: model.FormTypeStandard,
	})
	assert.ErrorIs(t, err, apperrors.ErrFormAppInvalid)

	// 配额超限
	quota.limit = 0
	_, err = svc.Create(tenantCtx(1), memberOfTenant(1), &model.CreateFormRequest{
		ApplicationID: 7, Name: "x", FormType: model.FormTypeStandard,
	})
	assert.ErrorIs(t, err, tenantservice.ErrQuotaExceeded)

	// 跨租户成员拒绝
	_, err = svc.Create(tenantCtx(1), memberOfTenant(2), &model.CreateFormRequest{
		ApplicationID: 7, Name: "x", FormType: model.FormTypeStandard,
	})
	assert.ErrorIs(t, err, apperrors.ErrForbidden)
}

// 指定分组创建：parentEntryCode 原样透传菜单维护端口（分组合法性由端口校验）
func TestCreateFormForwardsParentEntryCode(t *testing.T) {
	menu := &fakeMenuPort{}
	svc := newTestService(&fakeQuota{limit: -1}, newFakeFormRepo(), newFakeVersionRepo(), &fakeRecordRepo{}, menu)

	_, err := svc.Create(tenantCtx(1), memberOfTenant(1),
		&model.CreateFormRequest{
			ApplicationID: 7, Name: "分组表单", FormType: model.FormTypeWorkflow,
			ParentEntryCode: " menu_group ",
		})
	assert.NoError(t, err)
	assert.Len(t, menu.attached, 1)
	assert.Equal(t, "menu_group", menu.attached[0].parent)
}

func TestSaveDraft(t *testing.T) {
	formRepo := newFakeFormRepo()
	svc := newTestService(&fakeQuota{limit: -1}, formRepo, newFakeVersionRepo(), &fakeRecordRepo{}, nil)
	created, _ := svc.Create(tenantCtx(1), memberOfTenant(1), &model.CreateFormRequest{
		ApplicationID: 7, Name: "报名表", FormType: model.FormTypeStandard,
	})

	// 非法协议：携带 issues 数据负载
	_, err := svc.SaveDraft(tenantCtx(1), memberOfTenant(1), created.Code, &model.SaveDraftRequest{
		DraftRevision: created.DraftRevision,
		Content:       model.JSONContent(`{"content":{"type":"form","items":[{"widget":{"type":"text","widgetName":"_widget_a","enable":true,"visible":true,"allowBlank":true},"label":"","description":"","labelHidden":false,"lineWidth":12}]}}`),
	})
	assert.ErrorIs(t, err, apperrors.ErrSchemaInvalid)
	var biz *httpx.BizError
	assert.True(t, errors.As(err, &biz))
	assert.NotNil(t, biz.Data)

	// 合法保存：口令递增
	result, err := svc.SaveDraft(tenantCtx(1), memberOfTenant(1), created.Code, &model.SaveDraftRequest{
		DraftRevision: created.DraftRevision,
		Content:       validDraft(),
	})
	assert.NoError(t, err)
	assert.EqualValues(t, 2, result.DraftRevision)

	// 旧口令重放：409
	_, err = svc.SaveDraft(tenantCtx(1), memberOfTenant(1), created.Code, &model.SaveDraftRequest{
		DraftRevision: created.DraftRevision,
		Content:       validDraft(),
	})
	assert.ErrorIs(t, err, apperrors.ErrRevisionConflict)
}

func TestPublishAndRuntimeAndSubmit(t *testing.T) {
	formRepo := newFakeFormRepo()
	versionRepo := newFakeVersionRepo()
	recordRepo := &fakeRecordRepo{}
	svc := newTestService(&fakeQuota{limit: -1}, formRepo, versionRepo, recordRepo, nil)
	ctx := tenantCtx(1)
	member := memberOfTenant(1)

	created, _ := svc.Create(ctx, member, &model.CreateFormRequest{
		ApplicationID: 7, Name: "报名表", FormType: model.FormTypeStandard,
	})
	// 白名单外控件（user）拒绝发布并给出 issues
	_, err := svc.SaveDraft(ctx, member, created.Code, &model.SaveDraftRequest{
		DraftRevision: 1,
		Content:       model.JSONContent(`{"content":{"type":"form","items":[{"widget":{"type":"user","widgetName":"_widget_u","enable":true,"visible":true,"allowBlank":true},"label":"审批人","description":"","labelHidden":false,"lineWidth":12}]}}`),
	})
	assert.NoError(t, err)
	_, err = svc.Publish(ctx, member, created.Code, &model.PublishRequest{DraftRevision: 2})
	assert.ErrorIs(t, err, apperrors.ErrPublishUnsupportedField)

	// 换成基础字段后发布成功：双口令
	_, _ = svc.SaveDraft(ctx, member, created.Code, &model.SaveDraftRequest{DraftRevision: 2, Content: validDraft()})
	published, err := svc.Publish(ctx, member, created.Code, &model.PublishRequest{DraftRevision: 3})
	assert.NoError(t, err)
	assert.Equal(t, 1, published.PublishedVersion)
	assert.NotEmpty(t, published.SchemaRevision)

	// 快照不可变：再次发布产生新版本，旧快照原样
	_, _ = svc.SaveDraft(ctx, member, created.Code, &model.SaveDraftRequest{
		DraftRevision: 3,
		Content:       model.JSONContent(`{"content":{"type":"form","items":[{"widget":{"type":"text","widgetName":"_widget_a","enable":true,"visible":true,"allowBlank":true},"label":"姓名2","description":"","labelHidden":false,"lineWidth":6}]}}`),
	})
	published2, err := svc.Publish(ctx, member, created.Code, &model.PublishRequest{DraftRevision: 4})
	assert.NoError(t, err)
	assert.Equal(t, 2, published2.PublishedVersion)

	// bootstrap：最新版本；归属不符/未发布路径
	runtime, err := svc.GetRuntime(ctx, "app_x", created.Code)
	assert.NoError(t, err)
	assert.Equal(t, created.Code, runtime.FormCode)
	assert.Equal(t, 2, runtime.PublishedVersion)
	assert.Equal(t, published2.SchemaRevision, runtime.SchemaRevision)
	var content map[string]any
	assert.NoError(t, json.Unmarshal([]byte(runtime.Content), &content))
	items := content["content"].(map[string]any)["items"].([]any)
	assert.Equal(t, "姓名2", items[0].(map[string]any)["label"])

	_, err = svc.GetRuntime(ctx, "app_archived", created.Code)
	assert.ErrorIs(t, err, apperrors.ErrFormNotFound)

	// 提交：历史版本（v1）合法；版本口令不符 409
	result, err := svc.SubmitRecord(ctx, member, &model.SubmitRecordRequest{
		FormCode: created.Code, PublishedVersion: 1, SchemaRevision: published.SchemaRevision,
		Values: map[string]model.JSONContent{"_widget_a": model.JSONContent(`"张三"`)},
	})
	assert.NoError(t, err)
	assert.NotZero(t, result.RecordID)
	assert.Equal(t, formRepo.formByCode(created.Code).ID, recordRepo.records[0].FormID)

	_, err = svc.SubmitRecord(ctx, member, &model.SubmitRecordRequest{
		FormCode: created.Code, PublishedVersion: 1, SchemaRevision: "999",
		Values: map[string]model.JSONContent{"_widget_a": model.JSONContent(`"张三"`)},
	})
	assert.ErrorIs(t, err, apperrors.ErrVersionConflict)

	// v1 快照 allowBlank=false：空值提交回填字段错误
	_, err = svc.SubmitRecord(ctx, member, &model.SubmitRecordRequest{
		FormCode: created.Code, PublishedVersion: 1, SchemaRevision: published.SchemaRevision,
		Values: map[string]model.JSONContent{"_widget_a": model.JSONContent(`null`)},
	})
	assert.ErrorIs(t, err, apperrors.ErrRecordInvalid)
	var biz *httpx.BizError
	assert.True(t, errors.As(err, &biz))
	fieldErrors := biz.Data.(map[string]any)["fieldErrors"]
	assert.Contains(t, fieldErrors, "_widget_a")
}

func TestGetFormAndDelete(t *testing.T) {
	formRepo := newFakeFormRepo()
	svc := newTestService(&fakeQuota{limit: -1}, formRepo, newFakeVersionRepo(), &fakeRecordRepo{}, nil)
	ctx := tenantCtx(1)
	member := memberOfTenant(1)
	created, _ := svc.Create(ctx, member, &model.CreateFormRequest{
		ApplicationID: 7, Name: "报名表", FormType: model.FormTypeStandard,
	})

	detail, err := svc.Get(ctx, member, created.Code)
	assert.NoError(t, err)
	assert.Equal(t, "报名表", detail.Name)

	updated, err := svc.Update(ctx, member, created.Code, &model.UpdateFormRequest{Name: strPtr("新名称")})
	assert.NoError(t, err)
	assert.Equal(t, "新名称", updated.Name)

	assert.NoError(t, svc.Delete(ctx, member, created.Code))
	_, err = svc.Get(ctx, member, created.Code)
	assert.ErrorIs(t, err, apperrors.ErrFormNotFound)
}

// M2-资产-1：表单创建/改名/删除事务内联动菜单节点维护端口
func TestFormLifecycleMaintainsMenuEntries(t *testing.T) {
	formRepo := newFakeFormRepo()
	menu := &fakeMenuPort{}
	svc := newTestService(&fakeQuota{limit: -1}, formRepo, newFakeVersionRepo(), &fakeRecordRepo{}, menu)
	ctx := tenantCtx(1)
	member := memberOfTenant(1)

	created, err := svc.Create(ctx, member, &model.CreateFormRequest{
		ApplicationID: 7, Name: "报名表", FormType: model.FormTypeStandard,
	})
	assert.NoError(t, err)
	createdModel := formRepo.formByCode(created.Code)
	assert.NotNil(t, createdModel)
	assert.Len(t, menu.attached, 1)
	assert.Equal(t, uint(7), menu.attached[0].appID)
	assert.Equal(t, createdModel.ID, menu.attached[0].formID)
	assert.Equal(t, "报名表", menu.attached[0].name)
	// 未指定分组：按根级挂载（空 parentEntryCode）
	assert.Empty(t, menu.attached[0].parent)

	_, err = svc.Update(ctx, member, created.Code, &model.UpdateFormRequest{Name: strPtr("新名称")})
	assert.NoError(t, err)
	assert.Len(t, menu.renamed, 1)
	assert.Equal(t, "新名称", menu.renamed[0].name)

	assert.NoError(t, svc.Delete(ctx, member, created.Code))
	assert.Len(t, menu.detached, 1)
	assert.Equal(t, createdModel.ID, menu.detached[0].formID)
}

func strPtr(value string) *string { return &value }
