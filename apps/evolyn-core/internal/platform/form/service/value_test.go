package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 与前端 schema/codec 对拍的提交值校验用例（错误文案逐字一致）。

func snapshot(items ...map[string]any) map[string]any {
	arr := make([]any, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		arr = append(arr, map[string]any(item))
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
