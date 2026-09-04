package formula

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectFieldsPreservesActualWidgetValueKinds(t *testing.T) {
	fields, err := ProjectFields([]byte(`{"content":{"items":[
		{"label":"姓名","widget":{"type":"text","widgetName":"name"}},
		{"label":"金额","widget":{"type":"number","widgetName":"amount"}},
		{"label":"标签","widget":{"type":"checkboxgroup","widgetName":"tags"}},
		{"label":"负责人","widget":{"type":"user","widgetName":"owner"}},
		{"label":"分割线","widget":{"type":"separator","widgetName":"divider"}}
	]}}`))
	require.NoError(t, err)
	require.Equal(t, []Field{
		{Key: "name", Label: "姓名", WidgetType: "text", ValueType: ValueTypeText, DisplayType: "文本", FormulaAllowed: true},
		{Key: "amount", Label: "金额", WidgetType: "number", ValueType: ValueTypeNumber, DisplayType: "数字", FormulaAllowed: true},
		{Key: "tags", Label: "标签", WidgetType: "checkboxgroup", ValueType: ValueTypeArray, DisplayType: "数组", FormulaAllowed: true},
		{Key: "owner", Label: "负责人", WidgetType: "user", ValueType: ValueTypeMember, DisplayType: "成员", FormulaAllowed: false},
	}, fields)
}

func TestProjectFieldsRejectsMalformedDocument(t *testing.T) {
	_, err := ProjectFields([]byte(`{`))
	require.Error(t, err)
}
