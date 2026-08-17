package authorization

import (
	"context"

	"evolyn/internal/model"

	"evolyn/internal/repository"
	"evolyn/pkg/utils/request"
)

var store repository.Repository

func InitAuthorization(repository repository.Repository) error {
	store = repository
	return nil
}

// Authorize 鉴权查询走请求 ctx：已认证用户经 TenantMiddleware 注入租户后，
// 组/角色数据同样按租户隔离；未认证请求无租户上下文，查询行为与单租户一致
func Authorize(ctx context.Context, user *model.User, ri *request.RequestInfo) (bool, error) {
	if user == nil || ri == nil {
		return false, nil
	}

	if user.ID == 0 {
		group, err := store.Group().GetGroupByName(ctx, model.UnAuthenticatedGroup)
		if err != nil {
			return false, err
		}
		user.Groups = append(user.Groups, *group)
	} else {
		group, err := store.Group().GetGroupByName(ctx, model.AuthenticatedGroup)
		if err != nil {
			return false, err
		}
		user.Groups = append(user.Groups, *group)
	}

	var err error
	if user.ID != 0 {
		user, err = store.User().GetUserByID(ctx, user.ID)
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
