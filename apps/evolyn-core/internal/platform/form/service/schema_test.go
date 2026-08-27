package service

import (
	"encoding/json"
	"testing"

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

func doc(items ...any) []byte {
	if items == nil {
		items = []any{}
	}
	raw, _ := json.Marshal(map[string]any{"content": map[string]any{"type": "form", "items": items}})
	return raw
}

func TestValidateFormSchemaValidSample(t *testing.T) {
	assert.Empty(t, ValidateFormSchema(doc(validTextItem())))
	assert.Empty(t, ValidateFormSchema(doc()))
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
		"widget": map[string]any{
			"type": "subform", "widgetName": "_widget_sub",
			"enable": true, "visible": true, "allowBlank": true,
			"items": []any{child},
		},
		"label": "子表单", "description": "", "labelHidden": false, "lineWidth": 12,
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

	// separator 允许空 label
	sep := map[string]any{
		"widget": map[string]any{
			"type": "separator", "widgetName": "_widget_sep",
			"enable": true, "visible": true, "allowBlank": true, "style": "dashed",
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
		"widget": map[string]any{
			"type": "subform", "widgetName": "_widget_sub",
			"enable": true, "visible": true, "allowBlank": true,
			"items": []any{map[string]any{
				"widget": map[string]any{
					"type": "subform", "widgetName": "_widget_inner",
					"enable": true, "visible": true, "allowBlank": true, "items": []any{},
				},
				"label": "嵌套", "description": "", "labelHidden": false, "lineWidth": 12,
			}},
		},
		"label": "子表单", "description": "", "labelHidden": false, "lineWidth": 12,
	}
	issues := ValidateFormSchema(doc(nested))
	assert.Contains(t, issues[0].Message, "子表单内不允许使用控件")
}

func TestValidatePublishable(t *testing.T) {
	// 基础 9 类可发布
	basic := []string{"text", "textarea", "number", "datetime", "radiogroup", "checkboxgroup", "combo", "combocheck", "separator"}
	items := make([]any, 0, len(basic))
	for i, widgetType := range basic {
		item := validTextItem()
		w := item["widget"].(map[string]any)
		w["type"] = widgetType
		w["widgetName"] = "_widget_b" + string(rune('0'+i))
		delete(w, "placeholder") // 仅 text/textarea/number/combo 系携带 placeholder
		if widgetType == "radiogroup" || widgetType == "checkboxgroup" || widgetType == "combo" || widgetType == "combocheck" {
			w["options"] = []any{map[string]any{"label": "A", "value": "a"}}
		}
		items = append(items, item)
	}
	assert.Empty(t, ValidatePublishable(doc(items...)))

	// 白名单外（user）给出精确路径
	user := validTextItem()
	userWidget := user["widget"].(map[string]any)
	userWidget["type"] = "user"
	userWidget["widgetName"] = "_widget_u1"
	delete(userWidget, "placeholder")
	issues := ValidatePublishable(doc(validTextItem(), user))
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
