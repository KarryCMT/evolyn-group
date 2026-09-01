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
	// GetMemberDetail 按 ID 加载成员及其账号/部门/角色（成员档案聚合读取：
	// 档案值跨 users/accounts/关系表，需一次带全关联，避免服务层拼装多跳查询）
	GetMemberDetail(ctx context.Context, id uint) (*model.User, error)
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
	// PurgeByAccount 物理清理账号在所有租户中的成员及关系表记录。
	PurgeByAccount(ctx context.Context, accountID uint) error
	Migrate() error
}

// MemberInvitationRepository 管理待接受成员邀请和租户公开邀请链接。
// 两类记录均为租户内资源，租户过滤统一由 GORM Callback 注入。
type MemberInvitationRepository interface {
	Create(ctx context.Context, invitation *model.MemberInvitation) (*model.MemberInvitation, error)
	CreateBatch(ctx context.Context, invitations []model.MemberInvitation) error
	// GetByToken 按单人邀请 token 全局定位邀请（接受链路尚未进入目标租户，
	// 需剥离调用方租户上下文）
	GetByToken(ctx context.Context, token string) (*model.MemberInvitation, error)
	// MarkAccepted 邀请状态流转为 accepted（租户内按 ID 更新，白名单列）
	MarkAccepted(ctx context.Context, invitation *model.MemberInvitation) error
	GetPublicLink(ctx context.Context) (*model.TenantPublicInvitationLink, error)
	GetPublicLinkByToken(ctx context.Context, token string) (*model.TenantPublicInvitationLink, error)
	CreatePublicLink(ctx context.Context, link *model.TenantPublicInvitationLink) (*model.TenantPublicInvitationLink, error)
	UpdatePublicLink(ctx context.Context, link *model.TenantPublicInvitationLink) (*model.TenantPublicInvitationLink, error)
	Migrate() error
}

// MemberFieldSettingRepository 租户级成员字段显示策略数据访问。
// field_key 的合法性由 Service 依字段注册表校验，数据库不做枚举约束（便于
// 后续增加预置字段，文档 4.1）
type MemberFieldSettingRepository interface {
	// ListByTenant 当前租户全部有效配置行（租户过滤由 Callback 注入）
	ListByTenant(ctx context.Context) ([]model.MemberFieldSetting, error)
	// GetByFieldKey 当前租户指定字段的配置行
	GetByFieldKey(ctx context.Context, fieldKey string) (*model.MemberFieldSetting, error)
	// CreateBatch 批量写入配置行（seed 路径：租户开通/读取兜底补齐）
	CreateBatch(ctx context.Context, settings []model.MemberFieldSetting) error
	// UpdateWithRevision 按 (id, revision) 乐观锁更新配置值：revision 匹配
	// 才落库并整体 +1，返回是否命中（false 即配置已被其他管理员修改）
	UpdateWithRevision(ctx context.Context, id uint, revision int64, updates map[string]interface{}) (bool, error)
	// BumpRevision 将当前租户全部配置行 revision +1：PATCH 单字段成功后
	// 同步推进整页版本号，使顶层 revision 成为真正的租户配置快照版本
	BumpRevision(ctx context.Context) error
	Migrate() error
}

// MemberProfileRepository 正式成员扩展档案数据访问。
// (tenant_id, member_id) 有效记录唯一，写入侧经 Upsert 保证
type MemberProfileRepository interface {
	// GetByMember 当前租户指定成员的档案；无档案时返回 NotFound（调用方
	// 以空档案语义兜底，不视为业务错误）
	GetByMember(ctx context.Context, memberID uint) (*model.MemberProfile, error)
	// Upsert 写入档案：已有记录按显式列白名单更新（identifier 与 attributes
	// 整体替换），无记录插入；attributes 只允许注册表扩展 key（服务层校验）
	Upsert(ctx context.Context, profile *model.MemberProfile) (*model.MemberProfile, error)
	// IdentifierExists 编号是否已被同租户其他有效成员占用（唯一性服务层校验，
	// 数据库部分唯一索引兜底）
	IdentifierExists(ctx context.Context, identifier string, excludeMemberID uint) (bool, error)
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
	// Purge 物理删除已无创建人归属的账号，并清理其第三方凭证。
	Purge(ctx context.Context, accountID uint) error
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

// AdminGroupRepository 管理组（权限中心-管理员模块）数据访问。
// 内置系统管理员组（built_in）的成员不落 tn_admin_group_members：经
// ResolveBuiltinRoleID + ListBuiltinMembers/CountBuiltinMembers 由
// tenant-admin 角色绑定实时推导，与租户域 seed 同一事实源。
// 注：tn_admin_group_members 含 tenant_id 列，与主表 join 会使 Callback 注入的
// 不限定租户条件产生歧义列——按成员取组一律走两段查询（ID 清单 + 批量取组）
type AdminGroupRepository interface {
	GetByID(ctx context.Context, id uint) (*model.AdminGroup, error)
	ListByTenant(ctx context.Context) ([]model.AdminGroup, error)
	GetByName(ctx context.Context, name string) (*model.AdminGroup, error)
	Create(ctx context.Context, group *model.AdminGroup) (*model.AdminGroup, error)
	UpdateConfig(ctx context.Context, id uint, config model.AdminGroupScopeConfig) error
	Rename(ctx context.Context, id uint, name string) error
	Delete(ctx context.Context, id uint) error
	ListMemberIDs(ctx context.Context, groupID uint) ([]uint, error)
	// ReplaceMembers 整体替换组成员绑定（tenantID 显式落行，seed 路径兜底）
	ReplaceMembers(ctx context.Context, groupID, tenantID uint, memberIDs []uint) error
	ListGroupIDsOfMember(ctx context.Context, memberID uint) ([]uint, error)
	ListByIDs(ctx context.Context, ids []uint) ([]model.AdminGroup, error)
	// MemberCounts 各组成员数（列表概要一次取齐）
	MemberCounts(ctx context.Context) (map[uint]int, error)
	DeleteMembersOfGroup(ctx context.Context, groupID uint) error
	// DeleteMembersOfMember 成员离职/删除路径清理其全部管理组绑定
	DeleteMembersOfMember(ctx context.Context, memberID uint) error
	// 内置组代理（成员读写经 tn_user_roles 的 tenant-admin 绑定）
	ResolveBuiltinRoleID(ctx context.Context) (uint, error)
	ListBuiltinMembers(ctx context.Context, roleID uint) ([]model.User, error)
	CountBuiltinMembers(ctx context.Context, roleID uint) (int64, error)
	Migrate() error
}
