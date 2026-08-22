package service

import (
	"context"

	"evolyn/internal/contextx"
	"evolyn/internal/platform/iam/authorization"
	iammodel "evolyn/internal/platform/iam/model"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ApplicationAccessEvaluator 应用级访问判定（§9.1）：capabilities 计算与
// Service 内部写路径的权限复核统一入口——Service 不直接依赖 HTTP 中间件
// 鉴权，内部调用路径（后续任务/开放接口等）经本判定同样受控。
// M2-A 实现为「与 AuthorizationMiddleware 同源的租户级 RBAC」；应用级
// 范围授权（owner 协作组、应用管理员分组）落地后替换实现即可
type ApplicationAccessEvaluator interface {
	// Permissions 返回成员在当前租户的有效权限集（键 "resource:verb"），
	// 必须与鉴权中间件判定同源：按 member.ID 在当前租户上下文内重载
	// 成员与角色，再合并 authenticated 系统组——不信任调用方传入的
	// TenantID/Roles 快照（可被伪造或已过期）
	Permissions(ctx context.Context, member *iammodel.User) map[string]bool
}

// memberLoader 按 ID 加载成员（预加载角色/分组）。ctx 携带租户时经
// GORM Callback 过滤，跨租户 ID 直接 NotFound——与 AuthorizationMiddleware
// 的鉴权加载同一仓储方法，天然同源（含缓存行为）
type memberLoader interface {
	GetUserByID(ctx context.Context, id uint) (*iammodel.User, error)
}

// groupRoleLoader 最小仓储面：按名加载系统组（预加载其角色）
type groupRoleLoader interface {
	GetGroupByName(ctx context.Context, name string) (*iammodel.Group, error)
}

// rbacAccessEvaluator M2-A 判定实现：先按 ID 重载成员（伪造/陈旧快照
// 无效），显式角色/分组的权限集再合并 authenticated 系统组角色（基线
// 种子给全体成员 applications:view 等），口径对齐 AuthorizationMiddleware
type rbacAccessEvaluator struct {
	members memberLoader
	groups  groupRoleLoader
}

func NewRBACAccessEvaluator(members memberLoader, groups groupRoleLoader) ApplicationAccessEvaluator {
	return &rbacAccessEvaluator{members: members, groups: groups}
}

func (e *rbacAccessEvaluator) Permissions(ctx context.Context, member *iammodel.User) map[string]bool {
	// 身份与租户前提：无成员标识或无租户上下文，权限集为空
	if member == nil || member.ID == 0 {
		return map[string]bool{}
	}
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return map[string]bool{}
	}

	// 不信任传入快照：按 ID 在当前租户上下文内重载成员与角色。
	// 调用方构造的「TenantID=目标租户 + applications:* 角色」不会生效——
	// 重载只认数据库里的真实角色；ID 不属于本租户时重载即 NotFound
	reloaded, err := e.members.GetUserByID(ctx, member.ID)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			logrus.Warnf("application access evaluator: load member %d failed: %v", member.ID, err)
		}
		return map[string]bool{}
	}
	// 双重防御：显式比对重载结果的租户归属（正常路径已被 Callback
	// 过滤保证，此处兜底无租户过滤的加载实现）
	if reloaded == nil || reloaded.TenantID != tenantID {
		return map[string]bool{}
	}

	perms := authorization.PermissionsOf(reloaded)

	// 中间件鉴权时会追加 authenticated 系统组；此处同源合并。
	// 系统组读取失败只降级为显式角色集并告警（写路径复核随之更严格，
	// 不会因读取失败放大权限）
	group, err := e.groups.GetGroupByName(ctx, iammodel.AuthenticatedGroup)
	if err != nil {
		logrus.Warnf("application access evaluator: load group %s failed: %v", iammodel.AuthenticatedGroup, err)
		return perms
	}
	if group != nil {
		for key, allowed := range authorization.PermissionsOf(&iammodel.User{Roles: group.Roles}) {
			if allowed {
				perms[key] = true
			}
		}
	}
	return perms
}
