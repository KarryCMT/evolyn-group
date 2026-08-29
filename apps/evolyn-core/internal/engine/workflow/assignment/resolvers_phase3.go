package assignment

import (
	"context"
	"strconv"
	"strings"

	"evolyn/internal/engine/workflow/model"
	"evolyn/internal/engine/workflow/provider"
)

// RoleResolver 指定角色解析器：按角色名解析租户内有效成员（组织端口）。
// RoleCode 语义 = 角色名称（租户内唯一，与 IAM 现行模型一致）；解析结果
// 在任务创建时一次性快照，运行中不随角色调整重算（v1.1 定版）。
type RoleResolver struct {
	org provider.OrganizationProvider
}

func (r *RoleResolver) Type() model.AssigneeType { return model.AssigneeTypeRole }

func (r *RoleResolver) Resolve(ctx context.Context, input ResolveInput) ([]model.Actor, error) {
	if r.org == nil {
		return nil, &ErrAssigneeNotFound{Type: r.Type(), Message: "组织端口未装配，无法解析角色成员"}
	}
	actors, err := r.org.ResolveRoleMembers(ctx, input.Ctx.TenantID, input.Spec.RoleCode)
	if err != nil {
		return nil, err
	}
	if len(actors) == 0 {
		return nil, &ErrAssigneeNotFound{Type: r.Type(), Message: "角色下无有效成员：" + input.Spec.RoleCode}
	}
	return actors, nil
}

// FormFieldResolver 表单用户字段解析器：从运行上下文 form.<formField> 读取
// 成员 ID（单值/数组均可，数字或数字字符串），作为审批人快照来源。
// 依赖 Runtime 在节点执行前经 BusinessDataProvider 填充 WorkflowContext.Form。
type FormFieldResolver struct {
	identity provider.IdentityProvider
}

func (r *FormFieldResolver) Type() model.AssigneeType { return model.AssigneeTypeFormField }

func (r *FormFieldResolver) Resolve(ctx context.Context, input ResolveInput) ([]model.Actor, error) {
	raw, ok := input.Ctx.Form[input.Spec.FormField]
	if !ok {
		return nil, &ErrAssigneeNotFound{Type: r.Type(),
			Message: "表单用户字段无值：" + input.Spec.FormField}
	}
	memberIDs := parseMemberIDs(raw)
	if len(memberIDs) == 0 {
		return nil, &ErrAssigneeNotFound{Type: r.Type(),
			Message: "表单用户字段不是有效成员：" + input.Spec.FormField}
	}
	if r.identity != nil {
		if err := r.identity.ValidateMembers(ctx, input.Ctx.TenantID, memberIDs); err != nil {
			return nil, err
		}
	}
	actors := make([]model.Actor, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		actor := model.Actor{MemberID: memberID}
		if r.identity != nil {
			actor.DisplayName = r.identity.MemberDisplayName(ctx, input.Ctx.TenantID, memberID)
		}
		actors = append(actors, actor)
	}
	return actors, nil
}

// parseMemberIDs 宽容解析成员 ID：JSON 解码后数值（float64）/数字字符串/
// 一层数组均接受，其余形态忽略；负数与 0 视为无效。
func parseMemberIDs(raw any) []uint {
	var out []uint
	appendValue := func(v any) {
		switch t := v.(type) {
		case float64:
			if t > 0 && t == float64(uint64(t)) {
				out = append(out, uint(t))
			}
		case string:
			if id, err := strconv.ParseUint(strings.TrimSpace(t), 10, 64); err == nil && id > 0 {
				out = append(out, uint(id))
			}
		}
	}
	switch t := raw.(type) {
	case []any:
		for _, item := range t {
			appendValue(item)
		}
	default:
		appendValue(raw)
	}
	return out
}

// DepartmentManagerResolver 部门负责人解析器：按部门 ID 解析 leader
// （迁移 000050 IAM 前置能力，禁止在流程域猜测组织语义）。
type DepartmentManagerResolver struct {
	org provider.OrganizationProvider
}

func (r *DepartmentManagerResolver) Type() model.AssigneeType {
	return model.AssigneeTypeDepartmentManager
}

func (r *DepartmentManagerResolver) Resolve(ctx context.Context, input ResolveInput) ([]model.Actor, error) {
	if r.org == nil {
		return nil, &ErrAssigneeNotFound{Type: r.Type(), Message: "组织端口未装配，无法解析部门负责人"}
	}
	actors, err := r.org.ResolveDepartmentManager(ctx, input.Ctx.TenantID, input.Spec.DeptID)
	if err != nil {
		return nil, err
	}
	if len(actors) == 0 {
		return nil, &ErrAssigneeNotFound{Type: r.Type(),
			Message: "部门未设置负责人或负责人无效"}
	}
	return actors, nil
}
