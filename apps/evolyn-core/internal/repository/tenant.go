package repository

import (
	"context"

	"evolyn/internal/database"
	"evolyn/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type tenantRepository struct {
	db  *gorm.DB
	rdb *database.RedisDB
}

func newTenantRepository(db *gorm.DB, rdb *database.RedisDB) TenantRepository {
	return &tenantRepository{
		db:  db,
		rdb: rdb,
	}
}

// withContext 以请求 ctx 打开新会话；tenants 表自身无 tenant_id 列，
// Callback 对其无副作用，保持签名统一只为与其余仓储一致
func (t *tenantRepository) withContext(ctx context.Context) *gorm.DB {
	return t.db.WithContext(ctx)
}

func (t *tenantRepository) GetByID(ctx context.Context, id uint) (*model.Tenant, error) {
	tenant := new(model.Tenant)
	if err := t.withContext(ctx).First(tenant, id).Error; err != nil {
		return nil, err
	}
	return tenant, nil
}

func (t *tenantRepository) GetByCode(ctx context.Context, code string) (*model.Tenant, error) {
	tenant := new(model.Tenant)
	if err := t.withContext(ctx).Where("code = ?", code).First(tenant).Error; err != nil {
		return nil, err
	}
	return tenant, nil
}

func (t *tenantRepository) Migrate() error {
	return t.db.AutoMigrate(&model.Tenant{})
}

// SeedDefaultTenant 确保默认租户存在，并将所有租户列为 0/NULL 的存量行归属到默认租户。
// 必须在其余模型 AutoMigrate（tenant_id 列已带 default:1 回填）之后调用：
// 若默认租户实际 ID 不是 1（如表经历过清理），此处负责把 1 修正为真实 ID。
func (t *tenantRepository) SeedDefaultTenant() error {
	tenant := &model.Tenant{
		Code:   model.DefaultTenantCode,
		Name:   "默认租户",
		Plan:   "free",
		Status: model.TenantActive,
	}
	if err := t.db.Clauses(clause.OnConflict{DoNothing: true}).Create(tenant).Error; err != nil {
		return err
	}

	created := new(model.Tenant)
	if err := t.db.Where("code = ?", model.DefaultTenantCode).First(created).Error; err != nil {
		return err
	}

	if created.ID == model.DefaultTenantID {
		return nil
	}

	// 默认租户 ID 与列默认值不一致时，统一把存量数据修正到真实 ID（PostgreSQL 不支持多表 UPDATE，逐表执行）
	for _, table := range []string{"users", "groups", "roles"} {
		if err := t.db.Exec(
			"UPDATE "+table+" SET tenant_id = ? WHERE tenant_id = ? OR tenant_id IS NULL",
			created.ID, model.DefaultTenantID,
		).Error; err != nil {
			return err
		}
	}

	return nil
}
