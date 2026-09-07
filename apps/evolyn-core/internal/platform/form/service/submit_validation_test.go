package service

import (
	"encoding/json"
	"os"
	"testing"

	"evolyn/internal/platform/form/model"

	"github.com/stretchr/testify/require"
)

func TestSubmitValidationGoldenFixtures(t *testing.T) {
	raw, err := os.ReadFile("testdata/submit_validation_golden.json")
	require.NoError(t, err)
	var fixtures []struct {
		Name, Formula, Remind string
		Value                 any
		Passed                bool `json:"passed"`
	}
	require.NoError(t, json.Unmarshal(raw, &fixtures))
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			root := map[string]any{"content": map[string]any{"items": []any{map[string]any{"widget": map[string]any{"type": "text", "widgetName": "_widget_phone", "visible": true}}, map[string]any{"widget": map[string]any{"type": "text", "widgetName": "_widget_option", "visible": true}}}, "validators": []any{map[string]any{"formula": fixture.Formula, "remind": fixture.Remind, "realtime": false, "failAction": 0, "remark": ""}}, "preSubmitConfirm": map[string]any{"enable": false, "title": "确认", "content": "确认"}}}
			compiled, err := CompileSubmitRules(root, 7)
			require.NoError(t, err)
			failures, err := ValidateCompiledSubmitRules(compiled, root, 7, map[string]any{"_widget_phone": fixture.Value, "_widget_option": fixture.Value})
			require.NoError(t, err)
			require.Equal(t, !fixture.Passed, len(failures) > 0)
		})
	}
}

func TestCompiledSubmitRulesUseFrozenVersionProgram(t *testing.T) {
	root := map[string]any{"content": map[string]any{
		"items": []any{
			map[string]any{"label": "联系电话", "widget": map[string]any{"type": "text", "widgetName": "_widget_phone", "visible": true}},
		},
		"validators":       []any{map[string]any{"formula": "LEN($_widget_phone#) == 11", "remind": "联系电话 ${_widget_phone} 不是 11 位", "realtime": true, "failAction": 0, "remark": ""}},
		"preSubmitConfirm": map[string]any{"enable": false, "title": "确认", "content": "${_widget_phone}"},
	}}
	compiled, err := CompileSubmitRules(root, 7)
	require.NoError(t, err)

	failures, err := ValidateCompiledSubmitRules(compiled, root, 7, map[string]any{"_widget_phone": "123"})
	require.NoError(t, err)
	require.Equal(t, []submitValidatorError{{Index: 0, Remind: "联系电话 123 不是 11 位", Fields: []string{"_widget_phone"}}}, failures)

	// Version content changes after publishing must not be silently combined with the
	// old compiled program: the digest catches the configuration integrity error.
	changed := map[string]any{}
	raw, _ := json.Marshal(root)
	require.NoError(t, json.Unmarshal(raw, &changed))
	changed["content"].(map[string]any)["validators"] = []any{}
	_, err = ValidateCompiledSubmitRules(compiled, changed, 7, map[string]any{"_widget_phone": "123"})
	require.Error(t, err)
}

func TestCompileSubmitRulesIgnoresLegacySnapshots(t *testing.T) {
	compiled, err := CompileSubmitRules(map[string]any{"content": map[string]any{}}, 6)
	require.NoError(t, err)
	require.Equal(t, model.JSONContent(`{}`), compiled)
}

func TestPublishRejectsExistingAddGroupMissingSubmitRuleField(t *testing.T) {
	content := map[string]any{"content": map[string]any{
		"items":            []any{map[string]any{"label": "联系电话", "widget": map[string]any{"type": "text", "widgetName": "phone", "visible": true}}},
		"validators":       []any{map[string]any{"formula": "LEN($phone#) == 11", "remind": "号码不正确", "realtime": false, "failAction": 0, "remark": ""}},
		"preSubmitConfirm": map[string]any{"enable": false, "title": "确认", "content": "确认"},
	}}
	group := model.AssetPermissionGroup{Code: "fpg_missing", Enabled: true,
		Operations: model.PermissionOperations{model.PermissionOpAdd}, FieldPermissions: model.PermissionFieldRules{}}
	require.Error(t, ValidateSubmitRulePermissionGroups(content, 7, []model.AssetPermissionGroup{group}))

	group.FieldPermissions = model.PermissionFieldRules{{Field: "phone", Visible: true, Editable: true}}
	require.NoError(t, ValidateSubmitRulePermissionGroups(content, 7, []model.AssetPermissionGroup{group}))
}
