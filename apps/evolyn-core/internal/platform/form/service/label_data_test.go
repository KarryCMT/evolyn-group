package service

import (
	"context"
	"testing"

	"evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/form/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type labelBatchRecordRepo struct {
	*fakeRecordRepo
	calls  int
	denyID uint
}

func (r *labelBatchRecordRepo) ListControlled(_ context.Context, params repository.RecordListParams) ([]model.FormRecord, int64, error) {
	r.calls++
	requested := make(map[uint]bool)
	for _, arg := range params.Args {
		if id, ok := arg.(uint); ok {
			requested[id] = true
		}
	}
	rows := make([]model.FormRecord, 0, len(requested))
	for _, record := range r.records {
		if record.FormID == params.FormID && requested[record.ID] && record.ID != r.denyID {
			rows = append(rows, *record)
		}
	}
	return rows, int64(len(rows)), nil
}

func labelDataService(t *testing.T, denyID uint) (*formService, *labelBatchRecordRepo) {
	t.Helper()
	forms := newFakeFormRepo()
	versions := newFakeVersionRepo()
	formAsset := &model.Form{Code: "form_users", AppID: 7, FormType: model.FormTypeStandard}
	formAsset.ID, formAsset.TenantID = 1, 1
	forms.forms[formAsset.ID] = formAsset
	version, err := versions.Create(tenantCtx(1), &model.FormVersion{
		FormID: formAsset.ID, VersionNo: 1, Content: model.JSONContent(permTestDoc), TenantID: 1,
	})
	require.NoError(t, err)
	formAsset.LatestVersionID = &version.ID
	formAsset.PublishedVersion = 1
	records := &labelBatchRecordRepo{fakeRecordRepo: &fakeRecordRepo{}, denyID: denyID}
	for id, name := range map[uint]string{1: "张三", 2: "李四"} {
		records.records = append(records.records, &model.FormRecord{
			ID: id, FormID: formAsset.ID, FormVersionID: version.ID, Values: model.JSONContent(`{"name":"` + name + `","amount":88}`),
			SubmittedByName: "提交人",
		})
		records.records[len(records.records)-1].TenantID = 1
	}
	svc := &formService{
		tx: passThroughTx{}, repo: forms, versions: versions, records: records,
		access: fakeAccess{perms: map[string]bool{"form-records:get": true}},
	}
	svc.UsePermissionEvaluator(fakePermEvaluator{resolved: &ResolvedFormPermission{
		Matched: []MatchedGroup{{
			Code: "fpg_print", Operations: map[string]bool{model.PermissionOpBatchPrint: true},
			Fields: map[string]FieldPermission{
				"name": {Visible: true}, "amount": {Visible: false},
			},
		}},
		fieldList: permFieldList(t),
	}})
	return svc, records
}

func TestReadLabelRecordsForOperationUsesOneQueryAndInputOrder(t *testing.T) {
	svc, records := labelDataService(t, 0)
	member := memberOfTenant(1)
	allowed, err := svc.CanUseLabelOperation(tenantCtx(1), member, 1, model.PermissionOpBatchPrint)
	require.NoError(t, err)
	require.True(t, allowed)
	result, err := svc.ReadLabelRecordsForOperation(tenantCtx(1), member, 1, []uint{2, 1}, model.PermissionOpBatchPrint)
	require.NoError(t, err)
	require.Equal(t, 1, records.calls)
	require.Equal(t, "2", result[0].System["recordId"])
	require.Equal(t, "1", result[1].System["recordId"])
	require.Equal(t, "李四", result[0].Fields["name"])
	assert.NotContains(t, result[0].Fields, "amount")
}

func TestReadLabelRecordsForOperationRejectsWholeBatchOnScopeMissAndCrossTenant(t *testing.T) {
	svc, _ := labelDataService(t, 2)
	_, err := svc.ReadLabelRecordsForOperation(tenantCtx(1), memberOfTenant(1), 1, []uint{1, 2}, model.PermissionOpBatchPrint)
	assert.ErrorIs(t, err, form.ErrForbidden)

	_, err = svc.ReadLabelRecordsForOperation(tenantCtx(1), memberOfTenant(2), 1, []uint{1}, model.PermissionOpBatchPrint)
	assert.ErrorIs(t, err, form.ErrForbidden)

	svc.UsePermissionEvaluator(fakePermEvaluator{resolved: &ResolvedFormPermission{fieldList: permFieldList(t)}})
	allowed, err := svc.CanUseLabelOperation(tenantCtx(1), memberOfTenant(1), 1, model.PermissionOpBatchPrint)
	require.NoError(t, err)
	require.False(t, allowed)
}
