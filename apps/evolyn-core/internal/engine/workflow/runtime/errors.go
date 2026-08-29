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
)
