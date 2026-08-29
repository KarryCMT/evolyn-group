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

// NewRegistry 构造执行器注册表（V1：start/approval/condition/cc/service/end；
// 运行期命中未注册类型即快速失败）。
func NewRegistry(resolvers assignment.Registry, identity provider.IdentityProvider) Registry {
	return &registry{byType: map[model.NodeType]NodeExecutor{
		model.NodeTypeStart:     &StartExecutor{},
		model.NodeTypeApproval:  &ApprovalExecutor{resolvers: resolvers, identity: identity},
		model.NodeTypeCondition: &ConditionExecutor{},
		model.NodeTypeCC:        &CCExecutor{resolvers: resolvers},
		model.NodeTypeService:   &ServiceExecutor{},
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

// CCExecutor 抄送节点（第 10.6 章）：解析抄送对象 → 产出抄送快照（落库由
// Runtime 统一承担）→ 瞬时完成。CC 不是审批任务，不参与节点完成判定，
// 解析不到任何对象同样报错禁止静默跳过（第 17 章补充语义）。
type CCExecutor struct {
	resolvers assignment.Registry
}

func (e *CCExecutor) Type() model.NodeType { return model.NodeTypeCC }

func (e *CCExecutor) Execute(ctx context.Context, input ExecuteInput) (ExecuteResult, error) {
	spec := input.Node.Config.Recipients
	if spec == nil {
		return ExecuteResult{}, fmt.Errorf("cc node %s has no recipients spec", input.Node.Key)
	}
	resolver, ok := e.resolvers.ResolverOf(spec.Type)
	if !ok {
		return ExecuteResult{}, &assignment.ErrAssigneeNotFound{Type: spec.Type}
	}
	recipients, err := resolver.Resolve(ctx, assignment.ResolveInput{Ctx: input.Ctx, Spec: *spec})
	if err != nil {
		return ExecuteResult{}, err
	}
	if len(recipients) == 0 {
		return ExecuteResult{}, &assignment.ErrAssigneeNotFound{Type: spec.Type}
	}
	return ExecuteResult{Complete: true, CCRecipients: recipients}, nil
}

// ServiceExecutor 服务节点（Phase 7，第 12.1/19 章）：校验配置存在后以
// Async 挂起——HTTP 调用不在业务事务内执行，由 Runtime 排期 service.invoke
// Job，Job Worker 独立事务经 ServiceInvoker 窄端口调用并续跑推进环；
// 失败经 wf_job 重试记账退避重试。
type ServiceExecutor struct{}

func (e *ServiceExecutor) Type() model.NodeType { return model.NodeTypeService }

func (e *ServiceExecutor) Execute(ctx context.Context, input ExecuteInput) (ExecuteResult, error) {
	if input.Node.Config.Service == nil {
		return ExecuteResult{}, fmt.Errorf("service node %s has no service config", input.Node.Key)
	}
	return ExecuteResult{Async: true}, nil
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
