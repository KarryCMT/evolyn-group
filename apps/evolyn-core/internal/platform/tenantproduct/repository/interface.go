// Package repository 产品中心域数据访问（ADR-007 域内小三层）：仅做持久化，
// 一律经 infrastructure.ResolveDB 取连接加入 ctx 传播事务（FIX-020/021）。
// 本域租户表带 tenant_id 但由服务层显式传 tenantID 条件定位（口径同
// edition 域）：租户开通事务（NewTenantContext 指向新租户）与访问判定
// 等调用方的 ctx 租户不必然等于目标租户，显式条件消除歧义，不依赖
// GORM 租户 Callback；users/departments/tn_department_users 的读取同理
package repository

import (
	"context"

	iammodel "evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/tenantproduct/model"
)

// Repository 产品中心域仓储
type Repository interface {
	// ---- 平台产品目录（平台级资源，无租户过滤）----

	// ListCatalog 全量目录，按 sort_order/id 升序（卡片展示顺序）
	ListCatalog(ctx context.Context) ([]model.ProductCatalog, error)
	// GetCatalogByCode 按稳定机器码取产品；不存在返回 gorm.ErrRecordNotFound
	GetCatalogByCode(ctx context.Context, code string) (*model.ProductCatalog, error)

	// ---- 租户产品配置（显式 tenantID 条件）----

	// ListConfigsByTenant 租户全部有效产品配置
	ListConfigsByTenant(ctx context.Context, tenantID uint) ([]model.TenantProductConfig, error)
	// GetConfig 取租户某产品的配置；未初始化返回 gorm.ErrRecordNotFound
	GetConfig(ctx context.Context, tenantID, productID uint) (*model.TenantProductConfig, error)
	// LockConfig 事务内 SELECT ... FOR UPDATE 锁定配置行（启停/范围更新前调用）
	LockConfig(ctx context.Context, tenantID, productID uint) (*model.TenantProductConfig, error)
	// CreateConfig 创建配置（TenantID 由调用方显式赋值；开通事务种子使用）
	CreateConfig(ctx context.Context, config *model.TenantProductConfig) error
	// UpdateEnabledWithRevision 乐观更新启停：revision 匹配才生效并 +1，
	// 返回是否命中（false = 版本过期，调用方按冲突拒绝）
	UpdateEnabledWithRevision(ctx context.Context, id uint, fromRevision int64, enabled bool) (bool, error)
	// UpdateScopeWithRevision 乐观更新范围模式：语义同上
	UpdateScopeWithRevision(ctx context.Context, id uint, fromRevision int64, scopeMode string) (bool, error)

	// ---- 范围关联（partial 模式，全量替换）----

	// ListScopeDepartments 配置当前关联的部门 ID 集合（含悬挂引用，由服务层过滤）
	ListScopeDepartments(ctx context.Context, configID uint) ([]uint, error)
	// ListScopeMembers 配置当前关联的成员 ID 集合（含悬挂引用，由服务层过滤）
	ListScopeMembers(ctx context.Context, configID uint) ([]uint, error)
	// ReplaceScope 全量替换范围关联（同事务先删后插；mode=all 时两清单为空
	// 即清空全部关联）。调用方必须已开启事务并锁定配置行
	ReplaceScope(ctx context.Context, config *model.TenantProductConfig, departmentIDs, memberIDs []uint) error

	// ---- iam 侧读取（显式 tenantID 条件；供范围校验、成员计数与访问判定）----

	// ListTenantDepartments 租户全量有效部门（未软删；含停用项由服务层按
	// status 过滤），用于部门归属校验与子部门递归展开
	ListTenantDepartments(ctx context.Context, tenantID uint) ([]iammodel.Department, error)
	// GetMember 取租户内成员（软删/跨租户表现为不存在）；状态由调用方判定
	GetMember(ctx context.Context, tenantID, memberID uint) (*iammodel.User, error)
	// ListMembersByIDs 按 ID 集合取租户成员（含状态，供校验与标签）
	ListMembersByIDs(ctx context.Context, tenantID uint, ids []uint) ([]iammodel.User, error)
	// CountActiveMembers 租户有效（active 且未软删）成员总数
	CountActiveMembers(ctx context.Context, tenantID uint) (int64, error)
	// CountActiveMembersInScope 范围内有效成员数：直接命中 memberIDs 或归属
	// deptIDs（含子部门展开后的集合）的 active 成员去重计数；两清单皆空返回 0
	CountActiveMembersInScope(ctx context.Context, tenantID uint, memberIDs, deptIDs []uint) (int64, error)
	// MemberInDepartments 成员是否直接归属 deptIDs 中任一部门
	// （tn_department_users 关联命中；deptIDs 应为展开后的有效部门集合）
	MemberInDepartments(ctx context.Context, tenantID, memberID uint, deptIDs []uint) (bool, error)

	// Migrate 开发/测试 AutoMigrate 路径（FIX-009：生产只走 SQL 迁移）
	Migrate() error
}
