package model

// Task 面向用户的人工待办（第 6.2 / 10 章）。原任务转办后关闭为
// TRANSFERRED 并另建新任务，不直接修改原审批人（第 10.5 章）。
type Task struct {
	ID uint
	// TenantID 归属租户
	TenantID uint
	// InstanceID / NodeInstanceID 归属实例与节点实例
	InstanceID     uint
	NodeInstanceID uint
	// NodeKey 对应设计态节点 key
	NodeKey string
	// Status 任务状态机（迁移表见 task/state_machine.go）
	Status TaskStatus
	// TransferredFromTaskID 转办来源任务（0=非转办产生；历史链可追溯）
	TransferredFromTaskID uint
	// TransferredToMemberID 转办目标成员（TRANSFERRED 任务记录去向）
	TransferredToMemberID uint
}

// TaskStatus 任务状态机（第 9.3 章，Phase 0 冻结）。
type TaskStatus string

const (
	// TaskStatusPENDING 待办（唯一可执行动作的状态）
	TaskStatusPENDING TaskStatus = "PENDING"
	// TaskStatusAPPROVED 已同意（终态）
	TaskStatusAPPROVED TaskStatus = "APPROVED"
	// TaskStatusREJECTED 已驳回（终态）
	TaskStatusREJECTED TaskStatus = "REJECTED"
	// TaskStatusTRANSFERRED 原任务被转办后关闭（终态），新任务另建
	TaskStatusTRANSFERRED TaskStatus = "TRANSFERRED"
	// TaskStatusCANCELLED 已取消（实例终止/撤回/或签淘汰/退回重开，终态）
	TaskStatusCANCELLED TaskStatus = "CANCELLED"
	// TaskStatusEXPIRED 超时过期（终态，wf_job task.timeout 触发）
	TaskStatusEXPIRED TaskStatus = "EXPIRED"
)

// TaskAction 人工任务动作（第 10 章；超时自动动作经 Task Engine
// 同一执行路径触发，不绕过状态机，第 19.4 章）。
type TaskAction string

const (
	TaskActionApprove         TaskAction = "approve"
	TaskActionReject          TaskAction = "reject"
	TaskActionReturnToStarter TaskAction = "return-to-starter"
	TaskActionTransfer        TaskAction = "transfer"
	TaskActionWithdraw        TaskAction = "withdraw"  // 实例级动作，作用于 Instance
	TaskActionTerminate       TaskAction = "terminate" // 管理员终止，独立权限
)
