package service

import (
	"context"
	iammodel "evolyn/internal/platform/iam/model"
)

// SubmittedWorkflowRecord 仅传递服务端已验证并落库的表单绑定，不接受客户端流程编码。
type SubmittedWorkflowRecord struct {
	FormCode                               string
	AppID, FormID, FormVersionID, RecordID uint
}

// WorkflowStarter 在记录提交事务内创建实例；失败必须向上传播，回滚记录。
// 表单提交已完成成员与 add 权限检查，此端口不额外要求设计权限。
type WorkflowStarter interface {
	StartSubmittedRecord(ctx context.Context, member *iammodel.User, record SubmittedWorkflowRecord) (string, error)
}

type WorkflowStarterInjector interface{ UseWorkflowStarter(WorkflowStarter) }

func (s *formService) UseWorkflowStarter(starter WorkflowStarter) { s.workflowStarter = starter }
