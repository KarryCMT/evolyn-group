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
		"field_layout": []any{"_layout_tabs"},
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
