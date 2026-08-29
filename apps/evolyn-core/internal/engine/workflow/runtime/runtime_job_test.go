// Phase 5 引擎测试：任务创建排期 Job、终态联动取消、超时自动动作走
// Task Engine 正常路径（AutoTimeout）、重试决策与 DSL 校验。
package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"evolyn/internal/engine/workflow/assignment"
	"evolyn/internal/engine/workflow/executor"
	"evolyn/internal/engine/workflow/model"
)

// timeoutDoc start → boss(user 2, 超时 approve 1s + 提醒 1s) → end。
func timeoutDoc() model.Document {
	return model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			{Key: "boss", Type: model.NodeTypeApproval, Name: "主管审批", Config: model.NodeConfig{
				ApprovalMode: model.ApprovalModeSingle,
				Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{2}},
				Timeout:      &model.TimeoutConfig{Seconds: 1, Action: model.TimeoutActionApprove},
				Reminder:     &model.ReminderConfig{Seconds: 1},
			}},
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "boss"},
			{Key: "e2", Source: "boss", Target: "end"},
		},
	}
}

func newJobHarness(t *testing.T, doc model.Document) (*harness, *fakeJobs, *fakeOperations) {
	t.Helper()
	definitions := newFakeDefinitions()
	instances := newFakeInstances()
	executions := newFakeExecutions()
	nodes := newFakeNodes()
	tasks := newFakeTasks()
	operations := &fakeOperations{}
	publisher := &fakePublisher{}
	jobs := newFakeJobs()

	definitions.publishDefinition("wf_test", doc)
	registry := executor.NewRegistry(assignment.NewRegistry(nil, nil), nil)
	rt := NewRuntime(definitions, instances, executions, nodes, tasks, operations, registry, publisher, nil, nil, nil, jobs)
	return &harness{runtime: rt, definitions: definitions, instances: instances, tasks: tasks, nodes: nodes, publisher: publisher}, jobs, operations
}

// TestScheduleJobsOnTaskCreation 任务创建时按节点配置排期 timeout/reminder。
func TestScheduleJobsOnTaskCreation(t *testing.T) {
	h, jobs, _ := newJobHarness(t, timeoutDoc())
	h.start(t)
	require.Len(t, h.tasks.byID, 1)
	require.Len(t, jobs.byID, 2)

	types := map[model.JobType]bool{}
	for _, job := range jobs.byID {
		types[job.Type] = true
		assert.Equal(t, h.tasks.byID[taskIDOf(h, 1)].ID, job.TaskID)
		assert.Equal(t, model.JobStatusPENDING, job.Status)
		assert.Equal(t, 3, job.MaxRetryCount)
	}
	assert.True(t, types[model.JobTypeTaskTimeout], "应有 task.timeout Job")
	assert.True(t, types[model.JobTypeTaskReminder], "应有 task.reminder Job")

	// 审批同意（任务终态）→ 排期 Job 联动取消
	taskID := taskIDOf(h, 1)
	_, err := h.runtime.Approve(context.Background(), ApproveInput{
		TenantID: 1, TaskID: taskID, OperatorMemberID: 2,
	})
	require.NoError(t, err)
	for _, job := range jobs.byID {
		assert.Equal(t, model.JobStatusCANCELLED, job.Status, "任务完成后 Job 应联动取消")
	}
}

// TestTimeoutAutoApproveViaTaskEngine 超时自动同意：必须走 Task Engine 正常
// 路径——任务 APPROVED、TIMEOUT 流水（操作人 0=系统）、节点/实例推进、事件。
func TestTimeoutAutoApproveViaTaskEngine(t *testing.T) {
	h, _, operations := newJobHarness(t, timeoutDoc())
	start := h.start(t)
	taskID := taskIDOf(h, 1)

	// 自动动作以操作人 0 触发（Actor 语义替代）：非参与人也无需校验
	res, err := h.runtime.Approve(context.Background(), ApproveInput{
		TenantID: 1, TaskID: taskID, OperatorMemberID: 0, AutoTimeout: true,
		Comment: "超时自动处理",
	})
	require.NoError(t, err)
	assert.Equal(t, model.InstanceStatusCOMPLETED, res.InstanceStatus)
	assert.Equal(t, model.InstanceStatusCOMPLETED, h.instances.byID[start.InstanceID].Status)
	assert.Equal(t, model.TaskStatusAPPROVED, h.tasks.byID[taskID].Status)

	// TIMEOUT 操作流水（操作人 0）
	var timeoutOp *model.Operation
	for _, op := range operations.ops {
		if op.Type == model.OperationTypeTimeout {
			op := op
			timeoutOp = &op
		}
	}
	require.NotNil(t, timeoutOp, "应有 TIMEOUT 操作流水")
	assert.Equal(t, uint(0), timeoutOp.OperatorMemberID)
	assert.Equal(t, bool(true), timeoutOp.Payload["auto"])

	// 人工动作路径不受影响：无 AutoTimeout 时操作人 0 仍被参与人校验拒绝
	h2, _, _ := newJobHarness(t, approvalDoc())
	h2.start(t)
	_, err = h2.runtime.Approve(context.Background(), ApproveInput{
		TenantID: 1, TaskID: taskIDOf(h2, 1), OperatorMemberID: 0,
	})
	assert.Error(t, err, "非自动路径的系统操作人必须被参与人校验拒绝")
}

// TestTimeoutAutoReject 超时自动驳回：terminate 联动实例 REJECTED。
func TestTimeoutAutoReject(t *testing.T) {
	doc := timeoutDoc()
	doc.Nodes[1].Config.Timeout.Action = model.TimeoutActionReject
	h, jobs, _ := newJobHarness(t, doc)
	start := h.start(t)
	taskID := taskIDOf(h, 1)

	res, err := h.runtime.Reject(context.Background(), RejectInput{
		TenantID: 1, TaskID: taskID, OperatorMemberID: 0, AutoTimeout: true,
	})
	require.NoError(t, err)
	assert.Equal(t, model.InstanceStatusREJECTED, res.InstanceStatus)
	assert.Equal(t, model.InstanceStatusREJECTED, h.instances.byID[start.InstanceID].Status)
	assert.Equal(t, model.TaskStatusREJECTED, h.tasks.byID[taskID].Status)
	for _, job := range jobs.byID {
		assert.Equal(t, model.JobStatusCANCELLED, job.Status)
	}
}

// TestJobCancelOnInstanceTerminate 实例终态（撤回）联动取消全部排期 Job。
func TestJobCancelOnInstanceTerminate(t *testing.T) {
	h, jobs, _ := newJobHarness(t, timeoutDoc())
	start := h.start(t)
	_, err := h.runtime.Withdraw(context.Background(), InstanceActionInput{
		TenantID: 1, InstanceID: start.InstanceID, OperatorMemberID: 1,
	})
	require.NoError(t, err)
	for _, job := range jobs.byID {
		assert.Equal(t, model.JobStatusCANCELLED, job.Status)
	}
}
