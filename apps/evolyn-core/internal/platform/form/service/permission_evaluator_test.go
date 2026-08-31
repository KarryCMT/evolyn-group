// 权限组判定器与执行点单测（表单权限 P1，设计 §8/§11 测试要点）：
// 组绑定（S2 串联越权回归）、S4/S5/S7 语义、字段矩阵按操作维度隔离、
// 管理员旁路与配置面分离、主体命中（成员/部门含子部门/角色）、数据范围
// NULL 语义表与形状守卫、switch-type/发布阻塞、提交流程权限执行点。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	iammodel "evolyn/internal/platform/iam/model"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ---- 测试桩 ----

// fakePermissionGroupRepo 权限组仓储桩：按资产组织组行与主体行
type fakePermissionGroupRepo struct {
	groups   map[uint]*model.AssetPermissionGroup // 含禁用组（软删场景单测不覆盖，走集成）
	subjects []model.AssetPermissionGroupSubject
	nextID   uint
}

func newFakePermissionGroupRepo() *fakePermissionGroupRepo {
	return &fakePermissionGroupRepo{groups: map[uint]*model.AssetPermissionGroup{}, nextID: 900}
}

func (f *fakePermissionGroupRepo) addGroup(t *testing.T, group *model.AssetPermissionGroup, subjects []model.AssetPermissionGroupSubject) *model.AssetPermissionGroup {
	t.Helper()
	f.nextID++
	group.ID = f.nextID
	clone := *group
	f.groups[group.ID] = &clone
	for i := range subjects {
		subject := subjects[i]
		subject.GroupID = group.ID
		f.subjects = append(f.subjects, subject)
	}
	return &clone
}

func (f *fakePermissionGroupRepo) Create(ctx context.Context, group *model.AssetPermissionGroup) (*model.AssetPermissionGroup, error) {
	f.nextID++
	group.ID = f.nextID
	clone := *group
	f.groups[group.ID] = &clone
	return &clone, nil
}

func (f *fakePermissionGroupRepo) GetByCode(ctx context.Context, code string) (*model.AssetPermissionGroup, error) {
	for _, group := range f.groups {
		if group.Code == code {
			clone := *group
			return &clone, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakePermissionGroupRepo) ListByAsset(ctx context.Context, assetType string, assetID uint) ([]model.AssetPermissionGroup, error) {
	result := make([]model.AssetPermissionGroup, 0)
	for _, group := range f.groups {
		if group.AssetType == assetType && group.AssetID == assetID {
			result = append(result, *group)
		}
	}
	return result, nil
}

func (f *fakePermissionGroupRepo) CountByAsset(ctx context.Context, assetType string, assetID uint) (int64, error) {
	rows, err := f.ListByAsset(ctx, assetType, assetID)
	return int64(len(rows)), err
}

func (f *fakePermissionGroupRepo) ExistsByAssetIDs(ctx context.Context, assetType string, assetIDs []uint) (map[uint]bool, error) {
	existing := map[uint]bool{}
	for _, id := range assetIDs {
		for _, group := range f.groups {
			if group.AssetType == assetType && group.AssetID == id {
				existing[id] = true
				break
			}
		}
	}
	return existing, nil
}

func (f *fakePermissionGroupRepo) ListEnabledByAssetIDs(ctx context.Context, assetType string, assetIDs []uint) ([]model.AssetPermissionGroup, error) {
	result := make([]model.AssetPermissionGroup, 0)
	for _, id := range assetIDs {
		for _, group := range f.groups {
			if group.AssetType == assetType && group.AssetID == id && group.Enabled {
				result = append(result, *group)
			}
		}
	}
	return result, nil
}

func (f *fakePermissionGroupRepo) UpdateWithRevision(ctx context.Context, id uint, fromRevision int64, fields map[string]interface{}) (bool, error) {
	group, ok := f.groups[id]
	if !ok || group.Revision != fromRevision {
		return false, nil
	}
	for key, value := range fields {
		switch key {
		case "name":
			group.Name = value.(string)
		case "enabled":
			group.Enabled = value.(bool)
		case "operations":
			group.Operations = value.(model.PermissionOperations)
		case "field_permissions":
			group.FieldPermissions = value.(model.PermissionFieldRules)
		case "data_scope":
			group.DataScope = value.(model.PermissionDataScopeValue)
		}
	}
	group.Revision = fromRevision + 1
	return true, nil
}

func (f *fakePermissionGroupRepo) SoftDelete(ctx context.Context, group *model.AssetPermissionGroup) error {
	delete(f.groups, group.ID)
	return nil
}

func (f *fakePermissionGroupRepo) ReplaceSubjects(ctx context.Context, groupID uint, subjects []model.AssetPermissionGroupSubject) error {
	kept := make([]model.AssetPermissionGroupSubject, 0)
	for _, subject := range f.subjects {
		if subject.GroupID != groupID {
			kept = append(kept, subject)
		}
	}
	f.subjects = kept
	f.subjects = append(f.subjects, subjects...)
	return nil
}

func (f *fakePermissionGroupRepo) DeleteSubjectsByGroupIDs(ctx context.Context, groupIDs []uint) error {
	kept := make([]model.AssetPermissionGroupSubject, 0)
	for _, subject := range f.subjects {
		removed := false
		for _, id := range groupIDs {
			if subject.GroupID == id {
				removed = true
				break
			}
		}
		if !removed {
			kept = append(kept, subject)
		}
	}
	f.subjects = kept
	return nil
}

func (f *fakePermissionGroupRepo) ListSubjectsByGroupIDs(ctx context.Context, groupIDs []uint) ([]model.AssetPermissionGroupSubject, error) {
	result := make([]model.AssetPermissionGroupSubject, 0)
	for _, subject := range f.subjects {
		for _, id := range groupIDs {
			if subject.GroupID == id {
				result = append(result, subject)
				break
			}
		}
	}
	return result, nil
}

func (f *fakePermissionGroupRepo) Migrate() error { return nil }

// fakeSubjectSource 主体解析桩：成员直系部门/角色集合直出
type fakeSubjectSource struct {
	departments map[uint][]uint // memberID → 直系部门（不展开祖先：祖先展开属装配适配层职责）
	roles       map[uint][]uint
}

func (f fakeSubjectSource) MemberSubject(ctx context.Context, memberID uint) ([]uint, []uint, error) {
	return f.departments[memberID], f.roles[memberID], nil
}

// fakePermissionAccess 权限集桩（区分数据面旁路与配置面键）
type fakePermissionAccess struct{ perms map[string]bool }

func (f fakePermissionAccess) Permissions(ctx context.Context, member *iammodel.User) map[string]bool {
	return f.perms
}

// ---- 快照文档构造助手 ----

const permTestDoc = `{"content":{"type":"form","layout":"grid-2","items":[
  {"widget":{"type":"text","widgetName":"name","visible":true,"allowBlank":false},"label":"姓名"},
  {"widget":{"type":"number","widgetName":"amount","visible":true,"allowBlank":true},"label":"金额"},
  {"widget":{"type":"text","widgetName":"secret","visible":true,"allowBlank":true},"label":"密级"},
  {"widget":{"type":"datetime","widgetName":"created_day","visible":true,"allowBlank":true,"format":"date"},"label":"日期"},
  {"widget":{"type":"checkboxgroup","widgetName":"tags","visible":true,"allowBlank":true,"options":[{"value":"a","label":"A"},{"value":"b","label":"B"}]},"label":"标签"}
]}}`

func permTestDocMap(t *testing.T) map[string]any {
	t.Helper()
	root := map[string]any{}
	assert.NoError(t, json.Unmarshal([]byte(permTestDoc), &root))
	return root
}

func permFieldList(t *testing.T) []permissionFieldMeta {
	t.Helper()
	list, err := buildPermissionFieldList(permTestDocMap(t))
	assert.NoError(t, err)
	return list
}

func permGroup(ops []string, rules []model.PermissionFieldRule, scope model.PermissionDataScopeSpec) *model.AssetPermissionGroup {
	return &model.AssetPermissionGroup{
		AssetType:        model.PermissionAssetTypeForm,
		AssetID:          1, // 测试表单统一 ID=1
		Code:             "fpg_test",
		Enabled:          true,
		Operations:       model.PermissionOperations(ops),
		FieldPermissions: model.PermissionFieldRules(rules),
		DataScope:        model.PermissionDataScopeValue(scope),
	}
}

func memberSubject(id uint, subjectType string, subjectID uint) model.AssetPermissionGroupSubject {
	return model.AssetPermissionGroupSubject{SubjectType: subjectType, SubjectID: subjectID}
}

// newPermTestEvaluator 构造判定器 + 依赖桩
type permTestEnv struct {
	groups    *fakePermissionGroupRepo
	forms     *fakeFormRepo
	versions  *fakeVersionRepo
	source    *fakeSubjectSource
	evaluator FormPermissionEvaluator
}

func newPermTestEvaluator(t *testing.T, access map[string]bool) *permTestEnv {
	t.Helper()
	env := &permTestEnv{
		groups:   newFakePermissionGroupRepo(),
		forms:    newFakeFormRepo(),
		versions: newFakeVersionRepo(),
		source:   &fakeSubjectSource{departments: map[uint][]uint{}, roles: map[uint][]uint{}},
	}
	env.evaluator = NewFormPermissionEvaluator(
		env.groups, env.forms, env.versions, fakePermissionAccess{perms: access}, env.source)
	return env
}

func (e *permTestEnv) addForm(t *testing.T, id uint, doc string) *model.Form {
	t.Helper()
	form := &model.Form{Code: "form_test", FormType: model.FormTypeStandard, DraftContent: model.JSONContent(doc)}
	form.ID = id
	form.TenantID = 1
	clone := *form
	e.forms.forms[id] = &clone
	return &clone
}

func (e *permTestEnv) publishSnapshot(t *testing.T, formID uint, doc string) {
	t.Helper()
	version := &model.FormVersion{FormID: formID, VersionNo: 1, Content: model.JSONContent(doc)}
	created, err := e.versions.Create(context.Background(), version)
	assert.NoError(t, err)
	form := e.forms.forms[formID]
	vid := created.ID
	form.LatestVersionID = &vid
}

// ---- 判定器语义 ----

// S4：表单不存在任何权限组行 → Baseline 放行，字段全量
func TestPermissionEvaluatorBaseline(t *testing.T) {
	env := newPermTestEvaluator(t, map[string]bool{})
	env.addForm(t, 1, permTestDoc)

	resolved, err := env.evaluator.Evaluate(context.Background(), &iammodel.User{ID: 7}, 1)
	assert.NoError(t, err)
	assert.True(t, resolved.Baseline)
	assert.False(t, resolved.Admin)
	assert.True(t, resolved.EntranceAllowed())
	assert.True(t, resolved.AllowsNewRecord(model.PermissionOpAdd))
	fields := resolved.FieldsForNew(model.PermissionOpView)
	assert.True(t, fields["name"].Visible && fields["name"].Editable)
	assert.True(t, resolved.AllowsOperation(model.PermissionOpDelete, map[string]any{"name": "x"}))
}

// S5：存在启用组但成员未命中任何主体 → 全量拒绝（含查看），菜单入口关闭
func TestPermissionEvaluatorNoMatchDenied(t *testing.T) {
	env := newPermTestEvaluator(t, map[string]bool{})
	env.addForm(t, 1, permTestDoc)
	env.groups.addGroup(t, permGroup([]string{model.PermissionOpView, model.PermissionOpAdd}, nil, model.PermissionDataScopeSpec{}),
		[]model.AssetPermissionGroupSubject{memberSubject(0, model.PermissionSubjectMember, 42)})

	resolved, err := env.evaluator.Evaluate(context.Background(), &iammodel.User{ID: 7}, 1)
	assert.NoError(t, err)
	assert.False(t, resolved.Baseline)
	assert.False(t, resolved.EntranceAllowed())
	assert.False(t, resolved.AllowsNewRecord(model.PermissionOpAdd))
	fields := resolved.FieldsForNew(model.PermissionOpView)
	assert.False(t, fields["name"].Visible)
}

// S5：禁用组同样维持收口——成员是禁用组主体，仍按未命中处理
func TestPermissionEvaluatorDisabledGroupStillLocks(t *testing.T) {
	env := newPermTestEvaluator(t, map[string]bool{})
	env.addForm(t, 1, permTestDoc)
	group := permGroup([]string{model.PermissionOpView}, nil, model.PermissionDataScopeSpec{})
	group.Enabled = false
	env.groups.addGroup(t, group, []model.AssetPermissionGroupSubject{memberSubject(0, model.PermissionSubjectMember, 7)})

	resolved, err := env.evaluator.Evaluate(context.Background(), &iammodel.User{ID: 7}, 1)
	assert.NoError(t, err)
	assert.False(t, resolved.Baseline, "禁用组行存在即收口")
	assert.False(t, resolved.EntranceAllowed(), "禁用组不授权：成员不可查看")
}

// S2 串联越权回归：A 组编辑 X 范围 + B 组查看 Y 范围 → 不得编辑 Y
func TestPermissionEvaluatorGroupBindingNoChain(t *testing.T) {
	env := newPermTestEvaluator(t, map[string]bool{})
	env.addForm(t, 1, permTestDoc)
	// A 组：edit + 数据范围 amount >= 1000；字段矩阵 name 可编辑
	groupA := permGroup([]string{model.PermissionOpEdit}, []model.PermissionFieldRule{{Field: "name", Visible: true, Editable: true}},
		model.PermissionDataScopeSpec{Match: model.PermissionScopeMatchAll, Conditions: []model.PermissionDataCondition{
			{Field: "amount", Operator: "gte", Value: []any{float64(1000)}},
		}})
	// B 组：view + 数据范围 amount < 1000
	groupB := permGroup([]string{model.PermissionOpView}, nil,
		model.PermissionDataScopeSpec{Match: model.PermissionScopeMatchAll, Conditions: []model.PermissionDataCondition{
			{Field: "amount", Operator: "lt", Value: []any{float64(1000)}},
		}})
	env.groups.addGroup(t, groupA, []model.AssetPermissionGroupSubject{memberSubject(0, model.PermissionSubjectMember, 7)})
	env.groups.addGroup(t, groupB, []model.AssetPermissionGroupSubject{memberSubject(0, model.PermissionSubjectMember, 7)})

	resolved, err := env.evaluator.Evaluate(context.Background(), &iammodel.User{ID: 7}, 1)
	assert.NoError(t, err)
	yRecord := map[string]any{"amount": float64(100)} // 仅 B 组范围
	assert.True(t, resolved.AllowsOperation(model.PermissionOpView, yRecord))
	assert.False(t, resolved.AllowsOperation(model.PermissionOpEdit, yRecord), "串联越权：B 组查看范围不得借 A 组 edit 放行")
	fields := resolved.FieldsFor(model.PermissionOpEdit, yRecord)
	assert.False(t, fields["name"].Editable, "字段矩阵同样按组绑定：Y 记录下 edit 不可编辑")
	assert.False(t, fields["name"].Visible)

	xRecord := map[string]any{"amount": float64(2000)} // A 组范围
	assert.True(t, resolved.AllowsOperation(model.PermissionOpEdit, xRecord))
	xf := resolved.FieldsFor(model.PermissionOpEdit, xRecord)
	assert.True(t, xf["name"].Editable)
}

// 字段矩阵按操作维度隔离：view 组与 add 组矩阵不同，FieldsForNew 不混用
func TestPermissionEvaluatorFieldsPerOperation(t *testing.T) {
	env := newPermTestEvaluator(t, map[string]bool{})
	env.addForm(t, 1, permTestDoc)
	viewOnly := permGroup([]string{model.PermissionOpView}, []model.PermissionFieldRule{
		{Field: "name", Visible: true, Editable: false},
		{Field: "secret", Visible: true, Editable: false},
	}, model.PermissionDataScopeSpec{})
	addOnly := permGroup([]string{model.PermissionOpAdd}, []model.PermissionFieldRule{
		{Field: "name", Visible: true, Editable: true},
	}, model.PermissionDataScopeSpec{})
	env.groups.addGroup(t, viewOnly, []model.AssetPermissionGroupSubject{memberSubject(0, model.PermissionSubjectMember, 7)})
	env.groups.addGroup(t, addOnly, []model.AssetPermissionGroupSubject{memberSubject(0, model.PermissionSubjectMember, 7)})

	resolved, err := env.evaluator.Evaluate(context.Background(), &iammodel.User{ID: 7}, 1)
	assert.NoError(t, err)
	assert.True(t, resolved.EntranceAllowed())

	viewFields := resolved.FieldsForNew(model.PermissionOpView)
	assert.True(t, viewFields["name"].Visible)
	assert.False(t, viewFields["name"].Editable)
	assert.True(t, viewFields["secret"].Visible, "view 组矩阵内字段随组生效")
	assert.False(t, viewFields["amount"].Visible, "S7：矩阵未覆盖字段 deny-by-default")

	addFields := resolved.FieldsForNew(model.PermissionOpAdd)
	assert.True(t, addFields["name"].Editable)
	assert.False(t, addFields["secret"].Visible)
	assert.False(t, addFields["secret"].Editable)
}

// S7 deny-by-default：发布新增字段默认不可见；矩阵中已删字段条目被忽略
func TestPermissionEvaluatorDenyByDefaultNewField(t *testing.T) {
	env := newPermTestEvaluator(t, map[string]bool{})
	env.addForm(t, 1, permTestDoc)
	env.publishSnapshot(t, 1, permTestDoc)
	group := permGroup([]string{model.PermissionOpView}, []model.PermissionFieldRule{
		{Field: "name", Visible: true, Editable: false},
		{Field: "ghost", Visible: true, Editable: true}, // 已不在快照的字段条目
	}, model.PermissionDataScopeSpec{})
	env.groups.addGroup(t, group, []model.AssetPermissionGroupSubject{memberSubject(0, model.PermissionSubjectMember, 7)})

	resolved, err := env.evaluator.Evaluate(context.Background(), &iammodel.User{ID: 7}, 1)
	assert.NoError(t, err)
	fields := resolved.FieldsForNew(model.PermissionOpView)
	assert.True(t, fields["name"].Visible)
	assert.False(t, fields["amount"].Visible, "清单内矩阵未覆盖字段默认拒绝")
	_, hasGhost := fields["ghost"]
	assert.False(t, hasGhost, "清单外矩阵条目被忽略")
}

// 主体命中：直接成员 / 部门（含子部门）/ 角色三种类型
func TestPermissionEvaluatorSubjectMatching(t *testing.T) {
	env := newPermTestEvaluator(t, map[string]bool{})
	env.addForm(t, 1, permTestDoc)
	// 部门组：配置父部门 10（成员所在子部门 11 的祖先）
	deptGroup := permGroup([]string{model.PermissionOpView}, nil, model.PermissionDataScopeSpec{})
	env.groups.addGroup(t, deptGroup, []model.AssetPermissionGroupSubject{memberSubject(0, model.PermissionSubjectDepartment, 10)})
	// 角色组：配置角色 5
	roleGroup := permGroup([]string{model.PermissionOpAdd}, nil, model.PermissionDataScopeSpec{})
	env.groups.addGroup(t, roleGroup, []model.AssetPermissionGroupSubject{memberSubject(0, model.PermissionSubjectRole, 5)})
	// 部门组配置为成员祖先链：source 桩直出「直系 ∪ 祖先」语义（装配层职责）
	env.source.departments[7] = []uint{10, 11}
	env.source.roles[7] = []uint{5, 9}

	resolved, err := env.evaluator.Evaluate(context.Background(), &iammodel.User{ID: 7}, 1)
	assert.NoError(t, err)
	assert.True(t, resolved.EntranceAllowed())
	assert.True(t, resolved.AllowsNewRecord(model.PermissionOpAdd))

	// 直系成员命中
	direct := permGroup([]string{model.PermissionOpEdit}, nil, model.PermissionDataScopeSpec{})
	env.groups.addGroup(t, direct, []model.AssetPermissionGroupSubject{memberSubject(0, model.PermissionSubjectMember, 7)})
	resolved, err = env.evaluator.Evaluate(context.Background(), &iammodel.User{ID: 7}, 1)
	assert.NoError(t, err)
	assert.True(t, resolved.AllowsOperation(model.PermissionOpEdit, nil))
}

// S3 管理员旁路：仅认 form-data:admin；form-permissions:* 不触发旁路
func TestPermissionEvaluatorAdminBypassKeys(t *testing.T) {
	env := newPermTestEvaluator(t, map[string]bool{"form-data:admin": true})
	env.addForm(t, 1, permTestDoc)
	group := permGroup([]string{model.PermissionOpView}, nil, model.PermissionDataScopeSpec{})
	env.groups.addGroup(t, group, []model.AssetPermissionGroupSubject{memberSubject(0, model.PermissionSubjectMember, 999)})

	resolved, err := env.evaluator.Evaluate(context.Background(), &iammodel.User{ID: 7}, 1)
	assert.NoError(t, err)
	assert.True(t, resolved.Admin, "form-data:admin 触发数据面旁路")
	assert.True(t, resolved.EntranceAllowed())
	fields := resolved.FieldsForNew(model.PermissionOpView)
	assert.True(t, fields["secret"].Visible && fields["secret"].Editable)

	// 持配置面权限键（即使全量）不触发旁路
	envNoBypass := newPermTestEvaluator(t, map[string]bool{
		"form-permissions:list": true, "form-permissions:create": true,
		"form-permissions:update": true, "form-permissions:delete": true,
	})
	envNoBypass.addForm(t, 1, permTestDoc)
	envNoBypass.groups.addGroup(t, group, []model.AssetPermissionGroupSubject{memberSubject(0, model.PermissionSubjectMember, 999)})
	resolved, err = envNoBypass.evaluator.Evaluate(context.Background(), &iammodel.User{ID: 7}, 1)
	assert.NoError(t, err)
	assert.False(t, resolved.Admin)
	assert.False(t, resolved.EntranceAllowed())
}

// ---- 数据范围内存匹配（§5.2 NULL 语义表） ----

func TestPermissionScopeMatchingNullSemantics(t *testing.T) {
	fieldList := permFieldList(t)
	condition := func(operator string, value []any) model.PermissionDataCondition {
		return model.PermissionDataCondition{Field: "name", Operator: operator, Value: value}
	}
	matchOne := func(operator string, value []any, record map[string]any) bool {
		scope := model.PermissionDataScopeSpec{Match: model.PermissionScopeMatchAll,
			Conditions: []model.PermissionDataCondition{condition(operator, value)}}
		return permissionScopeMatches(scope, fieldList, record)
	}

	assert.True(t, matchOne("eq", []any{"张三"}, map[string]any{"name": "张三"}))
	assert.False(t, matchOne("eq", []any{"李四"}, map[string]any{"name": "张三"}))
	// ne 三值逻辑：NULL 不命中
	assert.False(t, matchOne("ne", []any{"李四"}, map[string]any{}), "缺键 → NULL，ne 不命中")
	assert.False(t, matchOne("ne", []any{"李四"}, map[string]any{"name": nil}), "JSON null → NULL，ne 不命中")
	assert.True(t, matchOne("ne", []any{"李四"}, map[string]any{"name": "张三"}))
	// empty 覆盖：缺键、JSON null、空串；用户文本 "null" 不视为空
	assert.True(t, matchOne("empty", []any{}, map[string]any{}))
	assert.True(t, matchOne("empty", []any{}, map[string]any{"name": nil}))
	assert.True(t, matchOne("empty", []any{}, map[string]any{"name": ""}))
	assert.False(t, matchOne("empty", []any{}, map[string]any{"name": "null"}))
	assert.True(t, matchOne("not_empty", []any{}, map[string]any{"name": "null"}))
	// not_in 显式命中 NULL
	assert.True(t, matchOne("not_in", []any{"x"}, map[string]any{}), "not_in 对 NULL 命中（显式 IS NULL OR 模板）")
	assert.False(t, matchOne("in", []any{"x"}, map[string]any{}))
	assert.True(t, matchOne("in", []any{"a", "b"}, map[string]any{"name": "b"}))
	// 数值守卫：数字字段脏值 → NULL；合法数字（float/字符串形态）参与比较
	numMatch := func(operator string, value []any, record map[string]any) bool {
		scope := model.PermissionDataScopeSpec{Conditions: []model.PermissionDataCondition{
			{Field: "amount", Operator: operator, Value: value}}}
		return permissionScopeMatches(scope, fieldList, record)
	}
	assert.True(t, numMatch("gte", []any{float64(100)}, map[string]any{"amount": float64(150)}))
	assert.True(t, numMatch("gte", []any{float64(100)}, map[string]any{"amount": "150"}), "数字字段字符串形态过守卫后可比较")
	assert.False(t, numMatch("gte", []any{float64(100)}, map[string]any{"amount": "abc"}), "脏值守卫 → NULL → 不命中")
	assert.True(t, numMatch("empty", []any{}, map[string]any{"amount": "abc"}), "脏值按 NULL 语义 empty 命中")
	// 日期形状守卫：date 形状定宽文本比较；跨形状脏值 → NULL
	dayMatch := func(operator string, value []any, record map[string]any) bool {
		scope := model.PermissionDataScopeSpec{Conditions: []model.PermissionDataCondition{
			{Field: "created_day", Operator: operator, Value: value}}}
		return permissionScopeMatches(scope, fieldList, record)
	}
	assert.True(t, dayMatch("gt", []any{"2026-01-01"}, map[string]any{"created_day": "2026-06-15"}), "定宽文本字典序 = 时间序")
	assert.False(t, dayMatch("gt", []any{"2026-01-01"}, map[string]any{"created_day": "2025-12-31"}))
	assert.False(t, dayMatch("eq", []any{"2026-01-01"}, map[string]any{"created_day": "2026/01/01"}), "形状不符守卫 → NULL")
	// 多选：contains / in / not_in 数组语义与标量数组兼容
	tagMatch := func(operator string, value []any, record map[string]any) bool {
		scope := model.PermissionDataScopeSpec{Conditions: []model.PermissionDataCondition{
			{Field: "tags", Operator: operator, Value: value}}}
		return permissionScopeMatches(scope, fieldList, record)
	}
	assert.True(t, tagMatch("contains", []any{"a"}, map[string]any{"tags": []any{"a", "b"}}))
	assert.False(t, tagMatch("contains", []any{"c"}, map[string]any{"tags": []any{"a", "b"}}))
	assert.True(t, tagMatch("in", []any{"c", "b"}, map[string]any{"tags": []any{"b"}}))
	assert.False(t, tagMatch("not_in", []any{"c", "b"}, map[string]any{"tags": []any{"b"}}))
	assert.True(t, tagMatch("not_in", []any{"c"}, map[string]any{"tags": []any{"a"}}))
	assert.True(t, tagMatch("empty", []any{}, map[string]any{"tags": []any{}}), "空数组 empty 命中")
	assert.False(t, tagMatch("not_empty", []any{}, map[string]any{"tags": []any{}}))
	// match=any：任一条件命中即命中
	anyScope := model.PermissionDataScopeSpec{Match: model.PermissionScopeMatchAny, Conditions: []model.PermissionDataCondition{
		{Field: "name", Operator: "eq", Value: []any{"李四"}},
		{Field: "name", Operator: "eq", Value: []any{"张三"}},
	}}
	assert.True(t, permissionScopeMatches(anyScope, fieldList, map[string]any{"name": "张三"}))
	// 空条件 = 全部数据（S6）
	assert.True(t, permissionScopeMatches(model.PermissionDataScopeSpec{}, fieldList, nil))
}

// ---- switch-type / 发布阻塞（§3.3/§5.2） ----

// fakeReadSource 权限组只读查询端口桩（switch-type/发布阻塞判定）
type fakeReadSource struct {
	hasWorkflow bool
	fields      map[string]bool
}

func (f *fakeReadSource) HasWorkflowOperations(ctx context.Context, formID uint) (bool, error) {
	return f.hasWorkflow, nil
}

func (f *fakeReadSource) EnabledDataScopeFields(ctx context.Context, formID uint) (map[string]bool, error) {
	if f.fields == nil {
		return map[string]bool{}, nil
	}
	return f.fields, nil
}

// §3.3：workflow → standard 方向存在含流程操作的权限组（含禁用组）→ 拒绝；
// 放行方向与无阻塞场景不受影响
func TestSwitchTypeBlockedByWorkflowOperations(t *testing.T) {
	member := memberOfTenant(1)
	formRepo := newFakeFormRepo()
	form := &model.Form{Code: "form_test", FormType: model.FormTypeWorkflow, ApplicationID: 1}
	form.ID = 1
	form.TenantID = 1
	formRepo.forms[1] = form
	svc := &formService{
		tx:     passThroughTx{},
		repo:   formRepo,
		access: fakeAccess{perms: map[string]bool{"forms:create": true, "form-actions:switch-type": true}},
		apps:   fakeApps{apps: map[uint]string{1: "active"}, codeToID: map[string]uint{"app_x": 1}},
	}
	source := &fakeReadSource{hasWorkflow: true}
	svc.UsePermissionGroupSource(source)

	_, err := svc.SwitchType(tenantCtx(1), member, "form_test", &model.SwitchFormTypeRequest{FormType: model.FormTypeStandard})
	assert.ErrorIs(t, err, apperrors.ErrPermissionBlockedTypeSwitch)

	// 无阻塞时 workflow → standard 正常放行（执行切换）
	source.hasWorkflow = false
	updated, err := svc.SwitchType(tenantCtx(1), member, "form_test", &model.SwitchFormTypeRequest{FormType: model.FormTypeStandard})
	assert.NoError(t, err)
	assert.Equal(t, model.FormTypeStandard, updated.FormType)
}

// §5.2 发布阻塞：新版删除/变更的字段被启用组 data_scope 引用 → 阻塞并列出
// 冲突字段；未被引用/引用已清理（禁用组不参与）时不阻塞
func TestPublishBlockedByDataScopeFieldLifecycle(t *testing.T) {
	formRepo := newFakeFormRepo()
	form := &model.Form{
		Code: "form_test", FormType: model.FormTypeStandard, ApplicationID: 1,
		DraftRevision: 2, ProtocolVersion: model.CurrentProtocolVersion,
	}
	form.ID = 1
	form.TenantID = 1
	formRepo.forms[1] = form
	versionRepo := newFakeVersionRepo()
	// 当前发布版含 amount/secret；新草稿删除 amount、secret 改类型
	oldDoc := `{"content":{"type":"form","layout":"grid-2","items":[
		{"widget":{"type":"number","widgetName":"amount","visible":true,"allowBlank":true},"label":"金额"},
		{"widget":{"type":"text","widgetName":"secret","visible":true,"allowBlank":true},"label":"密级"},
		{"widget":{"type":"text","widgetName":"name","visible":true,"allowBlank":true},"label":"姓名"}]}}`
	version := &model.FormVersion{FormID: 1, VersionNo: 1, Content: model.JSONContent(oldDoc)}
	created, err := versionRepo.Create(tenantCtx(1), version)
	assert.NoError(t, err)
	vid := created.ID
	form.LatestVersionID = &vid
	// 草稿：删除 amount、secret 类型 text→number（形状变更）
	newDoc := `{"content":{"type":"form","layout":"grid-2","items":[
		{"widget":{"type":"number","widgetName":"secret","visible":true,"allowBlank":true},"label":"密级"},
		{"widget":{"type":"text","widgetName":"name","visible":true,"allowBlank":true},"label":"姓名"}]}}`
	form.DraftContent = model.JSONContent(newDoc)

	svc := &formService{
		tx:       passThroughTx{},
		repo:     formRepo,
		versions: versionRepo,
		access:   fakeAccess{perms: map[string]bool{"forms:create": true}},
		apps:     fakeApps{apps: map[uint]string{1: "active"}, codeToID: map[string]uint{"app_x": 1}},
	}
	source := &fakeReadSource{fields: map[string]bool{"amount": true, "secret": true}}
	svc.UsePermissionGroupSource(source)

	blocked, fields, err := svc.publishBlockedFields(tenantCtx(1), form)
	assert.NoError(t, err)
	assert.True(t, blocked)
	assert.ElementsMatch(t, []string{"amount", "secret"}, fields, "删除与类型变更字段均列入冲突清单")

	// 引用清理（仅剩余未引用字段）后放行
	source.fields = map[string]bool{"name": true}
	blocked, fields, err = svc.publishBlockedFields(tenantCtx(1), form)
	assert.NoError(t, err)
	assert.False(t, blocked)
	assert.Nil(t, fields)

	// 首次发布（无当前发布版）不阻塞
	form.LatestVersionID = nil
	blocked, _, err = svc.publishBlockedFields(tenantCtx(1), form)
	assert.NoError(t, err)
	assert.False(t, blocked)
}

// fakePermEvaluator 判定器桩：直出构造好的判定结果
type fakePermEvaluator struct{ resolved *ResolvedFormPermission }

func (f fakePermEvaluator) Evaluate(ctx context.Context, member *iammodel.User, formID uint) (*ResolvedFormPermission, error) {
	return f.resolved, nil
}

func (f fakePermEvaluator) EvaluateForForms(ctx context.Context, member *iammodel.User, formIDs []uint) (map[uint]*ResolvedFormPermission, error) {
	result := map[uint]*ResolvedFormPermission{}
	for _, id := range formIDs {
		result[id] = f.resolved
	}
	return result, nil
}

// 提交执行点（S8/S5）：无 add 组 → FORM_PERMISSION_DENIED；命中 add 组时
// 走权限感知管线（不可编辑字段携带数据整体拒绝；权限隐藏字段空 data 通过）
func TestSubmitRecordPermissionExecution(t *testing.T) {
	// 信封协议要求全部数据字段显式携带可见状态；entries 仅覆盖携带数据的字段
	submitValues := func(entries map[string]string) map[string]model.SubmitFieldValue {
		values := map[string]model.SubmitFieldValue{}
		for _, field := range []string{"name", "amount", "secret", "created_day", "tags"} {
			values[field] = model.SubmitFieldValue{Visible: submitBool(true)}
		}
		for name, data := range entries {
			values[name] = model.SubmitFieldValue{Data: model.JSONContent(data), Visible: submitBool(true)}
		}
		return values
	}
	member := memberOfTenant(1)
	formRepo := newFakeFormRepo()
	versionRepo := newFakeVersionRepo()
	recordRepo := &fakeRecordRepo{}
	form := &model.Form{
		Code: "form_test", FormType: model.FormTypeStandard, ApplicationID: 1,
		DraftRevision: 1, ProtocolVersion: model.CurrentProtocolVersion,
		DraftContent: model.JSONContent(permTestDoc),
	}
	form.ID = 1
	form.TenantID = 1
	formRepo.forms[1] = form
	published, err := publishForTest(tenantCtx(1), formRepo, versionRepo, form, permTestDoc)
	assert.NoError(t, err)

	svc := &formService{
		tx:       passThroughTx{},
		repo:     formRepo,
		versions: versionRepo,
		records:  recordRepo,
		access:   fakeAccess{perms: map[string]bool{"form-records:create": true}},
		apps:     fakeApps{apps: map[uint]string{1: "active"}, codeToID: map[string]uint{"app_x": 1}},
	}
	// S5 收口场景：组存在但成员未命中（Matched 空）→ 提交拒绝
	svc.UsePermissionEvaluator(fakePermEvaluator{resolved: &ResolvedFormPermission{
		fieldList: permFieldList(t),
	}})
	_, err = svc.SubmitRecord(tenantCtx(1), member, &model.SubmitRecordRequest{
		AppCode: "app_x", FormCode: "form_test", PublishedVersion: 1,
		SchemaRevision: published.SchemaRevision, HasResult: submitBool(true),
		DataOpID: "6e243bbb-7d57-4e59-952b-d530c53c6561",
		Values:   submitValues(map[string]string{`"name": `: ``}),
	})
	assert.ErrorIs(t, err, apperrors.ErrPermissionDenied)

	// 命中 add 组：矩阵仅 name/amount 可编辑，secret 不可编辑
	fields := map[string]FieldPermission{
		"name": {Visible: true, Editable: true}, "amount": {Visible: true, Editable: true},
		"secret": {}, "created_day": {}, "tags": {},
	}
	svc.UsePermissionEvaluator(fakePermEvaluator{resolved: &ResolvedFormPermission{
		Matched:   []MatchedGroup{{Code: "fpg_a", Operations: map[string]bool{model.PermissionOpAdd: true}, Fields: fields}},
		fieldList: permFieldList(t),
	}})
	// secret 不可编辑携带数据 → 整体拒绝（FORM_RECORD_INVALID + 按键回填）
	_, err = svc.SubmitRecord(tenantCtx(1), member, &model.SubmitRecordRequest{
		AppCode: "app_x", FormCode: "form_test", PublishedVersion: 1,
		SchemaRevision: published.SchemaRevision, HasResult: submitBool(true),
		DataOpID: "6e243bbb-7d57-4e59-952b-d530c53c6561",
		Values:   submitValues(map[string]string{"name": `"张三"`, "secret": `"越权"`}),
	})
	assert.ErrorIs(t, err, apperrors.ErrRecordInvalid)

	// 权限隐藏字段（secret）空 data + 快照可见性信封 → 通过，落库值不含 secret
	result, err := svc.SubmitRecord(tenantCtx(1), member, &model.SubmitRecordRequest{
		AppCode: "app_x", FormCode: "form_test", PublishedVersion: 1,
		SchemaRevision: published.SchemaRevision, HasResult: submitBool(true),
		DataOpID: "6e243bbb-7d57-4e59-952b-d530c53c6561",
		Values:   submitValues(map[string]string{"name": `"张三"`, "amount": "88"}),
	})
	assert.NoError(t, err)
	assert.NotZero(t, result.RecordID)
	var stored map[string]any
	assert.NoError(t, json.Unmarshal(recordRepo.records[0].Values, &stored))
	assert.Equal(t, "张三", stored["name"])
	// 不可编辑的 secret 以 null 落库（与既有空值语义一致），携带的「越权」值被拒收
	assert.Nil(t, stored["secret"])
	assert.Nil(t, stored["tags"])
}

// publishForTest 测试助手：直接走版本仓储发布（绕过 Service 权限复核）
func publishForTest(ctx context.Context, formRepo *fakeFormRepo, versionRepo *fakeVersionRepo, form *model.Form, doc string) (*model.PublishResult, error) {
	nextNo := 1
	version := &model.FormVersion{FormID: form.ID, VersionNo: nextNo, Content: model.JSONContent(doc), ProtocolVersion: model.CurrentProtocolVersion}
	version.TenantID = form.TenantID
	created, err := versionRepo.Create(ctx, version)
	if err != nil {
		return nil, err
	}
	if err := versionRepo.SetSchemaRevision(ctx, created.ID, int64(created.ID)); err != nil {
		return nil, err
	}
	if err := formRepo.MarkPublished(ctx, form.ID, created.ID, nextNo); err != nil {
		return nil, err
	}
	return &model.PublishResult{PublishedVersion: nextNo, SchemaRevision: strconvFormatInt(int64(created.ID))}, nil
}

func strconvFormatInt(v int64) string {
	return fmt.Sprintf("%d", v)
}
