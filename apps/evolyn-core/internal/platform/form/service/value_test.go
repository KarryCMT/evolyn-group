package service

import (
	"encoding/json"
	"testing"

	"evolyn/internal/platform/form/model"

	"github.com/stretchr/testify/assert"
)

// 与前端 schema/codec 对拍的提交值校验用例（错误文案逐字一致）。

func snapshot(items ...map[string]any) map[string]any {
	arr := make([]any, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		arr = append(arr, item)
	}
	return map[string]any{"content": map[string]any{"type": "form", "items": arr}}
}

func snapItem(widgetType, name, label string, extra map[string]any) map[string]any {
	widget := map[string]any{
		"type": widgetType, "widgetName": name,
		"enable": true, "visible": true, "allowBlank": true,
	}
	for key, value := range extra {
		widget[key] = value
	}
	return map[string]any{
		"widget": widget, "label": label, "description": "",
		"labelHidden": false, "lineWidth": 12,
	}
}

func values(pairs ...string) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	for i := 0; i+1 < len(pairs); i += 2 {
		out[pairs[i]] = json.RawMessage(pairs[i+1])
	}
	return out
}

func wrappedValue(data string, visible bool) model.SubmitFieldValue {
	value := visible
	return model.SubmitFieldValue{Data: model.JSONContent(data), Visible: &value}
}

// submitValues 测试便捷封装：默认匿名口径，需要时以第三参注入提交成员。
func submitValues(
	content map[string]any, submitted map[string]model.SubmitFieldValue, member ...string,
) (map[string]any, RecordFieldErrors) {
	currentMemberID := ""
	if len(member) > 0 {
		currentMemberID = member[0]
	}
	return ValidateSubmittedRecordValues(content, submitted, currentMemberID)
}

func TestValidateSubmittedRecordValuesEnvelope(t *testing.T) {
	content := snapshot(
		snapItem("text", "_widget_visible", "姓名", map[string]any{"allowBlank": false}),
		snapItem("text", "_widget_hidden", "内部字段", map[string]any{"visible": false}),
		snapItem("separator", "_widget_sep", "分割线", nil),
	)
	cleaned, errs := submitValues(content, map[string]model.SubmitFieldValue{
		"_widget_visible": wrappedValue(`"张三"`, true),
		"_widget_hidden":  wrappedValue("", false),
	})
	assert.Empty(t, errs)
	assert.Equal(t, "张三", cleaned["_widget_visible"])
	assert.NotContains(t, cleaned, "_widget_hidden")

	_, errs = submitValues(content, map[string]model.SubmitFieldValue{
		"_widget_visible": wrappedValue(`"张三"`, false),
		"_widget_hidden":  wrappedValue(`"越权值"`, false),
		"_widget_sep":     wrappedValue("", true),
	})
	assert.Equal(t, []string{"字段可见状态与发布快照不一致"}, errs["_widget_visible"])
	assert.Equal(t, []string{"隐藏字段不能提交值"}, errs["_widget_hidden"])
	assert.Equal(t, []string{"分割线等布局字段不能进入提交值"}, errs["_widget_sep"])
}

func TestValidateRecordValuesRequired(t *testing.T) {
	content := snapshot(snapItem("text", "_widget_t", "姓名", map[string]any{"allowBlank": false}))
	_, errs := ValidateRecordValues(content, values("_widget_t", `null`))
	assert.Equal(t, []string{"请输入姓名"}, errs["_widget_t"])

	// 选择动词
	content = snapshot(snapItem("combo", "_widget_c", "城市", map[string]any{
		"allowBlank": false,
		"options":    []any{map[string]any{"label": "A", "value": "a"}},
	}))
	_, errs = ValidateRecordValues(content, values("_widget_c", `null`))
	assert.Equal(t, []string{"请选择城市"}, errs["_widget_c"])

	// allowBlank=true 空值直接通过并清洗为 null
	content = snapshot(snapItem("text", "_widget_t", "备注", nil))
	cleaned, errs := ValidateRecordValues(content, values("_widget_t", `null`))
	assert.Empty(t, errs)
	assert.Nil(t, cleaned["_widget_t"])
}

func TestValidateRecordValuesTextAndNumber(t *testing.T) {
	content := snapshot(snapItem("text", "_widget_t", "备注", map[string]any{
		"minLength": 2, "maxLength": 4,
	}))
	_, errs := ValidateRecordValues(content, values("_widget_t", `"a"`))
	assert.Equal(t, []string{"备注最少输入 2 个字符"}, errs["_widget_t"])
	_, errs = ValidateRecordValues(content, values("_widget_t", `"abcde"`))
	assert.Equal(t, []string{"备注不能超过 4 个字符"}, errs["_widget_t"])

	content = snapshot(snapItem("number", "_widget_n", "年龄", map[string]any{
		"min": 0.0, "max": 10.0, "precision": 1,
	}))
	_, errs = ValidateRecordValues(content, values("_widget_n", `-1`))
	assert.Equal(t, []string{"年龄不能小于 0"}, errs["_widget_n"])
	_, errs = ValidateRecordValues(content, values("_widget_n", `11`))
	assert.Equal(t, []string{"年龄不能大于 10"}, errs["_widget_n"])
	_, errs = ValidateRecordValues(content, values("_widget_n", `1.23`))
	assert.Equal(t, []string{"年龄最多支持 1 位小数"}, errs["_widget_n"])

	// 类型不符（字符串传数字）
	_, errs = ValidateRecordValues(content, values("_widget_n", `"3"`))
	assert.Equal(t, []string{"年龄的值类型不正确"}, errs["_widget_n"])
}

func TestValidateRecordValuesDateTime(t *testing.T) {
	content := snapshot(snapItem("datetime", "_widget_d", "生日", map[string]any{"format": "datetime"}))
	_, errs := ValidateRecordValues(content, values("_widget_d", `"2026-02-30 10:00:00"`))
	assert.Equal(t, []string{"生日的日期格式不正确"}, errs["_widget_d"])
	cleaned, errs := ValidateRecordValues(content, values("_widget_d", `"2024-02-29 23:59:59"`))
	assert.Empty(t, errs)
	assert.Equal(t, "2024-02-29 23:59:59", cleaned["_widget_d"])
}

func TestValidateRecordValuesOptions(t *testing.T) {
	options := []any{
		map[string]any{"label": "A", "value": "a"},
		map[string]any{"label": "B", "value": "b"},
	}
	content := snapshot(
		snapItem("combo", "_widget_c", "城市", map[string]any{"options": options}),
		snapItem("checkboxgroup", "_widget_m", "标签", map[string]any{"options": options}),
	)
	_, errs := ValidateRecordValues(content, values("_widget_c", `"c"`))
	assert.Equal(t, []string{"城市的值不在选项范围内"}, errs["_widget_c"])

	_, errs = ValidateRecordValues(content, values("_widget_m", `["a","a"]`))
	assert.Equal(t, []string{"标签的值存在重复选项"}, errs["_widget_m"])

	cleaned, errs := ValidateRecordValues(content, values("_widget_m", `["a","b"]`))
	assert.Empty(t, errs)
	assert.Equal(t, []any{"a", "b"}, cleaned["_widget_m"])
}

func TestValidateRecordValuesMembers(t *testing.T) {
	content := snapshot(
		snapItem("user", "_widget_u", "负责人", map[string]any{"allowBlank": false}),
		snapItem("usergroup", "_widget_ug", "参与成员", map[string]any{"allowBlank": false}),
	)

	_, errs := ValidateRecordValues(content, values("_widget_u", `{"id":"m1"}`, "_widget_ug", `["m1","m1"]`))
	assert.Equal(t, []string{"负责人的值类型不正确"}, errs["_widget_u"])
	assert.Equal(t, []string{"参与成员的值存在重复成员"}, errs["_widget_ug"])

	cleaned, errs := ValidateRecordValues(content, values("_widget_u", `"m1"`, "_widget_ug", `["m1","m2"]`))
	assert.Empty(t, errs)
	assert.Equal(t, "m1", cleaned["_widget_u"])
	assert.Equal(t, []any{"m1", "m2"}, cleaned["_widget_ug"])
}

func TestValidateRecordValuesDepartments(t *testing.T) {
	content := snapshot(
		snapItem("dept", "_widget_d", "所属部门", map[string]any{"allowBlank": false}),
		snapItem("deptgroup", "_widget_dg", "协作部门", map[string]any{"allowBlank": false}),
	)

	_, errs := ValidateRecordValues(content, values("_widget_d", `{"id":"d1"}`, "_widget_dg", `["d1","d1"]`))
	assert.Equal(t, []string{"所属部门的值类型不正确"}, errs["_widget_d"])
	assert.Equal(t, []string{"协作部门的值存在重复部门"}, errs["_widget_dg"])

	cleaned, errs := ValidateRecordValues(content, values("_widget_d", `"d1"`, "_widget_dg", `["d1","d2"]`))
	assert.Empty(t, errs)
	assert.Equal(t, "d1", cleaned["_widget_d"])
	assert.Equal(t, []any{"d1", "d2"}, cleaned["_widget_dg"])
}

func TestValidateRecordValuesLayoutAndHiddenAndUnknown(t *testing.T) {
	content := snapshot(
		snapItem("separator", "_widget_sep", "分割线", nil),
		snapItem("text", "_widget_h", "隐藏字段", map[string]any{"visible": false}),
		snapItem("text", "_widget_t", "可见", nil),
	)
	// 布局项携值拒绝
	_, errs := ValidateRecordValues(content, values("_widget_sep", `"x"`))
	assert.Equal(t, []string{"分割线等布局字段不能携带值"}, errs["_widget_sep"])
	// 布局项 null 允许（不落库）
	cleaned, errs := ValidateRecordValues(content, values("_widget_sep", `null`, "_widget_t", `"ok"`))
	assert.Empty(t, errs)
	assert.NotContains(t, cleaned, "_widget_sep")
	// 隐藏字段携值拒绝
	_, errs = ValidateRecordValues(content, values("_widget_h", `"x"`))
	assert.Equal(t, []string{"隐藏字段不能提交值"}, errs["_widget_h"])
	// 未知键拒绝
	_, errs = ValidateRecordValues(content, values("_widget_unknown", `1`))
	assert.Equal(t, []string{"提交了表单中不存在的字段"}, errs["_widget_unknown"])
}

func TestExtractSnapshotTopFieldKeys(t *testing.T) {
	content := snapshot(
		snapItem("text", "_widget_t", "a", nil),
		snapItem("separator", "_widget_sep", "", nil),
	)
	assert.Equal(t, []string{"_widget_t", "_widget_sep"}, ExtractSnapshotTopFieldKeys(content))
}

func TestExtractSnapshotFieldMappings(t *testing.T) {
	content := snapshot(
		snapItem("number", "_widget_amount", "合同金额", nil),
		snapItem("separator", "_widget_sep", "", nil),
	)

	assert.Equal(t, []SnapshotFieldMapping{{
		WidgetName:     "_widget_amount",
		WidgetType:     "number",
		JSONBKey:       "_widget_amount",
		PhysicalColumn: "f_amount",
	}}, ExtractSnapshotFieldMappings(content))
}

// ---- v5 字段显隐规则：提交终审动态可见性 ----

func snapshotWithRules(rules []any, items ...map[string]any) map[string]any {
	content := snapshot(items...)
	content["content"].(map[string]any)["fieldShowRules"] = rules
	return content
}

func eqTextRule(id, field, target string, expected string) map[string]any {
	return map[string]any{
		"id": id,
		"filter": map[string]any{"rel": "and", "cond": []any{map[string]any{
			"field": field, "type": "text", "method": "eq", "value": []any{expected},
		}}},
		"fields": []any{target},
	}
}

func TestValidateSubmittedRecordValuesWithShowRules(t *testing.T) {
	content := snapshotWithRules(
		[]any{eqTextRule("r1", "_widget_src", "_widget_target", "是")},
		snapItem("text", "_widget_src", "是否外出", nil),
		snapItem("text", "_widget_target", "外出城市", map[string]any{"allowBlank": false}),
	)

	// 条件成立：目标字段按动态可见性提交，通过并落库。
	cleaned, errs := submitValues(content, map[string]model.SubmitFieldValue{
		"_widget_src":    wrappedValue(`"是"`, true),
		"_widget_target": wrappedValue(`"上海"`, true),
	})
	assert.Empty(t, errs)
	assert.Equal(t, "上海", cleaned["_widget_target"])

	// 条件不成立：目标字段必须声明隐藏且不得携带 data。
	cleaned, errs = submitValues(content, map[string]model.SubmitFieldValue{
		"_widget_src":    wrappedValue(`"否"`, true),
		"_widget_target": wrappedValue("", false),
	})
	assert.Empty(t, errs)
	assert.Nil(t, cleaned["_widget_target"])

	// 伪造可见性：规则求值结果为隐藏却声明可见 → 信封不一致。
	_, errs = submitValues(content, map[string]model.SubmitFieldValue{
		"_widget_src":    wrappedValue(`"否"`, true),
		"_widget_target": wrappedValue("", true),
	})
	assert.Equal(t, []string{"字段可见状态与发布快照不一致"}, errs["_widget_target"])

	// 伪造隐藏：规则求值结果为可见却声明隐藏 → 信封不一致。
	_, errs = submitValues(content, map[string]model.SubmitFieldValue{
		"_widget_src":    wrappedValue(`"是"`, true),
		"_widget_target": wrappedValue("", false),
	})
	assert.Equal(t, []string{"字段可见状态与发布快照不一致"}, errs["_widget_target"])

	// 隐藏字段携带 data → 拒绝。
	_, errs = submitValues(content, map[string]model.SubmitFieldValue{
		"_widget_src":    wrappedValue(`"否"`, true),
		"_widget_target": wrappedValue(`"上海"`, false),
	})
	assert.Equal(t, []string{"隐藏字段不能提交值"}, errs["_widget_target"])
}

func TestValidateSubmittedRecordValuesShowRuleCascade(t *testing.T) {
	// 多级：src eq 是 → mid 显示；mid notEmpty → detail 显示。
	content := snapshotWithRules([]any{
		eqTextRule("r1", "_widget_src", "_widget_mid", "是"),
		map[string]any{
			"id": "r2",
			"filter": map[string]any{"rel": "and", "cond": []any{map[string]any{
				"field": "_widget_mid", "type": "text", "method": "notEmpty",
			}}},
			"fields": []any{"_widget_detail"},
		},
	},
		snapItem("text", "_widget_src", "是否外出", nil),
		snapItem("text", "_widget_mid", "外出城市", nil),
		snapItem("text", "_widget_detail", "住宿说明", map[string]any{"allowBlank": false}),
	)

	// 上游条件不成立：mid/detail 均需声明隐藏且不得携带 data（隐藏值不参与求值）。
	_, errs := submitValues(content, map[string]model.SubmitFieldValue{
		"_widget_src":    wrappedValue(`"否"`, true),
		"_widget_mid":    wrappedValue("", false),
		"_widget_detail": wrappedValue("", false),
	})
	assert.Empty(t, errs)

	// 隐藏仍携带 data → 拒绝（伪造隐藏值不进入求值也不进入记录）。
	_, errs = submitValues(content, map[string]model.SubmitFieldValue{
		"_widget_src":    wrappedValue(`"否"`, true),
		"_widget_mid":    wrappedValue(`"上海"`, false),
		"_widget_detail": wrappedValue("", false),
	})
	assert.Equal(t, []string{"隐藏字段不能提交值"}, errs["_widget_mid"])

	// 上游条件不成立却伪造 mid 可见 → 与求值结果不一致（隐藏值不参与求值）。
	_, errs = submitValues(content, map[string]model.SubmitFieldValue{
		"_widget_src":    wrappedValue(`"否"`, true),
		"_widget_mid":    wrappedValue(`"上海"`, true),
		"_widget_detail": wrappedValue("", false),
	})
	assert.Equal(t, []string{"字段可见状态与发布快照不一致"}, errs["_widget_mid"])
}

func TestValidateRecordValuesDropsRuleHiddenFieldValues(t *testing.T) {
	// 无信封入口（流程写回路径）：规则隐藏字段的既有值收网为 null，必填跳过。
	content := snapshotWithRules(
		[]any{eqTextRule("r1", "_widget_src", "_widget_target", "是")},
		snapItem("text", "_widget_src", "是否外出", nil),
		snapItem("text", "_widget_target", "外出城市", map[string]any{"allowBlank": false}),
	)
	cleaned, errs := ValidateRecordValues(content, values(
		"_widget_src", `"否"`,
		"_widget_target", `"上海"`,
	))
	assert.Empty(t, errs)
	assert.Nil(t, cleaned["_widget_target"])

	// 条件成立时既有值保留并正常校验。
	cleaned, errs = ValidateRecordValues(content, values(
		"_widget_src", `"是"`,
		"_widget_target", `"上海"`,
	))
	assert.Empty(t, errs)
	assert.Equal(t, "上海", cleaned["_widget_target"])
}

func TestValidateSubmittedRecordValuesIncludeCurrentMember(t *testing.T) {
	// 负责人 eq member_a（includeCurrentMember）：服务端以提交人身份求值。
	content := snapshotWithRules(
		[]any{
			map[string]any{
				"id": "r1",
				"filter": map[string]any{"rel": "and", "cond": []any{map[string]any{
					"field": "_widget_owner", "type": "user", "method": "eq",
					"value": []any{"member_a"}, "includeCurrentMember": true,
				}}},
				"fields": []any{"_widget_secret"},
			},
		},
		snapItem("user", "_widget_owner", "负责人", nil),
		snapItem("text", "_widget_secret", "专属项", map[string]any{"allowBlank": false}),
	)

	// 提交人即当前成员：条件成立，专属项可见可提交。
	cleaned, errs := submitValues(content, map[string]model.SubmitFieldValue{
		"_widget_owner":  wrappedValue(`"member_current"`, true),
		"_widget_secret": wrappedValue(`"内容"`, true),
	}, "member_current")
	assert.Empty(t, errs)
	assert.Equal(t, "内容", cleaned["_widget_secret"])

	// 匿名（未注入成员）：includeCurrentMember 不加入任何值，条件不成立。
	_, errs = submitValues(content, map[string]model.SubmitFieldValue{
		"_widget_owner":  wrappedValue(`"member_current"`, true),
		"_widget_secret": wrappedValue("", false),
	})
	assert.Empty(t, errs)

	// 提交人不同且值不命中：条件不成立，声明可见即与求值不一致。
	_, errs = submitValues(content, map[string]model.SubmitFieldValue{
		"_widget_owner":  wrappedValue(`"member_other"`, true),
		"_widget_secret": wrappedValue("", true),
	}, "member_current")
	assert.Equal(t, []string{"字段可见状态与发布快照不一致"}, errs["_widget_secret"])
}
