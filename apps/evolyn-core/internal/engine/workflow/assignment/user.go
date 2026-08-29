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

// NewRegistry 构造审批人解析注册表：注册类型必须与 v1ResolverCapabilities
// 启用集一致（发布校验器据此拒绝未启用类型，双保险）。
//   - user：指定用户（身份端口校验 + 显示名快照）；
//   - role：指定角色（组织端口解析角色成员）；
//   - form_field：表单用户字段（运行上下文 form.* 取成员 ID）；
//   - department_manager：部门负责人（迁移 000050 leader 前置）。
func NewRegistry(identity provider.IdentityProvider, org provider.OrganizationProvider) Registry {
	resolvers := map[model.AssigneeType]AssigneeResolver{
		model.AssigneeTypeUser:              &UserResolver{identity: identity},
		model.AssigneeTypeRole:              &RoleResolver{org: org},
		model.AssigneeTypeFormField:         &FormFieldResolver{identity: identity},
		model.AssigneeTypeDepartmentManager: &DepartmentManagerResolver{org: org},
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
