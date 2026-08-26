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

// UserRepository 成员（users 表）数据访问。登录身份见 AccountRepository（ADR-006）
type UserRepository interface {
	GetUserByID(ctx context.Context, id uint) (*model.User, error)
	ListByAccount(ctx context.Context, accountID uint) (model.Users, error)
	GetByAccountAndTenant(ctx context.Context, accountID, tenantID uint) (*model.User, error)
	List(ctx context.Context) (model.Users, error)
	ListPage(ctx context.Context, query model.MemberListQuery) (model.Users, int64, error)
	// CountByTenant 指定租户的有效成员数（配额执行用，FIX-011）。
	// 显式按租户计数：调用方（配额/运营路径）可能无租户上下文
	CountByTenant(ctx context.Context, tenantID uint) (int64, error)
	Create(ctx context.Context, member *model.User) (*model.User, error)
	Update(ctx context.Context, member *model.User) (*model.User, error)
	UpdateStatus(ctx context.Context, member *model.User) (*model.User, error)
	Delete(ctx context.Context, member *model.User) error
	AddRole(ctx context.Context, role *model.Role, user *model.User) error
	DelRole(ctx context.Context, role *model.Role, user *model.User) error
	GetGroups(ctx context.Context, user *model.User) ([]model.Group, error)
	Migrate() error
}

// MemberInvitationRepository 管理待接受成员邀请和租户公开邀请链接。
// 两类记录均为租户内资源，租户过滤统一由 GORM Callback 注入。
type MemberInvitationRepository interface {
	Create(ctx context.Context, invitation *model.MemberInvitation) (*model.MemberInvitation, error)
	CreateBatch(ctx context.Context, invitations []model.MemberInvitation) error
	GetPublicLink(ctx context.Context) (*model.TenantPublicInvitationLink, error)
	GetPublicLinkByToken(ctx context.Context, token string) (*model.TenantPublicInvitationLink, error)
	CreatePublicLink(ctx context.Context, link *model.TenantPublicInvitationLink) (*model.TenantPublicInvitationLink, error)
	UpdatePublicLink(ctx context.Context, link *model.TenantPublicInvitationLink) (*model.TenantPublicInvitationLink, error)
	Migrate() error
}

// AccountRepository 平台账号（accounts 表）数据访问；平台级表无租户上下文
type AccountRepository interface {
	GetByID(ctx context.Context, id uint) (*model.Account, error)
	GetByName(ctx context.Context, name string) (*model.Account, error)
	GetByPhone(ctx context.Context, phone string) (*model.Account, error)
	GetByAuthID(ctx context.Context, authType, authID string) (*model.Account, error)
	List(ctx context.Context) ([]model.Account, error)
	Create(ctx context.Context, account *model.Account) (*model.Account, error)
	Update(ctx context.Context, account *model.Account) (*model.Account, error)
	AddAuthInfo(ctx context.Context, authInfo *model.AuthInfo) error
	DelAuthInfo(ctx context.Context, authInfo *model.AuthInfo) error
	// UpdatePassword 密码重置（已散列值，散列在服务层完成），同语句落 password_initialized
	UpdatePassword(ctx context.Context, id uint, hashed string, initialized bool) error
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

// DepartmentRepository 部门（租户内组织架构）数据访问
type DepartmentRepository interface {
	List(ctx context.Context) ([]model.Department, error)
	GetByID(ctx context.Context, id uint) (*model.Department, error)
	Create(ctx context.Context, dept *model.Department) (*model.Department, error)
	Update(ctx context.Context, dept *model.Department) (*model.Department, error)
	Delete(ctx context.Context, id uint) error
	// SetMemberDepartments 整体替换成员的部门归属（多部门）
	SetMemberDepartments(ctx context.Context, member *model.User, departmentIDs []uint) error
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
	// ClearRoleGroup 在删除展示分组前清除其中角色的分组归属，角色本身不删除。
	ClearRoleGroup(ctx context.Context, groupID uint) error
	Delete(ctx context.Context, id uint) error
	DeleteResource(ctx context.Context, id uint) error
	Migrate() error
}

// RoleGroupRepository 管理内部组织页中的角色展示分组。它与权限分组 Group
// 独立，避免角色分类意外影响成员的权限继承。
type RoleGroupRepository interface {
	List(ctx context.Context) ([]model.RoleGroup, error)
	GetByID(ctx context.Context, id uint) (*model.RoleGroup, error)
	GetByName(ctx context.Context, name string) (*model.RoleGroup, error)
	Create(ctx context.Context, group *model.RoleGroup) (*model.RoleGroup, error)
	Update(ctx context.Context, group *model.RoleGroup) (*model.RoleGroup, error)
	Delete(ctx context.Context, group *model.RoleGroup) error
	Migrate() error
}
