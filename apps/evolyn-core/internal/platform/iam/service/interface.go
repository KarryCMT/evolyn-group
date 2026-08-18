package service

import (
	"context"

	"evolyn/internal/platform/iam/model"
)

// ctx 约定：触及数据访问的方法统一以 ctx 为首参，由 controller 从
// c.Request.Context() 透传；租户 ID 藏在 ctx 中由仓储层 GORM Callback
// 自动注入，service 不感知租户细节。纯计算方法（Validate/Default/
// ListOperations）不落库，不携带 ctx。

// AccountService 账号服务（登录身份，平台级）
type AccountService interface {
	// Auth 账号密码校验并解析登录成员（TenantCode 可选指定目标租户）
	Auth(ctx context.Context, auser *model.AuthUser) (*model.Account, *model.User, error)
	// Register 注册：创建账号 + 默认租户成员
	Register(ctx context.Context, account *model.Account) (*model.Account, *model.User, error)
	// CreateOAuthAccount OAuth 链路：复用或创建账号，返回默认成员
	CreateOAuthAccount(ctx context.Context, account *model.Account) (*model.Account, *model.User, error)
	Validate(*model.Account) error
	Default(*model.Account)
}

// UserService 成员服务（租户内身份）
type UserService interface {
	List(ctx context.Context) (model.Users, error)
	Get(ctx context.Context, id string) (*model.User, error)
	Update(ctx context.Context, id string, member *model.User) (*model.User, error)
	Delete(ctx context.Context, id string) error
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
