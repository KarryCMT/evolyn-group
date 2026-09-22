package service

import (
	"encoding/json"
	"testing"

	"evolyn/internal/platform/form/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fieldFormulaDocument(t *testing.T) map[string]any {
	t.Helper()
	first := validTextItem()
	second := validTextItem()
	second["widget"].(map[string]any)["widgetName"] = "_widget_b1"
	third := validTextItem()
	third["widget"].(map[string]any)["widgetName"] = "_widget_c1"
	var document map[string]any
	require.NoError(t, json.Unmarshal(doc(first, second, third), &document))
	document["content"].(map[string]any)["fieldFormulas"] = []any{
		map[string]any{
			"id": "formula_product_name", "version": 1, "enabled": true,
			"targetFieldId": "_widget_c1",
			"formula":       `CONCATENATE($_widget_a1#, "-", UPPER($_widget_b1#))`,
			"remark":        "产品名称",
		},
	}
	return document
}

func TestCompileAndApplyFieldFormulas(t *testing.T) {
	document := fieldFormulaDocument(t)
	assert.Empty(t, ValidateFormSchema(mustJSON(t, document), model.CurrentProtocolVersion))
	compiled, err := CompileFieldFormulas(document, model.CurrentProtocolVersion)
	require.NoError(t, err)

	values := map[string]any{"_widget_a1": "P100", "_widget_b1": "blue", "_widget_c1": "伪造值"}
	require.NoError(t, ApplyCompiledFieldFormulas(compiled, document, model.CurrentProtocolVersion, values))
	assert.Equal(t, "P100-BLUE", values["_widget_c1"])
}

func TestFieldFormulaRejectsCycleAndLinkageTargetConflict(t *testing.T) {
	document := fieldFormulaDocument(t)
	content := document["content"].(map[string]any)
	content["fieldFormulas"] = append(content["fieldFormulas"].([]any), map[string]any{
		"id": "formula_cycle_back", "version": 1, "enabled": true,
		"targetFieldId": "_widget_a1", "formula": `$_widget_c1#`, "remark": "cycle",
	})
	assert.True(t, containsPath(ValidateFormSchema(mustJSON(t, document)), "content.fieldFormulas"))

	document = fieldFormulaDocument(t)
	content = document["content"].(map[string]any)
	content["linkages"] = []any{map[string]any{
		"id": "linkage_conflict", "version": 1, "enabled": true,
		"source": map[string]any{"type": "form", "appId": 1, "sourceId": "form_products"},
		"filter": map[string]any{"logic": "and", "conditions": []any{map[string]any{
			"id": "condition_1", "sourceFieldId": "name", "operator": "eq",
			"value": map[string]any{"type": "field", "fieldId": "_widget_a1"},
		}}},
		"mappings": []any{map[string]any{"sourceFieldId": "name", "targetFieldId": "_widget_c1"}},
		"result":   map[string]any{"mode": "first"},
		"runtime":  map[string]any{"trigger": "dependency_change", "runOnInit": true, "debounceMs": 250, "emptyStrategy": "clear", "errorStrategy": "keep"},
	}}
	assert.True(t, containsPath(ValidateFormSchema(mustJSON(t, document)), "content.fieldFormulas"))
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	return raw
}
