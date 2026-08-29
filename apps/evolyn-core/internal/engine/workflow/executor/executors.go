package executor

import (
	"context"
	"errors"
	"fmt"

	"evolyn/internal/engine/workflow/assignment"
	"evolyn/internal/engine/workflow/model"
	"evolyn/internal/engine/workflow/provider"
)

// registry Registry 的默认实现：按 NodeType 注册执行器。
type registry struct {
	byType map[model.NodeType]NodeExecutor
}

func (r *registry) ExecutorOf(nodeType model.NodeType) (NodeExecutor, bool) {
	e, ok := r.byType[nodeType]
	return e, ok
}

// NewRegistry 构造执行器注册表（V1：start/approval/condition/end；
// service 执行器 Phase 7 注册，cc 执行器 Phase 4 随人工任务全量动作注册，
// 运行期命中未注册类型即快速失败）。
func NewRegistry(resolvers assignment.Registry, identity provider.IdentityProvider) Registry {
	return &registry{byType: map[model.NodeType]NodeExecutor{
		model.NodeTypeStart:     &StartExecutor{},
		model.NodeTypeApproval:  &ApprovalExecutor{resolvers: resolvers, identity: identity},
		model.NodeTypeCondition: &ConditionExecutor{},
		model.NodeTypeEnd:       &EndExecutor{},
	}}
}

// StartExecutor 发起节点：瞬时完成，无业务动作。
type StartExecutor struct{}

func (e *StartExecutor) Type() model.NodeType { return model.NodeTypeStart }

func (e *StartExecutor) Execute(ctx context.Context, input ExecuteInput) (ExecuteResult, error) {
	return ExecuteResult{Complete: true}, nil
}

// ConditionExecutor 条件节点：瞬时完成，无业务动作；分支选择由 Navigator
// 按出边表达式求值承担（第 12.2 章「Navigator.FindNext」），执行器只负责
// 让推进环越过本节点。
type ConditionExecutor struct{}

func (e *ConditionExecutor) Type() model.NodeType { return model.NodeTypeCondition }

func (e *ConditionExecutor) Execute(ctx context.Context, input ExecuteInput) (ExecuteResult, error) {
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
// 解析不到审批人时按 v1.1 补充语义兜底转交租户管理员（identity 端口），
// 兜底仍为空才返回 WORKFLOW_ASSIGNEE_NOT_FOUND，禁止静默跳过节点。
type ApprovalExecutor struct {
	resolvers assignment.Registry
	identity  provider.IdentityProvider
}

func (e *ApprovalExecutor) Type() model.NodeType { return model.NodeTypeApproval }

func (e *ApprovalExecutor) Execute(ctx context.Context, input ExecuteInput) (ExecuteResult, error) {
	spec := input.Node.Config.Assignee
	if spec == nil {
		return ExecuteResult{}, fmt.Errorf("approval node %s has no assignee spec", input.Node.Key)
	}
	resolver, ok := e.resolvers.ResolverOf(spec.Type)
	if !ok {
		// 未注册类型经能力矩阵双重确认（发布校验器同口径）
		return ExecuteResult{}, &assignment.ErrAssigneeNotFound{Type: spec.Type}
	}
	actors, err := resolver.Resolve(ctx, assignment.ResolveInput{Ctx: input.Ctx, Spec: *spec})
	if err != nil {
		// 解析为空 → 兜底转交租户管理员；兜底也为空/端口未装配 → 原错误上抛
		var notFound *assignment.ErrAssigneeNotFound
		if errors.As(err, &notFound) && e.identity != nil {
			fallback, fbErr := e.identity.ResolveTenantAdmins(ctx, input.Ctx.TenantID)
			if fbErr == nil && len(fallback) > 0 {
				actors = fallback
			} else {
				return ExecuteResult{}, err
			}
		} else {
			return ExecuteResult{}, err
		}
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
