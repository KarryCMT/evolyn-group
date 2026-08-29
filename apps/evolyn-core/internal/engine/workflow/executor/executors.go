package executor

import (
	"context"
	"fmt"

	"evolyn/internal/engine/workflow/assignment"
	"evolyn/internal/engine/workflow/model"
)

// registry Registry 的默认实现：按 NodeType 注册执行器。
type registry struct {
	byType map[model.NodeType]NodeExecutor
}

func (r *registry) ExecutorOf(nodeType model.NodeType) (NodeExecutor, bool) {
	e, ok := r.byType[nodeType]
	return e, ok
}

// NewRegistry 构造执行器注册表（V1：start/approval/condition/cc/end；
// service 执行器 Phase 7 注册，Phase 2 运行期命中即快速失败）。
func NewRegistry(resolvers assignment.Registry) Registry {
	return &registry{byType: map[model.NodeType]NodeExecutor{
		model.NodeTypeStart:    &StartExecutor{},
		model.NodeTypeApproval: &ApprovalExecutor{resolvers: resolvers},
		model.NodeTypeEnd:      &EndExecutor{},
	}}
}

// StartExecutor 发起节点：瞬时完成，无业务动作。
type StartExecutor struct{}

func (e *StartExecutor) Type() model.NodeType { return model.NodeTypeStart }

func (e *StartExecutor) Execute(ctx context.Context, input ExecuteInput) (ExecuteResult, error) {
	return ExecuteResult{Complete: true}, nil
}

// EndExecutor 结束节点：节点瞬时完成；实例终态由 Runtime 依据节点类型落账。
type EndExecutor struct{}

func (e *EndExecutor) Type() model.NodeType { return model.NodeTypeEnd }

func (e *EndExecutor) Execute(ctx context.Context, input ExecuteInput) (ExecuteResult, error) {
	return ExecuteResult{Complete: true}, nil
}

// ApprovalExecutor 审批节点：解析审批人 → 构建任务蓝图（落库由 Runtime
// 统一承担，第 13.2 章事务模板第 11 步）→ WAIT 挂起。
type ApprovalExecutor struct {
	resolvers assignment.Registry
}

func (e *ApprovalExecutor) Type() model.NodeType { return model.NodeTypeApproval }

func (e *ApprovalExecutor) Execute(ctx context.Context, input ExecuteInput) (ExecuteResult, error) {
	spec := input.Node.Config.Assignee
	if spec == nil {
		return ExecuteResult{}, fmt.Errorf("approval node %s has no assignee spec", input.Node.Key)
	}
	resolver, ok := e.resolvers.ResolverOf(spec.Type)
	if !ok {
		// 解析不到任何审批人返回稳定错误（WORKFLOW_ASSIGNEE_NOT_FOUND），
		// 禁止静默跳过节点（v1.1 第 17 章补充语义）
		return ExecuteResult{}, &assignment.ErrAssigneeNotFound{Type: spec.Type}
	}
	actors, err := resolver.Resolve(ctx, assignment.ResolveInput{Ctx: input.Ctx, Spec: *spec})
	if err != nil {
		return ExecuteResult{}, err
	}
	if len(actors) == 0 {
		return ExecuteResult{}, &assignment.ErrAssigneeNotFound{Type: spec.Type}
	}

	// V1 每名参与人一个任务（Node ≠ NodeInstance ≠ Task，第 6.2 章）；
	// 节点完成判定由 Runtime 按审批模式裁决
	tasks := make([]model.Task, 0, len(actors))
	taskActors := make([][]model.Actor, 0, len(actors))
	for _, actor := range actors {
		tasks = append(tasks, model.Task{
			TenantID:   input.Ctx.TenantID,
			InstanceID: input.Ctx.Instance.InstanceID,
			NodeKey:    input.Node.Key,
			Status:     model.TaskStatusPENDING,
		})
		taskActors = append(taskActors, []model.Actor{actor})
	}
	return ExecuteResult{
		Wait:          true,
		CreatedTasks:  tasks,
		CreatedActors: taskActors,
	}, nil
}
