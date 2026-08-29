package runtime

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"evolyn/internal/engine/workflow/assignment"
	"evolyn/internal/engine/workflow/event"
	"evolyn/internal/engine/workflow/executor"
	"evolyn/internal/engine/workflow/model"
	"evolyn/internal/engine/workflow/provider"
	"evolyn/internal/engine/workflow/repository"
	"evolyn/internal/engine/workflow/task"
)

// ---- 内存仓储桩：模拟行语义（ID 回填、部分唯一索引、追加写） ----

type fakeDefinitions struct {
	defs     map[string]*model.Definition
	versions map[uint]*model.DefinitionVersion
}

func newFakeDefinitions() *fakeDefinitions {
	return &fakeDefinitions{defs: map[string]*model.Definition{}, versions: map[uint]*model.DefinitionVersion{}}
}

// publishDefinition 造一个已发布定义（快照 = 草稿）。
func (f *fakeDefinitions) publishDefinition(code string, doc model.Document) *model.Definition {
	defID := uint(len(f.defs) + 1)
	versionID := uint(100 + len(f.versions))
	f.versions[versionID] = &model.DefinitionVersion{ID: versionID, DefinitionID: defID, VersionNo: 1, Snapshot: doc}
	latest := versionID
	def := &model.Definition{
		ID: defID, TenantID: 1, Code: code, Status: model.DefinitionStatusPublished,
		LatestVersionID: &latest, PublishedVersion: 1,
	}
	f.defs[code] = def
	return def
}

func (f *fakeDefinitions) FindDefinitionByCode(ctx context.Context, tenantID uint, code string) (*model.Definition, error) {
	if def, ok := f.defs[code]; ok {
		return def, nil
	}
	return nil, fmt.Errorf("not found")
}

func (f *fakeDefinitions) FindVersion(ctx context.Context, tenantID, definitionID uint, versionNo int) (*model.DefinitionVersion, error) {
	for _, v := range f.versions {
		if v.DefinitionID == definitionID && v.VersionNo == versionNo {
			return v, nil
		}
	}
	return nil, fmt.Errorf("version not found")
}

func (f *fakeDefinitions) FindVersionByID(ctx context.Context, tenantID, versionID uint) (*model.DefinitionVersion, error) {
	if v, ok := f.versions[versionID]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("version not found")
}

// 未在 Runtime 路径使用的接口方法：显式 panic 防止测试误用
func (f *fakeDefinitions) CreateDefinition(ctx context.Context, def *model.Definition) error {
	panic("not used")
}
func (f *fakeDefinitions) ListDefinitions(ctx context.Context, tenantID uint) ([]model.Definition, error) {
	panic("not used")
}
func (f *fakeDefinitions) SaveDraft(ctx context.Context, def *model.Definition) error {
	panic("not used")
}
func (f *fakeDefinitions) SoftDeleteDefinition(ctx context.Context, tenantID uint, code string) error {
	panic("not used")
}
func (f *fakeDefinitions) CreateVersion(ctx context.Context, version *model.DefinitionVersion) error {
	panic("not used")
}
func (f *fakeDefinitions) ListVersions(ctx context.Context, definitionID uint) ([]model.DefinitionVersion, error) {
	panic("not used")
}

type fakeInstances struct {
	mu     sync.Mutex
	byID   map[uint]*model.Instance
	nextID uint
}

func newFakeInstances() *fakeInstances {
	return &fakeInstances{byID: map[uint]*model.Instance{}, nextID: 1}
}

func (f *fakeInstances) CreateInstance(ctx context.Context, instance *model.Instance) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	// 模拟 RUNNING 部分唯一索引
	for _, other := range f.byID {
		if other.TenantID == instance.TenantID && other.BusinessType == instance.BusinessType &&
			other.BusinessID == instance.BusinessID && other.Status == model.InstanceStatusRUNNING {
			return fmt.Errorf("unique constraint wf_instance_running_business")
		}
	}
	// 模拟幂等键部分唯一索引
	if instance.IdempotencyKey != "" {
		for _, other := range f.byID {
			if other.TenantID == instance.TenantID && other.IdempotencyKey == instance.IdempotencyKey {
				return fmt.Errorf("unique constraint wf_instance_idempotency_key")
			}
		}
	}
	instance.ID = f.nextID
	f.nextID++
	f.byID[instance.ID] = instance
	return nil
}

func (f *fakeInstances) FindInstanceByID(ctx context.Context, tenantID, instanceID uint) (*model.Instance, error) {
	if instance, ok := f.byID[instanceID]; ok {
		return instance, nil
	}
	return nil, fmt.Errorf("instance not found")
}

func (f *fakeInstances) FindInstanceByIDForUpdate(ctx context.Context, tenantID, instanceID uint) (*model.Instance, error) {
	if instance, ok := f.byID[instanceID]; ok {
		return instance, nil
	}
	return nil, fmt.Errorf("instance not found")
}

func (f *fakeInstances) FindRunningInstanceByBusiness(ctx context.Context, tenantID uint, businessType, businessID string) (*model.Instance, error) {
	for _, instance := range f.byID {
		if instance.TenantID == tenantID && instance.BusinessType == businessType &&
			instance.BusinessID == businessID && instance.Status == model.InstanceStatusRUNNING {
			return instance, nil
		}
	}
	return nil, nil
}

func (f *fakeInstances) FindInstanceByIdempotencyKey(ctx context.Context, tenantID uint, key string) (*model.Instance, error) {
	for _, instance := range f.byID {
		if instance.TenantID == tenantID && instance.IdempotencyKey == key {
			return instance, nil
		}
	}
	return nil, nil
}

func (f *fakeInstances) SaveInstance(ctx context.Context, instance *model.Instance) error {
	f.byID[instance.ID] = instance
	return nil
}

func (f *fakeInstances) HasRunningInstanceByDefinition(ctx context.Context, definitionID uint) (bool, error) {
	panic("not used")
}

type fakeExecutions struct {
	byInstance map[uint][]*model.Execution
	next       uint
}

func newFakeExecutions() *fakeExecutions {
	return &fakeExecutions{byInstance: map[uint][]*model.Execution{}}
}

func (f *fakeExecutions) CreateExecution(ctx context.Context, execution *model.Execution) error {
	execution.ID = f.next
	f.next++
	f.byInstance[execution.InstanceID] = append(f.byInstance[execution.InstanceID], execution)
	return nil
}

func (f *fakeExecutions) ListExecutionsByInstance(ctx context.Context, instanceID uint) ([]model.Execution, error) {
	rows := make([]model.Execution, 0)
	for _, e := range f.byInstance[instanceID] {
		rows = append(rows, *e)
	}
	return rows, nil
}

func (f *fakeExecutions) SaveExecution(ctx context.Context, execution *model.Execution) error {
	for i, e := range f.byInstance[execution.InstanceID] {
		if e.ID == execution.ID {
			f.byInstance[execution.InstanceID][i] = execution
		}
	}
	return nil
}

func (f *fakeExecutions) FindExecutionByID(ctx context.Context, tenantID, executionID uint) (*model.Execution, error) {
	panic("not used")
}

type fakeNodes struct {
	byID map[uint]*model.NodeInstance
	next uint
}

func newFakeNodes() *fakeNodes {
	return &fakeNodes{byID: map[uint]*model.NodeInstance{}}
}

func (f *fakeNodes) CreateNodeInstance(ctx context.Context, nodeInstance *model.NodeInstance) error {
	nodeInstance.ID = f.next
	f.next++
	f.byID[nodeInstance.ID] = nodeInstance
	return nil
}

func (f *fakeNodes) FindNodeInstanceByID(ctx context.Context, tenantID, nodeInstanceID uint) (*model.NodeInstance, error) {
	if n, ok := f.byID[nodeInstanceID]; ok {
		return n, nil
	}
	return nil, fmt.Errorf("node instance not found")
}

func (f *fakeNodes) ListNodeInstancesByInstance(ctx context.Context, instanceID uint) ([]model.NodeInstance, error) {
	panic("not used")
}

func (f *fakeNodes) SaveNodeInstance(ctx context.Context, nodeInstance *model.NodeInstance) error {
	f.byID[nodeInstance.ID] = nodeInstance
	return nil
}

type fakeTasks struct {
	mu     sync.Mutex
	byID   map[uint]*model.Task
	actors map[uint][]model.Actor
	next   uint
}

func newFakeTasks() *fakeTasks {
	return &fakeTasks{byID: map[uint]*model.Task{}, actors: map[uint][]model.Actor{}}
}

func (f *fakeTasks) CreateTask(ctx context.Context, task *model.Task) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	task.ID = f.next
	f.next++
	f.byID[task.ID] = task
	return nil
}

func (f *fakeTasks) ReplaceActors(ctx context.Context, taskID uint, actors []model.Actor) error {
	f.actors[taskID] = actors
	return nil
}

func (f *fakeTasks) FindTaskByIDForUpdate(ctx context.Context, tenantID, taskID uint) (*model.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if task, ok := f.byID[taskID]; ok {
		return task, nil
	}
	return nil, fmt.Errorf("task not found")
}

func (f *fakeTasks) ListTasksByInstance(ctx context.Context, instanceID uint) ([]model.Task, error) {
	rows := make([]model.Task, 0)
	for _, task := range f.byID {
		if task.InstanceID == instanceID {
			rows = append(rows, *task)
		}
	}
	return rows, nil
}

func (f *fakeTasks) ListTasksByNodeInstance(ctx context.Context, nodeInstanceID uint) ([]model.Task, error) {
	rows := make([]model.Task, 0)
	for _, task := range f.byID {
		if task.NodeInstanceID == nodeInstanceID {
			rows = append(rows, *task)
		}
	}
	return rows, nil
}

func (f *fakeTasks) ListActorsOfTask(ctx context.Context, taskID uint) ([]model.Actor, error) {
	return f.actors[taskID], nil
}

func (f *fakeTasks) SaveTask(ctx context.Context, task *model.Task) error {
	f.byID[task.ID] = task
	return nil
}

func (f *fakeTasks) CancelPendingTasksByNode(ctx context.Context, nodeInstanceID uint) (int64, error) {
	var cancelled int64
	for _, task := range f.byID {
		if task.NodeInstanceID == nodeInstanceID && task.Status == model.TaskStatusPENDING {
			task.Status = model.TaskStatusCANCELLED
			cancelled++
		}
	}
	return cancelled, nil
}

type fakeOperations struct {
	ops []model.Operation
}

func (f *fakeOperations) AppendOperation(ctx context.Context, operation *model.Operation) error {
	f.ops = append(f.ops, *operation)
	return nil
}

func (f *fakeOperations) ListOperationsByInstance(ctx context.Context, instanceID uint) ([]model.Operation, error) {
	panic("not used")
}

type fakePublisher struct {
	events []string
}

func (f *fakePublisher) PublishInTx(ctx context.Context, e provider.Event) error {
	f.events = append(f.events, e.EventName)
	return nil
}

// ---- 构造 ----

// approvalDoc start → approval(user 2) → end 的单审批人快照。
func approvalDoc() model.Document {
	return model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			{Key: "boss", Type: model.NodeTypeApproval, Name: "主管审批", Config: model.NodeConfig{
				ApprovalMode: model.ApprovalModeSingle,
				Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{2}},
			}},
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "boss"},
			{Key: "e2", Source: "boss", Target: "end"},
		},
	}
}

type harness struct {
	runtime     *Runtime
	definitions *fakeDefinitions
	instances   *fakeInstances
	tasks       *fakeTasks
	nodes       *fakeNodes
	publisher   *fakePublisher
}

func newHarness(t *testing.T, doc model.Document) *harness {
	t.Helper()
	definitions := newFakeDefinitions()
	instances := newFakeInstances()
	executions := newFakeExecutions()
	nodes := newFakeNodes()
	tasks := newFakeTasks()
	operations := &fakeOperations{}
	publisher := &fakePublisher{}

	definitions.publishDefinition("wf_test", doc)
	registry := executor.NewRegistry(assignment.NewRegistry(nil, nil), nil)
	rt := NewRuntime(definitions, instances, executions, nodes, tasks, operations, registry, publisher, nil, nil)
	return &harness{runtime: rt, definitions: definitions, instances: instances, tasks: tasks, nodes: nodes, publisher: publisher}
}

func (h *harness) start(t *testing.T) *StartResult {
	t.Helper()
	result, err := h.runtime.Start(context.Background(), StartInput{
		TenantID: 1, Code: "wf_test",
		BusinessType: "expense", BusinessID: "biz-1",
		StarterMemberID: 1, IdempotencyKey: "req-1",
	})
	require.NoError(t, err)
	return result
}

// ---- 用例 ----

func TestEndToEndStartApproveComplete(t *testing.T) {
	h := newHarness(t, approvalDoc())

	// Start：实例 RUNNING，推进至审批节点挂起
	result := h.start(t)
	assert.Equal(t, model.InstanceStatusRUNNING, result.Status)
	assert.False(t, result.IdempotentReplay)

	// 人工任务已创建，参与人快照为 user 2
	var pending *model.Task
	for _, task := range h.tasks.byID {
		if task.Status == model.TaskStatusPENDING {
			pending = task
		}
	}
	require.NotNil(t, pending, "审批节点应创建 PENDING 任务")
	assert.Equal(t, "boss", pending.NodeKey)
	actors, err := h.tasks.ListActorsOfTask(context.Background(), pending.ID)
	require.NoError(t, err)
	require.Len(t, actors, 1)
	assert.Equal(t, uint(2), actors[0].MemberID)

	// 节点实例挂起 WAITING
	nodeInstance, err := h.nodes.FindNodeInstanceByID(context.Background(), 1, pending.NodeInstanceID)
	require.NoError(t, err)
	assert.Equal(t, model.NodeInstanceStatusWAITING, nodeInstance.Status)

	// 非参与人拒绝
	_, err = h.runtime.Approve(context.Background(), ApproveInput{TenantID: 1, TaskID: pending.ID, OperatorMemberID: 99})
	assert.ErrorIs(t, err, task.ErrTaskForbidden)

	// 参与人同意：实例完成（start → approval → end 顺序流）
	done, err := h.runtime.Approve(context.Background(), ApproveInput{TenantID: 1, TaskID: pending.ID, OperatorMemberID: 2, Comment: "同意"})
	require.NoError(t, err)
	assert.True(t, done.NodeCompleted)
	assert.Equal(t, model.InstanceStatusCOMPLETED, done.InstanceStatus)

	instance, err := h.instances.FindInstanceByIDForUpdate(context.Background(), 1, result.InstanceID)
	require.NoError(t, err)
	assert.Equal(t, model.InstanceStatusCOMPLETED, instance.Status)

	// 事件目录：start / task.created / node.entered / task.approved / node.completed / instance.completed
	assert.Contains(t, h.publisher.events, event.InstanceStarted)
	assert.Contains(t, h.publisher.events, event.TaskCreated)
	assert.Contains(t, h.publisher.events, event.NodeEntered)
	assert.Contains(t, h.publisher.events, event.TaskApproved)
	assert.Contains(t, h.publisher.events, event.NodeCompleted)
	assert.Contains(t, h.publisher.events, event.InstanceCompleted)
}

func TestDoubleApproveDoesNotAdvanceTwice(t *testing.T) {
	h := newHarness(t, approvalDoc())
	h.start(t)

	var pending *model.Task
	for _, task := range h.tasks.byID {
		if task.Status == model.TaskStatusPENDING {
			pending = task
		}
	}
	require.NotNil(t, pending)
	_, err := h.runtime.Approve(context.Background(), ApproveInput{TenantID: 1, TaskID: pending.ID, OperatorMemberID: 2})
	require.NoError(t, err)

	// 双击防护：第二次 Approve 命中 TASK_NOT_PENDING，实例不再推进
	_, err = h.runtime.Approve(context.Background(), ApproveInput{TenantID: 1, TaskID: pending.ID, OperatorMemberID: 2})
	assert.ErrorIs(t, err, task.ErrTaskNotPending)
	assert.Equal(t, 1, countStatus(h.publisher.events, event.InstanceCompleted))
}

func TestDoubleStartBusinessIdempotency(t *testing.T) {
	h := newHarness(t, approvalDoc())
	h.start(t)

	// 同业务键第二个 RUNNING 实例被拒（第 14.1 章）
	_, err := h.runtime.Start(context.Background(), StartInput{
		TenantID: 1, Code: "wf_test", BusinessType: "expense", BusinessID: "biz-1", StarterMemberID: 1,
	})
	assert.ErrorIs(t, err, ErrInstanceAlreadyRunning)
}

func TestStartRequestIdempotencyReplay(t *testing.T) {
	h := newHarness(t, approvalDoc())
	first := h.start(t)

	// 同幂等键重发：重放返回同一实例，不新建（第 14.2 章）
	replay, err := h.runtime.Start(context.Background(), StartInput{
		TenantID: 1, Code: "wf_test",
		BusinessType: "expense", BusinessID: "biz-2", // 换业务键，仅幂等键命中
		StarterMemberID: 1, IdempotencyKey: "req-1",
	})
	require.NoError(t, err)
	assert.True(t, replay.IdempotentReplay)
	assert.Equal(t, first.InstanceID, replay.InstanceID)
	assert.Equal(t, 1, len(h.instances.byID), "重放不得创建第二个实例")
}

func TestStartRejectsUnpublishedDefinition(t *testing.T) {
	h := newHarness(t, approvalDoc())
	h.definitions.defs["wf_test"].PublishedVersion = 0

	_, err := h.runtime.Start(context.Background(), StartInput{
		TenantID: 1, Code: "wf_test", BusinessType: "expense", BusinessID: "biz-1", StarterMemberID: 1,
	})
	assert.ErrorIs(t, err, ErrDefinitionNotPublished)
}

func TestStartRecordsFormBinding(t *testing.T) {
	h := newHarness(t, approvalDoc())
	result, err := h.runtime.Start(context.Background(), StartInput{
		TenantID: 1, Code: "wf_test",
		BusinessType: "expense", BusinessID: "biz-1",
		StarterMemberID: 1, AppID: 7, FormID: 8, FormVersionID: 9,
	})
	require.NoError(t, err)
	_ = result
	instance, err := h.instances.FindInstanceByIDForUpdate(context.Background(), 1, result.InstanceID)
	require.NoError(t, err)
	assert.Equal(t, uint(7), instance.AppID)
	assert.Equal(t, uint(8), instance.FormID)
	assert.Equal(t, uint(9), instance.FormVersionID)
}

func countStatus(events []string, name string) int {
	count := 0
	for _, e := range events {
		if e == name {
			count++
		}
	}
	return count
}

var _ repository.DefinitionRepository = (*fakeDefinitions)(nil)
var _ repository.InstanceRepository = (*fakeInstances)(nil)
var _ repository.ExecutionRepository = (*fakeExecutions)(nil)
var _ repository.NodeInstanceRepository = (*fakeNodes)(nil)
var _ repository.TaskRepository = (*fakeTasks)(nil)
var _ repository.OperationRepository = (*fakeOperations)(nil)
var _ provider.EventPublisher = (*fakePublisher)(nil)
