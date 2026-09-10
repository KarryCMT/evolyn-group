package service

import (
	"encoding/json"
	"testing"

	storagepkg "evolyn/internal/engine/data/storage"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 数值字段族（decimal/money/percent）镜像测试：与前端 packages/form/src/
// schema/__tests__/numeric.spec.ts 同向量对拍，同一 JSON 两端结论必须一致。

func TestNumericFieldSemantics(t *testing.T) {
	// 有效精度/小数位：显式配置优先，否则按类型取默认。
	precision, scale := effectiveNumericPrecisionScale("decimal", map[string]any{})
	assert.Equal(t, 20, precision)
	assert.Equal(t, 6, scale)
	_, scale = effectiveNumericPrecisionScale("money", map[string]any{})
	assert.Equal(t, 2, scale)
	precision, _ = effectiveNumericPrecisionScale("percent", map[string]any{})
	assert.Equal(t, 10, precision)
	precision, scale = effectiveNumericPrecisionScale("decimal", map[string]any{"precision": 30, "scale": 4})
	assert.Equal(t, 30, precision)
	assert.Equal(t, 4, scale)

	// 位数计数按值语义：尾随零/前导零不计位。
	assert.Equal(t, 1, decimalFractionDigits("1.500"))
	assert.Equal(t, 0, decimalFractionDigits("100"))
	assert.Equal(t, 0, decimalFractionDigits("0.000"))
	assert.Equal(t, 1, decimalIntegerDigits("007.5"))
	assert.Equal(t, 0, decimalIntegerDigits("0"))
	assert.Equal(t, 0, decimalIntegerDigits("-0.5"))
	assert.Equal(t, 3, decimalIntegerDigits("123.45"))
	assert.Equal(t, "scale", decimalDigitIssue("1.234", 20, 2))
	assert.Equal(t, "precision", decimalDigitIssue("1234567890123456789.5", 20, 2))
	assert.Equal(t, "", decimalDigitIssue("12.34", 20, 2))

	// 十进制文本形状：拒绝指数记法、正号、裸小数点与空白。
	for _, ok := range []string{"-3.5", "0", "100.125", "007.500"} {
		assert.True(t, validDecimalText(ok), ok)
	}
	for _, bad := range []string{"1e3", "+1", ".5", "3.", " 1", "1 ", "abc", "1.2.3", ""} {
		assert.False(t, validDecimalText(bad), bad)
	}

	// 十进制比较走精确值序（与前端对拍）。
	order, ok := compareDecimalText("0.1", "0.10000000000000000000001")
	assert.True(t, ok)
	assert.Equal(t, -1, order)
	order, ok = compareDecimalText("100", "99.999999999999999999999")
	assert.True(t, ok)
	assert.Equal(t, 1, order)
	order, ok = compareDecimalText("-1.5", "-1.50")
	assert.True(t, ok)
	assert.Equal(t, 0, order)
}

func TestNumericFamilySchemaValidation(t *testing.T) {
	numericItem := func(widgetType string, extra map[string]any) map[string]any {
		item := validTextItem()
		w := item["widget"].(map[string]any)
		w["type"] = widgetType
		w["widgetName"] = "_widget_n1"
		delete(w, "placeholder")
		for key, value := range extra {
			w[key] = value
		}
		return item
	}
	find := func(issues []SchemaIssue, path, message string) bool {
		for _, issue := range issues {
			if issue.Path == path && issue.Message == message {
				return true
			}
		}
		return false
	}

	// 三种类型的最小合法形态并可发布。
	for _, widgetType := range []string{"decimal", "money", "percent"} {
		assert.Empty(t, ValidateFormSchema(doc(numericItem(widgetType, nil))))
		assert.Empty(t, ValidatePublishable(doc(numericItem(widgetType, nil))))
	}

	// min/max/defaultValue 拒绝非十进制字符串形状。
	assert.True(t, find(
		ValidateFormSchema(doc(numericItem("decimal", map[string]any{"min": "1e3"}))),
		"content.items[0].widget.min", "min 必须是十进制数字字符串（null 表示未启用）"))
	assert.True(t, find(
		ValidateFormSchema(doc(numericItem("money", map[string]any{"defaultValue": 12.5}))),
		"content.items[0].widget.defaultValue", "defaultValue 必须是十进制数字字符串（null 表示未启用）"))

	// precision/scale 超出护栏即拒绝。
	assert.True(t, find(
		ValidateFormSchema(doc(numericItem("decimal", map[string]any{"precision": 41}))),
		"content.items[0].widget.precision", "precision 不在允许范围 1–40 内"))
	assert.True(t, find(
		ValidateFormSchema(doc(numericItem("money", map[string]any{"scale": 19}))),
		"content.items[0].widget.scale", "scale 不在允许范围 0–18 内"))

	// rounding 只接受七种平台舍入模式。
	assert.True(t, find(
		ValidateFormSchema(doc(numericItem("percent", map[string]any{"rounding": "HALF_CEIL"}))),
		"content.items[0].widget.rounding",
		"rounding 必须是以下枚举值之一：UP / DOWN / CEIL / FLOOR / HALF_UP / HALF_DOWN / HALF_EVEN"))
	assert.Empty(t, ValidateFormSchema(doc(numericItem("percent", map[string]any{"rounding": "HALF_EVEN"}))))

	// 交叉规则：precision 显式 5 时，缺省 scale=6 生效后超限，必须拒绝。
	assert.True(t, find(
		ValidateFormSchema(doc(numericItem("decimal", map[string]any{"precision": 5}))),
		"content.items[0].widget.scale", "scale 不能大于 precision"))

	// 交叉规则：min ≤ max 与 defaultValue 范围（decimal 值序）。
	assert.True(t, find(
		ValidateFormSchema(doc(numericItem("money", map[string]any{"min": "100.5", "max": "100.4"}))),
		"content.items[0].widget.max", "max 不能小于 min"))
	assert.True(t, find(
		ValidateFormSchema(doc(numericItem("money", map[string]any{"min": "10", "max": "20", "defaultValue": "20.01"}))),
		"content.items[0].widget.defaultValue", "defaultValue 不能大于 max"))

	// 交叉规则：min/max/defaultValue 自身受位数约束（防不可满足范围）。
	assert.True(t, find(
		ValidateFormSchema(doc(numericItem("money", map[string]any{"min": "0.123"}))),
		"content.items[0].widget.min", "min 最多支持 2 位小数"))
	assert.True(t, find(
		ValidateFormSchema(doc(numericItem("percent", map[string]any{"max": "12345678901234.5"}))),
		"content.items[0].widget.max", "max 整数位最多 4 位"))

	// 未知属性拒绝。
	assert.True(t, find(
		ValidateFormSchema(doc(numericItem("decimal", map[string]any{"currency": "CNY"}))),
		"content.items[0].widget.currency", "未知属性「currency」"))

	// 显隐条件值必须是 decimal string。
	ruleItem := numericItem("money", nil)
	moneyName := ruleItem["widget"].(map[string]any)["widgetName"].(string)
	rule := map[string]any{
		"id": "_field_show_rule_1",
		"filter": map[string]any{"rel": "and", "cond": []any{map[string]any{
			"field": moneyName, "type": "money", "method": "gt", "value": []any{100.0},
		}}},
		"fields": []any{},
	}
	items := []any{ruleItem}
	ensureTestFieldIDs(items)
	raw := map[string]any{"content": map[string]any{
		"type": "form", "layout": "normal", "items": items, "layout_fields": []any{moneyName},
		"fieldShowRules": []any{rule}, "submitRule": 2, "widget_submit_rules": map[string]any{},
		"validators": []any{}, "preSubmitConfirm": map[string]any{"enable": false, "title": "请确认提交", "content": "确认提交当前内容？"},
	}}
	encoded, err := json.Marshal(raw)
	assert.NoError(t, err)
	assert.True(t, find(ValidateFormSchema(encoded),
		"content.fieldShowRules[0].filter.cond[0].value[0]", "value 条目必须是十进制数字字符串"))
}

func TestValidateRecordValuesNumericFamily(t *testing.T) {
	// 类型与形状错误逐字回填（与前端对拍）。
	content := snapshot(snapItem("decimal", "_widget_n", "数值", nil))
	_, errs := ValidateRecordValues(content, values("_widget_n", `"123.45"`))
	assert.Empty(t, errs["_widget_n"])
	_, errs = ValidateRecordValues(content, values("_widget_n", `"-0.5"`))
	assert.Empty(t, errs["_widget_n"])
	_, errs = ValidateRecordValues(content, values("_widget_n", `123.45`))
	assert.Equal(t, []string{"数值的值类型不正确"}, errs["_widget_n"])
	_, errs = ValidateRecordValues(content, values("_widget_n", `"1e3"`))
	assert.Equal(t, []string{"数值格式不正确"}, errs["_widget_n"])

	// 位数与范围按有效精度复核（尾随零不计位）。
	content = snapshot(snapItem("money", "_widget_n", "数值", nil))
	_, errs = ValidateRecordValues(content, values("_widget_n", `"1.500"`))
	assert.Empty(t, errs["_widget_n"])
	_, errs = ValidateRecordValues(content, values("_widget_n", `"1.234"`))
	assert.Equal(t, []string{"数值最多支持 2 位小数"}, errs["_widget_n"])
	content = snapshot(snapItem("percent", "_widget_n", "数值", nil))
	_, errs = ValidateRecordValues(content, values("_widget_n", `"12345.5"`))
	assert.Equal(t, []string{"数值整数位最多 4 位"}, errs["_widget_n"])
	content = snapshot(snapItem("money", "_widget_n", "数值", map[string]any{
		"min": "10.5", "max": "20",
	}))
	_, errs = ValidateRecordValues(content, values("_widget_n", `"10.49"`))
	assert.Equal(t, []string{"数值不能小于 10.5"}, errs["_widget_n"])
	_, errs = ValidateRecordValues(content, values("_widget_n", `"20.5"`))
	assert.Equal(t, []string{"数值不能大于 20"}, errs["_widget_n"])

	// 必填与空值语义。
	required := snapshot(snapItem("money", "_widget_n", "金额", map[string]any{"allowBlank": false}))
	_, errs = ValidateRecordValues(required, values("_widget_n", `null`))
	assert.Equal(t, []string{"请输入金额"}, errs["_widget_n"])
}

func TestNumericFamilyShowRuleEvaluation(t *testing.T) {
	// decimal 条件求值：gt/between 走精确值序（与前端 rules.ts 对拍）。
	content := snapshotWithRules([]any{map[string]any{
		"id": "_field_show_rule_1",
		"filter": map[string]any{"rel": "and", "cond": []any{
			map[string]any{"field": "_widget_m", "type": "money", "method": "gt", "value": []any{"100.5"}},
		}},
		"fields": []any{"_widget_t"},
	}},
		snapItem("money", "_widget_m", "金额", nil),
		snapItem("text", "_widget_t", "备注", nil),
	)
	// 规则隐藏：金额 100.5 不大于 100.5 → 目标隐藏；100.5001 → 显示。
	cleaned, errs := ValidateRecordValues(content, values("_widget_m", `"100.5"`, "_widget_t", `null`))
	assert.Empty(t, errs)
	assert.Nil(t, cleaned["_widget_t"])
	cleaned, errs = ValidateRecordValues(content, values("_widget_m", `"100.51"`, "_widget_t", `"备注"`))
	assert.Empty(t, errs)
	assert.Equal(t, "备注", cleaned["_widget_t"])
}

func TestNumericColumnSpecResolution(t *testing.T) {
	// 发布期物理列精度解析：显式优先、缺省按类型默认（NUMERIC(p,s)）。
	p, s := resolveNumericColumnSpec("money", map[string]any{})
	assert.Equal(t, int16(20), p)
	assert.Equal(t, int16(2), s)
	p, s = resolveNumericColumnSpec("percent", map[string]any{})
	assert.Equal(t, int16(10), p)
	assert.Equal(t, int16(6), s)
	p, s = resolveNumericColumnSpec("decimal", map[string]any{})
	assert.Equal(t, int16(20), p)
	assert.Equal(t, int16(6), s)
	p, s = resolveNumericColumnSpec("decimal", map[string]any{"precision": 12, "scale": 0})
	assert.Equal(t, int16(12), p)
	assert.Equal(t, int16(0), s)
}

// 数值字段族发布链路：目标物理模型落 NUMERIC(p,s)（显式与缺省精度），
// 且发布后精度修饰变更按类型变更拒绝（FORM_STORAGE_TYPE_CHANGE_UNSUPPORTED）。
func TestPublishNumericFamilyColumnSpec(t *testing.T) {
	env := newPhysicalTestEnv()
	money := `{"widget":{"type":"money","widgetName":"_widget_m","fieldId":"aaaaaaaa03","enable":true,"visible":true,"allowBlank":true,"precision":18,"scale":4},"label":"金额","description":"","labelHidden":false,"lineWidth":6}`
	code, revision := env.createPhysicalForm(t, v8PhysicalDraft(money))
	result, err := env.svc.Publish(tenantCtx(1), memberOfTenant(1), code, &model.PublishRequest{DraftRevision: revision})
	require.NoError(t, err)
	assert.True(t, result.Async)

	// 待应用 SchemaVersion 的目标模型：money 列 NUMERIC(18,4)。
	require.NotEmpty(t, env.versions.versions)
	pending := env.versions.versions[len(env.versions.versions)-1]
	var parsed storagepkg.StorageModel
	require.NoError(t, decodeJSONB(pending.Model, &parsed))
	var moneyColumn *storagepkg.ColumnSpec
	for i := range parsed.Columns {
		if parsed.Columns[i].WidgetType == "money" {
			moneyColumn = &parsed.Columns[i]
		}
	}
	require.NotNil(t, moneyColumn, "money column must exist")
	assert.Equal(t, storagepkg.KindDecimal, moneyColumn.Kind)
	assert.Equal(t, storagepkg.ColumnTypeNumeric, moneyColumn.Type)
	assert.EqualValues(t, 18, moneyColumn.Precision)
	assert.EqualValues(t, 4, moneyColumn.Scale)
	assert.Equal(t, "NUMERIC(18,4)", moneyColumn.ColumnDDLType())
	// 存量 number 列不受影响（裸 NUMERIC，无修饰）。
	for _, column := range parsed.Columns {
		if column.WidgetType == "number" {
			assert.Equal(t, storagepkg.KindNumber, column.Kind)
			assert.Zero(t, column.Precision)
			assert.Zero(t, column.Scale)
			assert.Equal(t, "NUMERIC", column.ColumnDDLType())
		}
	}

	// 完成 Job 后改精度再发布：类型变更拒绝。
	completeFirstDDLJob(t, env, code)
	narrowed := `{"widget":{"type":"money","widgetName":"_widget_m","fieldId":"aaaaaaaa03","enable":true,"visible":true,"allowBlank":true,"precision":18,"scale":2},"label":"金额","description":"","labelHidden":false,"lineWidth":6}`
	saved, err := env.svc.SaveDraft(tenantCtx(1), memberOfTenant(1), code, &model.SaveDraftRequest{
		DraftRevision: 2, ProtocolVersion: model.CurrentProtocolVersion,
		Content: v8PhysicalDraft(narrowed),
	})
	require.NoError(t, err)
	_, err = env.svc.Publish(tenantCtx(1), memberOfTenant(1), code, &model.PublishRequest{DraftRevision: saved.DraftRevision})
	assert.ErrorIs(t, err, apperrors.ErrStorageTypeChangeUnsupported)
}

// 缺省精度的数值字段族发布：按类型默认解析（money → NUMERIC(20,2)）。
func TestPublishNumericFamilyDefaultPrecision(t *testing.T) {
	env := newPhysicalTestEnv()
	percent := `{"widget":{"type":"percent","widgetName":"_widget_m","fieldId":"aaaaaaaa03","enable":true,"visible":true,"allowBlank":true},"label":"税率","description":"","labelHidden":false,"lineWidth":6}`
	code, revision := env.createPhysicalForm(t, v8PhysicalDraft(percent))
	_, err := env.svc.Publish(tenantCtx(1), memberOfTenant(1), code, &model.PublishRequest{DraftRevision: revision})
	require.NoError(t, err)
	require.NotEmpty(t, env.versions.versions)
	pending := env.versions.versions[len(env.versions.versions)-1]
	var parsed storagepkg.StorageModel
	require.NoError(t, decodeJSONB(pending.Model, &parsed))
	for _, column := range parsed.Columns {
		if column.WidgetType == "percent" {
			assert.Equal(t, "NUMERIC(10,6)", column.ColumnDDLType())
		}
	}
}
