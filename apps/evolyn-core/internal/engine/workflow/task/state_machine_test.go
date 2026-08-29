package task

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"evolyn/internal/engine/workflow/model"
)

// TestInstanceTransitions 状态机 DoD：迁移表无未定义转换——
// 枚举全量遍历，仅迁移表登记的迁移合法，其余（含终态自迁移）非法。
func TestInstanceTransitions(t *testing.T) {
	allowed := map[model.InstanceStatus][]model.InstanceStatus{
		model.InstanceStatusDRAFT:   {model.InstanceStatusRUNNING, model.InstanceStatusCANCELLED},
		model.InstanceStatusRUNNING: {model.InstanceStatusCOMPLETED, model.InstanceStatusREJECTED, model.InstanceStatusCANCELLED},
	}
	all := []model.InstanceStatus{
		model.InstanceStatusDRAFT, model.InstanceStatusRUNNING,
		model.InstanceStatusCOMPLETED, model.InstanceStatusREJECTED, model.InstanceStatusCANCELLED,
	}
	for _, from := range all {
		for _, to := range all {
			want := false
			for _, a := range allowed[from] {
				if a == to {
					want = true
				}
			}
			assert.Equal(t, want, CanTransitionInstance(from, to), "%s -> %s", from, to)
		}
	}
}

func TestNodeInstanceTransitions(t *testing.T) {
	// 合法主链：PENDING → RUNNING → WAITING → COMPLETED
	assert.True(t, CanTransitionNodeInstance(model.NodeInstanceStatusPENDING, model.NodeInstanceStatusRUNNING))
	assert.True(t, CanTransitionNodeInstance(model.NodeInstanceStatusRUNNING, model.NodeInstanceStatusWAITING))
	assert.True(t, CanTransitionNodeInstance(model.NodeInstanceStatusWAITING, model.NodeInstanceStatusCOMPLETED))

	// 退回发起人：WAITING_RESUBMIT 仅可回到 RUNNING 或取消
	assert.True(t, CanTransitionNodeInstance(model.NodeInstanceStatusWAITING_RESUBMIT, model.NodeInstanceStatusRUNNING))
	assert.False(t, CanTransitionNodeInstance(model.NodeInstanceStatusWAITING_RESUBMIT, model.NodeInstanceStatusCOMPLETED))

	// 终态无出边
	for _, terminal := range []model.NodeInstanceStatus{
		model.NodeInstanceStatusCOMPLETED, model.NodeInstanceStatusREJECTED, model.NodeInstanceStatusCANCELLED,
	} {
		assert.False(t, CanTransitionNodeInstance(terminal, model.NodeInstanceStatusRUNNING))
		assert.False(t, CanTransitionNodeInstance(terminal, terminal))
	}
}

func TestTaskTransitions(t *testing.T) {
	// PENDING 是唯一可执行动作状态
	for _, to := range []model.TaskStatus{
		model.TaskStatusAPPROVED, model.TaskStatusREJECTED, model.TaskStatusTRANSFERRED,
		model.TaskStatusCANCELLED, model.TaskStatusEXPIRED,
	} {
		assert.True(t, CanTransitionTask(model.TaskStatusPENDING, to))
	}
	// 终态无出边（TRANSFERRED 后续在新任务上发生）
	for _, terminal := range []model.TaskStatus{
		model.TaskStatusAPPROVED, model.TaskStatusREJECTED, model.TaskStatusTRANSFERRED,
		model.TaskStatusCANCELLED, model.TaskStatusEXPIRED,
	} {
		assert.False(t, CanTransitionTask(terminal, model.TaskStatusPENDING))
		assert.False(t, CanTransitionTask(terminal, terminal))
	}
}

func TestJobTransitions(t *testing.T) {
	assert.True(t, CanTransitionJob(model.JobStatusPENDING, model.JobStatusPROCESSING))
	assert.True(t, CanTransitionJob(model.JobStatusPROCESSING, model.JobStatusPENDING)) // 重试回队
	assert.True(t, CanTransitionJob(model.JobStatusPROCESSING, model.JobStatusFAILED))
	assert.False(t, CanTransitionJob(model.JobStatusFAILED, model.JobStatusPENDING)) // 超限终态
	assert.False(t, CanTransitionJob(model.JobStatusSUCCEEDED, model.JobStatusPENDING))
}

func TestWithdrawAllowed(t *testing.T) {
	pending := model.Task{Status: model.TaskStatusPENDING}
	cancelled := model.Task{Status: model.TaskStatusCANCELLED}
	// 无任何已完成人工审批任务 → 允许撤回（第 10.4 章）
	assert.True(t, WithdrawAllowed([]model.Task{pending, pending, cancelled}))

	// 任一任务已通过/驳回/转办/过期 → 禁止撤回
	approved := model.Task{Status: model.TaskStatusAPPROVED}
	assert.False(t, WithdrawAllowed([]model.Task{pending, approved}))
	assert.False(t, WithdrawAllowed([]model.Task{approved}))
	assert.False(t, WithdrawAllowed([]model.Task{{Status: model.TaskStatusTRANSFERRED}}))
	assert.False(t, WithdrawAllowed([]model.Task{{Status: model.TaskStatusEXPIRED}}))

	// 空任务集（尚无待办）允许撤回
	assert.True(t, WithdrawAllowed(nil))
}

func TestRequiredApprovals(t *testing.T) {
	// 第 11.3 章：ceil(total * passRatio)
	assert.Equal(t, 2, RequiredApprovals(4, 0.5))
	assert.Equal(t, 3, RequiredApprovals(5, 0.5))
	assert.Equal(t, 4, RequiredApprovals(4, 0.8))
	assert.Equal(t, 5, RequiredApprovals(5, 1.0))
	assert.Equal(t, 1, RequiredApprovals(1, 0.5))
	assert.Equal(t, 0, RequiredApprovals(0, 0.5))
	// passRatio 未配置兜底全票
	assert.Equal(t, 3, RequiredApprovals(3, 0))
}
