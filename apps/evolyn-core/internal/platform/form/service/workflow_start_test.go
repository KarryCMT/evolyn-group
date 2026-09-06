package service

import (
	"context"
	"errors"
	"testing"

	"evolyn/internal/platform/form/model"
	iammodel "evolyn/internal/platform/iam/model"
	"github.com/stretchr/testify/require"
)

type submissionTxKey struct{}

// rollbackRecordTx 模拟提交事务，以验证启动失败向上传播时业务记录也会撤销。
type rollbackRecordTx struct{ records *fakeRecordRepo }

func (tx rollbackRecordTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	before := len(tx.records.records)
	err := fn(context.WithValue(ctx, submissionTxKey{}, true))
	if err != nil {
		tx.records.records = tx.records.records[:before]
	}
	return err
}

type submissionStarter struct {
	calls         []SubmittedWorkflowRecord
	err           error
	inTransaction bool
}

func (s *submissionStarter) StartSubmittedRecord(ctx context.Context, member *iammodel.User, record SubmittedWorkflowRecord) (string, error) {
	s.inTransaction, _ = ctx.Value(submissionTxKey{}).(bool)
	s.calls = append(s.calls, record)
	return "WF-20260905-000001", s.err
}

func TestSubmitWorkflowRecordStartsAtomicallyAndDoesNotReplay(t *testing.T) {
	for _, formType := range []model.FormType{model.FormTypeStandard, model.FormTypeWorkflow} {
		t.Run(string(formType), func(t *testing.T) {
			forms, versions, records := newFakeFormRepo(), newFakeVersionRepo(), &fakeRecordRepo{}
			svc := newTestService(&fakeQuota{limit: -1}, forms, versions, records, nil).(*formService)
			svc.tx = rollbackRecordTx{records: records}
			starter := &submissionStarter{}
			svc.UseWorkflowStarter(starter)
			ctx, member := tenantCtx(1), memberOfTenant(1)
			created, err := svc.Create(ctx, member, &model.CreateFormRequest{ApplicationID: 7, Name: "流程提交回归", FormType: formType})
			require.NoError(t, err)
			_, err = svc.SaveDraft(ctx, member, created.Code, &model.SaveDraftRequest{DraftRevision: 1, ProtocolVersion: model.CurrentProtocolVersion, Content: validDraft()})
			require.NoError(t, err)
			published, err := svc.Publish(ctx, member, created.Code, &model.PublishRequest{DraftRevision: 2})
			require.NoError(t, err)
			req := &model.SubmitRecordRequest{AppCode: "app_x", FormCode: created.Code, PublishedVersion: 1,
				SchemaRevision: published.SchemaRevision, HasResult: submitBool(true), DataOpID: "6e243bbb-7d57-4e59-952b-d530c53c6561",
				Values: map[string]model.SubmitFieldValue{"_widget_a": submittedValue(`"申请人"`, true)}}
			result, err := svc.SubmitRecord(ctx, member, req)
			require.NoError(t, err)
			replay, err := svc.SubmitRecord(ctx, member, req)
			require.NoError(t, err)
			require.Equal(t, result.RecordID, replay.RecordID)
			require.Len(t, records.records, 1)
			if formType == model.FormTypeStandard {
				require.Empty(t, starter.calls)
				return
			}
			require.Equal(t, "WF-20260905-000001", result.WorkflowInstanceNo)
			require.Equal(t, result.WorkflowInstanceNo, replay.WorkflowInstanceNo)
			require.Equal(t, result.WorkflowInstanceNo, records.records[0].WorkflowInstanceNo)
			require.Len(t, starter.calls, 1)
			require.True(t, starter.inTransaction)
			require.Equal(t, SubmittedWorkflowRecord{FormCode: created.Code, AppID: 7, FormID: records.records[0].FormID, FormVersionID: records.records[0].FormVersionID, RecordID: result.RecordID}, starter.calls[0])
			// 模拟未发布/引擎发起失败：不能再返回“提交成功”或留下第二条记录。
			starter.err = errors.New("workflow not published")
			req.DataOpID = "6e243bbb-7d57-4e59-952b-d530c53c6562"
			_, err = svc.SubmitRecord(ctx, member, req)
			require.ErrorIs(t, err, starter.err)
			require.Len(t, records.records, 1)
		})
	}
}
