package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"evolyn/internal/contextx"
	engineruntime "evolyn/internal/engine/workflow/runtime"
	"evolyn/internal/infrastructure"
	formservice "evolyn/internal/platform/form/service"
	iammodel "evolyn/internal/platform/iam/model"
	wfapp "evolyn/internal/platform/workflow"
	"gorm.io/gorm"
)

var _ formservice.WorkflowStarter = (*runtimeService)(nil)

// StartSubmittedRecord 是表单提交内部端口。调用方完成表单 add 权限校验；
// 强制复用记录事务，实例、审批任务、通知事件与业务记录共同提交或回滚。
func (s *runtimeService) StartSubmittedRecord(ctx context.Context, member *iammodel.User, record formservice.SubmittedWorkflowRecord) (string, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok || member == nil || member.ID == 0 || member.TenantID != tenantID {
		return "", wfapp.ErrForbidden
	}
	if !infrastructure.InTransaction(ctx) {
		return "", fmt.Errorf("workflow form submission requires record transaction")
	}
	if record.RecordID == 0 || record.FormID == 0 || record.FormVersionID == 0 || record.AppID == 0 {
		return "", wfapp.ErrFormVersionInvalid
	}
	code, err := s.definitions.FindCodeByFormCode(ctx, record.FormCode)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", wfapp.ErrNotPublished
	}
	if err != nil {
		return "", err
	}
	result, err := s.engine.Start(ctx, engineruntime.StartInput{
		TenantID: tenantID, Code: code, BusinessType: "form_record",
		BusinessID:      strconv.FormatUint(uint64(record.RecordID), 10),
		StarterMemberID: member.ID, AppID: record.AppID,
		FormID: record.FormID, FormVersionID: record.FormVersionID,
		IdempotencyKey: fmt.Sprintf("form-record:%d", record.RecordID),
	})
	if err != nil {
		return "", mapEngineError(err)
	}
	instance, err := s.reader.FindInstanceRow(ctx, result.InstanceID)
	if err != nil {
		return "", err
	}
	return instance.InstanceNo, nil
}
