// Phase 3 引擎测试：条件路由（Expr + 预编译产物）、form.* 表达式数据源、
// 表单用户字段/部门负责人审批人解析、审批编辑字段权限过滤与同事务写回。
package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"evolyn/internal/engine/workflow/assignment"
	"evolyn/internal/engine/workflow/executor"
	"evolyn/internal/engine/workflow/model"
	"evolyn/internal/engine/workflow/provider"
)

// fakeBusinessData BusinessDataProvider 内存桩：模拟 form_records 值读写。
type fakeBusinessData struct {
	values   map[string]any
	updated  []map[string]any
	getCalls int
}

func (f *fakeBusinessData) GetData(ctx context.Context, ref provider.BusinessRef) (map[string]any, error) {
	f.getCalls++
	if f.values == nil {
		return map[string]any{}, nil
	}
	return f.values, nil
}

func (f *fakeBusinessData) UpdateData(ctx context.Context, ref provider.BusinessRef, values map[string]any) error {
	f.updated = append(f.updated, values)
	for k, v := range values {
		f.values[k] = v
	}
	return nil
}

// fakeIdentity IdentityProvider 桩：成员上下文与显示名。
type fakeIdentity struct {
	admins []model.Actor
}

func (f *fakeIdentity) ValidateMembers(ctx context.Context, tenantID uint, memberIDs []uint) error {
	return nil
}

func (f *fakeIdentity) MemberDisplayName(ctx context.Context, tenantID, memberID uint) string {
	return "成员"
}

func (f *fakeIdentity) ResolveTenantAdmins(ctx context.Context, tenantID uint) ([]model.Actor, error) {
	return f.admins, nil
}

func (f *fakeIdentity) MemberContext(ctx context.Context, tenantID, memberID uint) (string, uint, error) {
	return "account_1", 5, nil
}

// newFormHarness 带业务数据/身份窄端口的推进环测试骨架。
func newFormHarness(t *testing.T, doc model.Document, business provider.BusinessDataProvider, identity provider.IdentityProvider) *harness {
	t.Helper()
	definitions := newFakeDefinitions()
	instances := newFakeInstances()
	executions := newFakeExecutions()
	nodes := newFakeNodes()
	tasks := newFakeTasks()
	operations := &fakeOperations{}
	publisher := &fakePublisher{}

	definitions.publishDefinition("wf_test", doc)
	registry := executor.NewRegistry(assignment.NewRegistry(identity, nil), identity)
	rt := NewRuntime(definitions, instances, executions, nodes, tasks, operations, registry, publisher, business, identity)
	return &harness{runtime: rt, definitions: definitions, instances: instances, tasks: tasks, nodes: nodes, publisher: publisher}
}

// conditionDoc start → boss（user 2，amount 可编辑）→ cond →
//   - expr form.amount > 100 → finance（user 3）
//   - default                → end
func conditionDoc() model.Document {
	return model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			{Key: "boss", Type: model.NodeTypeApproval, Name: "主管审批", Config: model.NodeConfig{
				ApprovalMode:    model.ApprovalModeSingle,
				Assignee:        &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{2}},
				FormPermissions: map[string]model.FieldPermission{"amount": model.FieldPermissionEditable},
			}},
			{Key: "cond", Type: model.NodeTypeCondition, Name: "金额路由"},
			{Key: "finance", Type: model.NodeTypeApproval, Name: "财务审批", Config: model.NodeConfig{
				ApprovalMode: model.ApprovalModeSingle,
				Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{3}},
			}},
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "boss"},
			{Key: "e2", Source: "boss", Target: "cond"},
			{Key: "e3", Source: "cond", Target: "finance", Condition: &model.EdgeCondition{Expression: "form.amount > 100"}},
			{Key: "e4", Source: "cond", Target: "end"},
			{Key: "e5", Source: "finance", Target: "end"},
		},
	}
}

// TestApproveFormValuesRouteCondition 审批编辑 amount=200 → 条件路由命中
// 财务节点；写回值经业务窄端口进入同一事务（顺序审批 Phase 3 主链路）。
func TestApproveFormValuesRouteCondition(t *testing.T) {
	business := &fakeBusinessData{values: map[string]any{"amount": float64(200)}}
	h := newFormHarness(t, conditionDoc(), business, &fakeIdentity{})
	start, err := h.runtime.Start(context.Background(), StartInput{
		TenantID: 1, Code: "wf_test", BusinessType: "expense", BusinessID: "biz-1",
		StarterMemberID: 1, FormVersionID: 9,
	})
	require.NoError(t, err)
	require.Len(t, h.tasks.byID, 1)
	bossTask := start.InstanceID

	// 主管审批并修改 amount（授权字段）→ 条件节点 → 财务待办
	_, err = h.runtime.Approve(context.Background(), ApproveInput{
		TenantID: 1, TaskID: taskIDOf(h, bossTask), OperatorMemberID: 2,
		FormValues: map[string]any{"amount": float64(300)},
	})
	require.NoError(t, err)
	require.Len(t, business.updated, 1)
	assert.Equal(t, float64(300), business.updated[0]["amount"])

	// amount=300 > 100 → 财务节点挂起
	pending := h.pendingTasks(t)
	require.Len(t, pending, 1)
	assert.Equal(t, "finance", h.tasks.byID[pending[0]].NodeKey)

	// 财务同意 → end → 实例完成
	_, err = h.runtime.Approve(context.Background(), ApproveInput{
		TenantID: 1, TaskID: pending[0], OperatorMemberID: 3,
	})
	require.NoError(t, err)
	instance := h.instances.byID[start.InstanceID]
	assert.Equal(t, model.InstanceStatusCOMPLETED, instance.Status)
}

// TestApproveFormFieldForbidden 未授权字段编辑整体拒绝（不写回、不推进）。
func TestApproveFormFieldForbidden(t *testing.T) {
	business := &fakeBusinessData{values: map[string]any{"amount": float64(200)}}
	h := newFormHarness(t, conditionDoc(), business, &fakeIdentity{})
	start, err := h.runtime.Start(context.Background(), StartInput{
		TenantID: 1, Code: "wf_test", BusinessType: "expense", BusinessID: "biz-2",
		StarterMemberID: 1, FormVersionID: 9,
	})
	require.NoError(t, err)

	_, err = h.runtime.Approve(context.Background(), ApproveInput{
		TenantID: 1, TaskID: taskIDOf(h, start.InstanceID), OperatorMemberID: 2,
		FormValues: map[string]any{"finance_code": "X001"}, // 未在 formPermissions 中授权
	})
	assert.ErrorIs(t, err, ErrFormFieldForbidden)
	assert.Empty(t, business.updated, "拒绝时不得触发写回")
	// 节点未推进：实例保持 RUNNING（内存桩无事务回滚，真实回滚由
	// TxManager 边界保证；此处验证的是错误语义与零写回副作用）
	assert.Equal(t, model.InstanceStatusRUNNING, h.instances.byID[start.InstanceID].Status)
}

// TestApproveFormValuesWithoutBinding 未绑定表单却携带编辑值 → 拒绝。
func TestApproveFormValuesWithoutBinding(t *testing.T) {
	h := newFormHarness(t, approvalDoc(), &fakeBusinessData{}, &fakeIdentity{})
	start := h.start(t)
	_, err := h.runtime.Approve(context.Background(), ApproveInput{
		TenantID: 1, TaskID: taskIDOf(h, start.InstanceID), OperatorMemberID: 2,
		FormValues: map[string]any{"amount": float64(1)},
	})
	assert.ErrorIs(t, err, ErrFormFieldForbidden)
}

// TestFormFieldAssignee 表单用户字段审批人：form.manager_id=3 → 任务落成员 3。
func TestFormFieldAssignee(t *testing.T) {
	doc := model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			{Key: "boss", Type: model.NodeTypeApproval, Name: "指定经理", Config: model.NodeConfig{
				ApprovalMode: model.ApprovalModeSingle,
				Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeFormField, FormField: "manager_id"},
			}},
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "boss"},
			{Key: "e2", Source: "boss", Target: "end"},
		},
	}
	business := &fakeBusinessData{values: map[string]any{"manager_id": float64(3)}}
	h := newFormHarness(t, doc, business, &fakeIdentity{})
	_, err := h.runtime.Start(context.Background(), StartInput{
		TenantID: 1, Code: "wf_test", BusinessType: "expense", BusinessID: "biz-1",
		StarterMemberID: 1, FormVersionID: 9,
	})
	require.NoError(t, err)
	require.Len(t, h.tasks.byID, 1)
	for _, task := range h.tasks.byID {
		assert.Equal(t, "boss", task.NodeKey)
	}
	actors := h.tasks.actors[taskIDOf(h, 1)]
	require.Len(t, actors, 1)
	assert.Equal(t, uint(3), actors[0].MemberID)
}

// TestFormFieldAssigneeEmptyField 字段无值 → 稳定错误（禁止静默跳过）。
func TestFormFieldAssigneeEmptyField(t *testing.T) {
	doc := model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			{Key: "boss", Type: model.NodeTypeApproval, Name: "指定经理", Config: model.NodeConfig{
				ApprovalMode: model.ApprovalModeSingle,
				Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeFormField, FormField: "manager_id"},
			}},
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "boss"},
			{Key: "e2", Source: "boss", Target: "end"},
		},
	}
	h := newFormHarness(t, doc, &fakeBusinessData{}, &fakeIdentity{})
	_, err := h.runtime.Start(context.Background(), StartInput{
		TenantID: 1, Code: "wf_test", BusinessType: "expense", BusinessID: "biz-1", StarterMemberID: 1,
	})
	require.Error(t, err)
	var notFound *assignment.ErrAssigneeNotFound
	assert.ErrorAs(t, err, &notFound)
}

// taskIDOf 取实例当前首个任务 ID（单任务场景辅助）。
func taskIDOf(h *harness, instanceID uint) uint {
	for id, task := range h.tasks.byID {
		if task.InstanceID == instanceID {
			_ = id
			return task.ID
		}
	}
	return 0
}

// pendingTasks 收集 PENDING 任务 ID。
func (h *harness) pendingTasks(t *testing.T) []uint {
	t.Helper()
	ids := make([]uint, 0)
	for _, task := range h.tasks.byID {
		if task.Status == model.TaskStatusPENDING {
			ids = append(ids, task.ID)
		}
	}
	return ids
}
