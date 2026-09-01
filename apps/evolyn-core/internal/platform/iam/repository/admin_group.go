package repository

import (
	"context"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
)

type adminGroupRepository struct{ db *gorm.DB }

func newAdminGroupRepository(db *gorm.DB) AdminGroupRepository {
	return &adminGroupRepository{db: db}
}

func (r *adminGroupRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *adminGroupRepository) GetByID(ctx context.Context, id uint) (*model.AdminGroup, error) {
	group := new(model.AdminGroup)
	// 跨租户 ID 由 Callback 过滤为 NotFound，不存在盲读
	if err := r.withContext(ctx).First(group, id).Error; err != nil {
		return nil, err
	}
	return group, nil
}

// ListByTenant 当前租户全部管理组：内置组排最前（前端系统管理组区块），
// 自定义组按创建序排列
func (r *adminGroupRepository) ListByTenant(ctx context.Context) ([]model.AdminGroup, error) {
	groups := make([]model.AdminGroup, 0)
	if err := r.withContext(ctx).Order("built_in DESC, id ASC").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (r *adminGroupRepository) GetByName(ctx context.Context, name string) (*model.AdminGroup, error) {
	group := new(model.AdminGroup)
	if err := r.withContext(ctx).Where("name = ?", name).First(group).Error; err != nil {
		return nil, err
	}
	return group, nil
}

// Create 写入管理组；seed 路径可能无租户上下文，TenantID 由调用方显式赋值
// （与 Callback 回填口径一致）。BuiltIn 为 bool 零值敏感列，Select 显式列出
// 全部列，避免 default tag 吞掉 false（口径同 memberFieldSettingRepository）
func (r *adminGroupRepository) Create(ctx context.Context, group *model.AdminGroup) (*model.AdminGroup, error) {
	if err := r.withContext(ctx).
		Select("TenantID", "Name", "Scope", "BuiltIn", "ScopeConfig", "CreatedAt", "UpdatedAt").
		Create(group).Error; err != nil {
		return nil, err
	}
	return group, nil
}

// UpdateConfig 整体替换 scope_config JSONB（先加载后写，服务层保证租户归属）
func (r *adminGroupRepository) UpdateConfig(ctx context.Context, id uint, config model.AdminGroupScopeConfig) error {
	return r.withContext(ctx).Model(&model.AdminGroup{}).
		Where("id = ?", id).
		Update("scope_config", config).Error
}

func (r *adminGroupRepository) Rename(ctx context.Context, id uint, name string) error {
	return r.withContext(ctx).Model(&model.AdminGroup{}).
		Where("id = ?", id).
		Update("name", name).Error
}

// Delete 软删管理组主表（TenantBaseModel 带 deleted_at）；成员行由服务层
// 在同一事务内显式清理
func (r *adminGroupRepository) Delete(ctx context.Context, id uint) error {
	return r.withContext(ctx).Delete(&model.AdminGroup{}, id).Error
}

func (r *adminGroupRepository) ListMemberIDs(ctx context.Context, groupID uint) ([]uint, error) {
	ids := make([]uint, 0)
	// 单表查询：tn_admin_group_members 含 tenant_id，Callback 注入过滤；
	// 不与主表 join，避免不带表名限定的 tenant_id 条件产生歧义列
	if err := r.withContext(ctx).Model(&model.AdminGroupMember{}).
		Where("admin_group_id = ?", groupID).
		Order("member_id").
		Pluck("member_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// ReplaceMembers 整体替换组成员绑定：先清空再重建（幂等）；原子性由服务层
// 事务保证，此处不开嵌套事务（ctx 携带事务 session 时 ResolveDB 已并入）
func (r *adminGroupRepository) ReplaceMembers(ctx context.Context, groupID uint, tenantID uint, memberIDs []uint) error {
	if err := r.withContext(ctx).
		Where("admin_group_id = ?", groupID).
		Delete(&model.AdminGroupMember{}).Error; err != nil {
		return err
	}
	if len(memberIDs) == 0 {
		return nil
	}
	rows := make([]model.AdminGroupMember, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		// 显式写 TenantID：成员表 Create Callback 依赖 ctx，seed 路径兜底同口径
		rows = append(rows, model.AdminGroupMember{AdminGroupID: groupID, MemberID: memberID, TenantID: tenantID})
	}
	return r.withContext(ctx).Create(&rows).Error
}

// ListGroupIDsOfMember 成员所属管理组 ID 清单（鉴权门/身份聚合用）。
// 单表查询经 Callback 注入租户过滤；调用方再按 ID 批量取组
func (r *adminGroupRepository) ListGroupIDsOfMember(ctx context.Context, memberID uint) ([]uint, error) {
	ids := make([]uint, 0)
	if err := r.withContext(ctx).Model(&model.AdminGroupMember{}).
		Where("member_id = ?", memberID).
		Order("admin_group_id").
		Pluck("admin_group_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// ListByIDs 按 ID 批量取组（配合 ListGroupIDsOfMember 的两段查询）
func (r *adminGroupRepository) ListByIDs(ctx context.Context, ids []uint) ([]model.AdminGroup, error) {
	groups := make([]model.AdminGroup, 0)
	if len(ids) == 0 {
		return groups, nil
	}
	if err := r.withContext(ctx).Where("id IN ?", ids).Order("built_in DESC, id ASC").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

// MemberCounts 各管理组的成员数（列表概要一次取齐，避免逐组计数）
func (r *adminGroupRepository) MemberCounts(ctx context.Context) (map[uint]int, error) {
	rows := make([]struct {
		AdminGroupID uint
		Total        int64
	}, 0)
	if err := r.withContext(ctx).Model(&model.AdminGroupMember{}).
		Select("admin_group_id, COUNT(*) AS total").
		Group("admin_group_id").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	counts := make(map[uint]int, len(rows))
	for _, row := range rows {
		counts[row.AdminGroupID] = int(row.Total)
	}
	return counts, nil
}

// DeleteMembersOfGroup 清空指定组的成员绑定（删组路径同事务调用）
func (r *adminGroupRepository) DeleteMembersOfGroup(ctx context.Context, groupID uint) error {
	return r.withContext(ctx).
		Where("admin_group_id = ?", groupID).
		Delete(&model.AdminGroupMember{}).Error
}

// ListBuiltinMembers 内置系统管理员组成员：由 tenant-admin 角色绑定实时推导
// （不落成员表）。roleID 由服务层先按名解析（Callback 已过滤租户）；
// tn_user_roles 无 tenant_id 列，join 后 Callback 注入的不限定条件落在 users 上
func (r *adminGroupRepository) ListBuiltinMembers(ctx context.Context, roleID uint) ([]model.User, error) {
	users := make([]model.User, 0)
	err := r.withContext(ctx).
		Joins("JOIN tn_user_roles ur ON ur.user_id = tn_users.id").
		Where("ur.role_id = ?", roleID).
		Where("tn_users.status <> ?", model.MemberStatusResigned).
		Preload("Account").Preload(model.DepartmentAssociation).
		Order("tn_users.id").
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

// CountBuiltinMembers 内置组成员数（清空守卫：系统管理员组至少保留一人）
func (r *adminGroupRepository) CountBuiltinMembers(ctx context.Context, roleID uint) (int64, error) {
	var count int64
	err := r.withContext(ctx).Model(&model.User{}).
		Joins("JOIN tn_user_roles ur ON ur.user_id = tn_users.id").
		Where("ur.role_id = ?", roleID).
		Where("tn_users.status <> ?", model.MemberStatusResigned).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// ResolveBuiltinRoleID 按名解析本租户的 tenant-admin 角色 ID（内置组成员
// 读写的代理目标）；ctx 无租户上下文时返回 NotFound（内置组只存在于租户域）
func (r *adminGroupRepository) ResolveBuiltinRoleID(ctx context.Context) (uint, error) {
	role := new(model.Role)
	if err := r.withContext(ctx).Where("name = ?", model.TenantAdminRoleName).First(role).Error; err != nil {
		return 0, err
	}
	return role.ID, nil
}

func (r *adminGroupRepository) Migrate() error {
	if err := r.db.AutoMigrate(&model.AdminGroup{}, &model.AdminGroupMember{}); err != nil {
		return err
	}
	// AutoMigrate 表达不了部分唯一索引与表级约束（FIX-009 口径），幂等 SQL
	// 补齐与 migrations 终态一致
	for _, stmt := range []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_admin_groups_tenant_name ON tn_admin_groups (tenant_id, name) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_tn_admin_groups_tenant_scope ON tn_admin_groups (tenant_id, scope) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_admin_group_member ON tn_admin_group_members (admin_group_id, member_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tn_admin_group_members_member ON tn_admin_group_members (tenant_id, member_id)`,
	} {
		if err := r.db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

// DeleteMembersOfMember 成员离职/删除路径清理其全部管理组绑定（同事务调用）
func (r *adminGroupRepository) DeleteMembersOfMember(ctx context.Context, memberID uint) error {
	return r.withContext(ctx).
		Where("member_id = ?", memberID).
		Delete(&model.AdminGroupMember{}).Error
}
