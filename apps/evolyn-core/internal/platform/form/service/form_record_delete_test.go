package service

import (
	"testing"

	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteRecordsRemovesOnlySelectedRecordsOfCurrentForm(t *testing.T) {
	forms := newFakeFormRepo()
	forms.forms[101] = &model.Form{ID: 101, Code: "form_orders", Name: "订单"}
	records := &fakeRecordRepo{records: []*model.FormRecord{
		{ID: 1, TenantID: 1, FormID: 101, Values: model.JSONContent(`{"_widget_a":"A"}`)},
		{ID: 2, TenantID: 1, FormID: 101, Values: model.JSONContent(`{"_widget_a":"B"}`)},
		{ID: 3, TenantID: 1, FormID: 102, Values: model.JSONContent(`{"_widget_a":"C"}`)},
	}}
	svc := newTestService(&fakeQuota{limit: -1}, forms, newFakeVersionRepo(), records, nil)

	result, err := svc.DeleteRecords(tenantCtx(1), memberOfTenant(1), "form_orders", &model.DeleteFormRecordsRequest{RecordIDs: []uint{1, 2}})
	require.NoError(t, err)
	assert.Equal(t, 2, result.DeletedCount)
	assert.Len(t, records.records, 1)
	assert.Equal(t, uint(3), records.records[0].ID)
}

func TestDeleteRecordsRejectsWorkflowRecordWithoutDeletingAnything(t *testing.T) {
	forms := newFakeFormRepo()
	forms.forms[101] = &model.Form{ID: 101, Code: "form_orders", Name: "订单"}
	records := &fakeRecordRepo{records: []*model.FormRecord{
		{ID: 1, TenantID: 1, FormID: 101, WorkflowInstanceNo: "WF-001"},
	}}
	svc := newTestService(&fakeQuota{limit: -1}, forms, newFakeVersionRepo(), records, nil)

	_, err := svc.DeleteRecords(tenantCtx(1), memberOfTenant(1), "form_orders", &model.DeleteFormRecordsRequest{RecordIDs: []uint{1}})
	assert.ErrorIs(t, err, apperrors.ErrRecordWorkflowActive)
	assert.Len(t, records.records, 1)
}
