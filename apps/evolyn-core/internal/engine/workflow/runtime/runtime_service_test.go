// Phase 7 引擎测试：service 节点异步挂起（Async + service.invoke 排期）、
// InvokeServiceNode 调用/变量写入/续跑推进、响应映射语义与幂等空跑。
package runtime

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"evolyn/internal/engine/workflow/assignment"
	"evolyn/internal/engine/workflow/executor"
	"evolyn/internal/engine/workflow/model"
	"evolyn/internal/engine/workflow/provider"
)

// errFakeInvoke 调用器替身的统一失败语义（非 2xx/传输失败归一为 error）。
var errFakeInvoke = fmt.Errorf("服务返回非 2xx 状态 500")

// serviceDoc start → svc（HTTP 调用映射变量）→ boss（条件经 variables 读取
// 由条件节点承担，此处仅顺序推进）→ end。
func serviceDoc() model.Document {
	maxRetries := 2
	return model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			{Key: "svc", Type: model.NodeTypeService, Name: "同步订单", Config: model.NodeConfig{
				Service: &model.ServiceConfig{
					Action: model.ServiceActionHTTP,
					Method: "POST",
					URL:    "https://api.example.com/orders/{{starter.member_id}}",
					Body:   `{"operator": {{starter.member_id}}}`,
					ResponseMapping: []model.ServiceResponseMapping{
						{Variable: "orderId", Path: "data.id", Required: true},
					},
					MaxRetries: &maxRetries,
				},
			}},
			{Key: "boss", Type: model.NodeTypeApproval, Name: "主管审批", Config: model.NodeConfig{
				ApprovalMode: model.ApprovalModeSingle,
				Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{2}},
			}},
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "svc"},
			{Key: "e2", Source: "svc", Target: "boss"},
			{Key: "e3", Source: "boss", Target: "end"},
		},
	}
}

// fakeVariables 变量仓储替身：记录 upsert 与实例内变量集。
type fakeVariables struct {
	saved []model.Variable
}

func (f *fakeVariables) SaveVariable(ctx context.Context, variable *model.Variable) error {
	f.saved = append(f.saved, *variable)
	return nil
}

func (f *fakeVariables) ListVariablesByInstance(ctx context.Context, instanceID uint) ([]model.Variable, error) {
	vars := make([]model.Variable, 0, len(f.saved))
	for _, v := range f.saved {
		if v.InstanceID == instanceID {
			vars = append(vars, v)
		}
	}
	return vars, nil
}

// fakeInvoker 调用器替身：记录请求并按脚本回放响应/错误。
type fakeInvoker struct {
	requests   []provider.ServiceRequest
	scriptedOK bool
	body       string
	err        error
}

func (f *fakeInvoker) Invoke(ctx context.Context, req provider.ServiceRequest) (*provider.ServiceResponse, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}
	if !f.scriptedOK {
		return nil, errFakeInvoke
	}
	return &provider.ServiceResponse{StatusCode: 200, Body: []byte(f.body)}, nil
}

func newServiceHarness(t *testing.T, doc model.Document, vars *fakeVariables, invoker *fakeInvoker) (*harness, *fakeJobs, *fakeOperations) {
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
	rt := NewRuntime(definitions, instances, executions, nodes, tasks, operations, registry, publisher,
		nil, nil, nil, jobs, vars, invoker)
	return &harness{runtime: rt, definitions: definitions, instances: instances, tasks: tasks, nodes: nodes, publisher: publisher}, jobs, operations
}

// TestServiceNodeAsyncSuspendAndResume service 节点端到端：Start 在服务节点
// 挂起并排期 service.invoke（业务事务内无外部调用）；Worker 触发调用后
// 变量落库、节点 COMPLETED、续跑至审批节点挂起。
func TestServiceNodeAsyncSuspendAndResume(t *testing.T) {
	vars := &fakeVariables{}
	invoker := &fakeInvoker{scriptedOK: true, body: `{"data":{"id":"order-9"}}`}
	h, jobs, operations := newServiceHarness(t, serviceDoc(), vars, invoker)

	// Start：推进在服务节点挂起，无任务创建，排期一个 service.invoke Job
	h.start(t)
	assert.Empty(t, h.tasks.byID, "业务事务内不应创建任务/发起调用")
	require.Len(t, jobs.byID, 1)
	var invokeJob *model.Job
	for _, job := range jobs.byID {
		invokeJob = job
		assert.Equal(t, model.JobTypeServiceInvoke, job.Type)
		assert.Equal(t, model.JobStatusPENDING, job.Status)
		assert.Equal(t, 2, job.MaxRetryCount, "重试上限应取节点配置")
	}
	assert.Empty(t, invoker.requests, "Start 事务内不得发起 HTTP 调用")

	// Worker 触发：调用发生（模板插值生效）、变量落库、续跑至审批挂起
	res, err := h.runtime.InvokeServiceNode(context.Background(), ServiceInvokeInput{
		TenantID:       1,
		InstanceID:     invokeJob.InstanceID,
		NodeInstanceID: invokeJob.NodeInstanceID,
	})
	require.NoError(t, err)
	assert.True(t, res.Completed)
	require.Len(t, invoker.requests, 1)
	req := invoker.requests[0]
	assert.Equal(t, "https://api.example.com/orders/1", req.URL, "URL 模板应按 starter.* 插值")
	assert.Equal(t, `{"operator": 1}`, req.Body)
	require.Len(t, vars.saved, 1)
	assert.Equal(t, "orderId", vars.saved[0].Key)
	assert.Equal(t, "order-9", vars.saved[0].Value)
	assert.Equal(t, model.VariableTypeString, vars.saved[0].ValueType)

	// 续跑后到达审批节点：任务已创建
	require.Len(t, h.tasks.byID, 1)
	// SERVICE 操作流水（成功语义）
	var svcOp *model.Operation
	for _, op := range operations.ops {
		if op.Type == model.OperationTypeService {
			op := op
			svcOp = &op
		}
	}
	require.NotNil(t, svcOp)
	assert.Equal(t, "SUCCEEDED", svcOp.Payload["status"])
}

// TestServiceNodeFailureNoStateChange 调用失败：整体无状态残留（事务回滚
// 语义），错误上抛由 Worker 重试记账承担。
func TestServiceNodeFailureNoStateChange(t *testing.T) {
	vars := &fakeVariables{}
	invoker := &fakeInvoker{err: errFakeInvoke}
	h, jobs, _ := newServiceHarness(t, serviceDoc(), vars, invoker)
	h.start(t)
	var invokeJob *model.Job
	for _, job := range jobs.byID {
		invokeJob = job
	}
	_, err := h.runtime.InvokeServiceNode(context.Background(), ServiceInvokeInput{
		TenantID: 1, InstanceID: invokeJob.InstanceID, NodeInstanceID: invokeJob.NodeInstanceID,
	})
	require.Error(t, err)
	assert.Empty(t, vars.saved, "调用失败不应写入变量")
	nodeInstance := h.nodes.byID[invokeJob.NodeInstanceID]
	assert.Equal(t, model.NodeInstanceStatusRUNNING, nodeInstance.Status, "失败后节点保持 RUNNING 等待重试")
}

// TestServiceNodeIdempotentRerun 节点已终态时重复触发：幂等空跑成功。
func TestServiceNodeIdempotentRerun(t *testing.T) {
	vars := &fakeVariables{}
	invoker := &fakeInvoker{scriptedOK: true, body: `{"data":{"id":"order-9"}}`}
	h, jobs, _ := newServiceHarness(t, serviceDoc(), vars, invoker)
	h.start(t)
	var invokeJob *model.Job
	for _, job := range jobs.byID {
		invokeJob = job
	}
	_, err := h.runtime.InvokeServiceNode(context.Background(), ServiceInvokeInput{
		TenantID: 1, InstanceID: invokeJob.InstanceID, NodeInstanceID: invokeJob.NodeInstanceID,
	})
	require.NoError(t, err)
	// 节点已 COMPLETED：再次触发空跑，不产生第二次调用
	requests := len(invoker.requests)
	res, err := h.runtime.InvokeServiceNode(context.Background(), ServiceInvokeInput{
		TenantID: 1, InstanceID: invokeJob.InstanceID, NodeInstanceID: invokeJob.NodeInstanceID,
	})
	require.NoError(t, err)
	assert.False(t, res.Completed)
	assert.Len(t, invoker.requests, requests)
}
