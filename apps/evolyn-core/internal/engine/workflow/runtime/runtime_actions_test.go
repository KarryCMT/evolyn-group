// Phase 4 引擎测试：驳回 terminate 联动、退回发起人 + 重提交、撤回窗口、
// 管理员终止、转办、抄送落库。
package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"evolyn/internal/engine/workflow/model"
	"evolyn/internal/engine/workflow/task"
)

// fakeCCRecords CCRepository 内存桩。
type fakeCCRecords struct {
	records []model.CCRecord
}

func (f *fakeCCRecords) CreateCCRecords(ctx context.Context, records []model.CCRecord) error {
	f.records = append(f.records, records...)
	return nil
}

// ccDoc start → cc(成员 9) → boss(user 2) → end 的含抄送快照。
func ccDoc() model.Document {
	return model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			{Key: "notice", Type: model.NodeTypeCC, Name: "抄送", Config: model.NodeConfig{
				Recipients: &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{9}},
			}},
			{Key: "boss", Type: model.NodeTypeApproval, Name: "主管审批", Config: model.NodeConfig{
				ApprovalMode: model.ApprovalModeSingle,
				Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{2}},
			}},
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "notice"},
			{Key: "e2", Source: "notice", Target: "boss"},
			{Key: "e3", Source: "boss", Target: "end"},
		},
	}
}

// newActionHarness 带抄送仓储的动作测试骨架。
func newActionHarness(t *testing.T, doc model.Document) (*harness, *fakeCCRecords) {
	t.Helper()
	cc := &fakeCCRecords{}
	h := newFormHarness(t, doc, nil, &fakeIdentity{})
	h.runtime.ccRecords = cc
	return h, cc
}

func pendingTaskID(t *testing.T, h *harness) uint {
	t.Helper()
	ids := h.pendingTasks(t)
	require.Len(t, ids, 1)
	return ids[0]
}

func TestRejectTerminatesInstance(t *testing.T) {
	h, _ := newActionHarness(t, approvalDoc())
	start := h.start(t)
	taskID := pendingTaskID(t, h)

	// 非参与人驳回 → 拒绝
	_, err := h.runtime.Reject(context.Background(), RejectInput{
		TenantID: 1, TaskID: taskID, OperatorMemberID: 999,
	})
	assert.ErrorIs(t, err, task.ErrTaskForbidden)

	res, err := h.runtime.Reject(context.Background(), RejectInput{
		TenantID: 1, TaskID: taskID, OperatorMemberID: 2, Comment: "不符合制度",
	})
	require.NoError(t, err)
	assert.Equal(t, model.InstanceStatusREJECTED, res.InstanceStatus)
	assert.Equal(t, model.InstanceStatusREJECTED, h.instances.byID[start.InstanceID].Status)
	// 任务已 REJECTED，无遗留 PENDING
	assert.Empty(t, h.pendingTasks(t))
	assert.Equal(t, model.TaskStatusREJECTED, h.tasks.byID[taskID].Status)
	// 双击防护：同一任务二次驳回 → TASK_NOT_PENDING
	_, err = h.runtime.Reject(context.Background(), RejectInput{
		TenantID: 1, TaskID: taskID, OperatorMemberID: 2,
	})
	assert.ErrorIs(t, err, task.ErrTaskNotPending)
	// 终态实例不可再操作
	_, err = h.runtime.Withdraw(context.Background(), InstanceActionInput{
		TenantID: 1, InstanceID: start.InstanceID, OperatorMemberID: 1,
	})
	assert.ErrorIs(t, err, task.ErrInstanceNotRunning)
}

func TestReturnAndResubmit(t *testing.T) {
	h, _ := newActionHarness(t, approvalDoc())
	start := h.start(t)
	taskID := pendingTaskID(t, h)

	_, err := h.runtime.ReturnToStarter(context.Background(), ReturnInput{
		TenantID: 1, TaskID: taskID, OperatorMemberID: 2, Comment: "请补充明细",
	})
	require.NoError(t, err)
	// 实例保持 RUNNING；原任务取消；无审批待办
	assert.Equal(t, model.InstanceStatusRUNNING, h.instances.byID[start.InstanceID].Status)
	assert.Equal(t, model.TaskStatusCANCELLED, h.tasks.byID[taskID].Status)
	assert.Empty(t, h.pendingTasks(t))

	// 非发起人重提交 → 拒绝
	_, err = h.runtime.Resubmit(context.Background(), ResubmitInput{
		TenantID: 1, InstanceID: start.InstanceID, OperatorMemberID: 2,
	})
	assert.ErrorIs(t, err, ErrNotStarter)

	// 发起人重提交 → 从退回节点重新执行，主管重新获得待办
	res, err := h.runtime.Resubmit(context.Background(), ResubmitInput{
		TenantID: 1, InstanceID: start.InstanceID, OperatorMemberID: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, model.InstanceStatusRUNNING, res.InstanceStatus)
	newTaskID := pendingTaskID(t, h)
	assert.Equal(t, "boss", h.tasks.byID[newTaskID].NodeKey)
	assert.NotEqual(t, taskID, newTaskID)
	// 退回节点的重提交节点实例已 COMPLETED
	nodes := h.nodes.byID
	resubmitSeen := false
	for _, node := range nodes {
		if node.Status == model.NodeInstanceStatusWAITING_RESUBMIT {
			resubmitSeen = true
		}
	}
	assert.False(t, resubmitSeen, "重提交后不应残留 WAITING_RESUBMIT")

	// 未退回状态下重提交 → 稳定错误
	_, err = h.runtime.Resubmit(context.Background(), ResubmitInput{
		TenantID: 1, InstanceID: start.InstanceID, OperatorMemberID: 1,
	})
	assert.ErrorIs(t, err, ErrResubmitNodeMissing)
}

// twoApprovalDoc start → boss(user 2) → manager(user 3) → end 的两连审批。
func twoApprovalDoc() model.Document {
	return model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			{Key: "boss", Type: model.NodeTypeApproval, Name: "主管审批", Config: model.NodeConfig{
				ApprovalMode: model.ApprovalModeSingle,
				Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{2}},
			}},
			{Key: "manager", Type: model.NodeTypeApproval, Name: "经理审批", Config: model.NodeConfig{
				ApprovalMode: model.ApprovalModeSingle,
				Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{3}},
			}},
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "boss"},
			{Key: "e2", Source: "boss", Target: "manager"},
			{Key: "e3", Source: "manager", Target: "end"},
		},
	}
}

func TestWithdrawWindow(t *testing.T) {
	h, _ := newActionHarness(t, twoApprovalDoc())
	start := h.start(t)

	// 撤回窗口内（无已完成审批任务）→ 成功
	res, err := h.runtime.Withdraw(context.Background(), InstanceActionInput{
		TenantID: 1, InstanceID: start.InstanceID, OperatorMemberID: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, model.InstanceStatusCANCELLED, res.InstanceStatus)
	assert.Equal(t, model.InstanceStatusCANCELLED, h.instances.byID[start.InstanceID].Status)
	assert.Empty(t, h.pendingTasks(t))

	// 窗口外场景：主管先同意 → 发起人撤回被拒（实例尚在第二审批节点）
	start2, err := h.runtime.Start(context.Background(), StartInput{
		TenantID: 1, Code: "wf_test", BusinessType: "expense", BusinessID: "biz-2", StarterMemberID: 1,
	})
	require.NoError(t, err)
	bossTask := pendingTaskID(t, h)
	_, err = h.runtime.Approve(context.Background(), ApproveInput{
		TenantID: 1, TaskID: bossTask, OperatorMemberID: 2,
	})
	require.NoError(t, err)
	_, err = h.runtime.Withdraw(context.Background(), InstanceActionInput{
		TenantID: 1, InstanceID: start2.InstanceID, OperatorMemberID: 1,
	})
	assert.ErrorIs(t, err, ErrActionNotAllowed)

	// 管理员终止不受窗口限制
	term, err := h.runtime.Terminate(context.Background(), InstanceActionInput{
		TenantID: 1, InstanceID: start2.InstanceID, OperatorMemberID: 99,
	})
	require.NoError(t, err)
	assert.Equal(t, model.InstanceStatusCANCELLED, term.InstanceStatus)
}

func TestTransferTask(t *testing.T) {
	h, _ := newActionHarness(t, approvalDoc())
	h.start(t)
	taskID := pendingTaskID(t, h)

	// 非参与人转办 → 拒绝
	_, err := h.runtime.Transfer(context.Background(), TransferInput{
		TenantID: 1, TaskID: taskID, OperatorMemberID: 999, TargetMemberID: 3,
	})
	assert.ErrorIs(t, err, task.ErrTaskForbidden)

	out, err := h.runtime.Transfer(context.Background(), TransferInput{
		TenantID: 1, TaskID: taskID, OperatorMemberID: 2, TargetMemberID: 3, Comment: "出差代审",
	})
	require.NoError(t, err)
	// 原任务 TRANSFERRED + 新任务 PENDING 且参与人为目标成员
	assert.Equal(t, model.TaskStatusTRANSFERRED, h.tasks.byID[taskID].Status)
	assert.Equal(t, uint(3), h.tasks.byID[taskID].TransferredToMemberID)
	assert.Equal(t, model.TaskStatusPENDING, out.NewTask.Status)
	assert.Equal(t, taskID, out.NewTask.TransferredFromTaskID)
	actors := h.tasks.actors[out.NewTask.ID]
	require.Len(t, actors, 1)
	assert.Equal(t, uint(3), actors[0].MemberID)
	// 实例保持 RUNNING（转办不推进节点）
	assert.Equal(t, model.InstanceStatusRUNNING, out.Instance.Status)
}

func TestCCNodePersistsRecords(t *testing.T) {
	h, cc := newActionHarness(t, ccDoc())
	h.start(t)
	// 推进环越过 CC 节点（瞬时完成），挂起在 boss 审批
	assert.Len(t, h.pendingTasks(t), 1)
	require.Len(t, cc.records, 1)
	assert.Equal(t, uint(9), cc.records[0].MemberID)
	assert.Equal(t, "notice", cc.records[0].NodeKey)
}
