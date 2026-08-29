// Package event Workflow 领域事件目录（第 18.2 章，Phase 0 冻结事件名）。
//
// V1 不一定全部对外消费，但事件名称先行冻结：接入既有 Outbox
// （notification_outbox_events / 既有 Dispatcher），不新建 domain_outbox；
// 消费方按事件名注册处理器，新增事件显式追加常量，禁止字符串散落。
package event

const (
	// 实例生命周期
	InstanceStarted   = "workflow.instance.started"
	InstanceCompleted = "workflow.instance.completed"
	InstanceRejected  = "workflow.instance.rejected"
	InstanceCancelled = "workflow.instance.cancelled"

	// 人工任务
	TaskCreated     = "workflow.task.created"
	TaskApproved    = "workflow.task.approved"
	TaskRejected    = "workflow.task.rejected"
	TaskTransferred = "workflow.task.transferred"
	TaskCancelled   = "workflow.task.cancelled"

	// 节点流转
	NodeEntered   = "workflow.node.entered"
	NodeCompleted = "workflow.node.completed"
)

// All 全部冻结事件名（平台层事件目录注册与 Dispatcher 消费校验使用）。
var All = []string{
	InstanceStarted,
	InstanceCompleted,
	InstanceRejected,
	InstanceCancelled,
	TaskCreated,
	TaskApproved,
	TaskRejected,
	TaskTransferred,
	TaskCancelled,
	NodeEntered,
	NodeCompleted,
}
