// 权限组配置期校验器单测（表单权限 P1，设计 §3/§4/§5）：
// 操作键 × 表单类型矩阵、字段矩阵必填协调两规则与矛盾组合、数据范围
// operator × 字段类型白名单与比较值形状。
package service

import (
	"encoding/json"
	"testing"

	"evolyn/internal/platform/form/model"

	"github.com/stretchr/testify/assert"
)

func TestValidatePermissionOperations(t *testing.T) {
	// 普通表单：standard 合法集内的键放行，重复键去重
	ops, err := ValidatePermissionOperations(model.FormTypeStandard, []string{"view", "add", "view"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"view", "add"}, ops)

	// 未知键拒绝
	_, err = ValidatePermissionOperations(model.FormTypeStandard, []string{"nope"})
	assert.Error(t, err)

	// standard 表单出现 workflow_* 键整体拒绝（§3.2）
	for _, op := range []string{model.PermissionOpWorkflowOwnerTransfer, model.PermissionOpWorkflowTerminate, model.PermissionOpWorkflowActivate} {
		_, err = ValidatePermissionOperations(model.FormTypeStandard, []string{"view", op})
		assert.Error(t, err, op)
	}

	// 流程表单合法集为超集
	ops, err = ValidatePermissionOperations(model.FormTypeWorkflow, []string{
		"view", "add", "copy", "edit", "delete", "batch_print", "batch_modify", "import", "export",
		model.PermissionOpWorkflowOwnerTransfer, model.PermissionOpWorkflowTerminate, model.PermissionOpWorkflowActivate,
	})
	assert.NoError(t, err)
	assert.Len(t, ops, 12)
}

func TestValidatePermissionFieldRules(t *testing.T) {
	fieldList := permFieldList(t) // name(必填)/amount/secret/created_day/tags

	// 正常配置
	err := ValidatePermissionFieldRules(fieldList, []model.PermissionFieldRule{
		{Field: "name", Visible: true, Editable: true},
		{Field: "secret", Visible: false, Editable: false},
	}, []string{"view", "add"})
	assert.NoError(t, err)

	// 键不在字段清单
	err = ValidatePermissionFieldRules(fieldList, []model.PermissionFieldRule{{Field: "ghost", Visible: true, Editable: true}}, nil)
	assert.Error(t, err)

	// 重复字段
	err = ValidatePermissionFieldRules(fieldList, []model.PermissionFieldRule{
		{Field: "name", Visible: true, Editable: true},
		{Field: "name", Visible: true, Editable: true},
	}, nil)
	assert.Error(t, err)

	// 矛盾组合：不可见却可编辑（校验器拒绝）
	err = ValidatePermissionFieldRules(fieldList, []model.PermissionFieldRule{{Field: "amount", Visible: false, Editable: true}}, nil)
	assert.Error(t, err)

	// 必填协调一：visible=false 仅允许非必填（name 必填 → 拒绝）
	err = ValidatePermissionFieldRules(fieldList, []model.PermissionFieldRule{{Field: "name", Visible: false, Editable: false}}, nil)
	assert.Error(t, err)

	// 必填协调二：operations 含 add 时必填字段必须 editable=true
	err = ValidatePermissionFieldRules(fieldList, []model.PermissionFieldRule{{Field: "name", Visible: true, Editable: false}}, []string{"view", "add"})
	assert.Error(t, err)
	// 无 add 时只读必填字段合法
	err = ValidatePermissionFieldRules(fieldList, []model.PermissionFieldRule{{Field: "name", Visible: true, Editable: false}}, []string{"view"})
	assert.NoError(t, err)
}

func TestValidatePermissionDataScope(t *testing.T) {
	fieldList := permFieldList(t)

	// 空条件 = 全部数据（S6）
	assert.NoError(t, ValidatePermissionDataScope(fieldList, &model.PermissionDataScopeSpec{}))

	// match 非法
	err := ValidatePermissionDataScope(fieldList, &model.PermissionDataScopeSpec{Match: "xor"})
	assert.Error(t, err)

	scope := func(conditions ...model.PermissionDataCondition) *model.PermissionDataScopeSpec {
		return &model.PermissionDataScopeSpec{Match: model.PermissionScopeMatchAll, Conditions: conditions}
	}

	// 字段不在清单
	err = ValidatePermissionDataScope(fieldList, scope(model.PermissionDataCondition{Field: "ghost", Operator: "empty"}))
	assert.Error(t, err)

	// 类型类白名单：contains 不适用于数字字段
	err = ValidatePermissionDataScope(fieldList, scope(model.PermissionDataCondition{Field: "amount", Operator: "contains", Value: []any{"x"}}))
	assert.Error(t, err)
	// eq 不适用于多选字段
	err = ValidatePermissionDataScope(fieldList, scope(model.PermissionDataCondition{Field: "tags", Operator: "eq", Value: []any{"a"}}))
	assert.Error(t, err)
	// 比较类不适用于文本字段
	err = ValidatePermissionDataScope(fieldList, scope(model.PermissionDataCondition{Field: "name", Operator: "gte", Value: []any{"x"}}))
	assert.Error(t, err)

	// 合法配置：文本 eq、数字 gte、日期形状、多选 contains
	assert.NoError(t, ValidatePermissionDataScope(fieldList, scope(
		model.PermissionDataCondition{Field: "name", Operator: "eq", Value: []any{"销售"}},
		model.PermissionDataCondition{Field: "amount", Operator: "gte", Value: []any{float64(1000)}},
		model.PermissionDataCondition{Field: "created_day", Operator: "lt", Value: []any{"2026-12-31"}},
		model.PermissionDataCondition{Field: "tags", Operator: "contains", Value: []any{"a"}},
	)))

	// empty/not_empty 须为空数组
	err = ValidatePermissionDataScope(fieldList, scope(model.PermissionDataCondition{Field: "name", Operator: "empty", Value: []any{"x"}}))
	assert.Error(t, err)
	assert.NoError(t, ValidatePermissionDataScope(fieldList, scope(model.PermissionDataCondition{Field: "name", Operator: "empty"})))

	// 标量 operator 要求单元素
	err = ValidatePermissionDataScope(fieldList, scope(model.PermissionDataCondition{Field: "name", Operator: "eq", Value: []any{"a", "b"}}))
	assert.Error(t, err)

	// 日期形状不符
	err = ValidatePermissionDataScope(fieldList, scope(model.PermissionDataCondition{Field: "created_day", Operator: "gt", Value: []any{"2026/01/01"}}))
	assert.Error(t, err)

	// in 值须为字符串集合
	err = ValidatePermissionDataScope(fieldList, scope(model.PermissionDataCondition{Field: "tags", Operator: "in", Value: []any{float64(1)}}))
	assert.Error(t, err)
	assert.NoError(t, ValidatePermissionDataScope(fieldList, scope(model.PermissionDataCondition{Field: "tags", Operator: "not_in", Value: []any{"a", "b"}})))

	// 不支持条件的字段类型（无确定值形状）
	doc := `{"content":{"type":"form","layout":"grid-2","items":[
		{"widget":{"type":"image","widgetName":"photo","visible":true,"allowBlank":true},"label":"照片"}]}}`
	imageList, err := buildPermissionFieldList(mustDocMap(doc))
	assert.NoError(t, err)
	err = ValidatePermissionDataScope(imageList, scope(model.PermissionDataCondition{Field: "photo", Operator: "eq", Value: []any{"x"}}))
	assert.Error(t, err)

	// 布局项不进权限域字段清单
	layoutDoc := `{"content":{"type":"form","layout":"grid-2","items":[
		{"widget":{"type":"separator","widgetName":"_sep","style":"solid"},"label":""},
		{"widget":{"type":"text","widgetName":"name","visible":true,"allowBlank":true},"label":"姓名"}]}}`
	layoutList, err := buildPermissionFieldList(mustDocMap(layoutDoc))
	assert.NoError(t, err)
	assert.Len(t, layoutList, 1, "separator 布局项不进权限域清单")
	assert.Equal(t, "name", layoutList[0].Key)
}

func mustDocMap(doc string) map[string]any {
	root := map[string]any{}
	if err := json.Unmarshal([]byte(doc), &root); err != nil {
		panic(err)
	}
	return root
}
