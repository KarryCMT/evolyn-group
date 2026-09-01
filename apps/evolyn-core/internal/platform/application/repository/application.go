package repository

import (
	"context"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/application/model"

	"gorm.io/gorm"
)

type applicationRepository struct {
	db *gorm.DB
}

// NewRepository 应用域仓储工厂（ADR-007 域模块化）
func NewRepository(db *gorm.DB) ApplicationRepository {
	return &applicationRepository{db: db}
}

// withContext 以请求 ctx 打开新会话：GORM 租户 Callback 自动注入过滤/
// 回填；ctx 携带事务 session 时加入外层事务（FIX-020/021 统一事务边界）
func (r *applicationRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *applicationRepository) Create(ctx context.Context, app *model.Application) (*model.Application, error) {
	if err := r.withContext(ctx).Create(app).Error; err != nil {
		return nil, err
	}
	return app, nil
}

func (r *applicationRepository) CreateInstallation(ctx context.Context, inst *model.Installation) error {
	return r.withContext(ctx).Create(inst).Error
}

func (r *applicationRepository) GetByID(ctx context.Context, id uint) (*model.Application, error) {
	app := new(model.Application)
	if err := r.withContext(ctx).First(app, id).Error; err != nil {
		return nil, err
	}
	return app, nil
}

// GetByCode 按应用编码加载：code 在租户内唯一（部分唯一索引），租户
// 过滤仍由 GORM Callback 兜底——跨租户 code 与不存在同口径 NotFound
func (r *applicationRepository) GetByCode(ctx context.Context, code string) (*model.Application, error) {
	app := new(model.Application)
	if err := r.withContext(ctx).Where("code = ?", code).First(app).Error; err != nil {
		return nil, err
	}
	return app, nil
}

// List 游标分页：固定序 sort_order ASC, id DESC，游标行之后取 limit 条，
// 多取一条探测 hasMore。keyword 命中 name（ILIKE），status 过滤可选
func (r *applicationRepository) List(ctx context.Context, params ListParams) ([]model.Application, bool, error) {
	query := r.withContext(ctx).Model(&model.Application{})
	if params.Keyword != "" {
		query = query.Where("name ILIKE ?", "%"+params.Keyword+"%")
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.HasCursor {
		query = query.Where(
			"(sort_order > ?) OR (sort_order = ? AND id < ?)",
			params.AfterSortID, params.AfterSortID, params.AfterID,
		)
	}

	apps := make([]model.Application, 0)
	// limit 上限归一化由 Service 完成；此处 limit+1 探测下一页
	if err := query.Order("sort_order ASC, id DESC").Limit(params.Limit + 1).Find(&apps).Error; err != nil {
		return nil, false, err
	}
	if len(apps) > params.Limit {
		return apps[:params.Limit], true, nil
	}
	return apps, false, nil
}

func (r *applicationRepository) UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	return r.withContext(ctx).Model(&model.Application{}).Where("id = ?", id).Updates(fields).Error
}

func (r *applicationRepository) SoftDelete(ctx context.Context, app *model.Application) error {
	return r.withContext(ctx).Delete(app).Error
}

// CountBillableByTenant 显式 Scope 而非依赖请求租户上下文（配额执行路径
// 可能在无上下文的事务内）；GORM 软删行自动排除
func (r *applicationRepository) CountBillableByTenant(ctx context.Context, tenantID uint) (int64, error) {
	var count int64
	err := r.withContext(ctx).Model(&model.Application{}).
		Scopes(infrastructure.TenantScope(tenantID)).Count(&count).Error
	return count, err
}

// Migrate 开发/测试路径：AutoMigrate 建表 + 补齐 GORM 标签表达不了的
// 软删部分唯一索引（与迁移链 uk_tn_applications_tenant_code 同名同构）
func (r *applicationRepository) Migrate() error {
	if err := r.db.AutoMigrate(&model.Application{}, &model.Installation{}); err != nil {
		return err
	}
	return r.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_applications_tenant_code
		ON applications (tenant_id, code) WHERE deleted_at IS NULL`).Error
}
