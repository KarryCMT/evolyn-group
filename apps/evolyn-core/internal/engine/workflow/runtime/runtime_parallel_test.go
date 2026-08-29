// Phase 8 引擎测试：并行执行树（第 31 章 Parallel Execution）——
// split 扇出子执行路径、分支人工审批互不阻塞、join token 到达判定、
// 分支驳回整体终止、含并行定义拒绝退回发起人、服务节点分支内续跑。
package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"evolyn/internal/engine/workflow/event"
	"evolyn/internal/engine/workflow/model"
)

// parallelApprovalDoc start → split → {bossA(user 2), bossB(user 3)} →
// join → end 的双分支并行快照。
func parallelApprovalDoc() model.Document {
	return model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			{Key: "split", Type: model.NodeTypeParallel, Name: "并行分流",
				Config: model.NodeConfig{Parallel: &model.ParallelConfig{Role: model.ParallelRoleSplit}}},
			{Key: "bossA", Type: model.NodeTypeApproval, Name: "审批A", Config: model.NodeConfig{
				ApprovalMode: model.ApprovalModeSingle,
				Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{2}},
			}},
			{Key: "bossB", Type: model.NodeTypeApproval, Name: "审批B", Config: model.NodeConfig{
				ApprovalMode: model.ApprovalModeSingle,
				Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{3}},
			}},
			{Key: "join", Type: model.NodeTypeParallel, Name: "并行汇聚",
				Config: model.NodeConfig{Parallel: &model.ParallelConfig{Role: model.ParallelRoleJoin}}},
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "split"},
			{Key: "p1", Source: "split", Target: "bossA"},
			{Key: "p2", Source: "split", Target: "bossB"},
			{Key: "j1", Source: "bossA", Target: "join"},
			{Key: "j2", Source: "bossB", Target: "join"},
			{Key: "e2", Source: "join", Target: "end"},
		},
	}
}

// pendingTaskByKey 取指定节点 key 的 PENDING 任务（并行分支按节点定位）。
func pendingTaskByKey(h *harness, nodeKey string) *model.Task {
	for i := range h.tasks.byID {
		tk := h.tasks.byID[i]
		if tk.Status == model.TaskStatusPENDING && tk.NodeKey == nodeKey {
			return tk
		}
	}
	return nil
}

func countExecutionsByStatus(h *harness, status model.ExecutionStatus) int {
	count := 0
	for _, executions := range h.executions.byInstance {
		for i := range executions {
			if executions[i].Status == status {
				count++
			}
		}
	}
	return count
}

func TestParallelStartForksBranches(t *testing.T) {
	h := newHarness(t, parallelApprovalDoc())

	result := h.start(t)
	require.Equal(t, model.InstanceStatusRUNNING, result.Status)

	// 执行树：根路径 + 两条子分支路径（ParentExecutionID 挂根）
	executions, err := h.runtime.executions.ListExecutionsByInstance(context.Background(), result.InstanceID)
	require.NoError(t, err)
	require.Len(t, executions, 3)
	var root model.Execution
	branches := 0
	for i := range executions {
		if executions[i].ParentExecutionID == 0 {
			root = executions[i]
		} else {
			branches++
			assert.Equal(t, model.ExecutionStatusRUNNING, executions[i].Status)
		}
	}
	assert.Equal(t, 2, branches)
	// split 节点瞬时完成（挂在根路径）
	assert.Equal(t, model.ExecutionStatusRUNNING, root.Status)

	// 两分支各自挂起人工审批，互不阻塞
	bossA := pendingTaskByKey(h, "bossA")
	bossB := pendingTaskByKey(h, "bossB")
	require.NotNil(t, bossA)
	require.NotNil(t, bossB)
	nodeA, err := h.nodes.FindNodeInstanceByID(context.Background(), 1, bossA.NodeInstanceID)
	require.NoError(t, err)
	assert.Equal(t, model.NodeInstanceStatusWAITING, nodeA.Status)
	nodeB, err := h.nodes.FindNodeInstanceByID(context.Background(), 1, bossB.NodeInstanceID)
	require.NoError(t, err)
	assert.Equal(t, model.NodeInstanceStatusWAITING, nodeB.Status)
	assert.NotEqual(t, nodeA.ExecutionID, nodeB.ExecutionID, "两分支节点实例应分属不同执行路径")

	// split 节点实例已完成
	var splitCompleted bool
	for i := range h.nodes.byID {
		if h.nodes.byID[i].NodeKey == "split" && h.nodes.byID[i].Status == model.NodeInstanceStatusCOMPLETED {
			splitCompleted = true
		}
	}
	assert.True(t, splitCompleted)
}

func TestParallelJoinWaitsForAllBranches(t *testing.T) {
	h := newHarness(t, parallelApprovalDoc())
	result := h.start(t)

	// 分支 A 同意：到达 join 但分支 B 未到，实例保持 RUNNING
	bossA := pendingTaskByKey(h, "bossA")
	require.NotNil(t, bossA)
	res, err := h.runtime.Approve(context.Background(), ApproveInput{
		TenantID: 1, TaskID: bossA.ID, OperatorMemberID: 2})
	require.NoError(t, err)
	assert.True(t, res.NodeCompleted)
	assert.Equal(t, model.InstanceStatusRUNNING, res.InstanceStatus)

	// join 到达 token 已落一个（COMPLETED），分支 A 执行路径收口
	var joinArrivals int
	for i := range h.nodes.byID {
		n := h.nodes.byID[i]
		if n.NodeKey == "join" && n.Status == model.NodeInstanceStatusCOMPLETED {
			joinArrivals++
		}
	}
	assert.Equal(t, 1, joinArrivals)
	assert.Equal(t, 1, countExecutionsByStatus(h, model.ExecutionStatusCOMPLETED), "分支 A 执行路径应完成")
	instance, err := h.instances.FindInstanceByIDForUpdate(context.Background(), 1, result.InstanceID)
	require.NoError(t, err)
	assert.Equal(t, model.InstanceStatusRUNNING, instance.Status)
}

func TestParallelEndToEnd(t *testing.T) {
	h := newHarness(t, parallelApprovalDoc())
	result := h.start(t)

	// 分支 A 同意 → 实例仍 RUNNING
	bossA := pendingTaskByKey(h, "bossA")
	_, err := h.runtime.Approve(context.Background(), ApproveInput{TenantID: 1, TaskID: bossA.ID, OperatorMemberID: 2})
	require.NoError(t, err)

	// 分支 B 同意 → 最后到达触发 join 放行，实例 COMPLETED
	bossB := pendingTaskByKey(h, "bossB")
	require.NotNil(t, bossB, "分支 B 任务不应被分支 A 的推进影响")
	done, err := h.runtime.Approve(context.Background(), ApproveInput{TenantID: 1, TaskID: bossB.ID, OperatorMemberID: 3})
	require.NoError(t, err)
	assert.Equal(t, model.InstanceStatusCOMPLETED, done.InstanceStatus)

	// join 到达 token 恰好两次（每个分支一次），实例仅完成一次
	joinArrivals := 0
	for i := range h.nodes.byID {
		n := h.nodes.byID[i]
		if n.NodeKey == "join" && n.Status == model.NodeInstanceStatusCOMPLETED {
			joinArrivals++
		}
	}
	assert.Equal(t, 2, joinArrivals)
	assert.Equal(t, 1, countStatus(h.publisher.events, event.InstanceCompleted))

	// 执行树收口：根 + 两分支全部 COMPLETED
	executions, err := h.runtime.executions.ListExecutionsByInstance(context.Background(), result.InstanceID)
	require.NoError(t, err)
	require.Len(t, executions, 3)
	for i := range executions {
		assert.Equal(t, model.ExecutionStatusCOMPLETED, executions[i].Status)
	}
}

func TestParallelBranchRejectTerminatesInstance(t *testing.T) {
	h := newHarness(t, parallelApprovalDoc())
	result := h.start(t)

	// 分支 A 驳回 → 整体终止：分支 B 的 PENDING 任务一并取消
	bossA := pendingTaskByKey(h, "bossA")
	require.NotNil(t, bossA)
	res, err := h.runtime.Reject(context.Background(), RejectInput{
		TenantID: 1, TaskID: bossA.ID, OperatorMemberID: 2, Comment: "不同意"})
	require.NoError(t, err)
	assert.Equal(t, model.InstanceStatusREJECTED, res.InstanceStatus)

	bossB := pendingTaskByKey(h, "bossB")
	assert.Nil(t, bossB, "其他分支的待办应随实例终止取消")

	// 全部执行路径（含未到达 join 的分支 B）CANCELLED
	executions, err := h.runtime.executions.ListExecutionsByInstance(context.Background(), result.InstanceID)
	require.NoError(t, err)
	require.Len(t, executions, 3)
	for i := range executions {
		assert.Equal(t, model.ExecutionStatusCANCELLED, executions[i].Status)
	}
	assert.Contains(t, h.publisher.events, event.InstanceRejected)
}

func TestParallelWithdrawCancelsAllBranches(t *testing.T) {
	h := newHarness(t, parallelApprovalDoc())
	result := h.start(t)

	// 撤回窗口未关闭（尚无已完成人工审批任务），发起人可撤回
	res, err := h.runtime.Withdraw(context.Background(), InstanceActionInput{
		TenantID: 1, InstanceID: result.InstanceID, OperatorMemberID: 1})
	require.NoError(t, err)
	assert.Equal(t, model.InstanceStatusCANCELLED, res.InstanceStatus)
	assert.Nil(t, pendingTaskByKey(h, "bossA"))
	assert.Nil(t, pendingTaskByKey(h, "bossB"))
	assert.Equal(t, 3, countExecutionsByStatus(h, model.ExecutionStatusCANCELLED))
}

func TestParallelReturnToStarterRejected(t *testing.T) {
	h := newHarness(t, parallelApprovalDoc())
	h.start(t)

	// 含并行网关的定义冻结不支持退回发起人（重提交会二次扇出分支，
	// join 到达计数失真）
	bossA := pendingTaskByKey(h, "bossA")
	require.NotNil(t, bossA)
	_, err := h.runtime.ReturnToStarter(context.Background(), ReturnInput{
		TenantID: 1, TaskID: bossA.ID, OperatorMemberID: 2})
	assert.ErrorIs(t, err, ErrActionNotAllowed)
}

// sequentialParallelDoc 串行双并行：start → split1{a} join1 → split2{c}
// join2 → end，验证多区域预编译产物与链式并行推进。
func sequentialParallelDoc() model.Document {
	return model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			{Key: "split1", Type: model.NodeTypeParallel, Name: "分流1",
				Config: model.NodeConfig{Parallel: &model.ParallelConfig{Role: model.ParallelRoleSplit}}},
			{Key: "a", Type: model.NodeTypeApproval, Name: "A", Config: model.NodeConfig{
				ApprovalMode: model.ApprovalModeSingle,
				Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{2}},
			}},
			{Key: "b", Type: model.NodeTypeApproval, Name: "B", Config: model.NodeConfig{
				ApprovalMode: model.ApprovalModeSingle,
				Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{3}},
			}},
			{Key: "join1", Type: model.NodeTypeParallel, Name: "汇聚1",
				Config: model.NodeConfig{Parallel: &model.ParallelConfig{Role: model.ParallelRoleJoin}}},
			{Key: "split2", Type: model.NodeTypeParallel, Name: "分流2",
				Config: model.NodeConfig{Parallel: &model.ParallelConfig{Role: model.ParallelRoleSplit}}},
			{Key: "c", Type: model.NodeTypeApproval, Name: "C", Config: model.NodeConfig{
				ApprovalMode: model.ApprovalModeSingle,
				Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{4}},
			}},
			{Key: "d", Type: model.NodeTypeApproval, Name: "D", Config: model.NodeConfig{
				ApprovalMode: model.ApprovalModeSingle,
				Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{5}},
			}},
			{Key: "join2", Type: model.NodeTypeParallel, Name: "汇聚2",
				Config: model.NodeConfig{Parallel: &model.ParallelConfig{Role: model.ParallelRoleJoin}}},
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "split1"},
			{Key: "p1", Source: "split1", Target: "a"},
			{Key: "p2", Source: "split1", Target: "b"},
			{Key: "j1", Source: "a", Target: "join1"},
			{Key: "j2", Source: "b", Target: "join1"},
			{Key: "m1", Source: "join1", Target: "split2"},
			{Key: "p3", Source: "split2", Target: "c"},
			{Key: "p4", Source: "split2", Target: "d"},
			{Key: "j3", Source: "c", Target: "join2"},
			{Key: "j4", Source: "d", Target: "join2"},
			{Key: "e2", Source: "join2", Target: "end"},
		},
	}
}

func TestSequentialParallelRegions(t *testing.T) {
	h := newHarness(t, sequentialParallelDoc())
	result := h.start(t)

	approve := func(nodeKey string, operator uint) {
		tk := pendingTaskByKey(h, nodeKey)
		require.NotNil(t, tk, "任务 %s 应存在", nodeKey)
		_, err := h.runtime.Approve(context.Background(), ApproveInput{
			TenantID: 1, TaskID: tk.ID, OperatorMemberID: operator})
		require.NoError(t, err, "审批 %s 失败", nodeKey)
	}

	approve("a", 2)
	approve("b", 3)
	instance, err := h.instances.FindInstanceByIDForUpdate(context.Background(), 1, result.InstanceID)
	require.NoError(t, err)
	assert.Equal(t, model.InstanceStatusRUNNING, instance.Status, "join1 放行后进入第二并行段")

	// join1 放行触发 split2 扇出：此刻执行路径 = 根 + 2（第一段）+ 2（第二段）
	executions, err := h.runtime.executions.ListExecutionsByInstance(context.Background(), result.InstanceID)
	require.NoError(t, err)
	assert.Len(t, executions, 5)

	approve("c", 4)
	approve("d", 5)
	instance, err = h.instances.FindInstanceByIDForUpdate(context.Background(), 1, result.InstanceID)
	require.NoError(t, err)
	assert.Equal(t, model.InstanceStatusCOMPLETED, instance.Status)

	// 全部执行路径 COMPLETED：根 + 4 分支
	executions, err = h.runtime.executions.ListExecutionsByInstance(context.Background(), result.InstanceID)
	require.NoError(t, err)
	require.Len(t, executions, 5)
	for i := range executions {
		assert.Equal(t, model.ExecutionStatusCOMPLETED, executions[i].Status)
	}
}

// TestParallelServiceNodeInBranch 服务节点位于并行分支：Worker 续跑必须
// 回到分支执行路径（而非根路径），join 到达计数落在正确分支上。
func TestParallelServiceNodeInBranch(t *testing.T) {
	doc := model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			{Key: "split", Type: model.NodeTypeParallel, Name: "分流",
				Config: model.NodeConfig{Parallel: &model.ParallelConfig{Role: model.ParallelRoleSplit}}},
			{Key: "svc", Type: model.NodeTypeService, Name: "调用", Config: model.NodeConfig{
				Service: &model.ServiceConfig{
					Action: model.ServiceActionHTTP,
					URL:    "https://api.example.com/notify",
				},
			}},
			{Key: "boss", Type: model.NodeTypeApproval, Name: "审批", Config: model.NodeConfig{
				ApprovalMode: model.ApprovalModeSingle,
				Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{2}},
			}},
			{Key: "join", Type: model.NodeTypeParallel, Name: "汇聚",
				Config: model.NodeConfig{Parallel: &model.ParallelConfig{Role: model.ParallelRoleJoin}}},
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "split"},
			{Key: "p1", Source: "split", Target: "svc"},
			{Key: "p2", Source: "split", Target: "boss"},
			{Key: "s1", Source: "svc", Target: "join"},
			{Key: "b1", Source: "boss", Target: "join"},
			{Key: "e2", Source: "join", Target: "end"},
		},
	}
	vars := &fakeVariables{}
	invoker := &fakeInvoker{scriptedOK: true, body: `{}`}
	h, jobs, _ := newServiceHarness(t, doc, vars, invoker)

	// Start：svc 异步挂起 + boss 人工挂起
	h.start(t)
	require.Len(t, jobs.byID, 1)
	var invokeJob *model.Job
	for _, job := range jobs.byID {
		invokeJob = job
	}
	require.NotNil(t, invokeJob)

	// Worker 续跑：svc 完成后分支到达 join（1 到达），实例仍等待 boss
	res, err := h.runtime.InvokeServiceNode(context.Background(), ServiceInvokeInput{
		TenantID: 1, InstanceID: invokeJob.InstanceID, NodeInstanceID: invokeJob.NodeInstanceID,
	})
	require.NoError(t, err)
	assert.Equal(t, model.InstanceStatusRUNNING, res.InstanceStatus)
	joinArrivals := 0
	for i := range h.nodes.byID {
		n := h.nodes.byID[i]
		if n.NodeKey == "join" && n.Status == model.NodeInstanceStatusCOMPLETED {
			joinArrivals++
		}
	}
	assert.Equal(t, 1, joinArrivals)

	// boss 同意：第二到达触发 join 放行 → 实例完成
	boss := pendingTaskByKey(h, "boss")
	require.NotNil(t, boss)
	done, err := h.runtime.Approve(context.Background(), ApproveInput{
		TenantID: 1, TaskID: boss.ID, OperatorMemberID: 2})
	require.NoError(t, err)
	assert.Equal(t, model.InstanceStatusCOMPLETED, done.InstanceStatus)
}
