package service

import (
	"context"

	"evolyn/internal/platform/iam/model"
)

// ctx 约定：触及数据访问的方法统一以 ctx 为首参，由 controller 从
// c.Request.Context() 透传；租户 ID 藏在 ctx 中由仓储层 GORM Callback
// 自动注入，service 不感知租户细节。纯计算方法（Validate/Default/
// ListOperations）不落库，不携带 ctx。

type UserService interface {
	List(ctx context.Context) (model.Users, error)
	Create(ctx context.Context, user *model.User) (*model.User, error)
	Get(ctx context.Context, id string) (*model.User, error)
	CreateOAuthUser(ctx context.Context, user *model.User) (*model.User, error)
	Update(ctx context.Context, id string, user *model.User) (*model.User, error)
	Delete(ctx context.Context, id string) error
	Validate(*model.User) error
	Auth(ctx context.Context, auser *model.AuthUser) (*model.User, error)
	Default(*model.User)
	GetGroups(ctx context.Context, id string) ([]model.Group, error)
	AddRole(ctx context.Context, id, rid string) error
	DelRole(ctx context.Context, id, rid string) error
}

type GroupService interface {
	List(ctx context.Context) ([]model.Group, error)
	Create(ctx context.Context, user *model.User, group *model.Group) (*model.Group, error)
	Get(ctx context.Context, id string) (*model.Group, error)
	Update(ctx context.Context, id string, group *model.Group) (*model.Group, error)
	Delete(ctx context.Context, id string) error
	GetUsers(ctx context.Context, gid string) (model.Users, error)
	AddUser(ctx context.Context, user *model.User, gid string) error
	DelUser(ctx context.Context, user *model.User, gid string) error
	AddRole(ctx context.Context, id, rid string) error
	DelRole(ctx context.Context, id, rid string) error
}

type RBACService interface {
	List(ctx context.Context) ([]model.Role, error)
	Create(ctx context.Context, role *model.Role) (*model.Role, error)
	Get(ctx context.Context, id string) (*model.Role, error)
	Update(ctx context.Context, id string, role *model.Role) (*model.Role, error)
	Delete(ctx context.Context, id string) error
	ListResources(ctx context.Context) ([]model.Resource, error)
	ListOperations() ([]model.Operation, error)
}
