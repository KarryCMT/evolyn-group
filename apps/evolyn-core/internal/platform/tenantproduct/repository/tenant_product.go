package repository

import (
	"context"

	"evolyn/internal/contextx"
	"evolyn/internal/infrastructure"
	iammodel "evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/tenantproduct/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type tenantProductRepository struct {
	db *gorm.DB
}

// NewRepository 产品中心域仓储工厂（ADR-007 域模块化）
func NewRepository(db *gorm.DB) Repository {
	return &tenantProductRepository{db: db}
}

// withContext 打开会话并剥离请求租户上下文：本域租户表以显式 tenantID
// 条件定位（租户开通事务/访问判定的 ctx 租户与目标租户不必然一致），
// 平台目录表本就无 tenant_id；ctx 携带事务 session 时仍加入外层事务
func (r *tenantProductRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(contextx.DetachTenant(ctx), r.db)
}

// ---- 平台产品目录 ----

func (r *tenantProductRepository) ListCatalog(ctx context.Context) ([]model.ProductCatalog, error) {
	catalogs := make([]model.ProductCatalog, 0)
	err := r.withContext(ctx).
		Order("sort_order ASC, id ASC").
		Find(&catalogs).Error
	return catalogs, err
}

func (r *tenantProductRepository) GetCatalogByCode(ctx context.Context, code string) (*model.ProductCatalog, error) {
	catalog := new(model.ProductCatalog)
	err := r.withContext(ctx).Where("code = ?", code).First(catalog).Error
	if err != nil {
		return nil, err
	}
	return catalog, nil
}

// ---- 租户产品配置 ----

func (r *tenantProductRepository) ListConfigsByTenant(ctx context.Context, tenantID uint) ([]model.TenantProductConfig, error) {
	configs := make([]model.TenantProductConfig, 0)
	err := r.withContext(ctx).
		Where("tenant_id = ?", tenantID).
		Find(&configs).Error
	return configs, err
}

func (r *tenantProductRepository) getConfig(ctx context.Context, tenantID, productID uint, lock bool) (*model.TenantProductConfig, error) {
	config := new(model.TenantProductConfig)
	query := r.withContext(ctx).Where("tenant_id = ? AND product_id = ?", tenantID, productID)
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.First(config).Error
	if err != nil {
		return nil, err
	}
	return config, nil
}

func (r *tenantProductRepository) GetConfig(ctx context.Context, tenantID, productID uint) (*model.TenantProductConfig, error) {
	return r.getConfig(ctx, tenantID, productID, false)
}

func (r *tenantProductRepository) LockConfig(ctx context.Context, tenantID, productID uint) (*model.TenantProductConfig, error) {
	return r.getConfig(ctx, tenantID, productID, true)
}

func (r *tenantProductRepository) CreateConfig(ctx context.Context, config *model.TenantProductConfig) error {
	return r.withContext(ctx).Create(config).Error
}

// updateWithRevision 乐观更新的公共实现：revision 匹配才写入并同句递增
// （revision+1 由 SQL 表达式完成，避免读改写竞态），0 行影响即版本过期
// （文档 6.2/6.3 的并发语义）
func (r *tenantProductRepository) updateWithRevision(ctx context.Context, id uint, fromRevision int64, updates map[string]interface{}) (bool, error) {
	updates["revision"] = gorm.Expr("revision + 1")
	res := r.withContext(ctx).
		Model(&model.TenantProductConfig{}).
		Where("id = ? AND revision = ?", id, fromRevision).
		Updates(updates)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *tenantProductRepository) UpdateEnabledWithRevision(ctx context.Context, id uint, fromRevision int64, enabled bool) (bool, error) {
	return r.updateWithRevision(ctx, id, fromRevision, map[string]interface{}{
		"enabled": enabled,
	})
}

func (r *tenantProductRepository) UpdateScopeWithRevision(ctx context.Context, id uint, fromRevision int64, scopeMode string) (bool, error) {
	return r.updateWithRevision(ctx, id, fromRevision, map[string]interface{}{
		"scope_mode": scopeMode,
	})
}

// ---- 范围关联 ----

func (r *tenantProductRepository) ListScopeDepartments(ctx context.Context, configID uint) ([]uint, error) {
	rows := make([]model.TenantProductDepartment, 0)
	err := r.withContext(ctx).
		Where("tenant_product_config_id = ?", configID).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.DepartmentID)
	}
	return ids, nil
}

func (r *tenantProductRepository) ListScopeMembers(ctx context.Context, configID uint) ([]uint, error) {
	rows := make([]model.TenantProductMember, 0)
	err := r.withContext(ctx).
		Where("tenant_product_config_id = ?", configID).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.MemberID)
	}
	return ids, nil
}

// ReplaceScope 全量替换范围关联（文档 6.3）：先删全部旧行再插入新清单，
// 保证旧关联不再生效；两清单皆空（mode=all）即清空。tenant_id 冗余自
// 配置行，读取侧用于归属校验
func (r *tenantProductRepository) ReplaceScope(ctx context.Context, config *model.TenantProductConfig, departmentIDs, memberIDs []uint) error {
	if err := r.withContext(ctx).
		Where("tenant_product_config_id = ?", config.ID).
		Delete(&model.TenantProductDepartment{}).Error; err != nil {
		return err
	}
	if err := r.withContext(ctx).
		Where("tenant_product_config_id = ?", config.ID).
		Delete(&model.TenantProductMember{}).Error; err != nil {
		return err
	}
	if len(departmentIDs) > 0 {
		rows := make([]model.TenantProductDepartment, 0, len(departmentIDs))
		for _, id := range departmentIDs {
			rows = append(rows, model.TenantProductDepartment{
				TenantProductConfigID: config.ID,
				DepartmentID:          id,
				TenantID:              config.TenantID,
			})
		}
		if err := r.withContext(ctx).Create(&rows).Error; err != nil {
			return err
		}
	}
	if len(memberIDs) > 0 {
		rows := make([]model.TenantProductMember, 0, len(memberIDs))
		for _, id := range memberIDs {
			rows = append(rows, model.TenantProductMember{
				TenantProductConfigID: config.ID,
				MemberID:              id,
				TenantID:              config.TenantID,
			})
		}
		if err := r.withContext(ctx).Create(&rows).Error; err != nil {
			return err
		}
	}
	return nil
}

// ---- iam 侧读取 ----

func (r *tenantProductRepository) ListTenantDepartments(ctx context.Context, tenantID uint) ([]iammodel.Department, error) {
	departments := make([]iammodel.Department, 0)
	err := r.withContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("id ASC").
		Find(&departments).Error
	return departments, err
}

func (r *tenantProductRepository) GetMember(ctx context.Context, tenantID, memberID uint) (*iammodel.User, error) {
	member := new(iammodel.User)
	err := r.withContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, memberID).
		First(member).Error
	if err != nil {
		return nil, err
	}
	return member, nil
}

func (r *tenantProductRepository) ListMembersByIDs(ctx context.Context, tenantID uint, ids []uint) ([]iammodel.User, error) {
	members := make([]iammodel.User, 0)
	if len(ids) == 0 {
		return members, nil
	}
	err := r.withContext(ctx).
		Where("tenant_id = ? AND id IN ?", tenantID, ids).
		Find(&members).Error
	return members, err
}

// activeMemberCondition 有效成员条件：同租户且状态 active（查询基于 User
// 模型，GORM 软删过滤自动追加 deleted_at IS NULL；跨租户/离职/禁用成员
// 一律不计，文档 3「产品可用成员」口径）
func activeMemberCondition(tenantID uint) (string, []interface{}) {
	return "tenant_id = ? AND status = ?",
		[]interface{}{tenantID, iammodel.MemberStatusActive}
}

func (r *tenantProductRepository) CountActiveMembers(ctx context.Context, tenantID uint) (int64, error) {
	cond, args := activeMemberCondition(tenantID)
	var count int64
	err := r.withContext(ctx).
		Model(&iammodel.User{}).
		Where(cond, args...).
		Count(&count).Error
	return count, err
}

func (r *tenantProductRepository) CountActiveMembersInScope(ctx context.Context, tenantID uint, memberIDs, deptIDs []uint) (int64, error) {
	if len(memberIDs) == 0 && len(deptIDs) == 0 {
		return 0, nil
	}
	cond, args := activeMemberCondition(tenantID)
	query := r.withContext(ctx).
		Model(&iammodel.User{}).
		Where(cond, args...)

	// 去重计数：直接命中成员清单 ∪ 归属选中部门（含子部门展开集）的成员
	query = query.Where(
		"(id IN ? OR EXISTS (SELECT 1 FROM tn_department_users du WHERE du.user_id = tn_users.id AND du.department_id IN ?))",
		memberIDs, deptIDs,
	)
	var count int64
	err := query.Distinct("tn_users.id").Count(&count).Error
	return count, err
}

func (r *tenantProductRepository) MemberInDepartments(ctx context.Context, tenantID, memberID uint, deptIDs []uint) (bool, error) {
	if len(deptIDs) == 0 {
		return false, nil
	}
	// 部门归属与租户绑定校验以 departments 行为准：跨租户/已删部门不会命中
	var count int64
	err := r.withContext(ctx).
		Table("tn_department_users du").
		Joins("JOIN tn_departments d ON d.id = du.department_id").
		Where("du.user_id = ? AND du.department_id IN ? AND d.tenant_id = ? AND d.deleted_at IS NULL", memberID, deptIDs, tenantID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ---- 开发/测试迁移 ----

func (r *tenantProductRepository) Migrate() error {
	if err := r.db.AutoMigrate(
		&model.ProductCatalog{},
		&model.TenantProductConfig{},
		&model.TenantProductDepartment{},
		&model.TenantProductMember{},
	); err != nil {
		return err
	}
	// GORM 标签表达不了部分唯一索引与 CHECK 约束，幂等 SQL 补齐，
	// 使开发库约束与 migrations 终态一致（口径同 iam 域 ensurePartialUniqueIndexes）
	for _, stmt := range []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_pf_product_catalogs_code ON pf_product_catalogs (code) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_product_configs_tenant_product_active ON tn_product_configs (tenant_id, product_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_tn_product_configs_tenant ON tn_product_configs (tenant_id) WHERE deleted_at IS NULL`,
		`ALTER TABLE tn_product_configs DROP CONSTRAINT IF EXISTS ck_tn_product_configs_scope_mode`,
		`ALTER TABLE tn_product_configs ADD CONSTRAINT ck_tn_product_configs_scope_mode CHECK (scope_mode IN ('all', 'partial'))`,
	} {
		if err := r.db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
