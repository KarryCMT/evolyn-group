package repository

import (
	"context"

	"evolyn/internal/platform/iam/model"

	"gorm.io/gorm/clause"
)

// ctx 约定：数据方法统一以 ctx 为首参，由调用链一路透传请求 context。
// ctx 中的租户 ID 经 RegisterTenantCallbacks 注册的 GORM Callback 自动注入
// 过滤/回填（架构文档 26.3），Repository 层禁止手写租户条件；
// 启动期路径（Migrate/Init/Seed）使用 context.Background()，无租户上下文即无副作用。

type UserRepository interface {
	GetUserByID(ctx context.Context, id uint) (*model.User, error)
	GetUserByAuthID(ctx context.Context, authType, authID string) (*model.User, error)
	GetUserByName(ctx context.Context, name string) (*model.User, error)
	List(ctx context.Context) (model.Users, error)
	Create(ctx context.Context, user *model.User) (*model.User, error)
	Update(ctx context.Context, user *model.User) (*model.User, error)
	Delete(ctx context.Context, user *model.User) error
	AddAuthInfo(ctx context.Context, authInfo *model.AuthInfo) error
	DelAuthInfo(ctx context.Context, authInfo *model.AuthInfo) error
	AddRole(ctx context.Context, role *model.Role, user *model.User) error
	DelRole(ctx context.Context, role *model.Role, user *model.User) error
	GetGroups(ctx context.Context, user *model.User) ([]model.Group, error)
	Migrate() error
}

type GroupRepository interface {
	GetGroupByID(ctx context.Context, id uint) (*model.Group, error)
	GetGroupByName(ctx context.Context, name string) (*model.Group, error)
	List(ctx context.Context) ([]model.Group, error)
	Create(ctx context.Context, user *model.User, group *model.Group) (*model.Group, error)
	CreateGroups(ctx context.Context, groups []model.Group, conds ...clause.Expression) error
	Update(ctx context.Context, group *model.Group) (*model.Group, error)
	Delete(ctx context.Context, id uint) error
	GetUsers(ctx context.Context, group *model.Group) (model.Users, error)
	AddUser(ctx context.Context, user *model.User, group *model.Group) error
	DelUser(ctx context.Context, user *model.User, group *model.Group) error
	AddRole(ctx context.Context, role *model.Role, group *model.Group) error
	DelRole(ctx context.Context, role *model.Role, group *model.Group) error
	RoleBinding(ctx context.Context, role *model.Role, group *model.Group) error
	Migrate() error
}

type RBACRepository interface {
	List(ctx context.Context) ([]model.Role, error)
	ListResources(ctx context.Context) ([]model.Resource, error)
	Create(ctx context.Context, role *model.Role) (*model.Role, error)
	CreateResource(ctx context.Context, resource *model.Resource) (*model.Resource, error)
	CreateResources(ctx context.Context, resources []model.Resource, conds ...clause.Expression) error
	GetRoleByID(ctx context.Context, id int) (*model.Role, error)
	GetResource(ctx context.Context, id int) (*model.Resource, error)
	GetRoleByName(ctx context.Context, name string) (*model.Role, error)
	Update(ctx context.Context, role *model.Role) (*model.Role, error)
	Delete(ctx context.Context, id uint) error
	DeleteResource(ctx context.Context, id uint) error
	Migrate() error
}
