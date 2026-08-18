package repository

import (
	"context"
	"time"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/tenant/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type tenantRepository struct {
	db  *gorm.DB
	rdb *infrastructure.RedisDB
}

// TenantRepository 租户域数据访问；管理面 CRUD 随 P3 运营域接口补充
type TenantRepository interface {
	GetByID(ctx context.Context, id uint) (*model.Tenant, error)
	GetByIDs(ctx context.Context, ids []uint) ([]model.Tenant, error)
	GetByCode(ctx context.Context, code string) (*model.Tenant, error)
	SeedDefaultTenant() error
	Migrate() error

	// 运营面 CRUD（P3-1）：内部剥离租户上下文，仅限 /platform 域调用
	List(ctx context.Context) ([]model.Tenant, error)
	Create(ctx context.Context, tenant *model.Tenant) (*model.Tenant, error)
	Update(ctx context.Context, tenant *model.Tenant) (*model.Tenant, error)
	UpdateStatus(ctx context.Context, id uint, status string) error
}

// NewRepository 租户域仓储工厂（ADR-007 域模块化）
func NewRepository(db *gorm.DB, rdb *infrastructure.RedisDB) TenantRepository {
	return &tenantRepository{
		db:  db,
		rdb: rdb,
	}
}

// withContext 以请求 ctx 打开新会话。注意：tenants 表因内嵌 BaseModel
// 同样带有 tenant_id 列（default:1），存在租户上下文时 Callback 会对
// tenants 查询照常追加过滤。当前租户读写均发生在启动期/登录前（无租户
// 上下文）；P1 运营面租户 CRUD 需以无租户上下文执行查询，避免自我过滤
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

// GetByIDs 批量取租户（成员关系列表组装用）；ids 为空返回空集
func (t *tenantRepository) GetByIDs(ctx context.Context, ids []uint) ([]model.Tenant, error) {
	tenants := make([]model.Tenant, 0)
	if len(ids) == 0 {
		return tenants, nil
	}
	if err := t.withContext(ctx).Where("id IN ?", ids).Find(&tenants).Error; err != nil {
		return nil, err
	}
	return tenants, nil
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

// platformCtx 运营面专用：刻意剥离请求携带的租户上下文（tenants 表带
// tenant_id 列，运营者自身会话的租户上下文会造成自我过滤，见 withContext 注释），
// 并施加独立超时（调用方 defer cancel）。仅限平台运营域（/platform/**）调用
func platformCtx(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

// List 运营面租户列表（无租户过滤，软删行不返回）
func (t *tenantRepository) List(ctx context.Context) ([]model.Tenant, error) {
	tenants := make([]model.Tenant, 0)
	pctx, cancel := platformCtx(ctx)
	defer cancel()
	if err := t.db.WithContext(pctx).Order("id").Find(&tenants).Error; err != nil {
		return nil, err
	}
	return tenants, nil
}

// Create 开通租户（运营面）：TenantID 列落默认值，OwnerAccountId 由服务层写入
func (t *tenantRepository) Create(ctx context.Context, tenant *model.Tenant) (*model.Tenant, error) {
	pctx, cancel := platformCtx(ctx)
	defer cancel()
	if err := t.db.WithContext(pctx).Create(tenant).Error; err != nil {
		return nil, err
	}
	return tenant, nil
}

// Update 运营面更新（名称/套餐/配置/配额覆盖/归属账号）
func (t *tenantRepository) Update(ctx context.Context, tenant *model.Tenant) (*model.Tenant, error) {
	pctx, cancel := platformCtx(ctx)
	defer cancel()
	if err := t.db.WithContext(pctx).Model(&model.Tenant{}).Where("id = ?", tenant.ID).
		Select("name", "plan", "owner_account_id", "config", "quotas").Updates(tenant).Error; err != nil {
		return nil, err
	}
	return tenant, nil
}

// UpdateStatus 生命周期流转：deleted 额外触发软删（数据保留期内可查库恢复）
func (t *tenantRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
	pctx, cancel := platformCtx(ctx)
	defer cancel()
	db := t.db.WithContext(pctx)
	if err := db.Model(&model.Tenant{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		return err
	}
	if status == model.TenantDeleted {
		return db.Delete(&model.Tenant{}, id).Error
	}
	return nil
}
