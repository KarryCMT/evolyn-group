package service

import (
	"encoding/json"
	enginemodel "evolyn/internal/engine/workflow/model"
	"testing"
)

func TestTaskCardFieldsProtectsPermissionsAndBoundsSummary(t *testing.T) {
	content := json.RawMessage(`{"content":{"items":[
 {"label":"未授权","widget":{"widgetName":"secret"}},
 {"label":"隐藏","widget":{"widgetName":"hidden"}},
 {"label":"复杂值","widget":{"widgetName":"object"}},
 {"label":"申请编号","widget":{"widgetName":"number"}},
 {"label":"数量","widget":{"widgetName":"count"}},
 {"label":"是否加急","widget":{"widgetName":"urgent"}},
 {"label":"备注","widget":{"widgetName":"note"}}
 ]}}`)
	fields := taskCardFields(content, map[string]any{
		"secret": "不可泄露", "hidden": "不可泄露", "object": map[string]any{"secret": "不可泄露"},
		"number": "AP-001", "count": float64(2), "urgent": false, "note": "第四条不展示",
	}, map[string]enginemodel.FieldPermission{
		"hidden": enginemodel.FieldPermissionHidden, "object": enginemodel.FieldPermissionReadonly,
		"number": enginemodel.FieldPermissionReadonly, "count": enginemodel.FieldPermissionEditable,
		"urgent": enginemodel.FieldPermissionReadonly, "note": enginemodel.FieldPermissionReadonly,
	})
	if len(fields) != 3 {
		t.Fatalf("expected three safe fields, got %#v", fields)
	}
	if fields[0].Label != "申请编号" || fields[0].Value != "AP-001" || fields[1].Value != "2" || fields[2].Value != "否" {
		t.Fatalf("unexpected summary: %#v", fields)
	}
}

func TestTaskCardFieldsMalformedDocumentIsEmpty(t *testing.T) {
	if fields := taskCardFields(json.RawMessage(`{"content":`), nil, nil); len(fields) != 0 {
		t.Fatalf("invalid snapshot leaked fields: %#v", fields)
	}
}
