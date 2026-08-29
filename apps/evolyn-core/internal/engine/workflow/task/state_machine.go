// Package task 人工任务引擎与状态机迁移表。
//
// 状态语义在 Phase 0 冻结（设计文档第 9/10/11/19 章），Runtime/Task Engine
// 代码不得自行解释：任何状态变更必须先经本包迁移表校验，未登记的迁移
// 一律非法（DoD：状态机无未定义转换）。迁移表是状态合法性的唯一事实源，
// 状态枚举定义在 model 包，此处只登记迁移关系。
package task

import (
	"evolyn/internal/engine/workflow/model"
)

// instanceTransitions 实例状态迁移表（第 9.1 章）：
//
//	DRAFT → RUNNING / CANCELLED
//	RUNNING → COMPLETED / REJECTED / CANCELLED
//	COMPLETED / REJECTED / CANCELLED → （终态，无出边）
var instanceTransitions = map[model.InstanceStatus][]model.InstanceStatus{
	model.InstanceStatusDRAFT:   {model.InstanceStatusRUNNING, model.InstanceStatusCANCELLED},
	model.InstanceStatusRUNNING: {model.InstanceStatusCOMPLETED, model.InstanceStatusREJECTED, model.InstanceStatusCANCELLED},
}

// nodeInstanceTransitions 节点实例状态迁移表（第 9.2 / 10.3 章）：
//
//	PENDING → RUNNING / CANCELLED
//	RUNNING → WAITING / COMPLETED / REJECTED / CANCELLED
//	WAITING → COMPLETED / REJECTED / CANCELLED
//	WAITING_RESUBMIT → RUNNING（发起人重新提交）/ CANCELLED
//	COMPLETED / REJECTED / CANCELLED → （终态，无出边）
//
// 退回发起人不等价于 REJECTED：当前节点以 operation 表达 RETURNED 后关闭，
// 新建（或激活）发起人修改 NodeInstance 处于 WAITING_RESUBMIT（第 10.3 章）。
var nodeInstanceTransitions = map[model.NodeInstanceStatus][]model.NodeInstanceStatus{
	model.NodeInstanceStatusPENDING:          {model.NodeInstanceStatusRUNNING, model.NodeInstanceStatusCANCELLED},
	model.NodeInstanceStatusRUNNING:          {model.NodeInstanceStatusWAITING, model.NodeInstanceStatusCOMPLETED, model.NodeInstanceStatusREJECTED, model.NodeInstanceStatusCANCELLED},
	model.NodeInstanceStatusWAITING:          {model.NodeInstanceStatusCOMPLETED, model.NodeInstanceStatusREJECTED, model.NodeInstanceStatusCANCELLED},
	model.NodeInstanceStatusWAITING_RESUBMIT: {model.NodeInstanceStatusRUNNING, model.NodeInstanceStatusCANCELLED},
}

// taskTransitions 任务状态迁移表（第 9.3 / 10 章）：
//
//	PENDING → APPROVED / REJECTED / TRANSFERRED / CANCELLED / EXPIRED
//	其余均为终态（TRANSFERRED 的后续审批在新 Task 上发生，不回写原任务）。
var taskTransitions = map[model.TaskStatus][]model.TaskStatus{
	model.TaskStatusPENDING: {
		model.TaskStatusAPPROVED,
		model.TaskStatusREJECTED,
		model.TaskStatusTRANSFERRED,
		model.TaskStatusCANCELLED,
		model.TaskStatusEXPIRED,
	},
}

// executionTransitions 执行路径迁移表：
//
//	RUNNING → COMPLETED / CANCELLED
//	COMPLETED / CANCELLED → （终态）
var executionTransitions = map[model.ExecutionStatus][]model.ExecutionStatus{
	model.ExecutionStatusRUNNING: {model.ExecutionStatusCOMPLETED, model.ExecutionStatusCANCELLED},
}

// jobTransitions Job 状态迁移表（第 19.1 章）：
//
//	PENDING → PROCESSING / CANCELLED
//	PROCESSING → SUCCEEDED / FAILED（超过上限）/ PENDING（未超限重试回队）
//	SUCCEEDED / FAILED / CANCELLED → （终态）
var jobTransitions = map[model.JobStatus][]model.JobStatus{
	model.JobStatusPENDING:    {model.JobStatusPROCESSING, model.JobStatusCANCELLED},
	model.JobStatusPROCESSING: {model.JobStatusSUCCEEDED, model.JobStatusFAILED, model.JobStatusPENDING},
}

// CanTransitionInstance 实例状态 from → to 是否为已登记合法迁移。
// 终态到任何状态（含自身）均非法。
func CanTransitionInstance(from, to model.InstanceStatus) bool {
	return canTransition(instanceTransitions, from, to)
}

// CanTransitionNodeInstance 节点实例状态 from → to 是否合法。
func CanTransitionNodeInstance(from, to model.NodeInstanceStatus) bool {
	return canTransition(nodeInstanceTransitions, from, to)
}

// CanTransitionTask 任务状态 from → to 是否合法。
func CanTransitionTask(from, to model.TaskStatus) bool {
	return canTransition(taskTransitions, from, to)
}

// CanTransitionExecution 执行路径状态 from → to 是否合法。
func CanTransitionExecution(from, to model.ExecutionStatus) bool {
	return canTransition(executionTransitions, from, to)
}

// CanTransitionJob Job 状态 from → to 是否合法。
func CanTransitionJob(from, to model.JobStatus) bool {
	return canTransition(jobTransitions, from, to)
}

// canTransition 迁移表通用查找：未登记状态一律视为非法迁移，
// 保证「状态机无未定义转换」。
func canTransition[T comparable](table map[T][]T, from, to T) bool {
	for _, allowed := range table[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// WithdrawAllowed 撤回资格判定（第 10.4 章冻结规则）：只有流程尚未存在
// 任何「已完成的人工审批 Task」时，发起人才允许撤回——即所有任务均处于
// PENDING（或已取消）状态。管理员 terminate 走独立权限，不经本判定。
func WithdrawAllowed(tasks []model.Task) bool {
	for i := range tasks {
		switch tasks[i].Status {
		case model.TaskStatusAPPROVED, model.TaskStatusREJECTED, model.TaskStatusTRANSFERRED, model.TaskStatusEXPIRED:
			return false
		}
	}
	return true
}

// RequiredApprovals 会签通过阈值（第 11.3 章冻结公式）：
// requiredApprovals = ceil(totalActors * passRatio)，如 4×0.5=2、5×0.5=3。
// totalActors 为本节点解析出的参与人总数（快照后不变）。
func RequiredApprovals(totalActors int, passRatio float64) int {
	if totalActors <= 0 {
		return 0
	}
	if passRatio <= 0 {
		passRatio = 1 // 未配置比例按全票通过兜底，避免除零语义漂移
	}
	required := int(float64(totalActors) * passRatio)
	if float64(required) < float64(totalActors)*passRatio {
		required++ // ceil
	}
	if required > totalActors {
		required = totalActors
	}
	return required
}
