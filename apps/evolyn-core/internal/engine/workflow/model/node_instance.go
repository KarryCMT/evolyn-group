package model

// NodeInstance 某个设计态 Node 在一次 Instance 中的一次实际运行。
// 必须坚持 Node ≠ NodeInstance ≠ Task（第 6.2 章）：三人会签是
// 1 NodeInstance → 3 Task。
type NodeInstance struct {
	ID uint
	// TenantID 归属租户
	TenantID uint
	// InstanceID / ExecutionID 归属实例与执行路径
	InstanceID  uint
	ExecutionID uint
	// NodeKey 对应设计态节点 key（配置运行时按 key 从发布快照读取）
	NodeKey string
	// Status 节点实例状态机（迁移表见 task/state_machine.go）
	Status NodeInstanceStatus
}

// NodeInstanceStatus 节点实例状态机（第 9.2 章，Phase 0 冻结）。
type NodeInstanceStatus string

const (
	// NodeInstanceStatusPENDING 已创建未激活执行
	NodeInstanceStatusPENDING NodeInstanceStatus = "PENDING"
	// NodeInstanceStatusRUNNING 自动执行中（start/condition/end 等瞬时节点）
	NodeInstanceStatusRUNNING NodeInstanceStatus = "RUNNING"
	// NodeInstanceStatusWAITING 等待人工任务（审批/抄送落库后挂起）
	NodeInstanceStatusWAITING NodeInstanceStatus = "WAITING"
	// NodeInstanceStatusWAITING_RESUBMIT 退回发起人后的修改等待态，
	// 发起人重新提交后回到 RUNNING 继续流程（第 10.3 章）
	NodeInstanceStatusWAITING_RESUBMIT NodeInstanceStatus = "WAITING_RESUBMIT"
	// NodeInstanceStatusCOMPLETED 节点完成（终态）
	NodeInstanceStatusCOMPLETED NodeInstanceStatus = "COMPLETED"
	// NodeInstanceStatusREJECTED 节点驳回（终态）
	NodeInstanceStatusREJECTED NodeInstanceStatus = "REJECTED"
	// NodeInstanceStatusCANCELLED 节点取消（实例终止/撤回/或签淘汰，终态）
	NodeInstanceStatusCANCELLED NodeInstanceStatus = "CANCELLED"
)
