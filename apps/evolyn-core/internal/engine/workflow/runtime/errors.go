// Package runtime 流程运行时：StartWorkflow 与人工任务后的推进编排。
// 内核只做状态机裁决与节点编排，事务由平台层经 TxManager 包裹后调用
// 本包方法（仓储经 ctx 传播加入同一事务）。
package runtime

import "errors"

// 运行时语义错误（sentinel）：内核返回本包错误，平台层经 errors.Is 映射为
// 稳定业务错误码（WORKFLOW_*），错误细节经 Wrap 只入日志。
// 任务域 sentinel（task 包）：ErrTaskNotFound/ErrTaskNotPending/
// ErrTaskForbidden/ErrInstanceNotFound/ErrInstanceNotRunning。
var (
	// ErrDefinitionNotFound 流程定义不存在或无权访问
	ErrDefinitionNotFound = errors.New("workflow definition not found")
	// ErrDefinitionNotPublished 流程尚未发布任何可执行版本
	ErrDefinitionNotPublished = errors.New("workflow definition not published")
	// ErrInstanceAlreadyRunning 同一业务键已存在 RUNNING 实例（业务幂等，第 14.1 章）
	ErrInstanceAlreadyRunning = errors.New("workflow instance already running")
	// ErrNodeUnsupported 遇到无执行器的节点类型（校验漏网快速失败）
	ErrNodeUnsupported = errors.New("node type has no executor")
	// ErrRouteStuck 寻路失败：当前节点无可用出边（校验器已保证，运行期兜底）
	ErrRouteStuck = errors.New("no available outgoing edge")
	// ErrAdvanceOverflow 推进步数超上限（防御快照异常导致的死循环）
	ErrAdvanceOverflow = errors.New("advance step overflow")
	// ErrFormFieldForbidden 审批编辑携带了节点字段权限未授权的字段，
	// 或实例未绑定表单却携带编辑值（第 15.4 章审批提交协议）
	ErrFormFieldForbidden = errors.New("form field not permitted on this node")
	// ErrActionNotAllowed 当前状态下不允许执行该动作（如撤回窗口已关闭：
	// 实例已存在完成的人工审批任务，第 10.4 章冻结规则）
	ErrActionNotAllowed = errors.New("action not allowed in current state")
	// ErrNotStarter 当前操作者不是实例发起人（撤回/重新提交为发起人专属动作）
	ErrNotStarter = errors.New("operator is not the instance starter")
	// ErrResubmitNodeMissing 实例不存在等待重提交的节点实例
	//（非退回状态下重提交，或状态已推进）
	ErrResubmitNodeMissing = errors.New("no node instance waiting for resubmit")
)
