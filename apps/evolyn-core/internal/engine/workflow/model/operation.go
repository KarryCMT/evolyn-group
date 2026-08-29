package model

// Operation 不可变业务操作记录（审批流水/时间线唯一事实源）。
// 追加写，禁止更新；与状态变更同事务写入（第 13.2 章事务模板第 8 步）。
type Operation struct {
	ID uint
	// TenantID 归属租户
	TenantID uint
	// InstanceID / TaskID 归属实例（TaskID 0=实例级操作，如 withdraw）
	InstanceID uint
	TaskID     uint
	// OperatorMemberID 操作人成员 ID（0=系统，如超时自动动作）
	OperatorMemberID uint
	// Type 操作类型
	Type OperationType
	// Payload 操作载荷（节点 key、意见、转办去向、字段修改摘要等；
	// JSONB 持久化，敏感字段出网前脱敏）
	Payload map[string]any
}

// OperationType 操作类型目录（Phase 0 冻结事件名/操作名，与
// event 包事件目录一一对应；新增操作类型必须显式追加，不复用语义）。
type OperationType string

const (
	OperationTypeStart           OperationType = "START"
	OperationTypeApprove         OperationType = "APPROVE"
	OperationTypeReject          OperationType = "REJECT"
	OperationTypeReturnToStarter OperationType = "RETURN_TO_STARTER"
	OperationTypeResubmit        OperationType = "RESUBMIT"
	OperationTypeWithdraw        OperationType = "WITHDRAW"
	OperationTypeTerminate       OperationType = "TERMINATE"
	OperationTypeTransfer        OperationType = "TRANSFER"
	OperationTypeCC              OperationType = "CC"
	OperationTypeTimeout         OperationType = "TIMEOUT"
	OperationTypeReminder        OperationType = "REMINDER"
)
