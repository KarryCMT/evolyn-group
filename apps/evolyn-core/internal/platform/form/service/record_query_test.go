package service

import (
	"encoding/json"
	"testing"

	"evolyn/internal/platform/form/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileRecordQueryConditionUsesFrozenMapping(t *testing.T) {
	compiled, err := CompileRecordQueryCondition([]SnapshotFieldMapping{{
		WidgetName: "_widget_name", JSONBKey: "_widget_name",
	}}, RecordQueryCondition{Field: "_widget_name", Operator: "contains", Value: json.RawMessage(`"100%"`)})
	require.NoError(t, err)
	// SQL 模板包含 JSON 类型守卫；字段键和值均只能作为绑定参数出现，不能被
	// 包含通配符的用户值改写成 SQL/LIKE 模板。
	assert.Contains(t, compiled.Where, "position(? in")
	assert.NotContains(t, compiled.Where, "100%")
	assert.Equal(t, "100%", compiled.Args[len(compiled.Args)-1])
}

func TestCompileRecordQueryConditionRejectsUnknownFieldAndOperator(t *testing.T) {
	mappings := []SnapshotFieldMapping{{WidgetName: "_widget_name", JSONBKey: "_widget_name"}}
	_, err := CompileRecordQueryCondition(mappings, RecordQueryCondition{Field: "unknown", Operator: "eq", Value: json.RawMessage(`"x"`)})
	assert.Error(t, err)
	_, err = CompileRecordQueryCondition(mappings, RecordQueryCondition{Field: "_widget_name", Operator: "gt", Value: json.RawMessage(`"x"`)})
	assert.Error(t, err)
}

func TestCompileRecordQueryConditionCompilesNumericMembership(t *testing.T) {
	compiled, err := CompileRecordQueryCondition([]SnapshotFieldMapping{{
		WidgetName: "amount", WidgetType: "number", JSONBKey: "amount",
	}}, RecordQueryCondition{Field: "amount", Operator: "in", Value: json.RawMessage(`[1, 2.5]`)})
	require.NoError(t, err)
	assert.Contains(t, compiled.Where, "::numeric = ?")
	assert.Contains(t, compiled.Args, float64(1))
	assert.Contains(t, compiled.Args, float64(2.5))
}

func TestSnapshotFieldMappingsDerivesLegacyPublishedSnapshot(t *testing.T) {
	version := &model.FormVersion{
		// 000065 之前的不可变版本仅有 content；读取侧不得清空这些历史
		// 记录，而是从同一版本的快照内容导出受控字段白名单。
		FieldMappings: model.JSONContent(`[]`),
		Content: model.JSONContent(`{"content":{"items":[{
			"label":"名称","widget":{"type":"text","widgetName":"name"}
		}]}}`),
	}
	mappings, err := snapshotFieldMappings(version)
	require.NoError(t, err)
	require.Equal(t, []SnapshotFieldMapping{{
		WidgetName: "name", WidgetType: "text", JSONBKey: "name", PhysicalColumn: "f_name",
	}}, mappings)
}
