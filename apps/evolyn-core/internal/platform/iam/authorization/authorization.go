package authorization

import (
	"context"

	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"
	"evolyn/internal/utils/request"
)

// Authorizer 鉴权器：显式持有 iam 仓储（拆除原全局单例，ADR-007/P0-4）。
// M1 后 Casbin 接入时在本结构内替换实现，调用方（中间件）无感。
type Authorizer struct {
	userRepo    repository.UserRepository
	groupRepo   repository.GroupRepository
	adminGroups repository.AdminGroupRepository
}

func NewAuthorizer(userRepo repository.UserRepository, groupRepo repository.GroupRepository, adminGroups repository.AdminGroupRepository) *Authorizer {
	return &Authorizer{
		userRepo:    userRepo,
		groupRepo:   groupRepo,
		adminGroups: adminGroups,
	}
}

// Authorize 鉴权查询走请求 ctx：已认证用户经 TenantMiddleware 注入租户后，
// 组/角色数据同样按租户隔离；未认证请求无租户上下文，查询行为与单租户一致
func (a *Authorizer) Authorize(ctx context.Context, user *model.User, ri *request.RequestInfo) (bool, error) {
	if user == nil || ri == nil {
		return false, nil
	}

	// 已认证用户先按 ID 重载完整成员关系（自身组/角色），再追加系统组——
	// 顺序不可颠倒：GetUserByID 返回的新对象会覆盖 user.Groups，若先追加
	// 系统组会被重载丢弃，导致无显式绑定的新成员（默认租户注册）鉴权恒假
	var err error
	if user.ID != 0 {
		user, err = a.userRepo.GetUserByID(ctx, user.ID)
		if err != nil {
			return false, err
		}
	}

	groupName := model.UnAuthenticatedGroup
	if user.ID != 0 {
		groupName = model.AuthenticatedGroup
	}
	group, err := a.groupRepo.GetGroupByName(ctx, groupName)
	if err != nil {
		return false, err
	}
	user.Groups = append(user.Groups, *group)

	roles := make([]model.Role, 0)
	roles = append(roles, user.Roles...)
	for _, g := range user.Groups {
		roles = append(roles, g.Roles...)
	}

	for _, role := range roles {
		for _, rule := range role.Rules {
			if (rule.Resource == model.All || rule.Resource == ri.Resource) && rule.Operation.Contain(ri.Verb) {
				return true, nil
			}
		}
	}

	// RBAC 未命中时回落管理组范围裁决（权限中心-管理员模块）：通讯录管理组/
	// 普通管理组的委托授权。已认证（非内置组）成员才可能持有管理组身份
	if user.ID != 0 {
		return a.authorizeByAdminGroup(ctx, user, ri)
	}
	return false, nil
}

// authorizeByAdminGroup 管理组范围裁决（保守门，权限中心-管理员模块一期）。
//
// 管理组授权是「带数据范围的委托」，不是 resource:verb 布尔：此处只放行
// 全量范围（mode=all / 全部应用）的能力；partial 清单范围的可见性与数据过滤
// 需要各域 Service 查询侧配合落地，门层放行会造成越权（partial 组成员可操作
// 清单外数据），故一律拒绝——数据范围执行批落地后按资源放开。
//
// 范围语义按组 scope 严格区分：
//   - system 组（通讯录管理组）：Department/Role 区块授予成员/部门/角色管理；
//   - application 组（普通管理组）：Department/Role 是应用使用权的分发范围，
//     绝不授予部门/角色管理；Application 区块授予应用编辑与增删。
//
// admin-groups 资源永不经管理组授予（防自我扩权），未列资源同样拒绝。
func (a *Authorizer) authorizeByAdminGroup(ctx context.Context, user *model.User, ri *request.RequestInfo) (bool, error) {
	switch ri.Resource {
	case model.MemberResource, "departments", model.RoleResource, "applications":
	default:
		return false, nil
	}

	ids, err := a.adminGroups.ListGroupIDsOfMember(ctx, user.ID)
	if err != nil {
		return false, err
	}
	groups, err := a.adminGroups.ListByIDs(ctx, ids)
	if err != nil {
		return false, err
	}

	// 并集聚合：成员属于多个管理组时能力只增不减
	var (
		deptManageAll   bool // system 组：部门范围全量 → 成员/部门管理
		roleVisibleAll  bool // system 组：角色可见（读）
		roleManageAll   bool // system 组：角色可管理（写）
		appCreateDelete bool // application 组：可添加/删除应用
		appEditAll      bool // application 组：全部应用可编辑
	)
	for _, group := range groups {
		if group.BuiltIn {
			continue // 内置组能力由 tenant-admin 角色经 RBAC 覆盖，无需重复判定
		}
		config := group.ScopeConfig
		switch group.Scope {
		case model.AdminGroupScopeSystem:
			if config.Department != nil && config.Department.Enabled && config.Department.Mode == model.AdminScopeAll {
				deptManageAll = true
			}
			if config.Role != nil && config.Role.Mode == model.AdminScopeAll {
				if config.Role.Visible {
					roleVisibleAll = true
				}
				if config.Role.Manage {
					roleManageAll = true
				}
			}
		case model.AdminGroupScopeApplication:
			if config.Application != nil {
				if config.Application.Manage {
					appCreateDelete = true
				}
				if config.Application.AllApplications {
					appEditAll = true
				}
			}
		}
	}

	switch ri.Resource {
	case model.MemberResource, "departments":
		// 成员管理与部门管理同属通讯录委托（可见/可管理为同一开关）
		return deptManageAll, nil
	case model.RoleResource:
		if model.ViewOperationSet.Has(ri.Verb) {
			return roleVisibleAll, nil
		}
		return roleManageAll, nil
	case "applications":
		if ri.Verb == request.CreateOperation || ri.Verb == request.DeleteOperation {
			return appCreateDelete, nil
		}
		// 应用读（view）由 authenticated 基线覆盖，此处只裁决编辑类动词
		return appEditAll, nil
	}
	return false, nil
}

// IsClusterAdmin 纯函数：按用户已有角色判定平台管理员，不触碰仓储
func IsClusterAdmin(user *model.User) bool {
	if user == nil || user.ID == 0 {
		return false
	}

	roles := make([]model.Role, 0)
	roles = append(roles, user.Roles...)
	for _, g := range user.Groups {
		roles = append(roles, g.Roles...)
	}

	for _, role := range roles {
		if role.Name == model.ClusterAdminRole {
			return true
		}
	}
	return false
}

// IsTenantAdmin 纯函数：按用户已有角色判定租户管理员（内置系统管理员组身份）
func IsTenantAdmin(user *model.User) bool {
	if user == nil || user.ID == 0 {
		return false
	}

	roles := make([]model.Role, 0)
	roles = append(roles, user.Roles...)
	for _, g := range user.Groups {
		roles = append(roles, g.Roles...)
	}

	for _, role := range roles {
		if role.Name == model.TenantAdminRoleName {
			return true
		}
	}
	return false
}
