package service

import (
	"encoding/json"
	"fmt"
	"testing"

	"evolyn/internal/platform/form/model"

	"github.com/stretchr/testify/assert"
)

// 与前端 packages/form/src/schema/__tests__/validate.spec.ts 对拍的核心用例集：
// 同一 JSON 两端结论必须一致（P1 验收条件 T14）。

func validTextItem() map[string]any {
	return map[string]any{
		"widget": map[string]any{
			"type":        "text",
			"widgetName":  "_widget_a1",
			"enable":      true,
			"visible":     true,
			"allowBlank":  true,
			"placeholder": "请输入",
		},
		"label":       "单行文本",
		"description": "",
		"labelHidden": false,
		"lineWidth":   12,
	}
}

func validSubformWidget(items []any) map[string]any {
	return map[string]any{
		"type": "subform", "widgetName": "_widget_sub",
		"enable": true, "visible": true, "allowBlank": true, "items": items,
		"subformCreate": true, "subformInsert": true, "subformEdit": true, "subformDelete": true,
		"quickFill":          true,
		"pcStickyColumn":     map[string]any{"enable": true, "limit": 1},
		"mobileStickyColumn": map[string]any{"enable": false, "limit": 1},
		"mobileViewStyle":    "vertical", "mobileSummaryFieldCount": 3,
	}
}

func doc(items ...any) []byte {
	if items == nil {
		items = []any{}
	}
	fieldLayout := make([]string, 0, len(items))
	for _, rawItem := range items {
		item, _ := rawItem.(map[string]any)
		widget, _ := item["widget"].(map[string]any)
		if name, ok := widget["widgetName"].(string); ok {
			fieldLayout = append(fieldLayout, name)
		}
	}
	raw, _ := json.Marshal(map[string]any{"content": map[string]any{
		"type": "form", "layout": "normal", "items": items, "layout_fields": []any{}, "field_layout": fieldLayout,
		"fieldShowRules": []any{}, "submitRule": 2, "widget_submit_rules": map[string]any{},
	}})
	return raw
}

func TestValidateFormSchemaMultitabWithSubformReference(t *testing.T) {
	child := validTextItem()
	child["widget"].(map[string]any)["widgetName"] = "_widget_child"
	subform := map[string]any{
		"widget": validSubformWidget([]any{child}),
		"label":  "子表单", "description": "", "labelHidden": false, "lineWidth": 12,
	}
	document := map[string]any{"content": map[string]any{
		"type": "form", "layout": "normal", "items": []any{subform},
		"layout_fields": []any{map[string]any{
			"name": "_layout_tabs", "type": "multitab", "tabStyle": "style2",
			"container": []any{map[string]any{
				"name": "_tab_detail", "title": "明细", "type": "tab",
				"field_layout": []any{"_widget_sub"},
			}},
		}},
		"field_layout":        []any{"_layout_tabs"},
		"fieldShowRules":      []any{},
		"submitRule":          2,
		"widget_submit_rules": map[string]any{},
	}}
	raw, _ := json.Marshal(document)
	assert.Empty(t, ValidateFormSchema(raw))

	// 子表单内部字段不是顶层字段，不能进入标签页引用。
	document["content"].(map[string]any)["layout_fields"].([]any)[0].(map[string]any)["container"].([]any)[0].(map[string]any)["field_layout"] = []any{"_widget_child"}
	raw, _ = json.Marshal(document)
	issues := ValidateFormSchema(raw)
	assert.True(t, containsPath(issues, "content.layout_fields[0].container[0].field_layout[0]"))
}

func TestValidateFormSchemaValidSample(t *testing.T) {
	assert.Empty(t, ValidateFormSchema(doc(validTextItem())))
	assert.Empty(t, ValidateFormSchema(doc()))
}

func TestValidateFormSchemaProtocolVersions(t *testing.T) {
	v1 := []byte(`{"content":{"type":"form","items":[]}}`)
	assert.Empty(t, ValidateFormSchema(v1, 1))
	assert.NotEmpty(t, ValidateFormSchema(v1, model.CurrentProtocolVersion))
	v2 := []byte(`{"content":{"type":"form","items":[],"layout_fields":[],"field_layout":[]}}`)
	assert.Empty(t, ValidateFormSchema(v2, 2))
	assert.NotEmpty(t, ValidateFormSchema(v2, model.CurrentProtocolVersion))

	issues := ValidateFormSchema(doc(), model.CurrentProtocolVersion+1)
	assert.Equal(t, "content", issues[0].Path)
	assert.Contains(t, issues[0].Message, "不支持的表单协议版本")
}

func TestValidateFormSchemaLayout(t *testing.T) {
	for _, layout := range []string{"normal", "grid-2", "grid-3", "grid-4"} {
		document := doc()
		var root map[string]any
		assert.NoError(t, json.Unmarshal(document, &root))
		root["content"].(map[string]any)["layout"] = layout
		raw, _ := json.Marshal(root)
		assert.Empty(t, ValidateFormSchema(raw), layout)
	}

	var root map[string]any
	assert.NoError(t, json.Unmarshal(doc(), &root))
	root["content"].(map[string]any)["layout"] = "grid-5"
	raw, _ := json.Marshal(root)
	issues := ValidateFormSchema(raw)
	assert.True(t, containsPath(issues, "content.layout"))
}

func TestValidateFormSchemaRootRules(t *testing.T) {
	// 根未知键
	issues := ValidateFormSchema([]byte(`{"content":{"type":"form","items":[]},"extra":1}`))
	assert.Equal(t, "content.extra", issues[0].Path)
	// content.type 固定 form
	issues = ValidateFormSchema([]byte(`{"content":{"type":"page","items":[]}}`))
	assert.Equal(t, "content.type", issues[0].Path)
}

func TestValidateFormSchemaUnknownKeys(t *testing.T) {
	item := validTextItem()
	item["placeholder"] = "错放 item 层"
	issues := ValidateFormSchema(doc(item))
	assert.Equal(t, "content.items[0].placeholder", issues[0].Path)

	item2 := validTextItem()
	item2["widget"].(map[string]any)["unknownProp"] = 1
	issues = ValidateFormSchema(doc(item2))
	assert.Equal(t, "content.items[0].widget.unknownProp", issues[0].Path)
	assert.Contains(t, issues[0].Message, "未知属性")
}

func TestValidateFormSchemaUnknownWidgetType(t *testing.T) {
	item := validTextItem()
	item["widget"].(map[string]any)["type"] = "magic"
	issues := ValidateFormSchema(doc(item))
	assert.Equal(t, "content.items[0].widget.type", issues[0].Path)
	assert.Contains(t, issues[0].Message, "未知的控件类型")
}

func TestValidateFormSchemaCommonBooleansRequired(t *testing.T) {
	item := validTextItem()
	delete(item["widget"].(map[string]any), "enable")
	assert.NotEmpty(t, ValidateFormSchema(doc(item)))

	widget2 := validTextItem()["widget"].(map[string]any)
	widget2["allowBlank"] = nil
	issues := ValidateFormSchema(doc(map[string]any{
		"widget":      widget2,
		"label":       "x",
		"description": "",
		"labelHidden": false,
		"lineWidth":   12,
	}))
	assert.True(t, containsPath(issues, "content.items[0].widget.allowBlank"))
}

func TestValidateFormSchemaWidgetNameRules(t *testing.T) {
	item := validTextItem()
	item["widget"].(map[string]any)["widgetName"] = "1bad"
	assert.NotEmpty(t, ValidateFormSchema(doc(item)))

	// 顶层重复
	dup := validTextItem()
	dup["widget"].(map[string]any)["widgetName"] = "_widget_a1"
	issues := ValidateFormSchema(doc(validTextItem(), dup))
	assert.Contains(t, issues[0].Message, "在当前作用域内重复")

	// 子表单作用域独立：与顶层同名合法；作用域内重复非法
	child := validTextItem()
	child["widget"].(map[string]any)["widgetName"] = "_widget_top"
	subform := map[string]any{
		"widget": validSubformWidget([]any{child}),
		"label":  "子表单", "description": "", "labelHidden": false, "lineWidth": 12,
	}
	top := validTextItem()
	top["widget"].(map[string]any)["widgetName"] = "_widget_top"
	assert.Empty(t, ValidateFormSchema(doc(top, subform)))

	child2 := validTextItem()
	child2["widget"].(map[string]any)["widgetName"] = "_widget_top"
	subform["widget"].(map[string]any)["items"] = []any{child, child2}
	assert.NotEmpty(t, ValidateFormSchema(doc(top, subform)))
}

func TestValidateFormSchemaLabelAndLimits(t *testing.T) {
	item := validTextItem()
	item["label"] = ""
	assert.NotEmpty(t, ValidateFormSchema(doc(item)))

	item = validTextItem()
	item["description"] = nil
	assert.NotEmpty(t, ValidateFormSchema(doc(item)))

	item = validTextItem()
	item["lineWidth"] = 13
	issues := ValidateFormSchema(doc(item))
	assert.Equal(t, "content.items[0].lineWidth", issues[0].Path)

	subform := map[string]any{
		"widget": validSubformWidget([]any{}),
		"label":  "子表单", "description": "", "labelHidden": false, "lineWidth": 6,
	}
	issues = ValidateFormSchema(doc(subform))
	assert.Equal(t, "子表单必须固定占整行（lineWidth=12）", issues[0].Message)

	// separator 允许空 label
	sep := map[string]any{
		"widget": map[string]any{
			"type": "separator", "widgetName": "_widget_sep",
			"enable": true, "visible": true, "allowBlank": true,
			"content": "区块分隔", "direction": "horizontal", "borderStyle": "double", "contentPosition": "left",
		},
		"label": "", "description": "", "labelHidden": false, "lineWidth": 12,
	}
	assert.Empty(t, ValidateFormSchema(doc(sep)))
}

func TestValidateFormSchemaOptions(t *testing.T) {
	radio := func(options any) map[string]any {
		item := validTextItem()
		w := item["widget"].(map[string]any)
		w["type"] = "radiogroup"
		w["options"] = options
		return item
	}
	// 必填
	assert.NotEmpty(t, ValidateFormSchema(doc(radio(nil))))
	// 至少一项
	assert.NotEmpty(t, ValidateFormSchema(doc(radio([]any{}))))
	// value 重复
	assert.NotEmpty(t, ValidateFormSchema(doc(radio([]any{
		map[string]any{"label": "A", "value": "a"},
		map[string]any{"label": "B", "value": "a"},
	}))))
	// 条目未知键
	assert.NotEmpty(t, ValidateFormSchema(doc(radio([]any{
		map[string]any{"label": "A", "value": "a", "extra": 1},
	}))))
}

func TestValidateFormSchemaCrossRules(t *testing.T) {
	item := validTextItem()
	w := item["widget"].(map[string]any)
	w["min"] = 10.0
	w["max"] = 1.0
	w["type"] = "number"
	assert.NotEmpty(t, ValidateFormSchema(doc(item)))

	item = validTextItem()
	w = item["widget"].(map[string]any)
	w["minLength"] = 10
	w["maxLength"] = 2
	assert.NotEmpty(t, ValidateFormSchema(doc(item)))

	// defaultValue 必须命中选项
	item = validTextItem()
	w = item["widget"].(map[string]any)
	w["type"] = "radiogroup"
	w["options"] = []any{map[string]any{"label": "A", "value": "a"}}
	w["defaultValue"] = "b"
	assert.NotEmpty(t, ValidateFormSchema(doc(item)))
}

func TestValidateFormSchemaSubformWhitelist(t *testing.T) {
	nested := map[string]any{
		"widget": validSubformWidget([]any{map[string]any{
			"widget": map[string]any{
				"type": "subform", "widgetName": "_widget_inner",
				"enable": true, "visible": true, "allowBlank": true, "items": []any{},
			},
			"label": "嵌套", "description": "", "labelHidden": false, "lineWidth": 12,
		}}),
		"label": "子表单", "description": "", "labelHidden": false, "lineWidth": 12,
	}
	issues := ValidateFormSchema(doc(nested))
	assert.Contains(t, issues[0].Message, "子表单内不允许使用控件")
}

func TestValidatePublishable(t *testing.T) {
	// 基础字段与成员选择字段可发布
	basic := []string{"text", "textarea", "number", "datetime", "radiogroup", "checkboxgroup", "combo", "combocheck", "separator", "user", "usergroup"}
	items := make([]any, 0, len(basic))
	for i, widgetType := range basic {
		item := validTextItem()
		w := item["widget"].(map[string]any)
		w["type"] = widgetType
		w["widgetName"] = fmt.Sprintf("_widget_b%d", i)
		delete(w, "placeholder") // 仅 text/textarea/number/combo 系携带 placeholder
		if widgetType == "radiogroup" || widgetType == "checkboxgroup" || widgetType == "combo" || widgetType == "combocheck" {
			w["options"] = []any{map[string]any{"label": "A", "value": "a"}}
		}
		items = append(items, item)
	}
	assert.Empty(t, ValidatePublishable(doc(items...)))

	// 白名单外（dept）给出精确路径
	dept := validTextItem()
	deptWidget := dept["widget"].(map[string]any)
	deptWidget["type"] = "dept"
	deptWidget["widgetName"] = "_widget_d1"
	delete(deptWidget, "placeholder")
	issues := ValidatePublishable(doc(validTextItem(), dept))
	assert.Equal(t, "content.items[1].widget.type", issues[0].Path)
}

func containsPath(issues []SchemaIssue, path string) bool {
	for _, issue := range issues {
		if issue.Path == path {
			return true
		}
	}
	return false
}

// ---- v5 字段显隐规则结构校验（与 TS validate.spec.ts 对拍） ----

func showRule(id, field, typ, method string, value []any, targets ...string) map[string]any {
	condition := map[string]any{"field": field, "type": typ, "method": method}
	if value != nil {
		condition["value"] = value
	}
	return map[string]any{
		"id":     id,
		"filter": map[string]any{"rel": "and", "cond": []any{condition}},
		"fields": toAnySlice(targets),
	}
}

func toAnySlice(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func rulesDoc(rules []any, items ...map[string]any) []byte {
	anyItems := make([]any, 0, len(items))
	for _, item := range items {
		anyItems = append(anyItems, item)
	}
	content := stringMapOfDoc(doc(anyItems...))
	content["fieldShowRules"] = rules
	raw, _ := json.Marshal(map[string]any{"content": content})
	return raw
}

func stringMapOfDoc(raw []byte) map[string]any {
	var parsed map[string]any
	_ = json.Unmarshal(raw, &parsed)
	content, _ := parsed["content"].(map[string]any)
	return content
}

func TestValidateFieldShowRulesAcceptsValidRule(t *testing.T) {
	source := validTextItem()
	source["widget"].(map[string]any)["widgetName"] = "_widget_src"
	target := validTextItem()
	target["widget"].(map[string]any)["widgetName"] = "_widget_target"
	assert.Empty(t, ValidateFormSchema(rulesDoc([]any{
		showRule("r1", "_widget_src", "text", "eq", []any{"甲"}, "_widget_target"),
	}, source, target)))
}

func TestValidateFieldShowRulesRequiredKey(t *testing.T) {
	// v5 起必填：缺失或非数组均拒绝。
	raw := doc(validTextItem())
	content := stringMapOfDoc(raw)
	delete(content, "fieldShowRules")
	rawWithout, _ := json.Marshal(map[string]any{"content": content})
	issues := ValidateFormSchema(rawWithout)
	assert.True(t, containsPath(issues, "content.fieldShowRules"))
}

func TestValidateFieldShowRulesUnknownFieldAndFingerprint(t *testing.T) {
	source := validTextItem()
	source["widget"].(map[string]any)["widgetName"] = "_widget_src"
	target := validTextItem()
	target["widget"].(map[string]any)["widgetName"] = "_widget_target"

	issues := ValidateFormSchema(rulesDoc([]any{
		showRule("r1", "_widget_missing", "text", "eq", []any{"甲"}, "_widget_target"),
	}, source, target))
	assert.True(t, containsPath(issues, "content.fieldShowRules[0].filter.cond[0].field"))

	issues = ValidateFormSchema(rulesDoc([]any{
		showRule("r1", "_widget_src", "number", "eq", []any{1.0}, "_widget_target"),
	}, source, target))
	assert.True(t, containsPath(issues, "content.fieldShowRules[0].filter.cond[0].type"))

	issues = ValidateFormSchema(rulesDoc([]any{
		showRule("r1", "_widget_src", "text", "between", []any{1.0, 2.0}, "_widget_target"),
	}, source, target))
	assert.True(t, containsPath(issues, "content.fieldShowRules[0].filter.cond[0].method"))
}

func TestValidateFieldShowRulesEmptyMethodAndValueShape(t *testing.T) {
	source := validTextItem()
	source["widget"].(map[string]any)["widgetName"] = "_widget_src"
	target := validTextItem()
	target["widget"].(map[string]any)["widgetName"] = "_widget_target"

	rule := showRule("r1", "_widget_src", "text", "isEmpty", []any{}, "_widget_target")
	issues := ValidateFormSchema(rulesDoc([]any{rule}, source, target))
	assert.True(t, containsPath(issues, "content.fieldShowRules[0].filter.cond[0].value"))

	rule = showRule("r1", "_widget_src", "text", "eq", nil, "_widget_target")
	issues = ValidateFormSchema(rulesDoc([]any{rule}, source, target))
	assert.True(t, containsPath(issues, "content.fieldShowRules[0].filter.cond[0].value"))

	// eq 恰 1 项。
	rule = showRule("r1", "_widget_src", "text", "eq", []any{"甲", "乙"}, "_widget_target")
	issues = ValidateFormSchema(rulesDoc([]any{rule}, source, target))
	assert.True(t, containsPath(issues, "content.fieldShowRules[0].filter.cond[0].value"))
}

func TestValidateFieldShowRulesStaticHiddenAndLayoutTargets(t *testing.T) {
	source := validTextItem()
	source["widget"].(map[string]any)["widgetName"] = "_widget_src"
	hidden := validTextItem()
	hidden["widget"].(map[string]any)["widgetName"] = "_widget_hidden"
	hidden["widget"].(map[string]any)["visible"] = false
	target := validTextItem()
	target["widget"].(map[string]any)["widgetName"] = "_widget_target"
	separator := map[string]any{
		"widget": map[string]any{
			"type": "separator", "widgetName": "_widget_sep",
			"enable": true, "visible": true, "allowBlank": true,
		},
		"label": "", "description": "", "labelHidden": false, "lineWidth": 12,
	}

	// 静态隐藏字段既不能作为条件源也不能作为目标。
	issues := ValidateFormSchema(rulesDoc([]any{
		showRule("r1", "_widget_hidden", "text", "eq", []any{"甲"}, "_widget_target"),
	}, source, hidden, target))
	assert.True(t, containsPath(issues, "content.fieldShowRules[0].filter.cond[0].field"))

	issues = ValidateFormSchema(rulesDoc([]any{
		showRule("r1", "_widget_src", "text", "eq", []any{"甲"}, "_widget_hidden"),
	}, source, hidden, target))
	assert.True(t, containsPath(issues, "content.fieldShowRules[0].fields[0]"))

	// 布局控件不能作为目标。
	issues = ValidateFormSchema(rulesDoc([]any{
		showRule("r1", "_widget_src", "text", "eq", []any{"甲"}, "_widget_sep"),
	}, source, target, separator))
	assert.True(t, containsPath(issues, "content.fieldShowRules[0].fields[0]"))

	// 同一目标被两条规则使用拒绝。
	issues = ValidateFormSchema(rulesDoc([]any{
		showRule("r1", "_widget_src", "text", "eq", []any{"甲"}, "_widget_target"),
		showRule("r2", "_widget_src", "text", "ne", []any{"乙"}, "_widget_target"),
	}, source, target))
	assert.True(t, containsPath(issues, "content.fieldShowRules[1].fields[0]"))
}

func TestValidateFieldShowRuleCycle(t *testing.T) {
	source := validTextItem()
	source["widget"].(map[string]any)["widgetName"] = "_widget_src"
	mid := validTextItem()
	mid["widget"].(map[string]any)["widgetName"] = "_widget_mid"
	numberField := map[string]any{
		"widget": map[string]any{
			"type": "number", "widgetName": "_widget_num",
			"enable": true, "visible": true, "allowBlank": true,
		},
		"label": "数字", "description": "", "labelHidden": false, "lineWidth": 12,
	}
	// src → mid → num → src 闭环。
	issues := ValidateFormSchema(rulesDoc([]any{
		showRule("r1", "_widget_src", "text", "eq", []any{"甲"}, "_widget_mid"),
		showRule("r2", "_widget_mid", "text", "notEmpty", nil, "_widget_num"),
		showRule("r3", "_widget_num", "number", "notEmpty", nil, "_widget_src"),
	}, source, mid, numberField))
	assert.True(t, len(issues) > 0)
	assert.Contains(t, issues[0].Message, "循环依赖")
	assert.Contains(t, issues[0].Message, "_widget_src → _widget_mid → _widget_num → _widget_src")
	assert.Regexp(t, `^content\.fieldShowRules\[\d+\]\.fields\[\d+\]$`, issues[0].Path)
}

func TestValidatePublishableConditionSource(t *testing.T) {
	dept := map[string]any{
		"widget": map[string]any{
			"type": "dept", "widgetName": "_widget_dept",
			"enable": true, "visible": true, "allowBlank": true,
		},
		"label": "部门", "description": "", "labelHidden": false, "lineWidth": 12,
	}
	target := validTextItem()
	target["widget"].(map[string]any)["widgetName"] = "_widget_target"
	issues := ValidatePublishable(rulesDoc([]any{
		showRule("r1", "_widget_dept", "dept", "eq", []any{"d1"}, "_widget_target"),
	}, dept, target))
	found := false
	for _, issue := range issues {
		if issue.Path == "content.fieldShowRules[0].filter.cond[0].field" &&
			issue.Message == "条件字段「_widget_dept」的运行能力尚未开放，暂不能发布" {
			found = true
		}
	}
	assert.True(t, found)
}

// ---- 不可见字段赋值校验（v6，与 TS validate.spec.ts 对拍） ----

// submitRulesDoc 在合法文档上叠加 v6 策略键（nil 表示显式删除该键）。
func submitRulesDoc(submitRule any, special map[string]any) []byte {
	var root map[string]any
	if err := json.Unmarshal(doc(validTextItem()), &root); err != nil {
		panic(err)
	}
	content := root["content"].(map[string]any)
	if submitRule == nil {
		delete(content, "submitRule")
	} else {
		content["submitRule"] = submitRule
	}
	if special == nil {
		delete(content, "widget_submit_rules")
	} else {
		content["widget_submit_rules"] = special
	}
	raw, _ := json.Marshal(root)
	return raw
}

func TestValidateFormSchemaSubmitRules(t *testing.T) {
	// 合法：默认策略 1/2、特殊规则 1/2、空对象。
	assert.Empty(t, ValidateFormSchema(submitRulesDoc(1, map[string]any{})))
	assert.Empty(t, ValidateFormSchema(submitRulesDoc(2, map[string]any{})))
	assert.Empty(t, ValidateFormSchema(submitRulesDoc(2, map[string]any{"_widget_a1": 1})))

	// 缺键 / 非法枚举 / 字符串数字。
	issues := ValidateFormSchema(submitRulesDoc(nil, nil))
	assert.True(t, containsPath(issues, "content.submitRule"))
	assert.True(t, containsPath(issues, "content.widget_submit_rules"))
	issues = ValidateFormSchema(submitRulesDoc("2", map[string]any{}))
	assert.True(t, containsPath(issues, "content.submitRule"))
	issues = ValidateFormSchema(submitRulesDoc(2, map[string]any{"_widget_a1": "1"}))
	assert.True(t, containsPath(issues, "content.widget_submit_rules._widget_a1"))

	// 未知字段与不可处理类型（布局/系统控件）。
	snItem := map[string]any{
		"widget": map[string]any{
			"type": "sn", "widgetName": "_widget_sn",
			"enable": true, "visible": true, "allowBlank": true,
		},
		"label": "流水号", "description": "", "labelHidden": false, "lineWidth": 12,
	}
	var root map[string]any
	assert.NoError(t, json.Unmarshal(doc(validTextItem(), snItem), &root))
	root["content"].(map[string]any)["submitRule"] = 2
	root["content"].(map[string]any)["widget_submit_rules"] = map[string]any{"_widget_ghost": 1, "_widget_sn": 1}
	raw, _ := json.Marshal(root)
	issues = ValidateFormSchema(raw)
	assert.True(t, containsPath(issues, "content.widget_submit_rules._widget_ghost"))
	assert.True(t, containsPath(issues, "content.widget_submit_rules._widget_sn"))

	// 与默认策略相同的冗余配置拒绝。
	issues = ValidateFormSchema(submitRulesDoc(2, map[string]any{"_widget_a1": 2}))
	assert.True(t, containsPath(issues, "content.widget_submit_rules._widget_a1"))

	// recompute 能力门控：特殊规则 3 与默认策略 3 均拒绝（P3 交付执行器前）。
	issues = ValidateFormSchema(submitRulesDoc(2, map[string]any{"_widget_a1": 3}))
	assert.True(t, containsPath(issues, "content.widget_submit_rules._widget_a1"))
	issues = ValidateFormSchema(submitRulesDoc(3, map[string]any{}))
	assert.True(t, containsPath(issues, "content.submitRule"))
	// submitRule=3 且全部可处理字段覆盖为 1/2：无未覆盖字段，不触发默认策略门控。
	assert.Empty(t, ValidateFormSchema(submitRulesDoc(3, map[string]any{"_widget_a1": 1})))
}

// 版本门控：v5 文档在 v5 合法、在 v6 缺键拒绝；v6 键在 v5 为未知键。
func TestValidateFormSchemaSubmitRuleVersionGating(t *testing.T) {
	v5Doc := []byte(`{"content":{"type":"form","layout":"normal","items":[],"layout_fields":[],"field_layout":[],"fieldShowRules":[]}}`)
	assert.Empty(t, ValidateFormSchema(v5Doc, 5))
	assert.True(t, containsPath(ValidateFormSchema(v5Doc, model.CurrentProtocolVersion), "content.submitRule"))

	v6Only := submitRulesDoc(2, map[string]any{})
	issues := ValidateFormSchema(v6Only, 5)
	assert.True(t, containsPath(issues, "content.submitRule"))
	assert.True(t, containsPath(issues, "content.widget_submit_rules"))
}
