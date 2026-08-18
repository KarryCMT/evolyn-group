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
	userRepo  repository.UserRepository
	groupRepo repository.GroupRepository
}

func NewAuthorizer(userRepo repository.UserRepository, groupRepo repository.GroupRepository) *Authorizer {
	return &Authorizer{
		userRepo:  userRepo,
		groupRepo: groupRepo,
	}
}

// Authorize 鉴权查询走请求 ctx：已认证用户经 TenantMiddleware 注入租户后，
// 组/角色数据同样按租户隔离；未认证请求无租户上下文，查询行为与单租户一致
func (a *Authorizer) Authorize(ctx context.Context, user *model.User, ri *request.RequestInfo) (bool, error) {
	if user == nil || ri == nil {
		return false, nil
	}

	if user.ID == 0 {
		group, err := a.groupRepo.GetGroupByName(ctx, model.UnAuthenticatedGroup)
		if err != nil {
			return false, err
		}
		user.Groups = append(user.Groups, *group)
	} else {
		group, err := a.groupRepo.GetGroupByName(ctx, model.AuthenticatedGroup)
		if err != nil {
			return false, err
		}
		user.Groups = append(user.Groups, *group)
	}

	var err error
	if user.ID != 0 {
		user, err = a.userRepo.GetUserByID(ctx, user.ID)
	}

	if err != nil {
		return false, err
	}

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

	return false, nil
}

// IsClusterAdmin 纯函数：按用户已有角色判定平台管理员，不触碰仓储
func IsClusterAdmin(user *model.User) bool {
	if user == nil || user.Name == "" {
		return false
	}

	roles := make([]model.Role, 0)
	roles = append(roles, user.Roles...)
	for _, g := range user.Groups {
		roles = append(roles, g.Roles...)
	}

	for _, role := range roles {
		if role.Name == "cluster-admin" {
			return true
		}
	}

	return false
}
