package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"evolyn/internal/contextx"
	kernel "evolyn/internal/model"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/workflow/model"
	"evolyn/internal/platform/workflow/repository"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	"workflows:create": true, "workflows:get": true,
	"workflows:update": true, "workflows:patch": true, "workflows:delete": true,
}

type fakeRecorder struct{ entries []auditservice.Entry }

func (f *fakeRecorder) Record(ctx context.Context, entry auditservice.Entry) {
	f.entries = append(f.entries, entry)
}

// fakeDefinitionRepo 内存桩：覆盖乐观锁与软删语义
type fakeDefinitionRepo struct {
	defs      map[uint]*model.WfDefinition
	next      uint
	createErr error
}

func newFakeDefinitionRepo() *fakeDefinitionRepo {
	return &fakeDefinitionRepo{defs: map[uint]*model.WfDefinition{}, next: 1}
}

func (f *fakeDefinitionRepo) Create(ctx context.Context, def *model.WfDefinition) (*model.WfDefinition, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	def.ID = f.next
	f.next++
	f.defs[def.ID] = def
	return def, nil
}

func (f *fakeDefinitionRepo) GetByCode(ctx context.Context, code string) (*model.WfDefinition, error) {
	for _, def := range f.defs {
		if def.Code == code && def.DeletedAt.Time.IsZero() {
			return def, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeDefinitionRepo) List(ctx context.Context, params repository.ListParams) ([]model.WfDefinition, bool, error) {
	rows := make([]model.WfDefinition, 0)
	for _, def := range f.defs {
		rows = append(rows, *def)
	}
	return rows, false, nil
}

func (f *fakeDefinitionRepo) UpdateMeta(ctx context.Context, id uint, name, description string) error {
	f.defs[id].Name = name
	f.defs[id].Description = description
	return nil
}

func (f *fakeDefinitionRepo) SaveDraft(ctx context.Context, id uint, fromRevision int64, content model.DSLContent) (bool, error) {
	def := f.defs[id]
	if def.DraftRevision != fromRevision {
		return false, nil
	}
	def.DraftContent = content
	def.DraftRevision = fromRevision + 1
	return true, nil
}

func (f *fakeDefinitionRepo) MarkPublished(ctx context.Context, id uint, versionID uint, versionNo int) error {
	f.defs[id].LatestVersionID = &versionID
	f.defs[id].PublishedVersion = versionNo
	return nil
}

func (f *fakeDefinitionRepo) SoftDelete(ctx context.Context, def *model.WfDefinition) error {
	delete(f.defs, def.ID)
	return nil
}

func (f *fakeDefinitionRepo) Migrate() error { return nil }

// fakeVersionRepo 内存桩：追加写，断言不可变语义
type fakeVersionRepo struct {
	versions map[uint][]*model.WfDefinitionVersion // definitionID → versions
	next     uint
}

func newFakeVersionRepo() *fakeVersionRepo {
	return &fakeVersionRepo{versions: map[uint][]*model.WfDefinitionVersion{}, next: 1}
}

func (f *fakeVersionRepo) MaxVersionNo(ctx context.Context, definitionID uint) (int, error) {
	max := 0
	for _, v := range f.versions[definitionID] {
		if v.VersionNo > max {
			max = v.VersionNo
		}
	}
	return max, nil
}

func (f *fakeVersionRepo) Create(ctx context.Context, version *model.WfDefinitionVersion) (*model.WfDefinitionVersion, error) {
	version.ID = f.next
	f.next++
	f.versions[version.DefinitionID] = append(f.versions[version.DefinitionID], version)
	return version, nil
}

func (f *fakeVersionRepo) GetByDefinitionAndVersionNo(ctx context.Context, definitionID uint, versionNo int) (*model.WfDefinitionVersion, error) {
	for _, v := range f.versions[definitionID] {
		if v.VersionNo == versionNo {
			return v, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeVersionRepo) ListByDefinition(ctx context.Context, definitionID uint) ([]model.WfDefinitionVersion, error) {
	rows := make([]model.WfDefinitionVersion, 0)
	for _, v := range f.versions[definitionID] {
		rows = append(rows, *v)
	}
	// 与真实仓储同口径：version_no 降序
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return rows, nil
}

func (f *fakeVersionRepo) Migrate() error { return nil }

// ---- 构造 ----

func newTestService(repo *fakeDefinitionRepo, versions *fakeVersionRepo, perms map[string]bool) DefinitionService {
	return NewDefinitionService(passThroughTx{}, repo, versions, fakeAccess{perms: perms}, &fakeRecorder{})
}

func adminMember() *iammodel.User {
	return &iammodel.User{
		ID: 11, Nickname: "管理员",
		TenantBaseModel: kernel.TenantBaseModel{TenantID: 1},
	}
}

func errCode(t *testing.T, err error) string {
	t.Helper()
	require.Error(t, err)
	var biz *httpx.BizError
	require.True(t, errors.As(err, &biz), "expected BizError, got %v", err)
	return biz.Code
}

// validDraftDoc 指定金额门槛的条件分支草稿（严格校验通过的完整 DSL）。
func validDraftDoc() string {
	return `{"schemaVersion":"1.0","nodes":[
		{"key":"start","type":"start","name":"发起"},
		{"key":"approval","type":"approval","name":"审批","config":{"approvalMode":"single","assignee":{"type":"user","userIds":[2]}}},
		{"key":"end","type":"end","name":"结束"}],
		"edges":[{"key":"e1","source":"start","target":"approval"},{"key":"e2","source":"approval","target":"end"}],
		"settings":{}}`
}

// ---- 用例 ----

func TestCreateWorkflow(t *testing.T) {
	repo, versions := newFakeDefinitionRepo(), newFakeVersionRepo()
	svc := newTestService(repo, versions, adminPerms)
	ctx := contextx.NewTenantContext(context.Background(), 1)

	detail, err := svc.Create(ctx, adminMember(), &model.CreateWorkflowRequest{Name: "报销审批", Description: "通用报销"})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(detail.Code, "wf_"), "code 必须为 wf_ 前缀")
	assert.Equal(t, int64(1), detail.DraftRevision)
	assert.Equal(t, 0, detail.PublishedVersion)

	// 草稿初值必须是最小合法 DSL（可直接发布）
	var doc map[string]any
	require.NoError(t, json.Unmarshal(detail.Draft, &doc))
	assert.Equal(t, "1.0", doc["schemaVersion"])
}

func TestCreateWorkflowValidation(t *testing.T) {
	repo, versions := newFakeDefinitionRepo(), newFakeVersionRepo()
	svc := newTestService(repo, versions, adminPerms)
	ctx := contextx.NewTenantContext(context.Background(), 1)

	// 名称校验
	_, err := svc.Create(ctx, adminMember(), &model.CreateWorkflowRequest{Name: "  "})
	assert.Equal(t, "WORKFLOW_NAME_INVALID", errCode(t, err))

	// 描述超长
	_, err = svc.Create(ctx, adminMember(), &model.CreateWorkflowRequest{
		Name: "报销", Description: strings.Repeat("长", 513),
	})
	assert.Equal(t, "WORKFLOW_DESCRIPTION_INVALID", errCode(t, err))

	// 无权限（空权限集）
	_, err = newTestService(repo, versions, map[string]bool{}).Create(ctx, adminMember(), &model.CreateWorkflowRequest{Name: "报销"})
	assert.Equal(t, "FORBIDDEN", errCode(t, err))
}

func TestCreateWorkflowMapsFormBindingUniqueViolation(t *testing.T) {
	repo, versions := newFakeDefinitionRepo(), newFakeVersionRepo()
	repo.createErr = &pgconn.PgError{Code: "23505", ConstraintName: "uk_wf_definition_form_code"}
	svc := newTestService(repo, versions, adminPerms)
	ctx := contextx.NewTenantContext(context.Background(), 1)

	_, err := svc.Create(ctx, adminMember(), &model.CreateWorkflowRequest{
		Name: "报销审批", FormCode: "form_1234567890abcdef",
	})
	assert.Equal(t, "WORKFLOW_FORM_ALREADY_BOUND", errCode(t, err))
}

func TestSaveDraftAndPublish(t *testing.T) {
	repo, versions := newFakeDefinitionRepo(), newFakeVersionRepo()
	svc := newTestService(repo, versions, adminPerms)
	ctx := contextx.NewTenantContext(context.Background(), 1)
	member := adminMember()

	detail, err := svc.Create(ctx, member, &model.CreateWorkflowRequest{Name: "报销审批"})
	require.NoError(t, err)

	// 保存合法草稿：口令递增
	result, err := svc.SaveDraft(ctx, member, detail.Code, &model.SaveDraftRequest{
		DraftRevision: detail.DraftRevision, Draft: json.RawMessage(validDraftDoc()),
	})
	require.NoError(t, err)
	assert.Equal(t, detail.DraftRevision+1, result.DraftRevision)

	// 发布：版本号 1，快照与草稿一致
	publish, err := svc.Publish(ctx, member, detail.Code, &model.PublishRequest{DraftRevision: result.DraftRevision})
	require.NoError(t, err)
	assert.Equal(t, 1, publish.VersionNo)

	// 再次修改发布：版本号 2
	result2, err := svc.SaveDraft(ctx, member, detail.Code, &model.SaveDraftRequest{
		DraftRevision: result.DraftRevision, Draft: json.RawMessage(validDraftDoc()),
	})
	require.NoError(t, err)
	publish2, err := svc.Publish(ctx, member, detail.Code, &model.PublishRequest{DraftRevision: result2.DraftRevision})
	require.NoError(t, err)
	assert.Equal(t, 2, publish2.VersionNo)

	// 历史版本不可变：v1 快照仍可读取且未受影响
	v1, err := svc.GetVersion(ctx, member, detail.Code, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, v1.VersionNo)
	var snapshot map[string]any
	require.NoError(t, json.Unmarshal(v1.DSL, &snapshot))
	assert.Equal(t, "1.0", snapshot["schemaVersion"])

	// 版本列表降序
	versionsList, err := svc.ListVersions(ctx, member, detail.Code)
	require.NoError(t, err)
	require.Len(t, versionsList, 2)
	assert.Equal(t, 2, versionsList[0].VersionNo)
}

func TestSaveDraftRevisionConflict(t *testing.T) {
	repo, versions := newFakeDefinitionRepo(), newFakeVersionRepo()
	svc := newTestService(repo, versions, adminPerms)
	ctx := contextx.NewTenantContext(context.Background(), 1)
	member := adminMember()

	detail, err := svc.Create(ctx, member, &model.CreateWorkflowRequest{Name: "报销审批"})
	require.NoError(t, err)

	// 口令过期（他人已保存）→ 409 稳定码
	_, err = svc.SaveDraft(ctx, member, detail.Code, &model.SaveDraftRequest{
		DraftRevision: detail.DraftRevision - 1, Draft: json.RawMessage(validDraftDoc()),
	})
	assert.Equal(t, "WORKFLOW_REVISION_CONFLICT", errCode(t, err))

	// 发布口令同样复核
	_, err = svc.Publish(ctx, member, detail.Code, &model.PublishRequest{DraftRevision: detail.DraftRevision + 99})
	assert.Equal(t, "WORKFLOW_REVISION_CONFLICT", errCode(t, err))
}

func TestSaveDraftInvalidDSL(t *testing.T) {
	repo, versions := newFakeDefinitionRepo(), newFakeVersionRepo()
	svc := newTestService(repo, versions, adminPerms)
	ctx := contextx.NewTenantContext(context.Background(), 1)
	member := adminMember()

	detail, err := svc.Create(ctx, member, &model.CreateWorkflowRequest{Name: "报销审批"})
	require.NoError(t, err)

	// 非法 DSL：重复节点 key + start 存在入边
	invalid := `{"schemaVersion":"1.0","nodes":[
		{"key":"start","type":"start","name":"发起"},
		{"key":"end","type":"end","name":"结束"},
		{"key":"end","type":"end","name":"结束2"}],
		"edges":[{"key":"e1","source":"start","target":"end"},{"key":"e2","source":"end","target":"start"}],
		"settings":{}}`
	_, err = svc.SaveDraft(ctx, member, detail.Code, &model.SaveDraftRequest{
		DraftRevision: detail.DraftRevision, Draft: json.RawMessage(invalid),
	})
	var biz *httpx.BizError
	require.ErrorAs(t, err, &biz)
	assert.Equal(t, "WORKFLOW_DEFINITION_INVALID", biz.Code)
	payload, ok := biz.Data.(map[string]any)
	require.True(t, ok, "issues 负载必须随错误返回")
	issues, ok := payload["issues"].([]map[string]string)
	require.True(t, ok)
	assert.NotEmpty(t, issues)
	assert.NotEmpty(t, issues[0]["path"])

	// 非法表达式：白名单外变量
	badExpr := `{"schemaVersion":"1.0","nodes":[
		{"key":"start","type":"start","name":"发起"},
		{"key":"cond","type":"condition","name":"条件"},
		{"key":"end","type":"end","name":"结束"}],
		"edges":[{"key":"e1","source":"start","target":"cond"},
			{"key":"e2","source":"cond","target":"end","condition":{"expression":"secret > 1"}},
			{"key":"e3","source":"cond","target":"end"}],
		"settings":{}}`
	_, err = svc.SaveDraft(ctx, member, detail.Code, &model.SaveDraftRequest{
		DraftRevision: detail.DraftRevision, Draft: json.RawMessage(badExpr),
	})
	require.ErrorAs(t, err, &biz)
	assert.Equal(t, "WORKFLOW_DEFINITION_INVALID", biz.Code)

	// 草稿未被污染（校验失败不落库）
	after, err := svc.Get(ctx, member, detail.Code)
	require.NoError(t, err)
	assert.Equal(t, detail.DraftRevision, after.DraftRevision)
}

func TestDeleteWorkflow(t *testing.T) {
	repo, versions := newFakeDefinitionRepo(), newFakeVersionRepo()
	svc := newTestService(repo, versions, adminPerms)
	ctx := contextx.NewTenantContext(context.Background(), 1)
	member := adminMember()

	detail, err := svc.Create(ctx, member, &model.CreateWorkflowRequest{Name: "报销审批"})
	require.NoError(t, err)
	require.NoError(t, svc.Delete(ctx, member, detail.Code))

	// 删除后不可见
	_, err = svc.Get(ctx, member, detail.Code)
	assert.Equal(t, "WORKFLOW_NOT_FOUND", errCode(t, err))

	// 无权限拒绝
	detail2, err := svc.Create(ctx, member, &model.CreateWorkflowRequest{Name: "第二个"})
	require.NoError(t, err)
	err = newTestService(repo, versions, map[string]bool{"workflows:get": true}).Delete(ctx, member, detail2.Code)
	assert.Equal(t, "FORBIDDEN", errCode(t, err))
}

func TestGetVersionNotFound(t *testing.T) {
	repo, versions := newFakeDefinitionRepo(), newFakeVersionRepo()
	svc := newTestService(repo, versions, adminPerms)
	ctx := contextx.NewTenantContext(context.Background(), 1)
	member := adminMember()

	detail, err := svc.Create(ctx, member, &model.CreateWorkflowRequest{Name: "报销审批"})
	require.NoError(t, err)
	_, err = svc.GetVersion(ctx, member, detail.Code, 1)
	assert.Equal(t, "WORKFLOW_VERSION_NOT_FOUND", errCode(t, err))
}
