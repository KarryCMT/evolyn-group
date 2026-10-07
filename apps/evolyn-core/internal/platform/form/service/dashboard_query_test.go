package service

import (
	"strings"
	"testing"

	queryengine "evolyn/internal/engine/query"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileDashboardProjectionUsesBoundPublishedFieldsAndStableOrder(t *testing.T) {
	fields := map[string]dashboardQueryField{
		"field_region": {
			fieldID: "field_region",
			query: recordQueryField{
				mapping: SnapshotFieldMapping{WidgetName: "region", JSONBKey: "region", WidgetType: "input", FieldID: "field_region"},
				meta:    permissionFieldMeta{Key: "region", Label: "区域", WidgetType: "input"},
				class:   permFieldClassText,
			},
			typeOf: queryengine.FieldText,
		},
	}
	plan := queryengine.LogicalPlan{
		Projection: []string{"field_region"},
		Sorts:      []queryengine.Sort{{Field: "field_region", Direction: "asc"}},
		Paging:     queryengine.Paging{Page: 2, PageSize: 20},
	}

	params, columns, _, err := compileDashboardProjection(plan, fields, RecordQueryCompileOptions{})
	require.NoError(t, err)
	require.Len(t, params.Selects, 1)
	assert.NotContains(t, params.Selects[0], "field_region")
	assert.Equal(t, []any{"region", "region", "region", "region"}, params.SelectArgs)
	assert.Equal(t, "c0 ASC, r.id DESC", params.OrderBy)
	assert.Equal(t, 20, params.Limit)
	assert.Equal(t, 20, params.Offset)
	assert.Equal(t, "field_region", columns[0].Key)

	plan.Projection = []string{"field_region; DROP TABLE tn_form_records"}
	_, _, _, err = compileDashboardProjection(plan, fields, RecordQueryCompileOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unavailable")
}

func TestCompileDashboardAggregatePreservesHighPrecisionDecimal(t *testing.T) {
	amount := recordQueryField{
		mapping: SnapshotFieldMapping{WidgetName: "amount", JSONBKey: "amount", WidgetType: "decimal", FieldID: "field_amount"},
		meta:    permissionFieldMeta{Key: "amount", Label: "销售额", WidgetType: "decimal"},
		class:   permFieldClassNumber,
	}
	fields := map[string]dashboardQueryField{
		"field_amount": {fieldID: "field_amount", query: amount, typeOf: queryengine.FieldDecimal},
	}
	plan := queryengine.LogicalPlan{
		Aggregate:  true,
		Aggregates: []queryengine.Aggregate{{Field: "field_amount", Operator: queryengine.AggregateSum, Alias: "total_amount"}},
		Sorts:      []queryengine.Sort{{Field: "total_amount", Direction: "desc"}},
	}

	params, columns, outputTypes, err := compileDashboardProjection(plan, fields, RecordQueryCompileOptions{})
	require.NoError(t, err)
	require.Len(t, params.Selects, 1)
	assert.True(t, strings.HasPrefix(params.Selects[0], "SUM(("))
	assert.Contains(t, params.Selects[0], ")::numeric)::text AS c0")
	assert.Equal(t, dashboardMaxAggregateGroups+1, params.Limit)
	assert.Equal(t, "c0 DESC", params.OrderBy)
	assert.Equal(t, queryengine.FieldDecimal, outputTypes["total_amount"])
	assert.Equal(t, queryengine.FieldDecimal, columns[0].Type)
	assert.Equal(t, "9007199254740993.123456", normalizeDashboardQueryValue([]byte("9007199254740993.123456"), queryengine.FieldDecimal))
}

func TestCompileDashboardCountUsesCanonicalNumberType(t *testing.T) {
	name := recordQueryField{
		mapping: SnapshotFieldMapping{WidgetName: "name", JSONBKey: "name", WidgetType: "text", FieldID: "field_name"},
		meta:    permissionFieldMeta{Key: "name", Label: "名称", WidgetType: "text"},
		class:   permFieldClassText,
	}
	params, columns, outputTypes, err := compileDashboardProjection(queryengine.LogicalPlan{
		Aggregate:  true,
		Aggregates: []queryengine.Aggregate{{Field: "field_name", Operator: queryengine.AggregateCount, Alias: "record_count"}},
	}, map[string]dashboardQueryField{
		"field_name": {fieldID: "field_name", query: name, typeOf: queryengine.FieldText},
	}, RecordQueryCompileOptions{})
	require.NoError(t, err)
	assert.Contains(t, params.Selects[0], "COUNT(")
	assert.Equal(t, queryengine.FieldNumber, columns[0].Type)
	assert.Equal(t, queryengine.FieldNumber, outputTypes["record_count"])
}
