package service

import (
	"encoding/json"
	"testing"

	"evolyn/internal/platform/form/model"
	"evolyn/internal/testsupport"

	"github.com/stretchr/testify/require"
)

// TestPermissionScopeSQLMatchesMemory is deliberately database-backed: it proves
// the generated PostgreSQL predicate stays aligned with permissionScopeMatches,
// including the non-obvious NULL/not_in semantics. testsupport skips it when a
// disposable TEST_PG_DSN is not configured.
func TestPermissionScopeSQLMatchesMemory(t *testing.T) {
	db := testsupport.NewPostgres(t)
	fields := []permissionFieldMeta{
		{Key: "text", WidgetType: "text"},
		{Key: "number", WidgetType: "number"},
		{Key: "date", WidgetType: "datetime", Format: "date"},
		{Key: "status", WidgetType: "radiogroup"},
		{Key: "tags", WidgetType: "checkboxgroup"},
	}
	mappings := []SnapshotFieldMapping{
		{WidgetName: "text", WidgetType: "text", JSONBKey: "text"},
		{WidgetName: "number", WidgetType: "number", JSONBKey: "number"},
		{WidgetName: "date", WidgetType: "datetime", JSONBKey: "date"},
		{WidgetName: "status", WidgetType: "radiogroup", JSONBKey: "status"},
		{WidgetName: "tags", WidgetType: "checkboxgroup", JSONBKey: "tags"},
	}
	records := []map[string]any{
		{"text": "alpha", "number": 12, "date": "2026-09-04", "status": "open", "tags": []any{"red", "blue"}},
		{"text": "", "number": "bad", "status": "closed", "tags": []any{}},
		{"text": nil, "number": 2, "date": "bad-date", "tags": []any{"green"}},
		{},
	}
	scopes := []model.PermissionDataScopeSpec{
		{Match: "all", Conditions: []model.PermissionDataCondition{{Field: "text", Operator: "eq", Value: []any{"alpha"}}, {Field: "number", Operator: "gt", Value: []any{10}}}},
		{Match: "all", Conditions: []model.PermissionDataCondition{{Field: "text", Operator: "ne", Value: []any{"alpha"}}, {Field: "number", Operator: "gte", Value: []any{2}}, {Field: "number", Operator: "lt", Value: []any{20}}}},
		{Match: "any", Conditions: []model.PermissionDataCondition{{Field: "status", Operator: "in", Value: []any{"open"}}, {Field: "text", Operator: "empty"}}},
		{Match: "all", Conditions: []model.PermissionDataCondition{{Field: "tags", Operator: "contains", Value: []any{"red"}}}},
		{Match: "all", Conditions: []model.PermissionDataCondition{{Field: "tags", Operator: "not_in", Value: []any{"red"}}}},
		{Match: "all", Conditions: []model.PermissionDataCondition{{Field: "date", Operator: "not_empty"}, {Field: "date", Operator: "lte", Value: []any{"2026-12-31"}}}},
	}

	for _, scope := range scopes {
		compiled, err := CompilePermissionScopeSQL(scope, mappings, fields)
		require.NoError(t, err)
		for _, record := range records {
			payload, err := json.Marshal(record)
			require.NoError(t, err)
			var matched bool
			args := append(append([]any{}, compiled.Args...), string(payload))
			require.NoError(t, db.Raw("SELECT "+compiled.Where+" AS matched FROM (SELECT ?::jsonb AS values) r", args...).Scan(&matched).Error)
			require.Equal(t, permissionScopeMatches(scope, fields, record), matched, "scope=%+v record=%s", scope, payload)
		}
	}
}

func TestCompilePermissionScopeSQLRejectsUnfrozenField(t *testing.T) {
	_, err := CompilePermissionScopeSQL(model.PermissionDataScopeSpec{Conditions: []model.PermissionDataCondition{{Field: "values ->> 'secret'", Operator: "eq", Value: []any{"x"}}}}, []SnapshotFieldMapping{{WidgetName: "text", WidgetType: "text", JSONBKey: "text"}}, []permissionFieldMeta{{Key: "text", WidgetType: "text"}})
	require.Error(t, err)
}
