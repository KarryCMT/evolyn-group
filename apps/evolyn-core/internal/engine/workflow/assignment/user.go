package assignment

import (
	"context"

	"evolyn/internal/engine/workflow/model"
	"evolyn/internal/engine/workflow/provider"
)

// registry Registry 默认实现：按类型注册 Resolver，并受 V1 能力矩阵约束
// （IAM 前置能力未落地类型拒绝注册，第 17 章）。
type registry struct {
	resolvers map[model.AssigneeType]AssigneeResolver
}

func (r *registry) ResolverOf(assigneeType model.AssigneeType) (AssigneeResolver, bool) {
	resolver, ok := r.resolvers[assigneeType]
	return resolver, ok
}

// NewRegistry 构造审批人解析注册表（Phase 2：指定用户；角色/表单字段
// 随 Phase 3 接入，部门负责人/直属主管待 IAM 前置能力落地）。
func NewRegistry(identity provider.IdentityProvider) Registry {
	resolvers := map[model.AssigneeType]AssigneeResolver{
		model.AssigneeTypeUser: &UserResolver{identity: identity},
	}
	return &registry{resolvers: resolvers}
}

// UserResolver 指定用户解析器：规格中的成员 ID 集合即为审批人快照来源；
// 身份端口可用时校验成员同租户有效并快照显示名（第 17 章：解析不到任何
// 审批人必须报错，禁止静默跳过）。
type UserResolver struct {
	identity provider.IdentityProvider
}

func (r *UserResolver) Type() model.AssigneeType { return model.AssigneeTypeUser }

func (r *UserResolver) Resolve(ctx context.Context, input ResolveInput) ([]model.Actor, error) {
	if len(input.Spec.UserIDs) == 0 {
		return nil, &ErrAssigneeNotFound{Type: r.Type(), Message: "指定用户审批人规格为空"}
	}
	if r.identity != nil {
		if err := r.identity.ValidateMembers(ctx, input.Ctx.TenantID, input.Spec.UserIDs); err != nil {
			return nil, err
		}
	}
	actors := make([]model.Actor, 0, len(input.Spec.UserIDs))
	for _, memberID := range input.Spec.UserIDs {
		actor := model.Actor{MemberID: memberID}
		if r.identity != nil {
			actor.DisplayName = r.identity.MemberDisplayName(ctx, input.Ctx.TenantID, memberID)
		}
		actors = append(actors, actor)
	}
	return actors, nil
}
