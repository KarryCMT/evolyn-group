package repository

import (
	"context"

	"evolyn/internal/contextx"
	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/workbench/model"

	"gorm.io/gorm"
)

type workbenchRepository struct {
	db *gorm.DB
}

// NewRepository 自定义工作台域仓储工厂（ADR-007 域模块化）
func NewRepository(db *gorm.DB) Repository {
	return &workbenchRepository{db: db}
}

// withContext 打开会话并剥离请求租户上下文：行定位由显式 tenantID 条件
// 表达（口径同 tenantproduct 域），ctx 携带事务 session 时仍加入外层事务
func (r *workbenchRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(contextx.DetachTenant(ctx), r.db)
}

func (r *workbenchRepository) FindByTenant(ctx context.Context, tenantID uint) (*model.TenantWorkbench, error) {
	record := new(model.TenantWorkbench)
	err := r.withContext(ctx).
		Where("tenant_id = ?", tenantID).
		First(record).Error
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (r *workbenchRepository) Create(ctx context.Context, record *model.TenantWorkbench) error {
	return r.withContext(ctx).Create(record).Error
}

func (r *workbenchRepository) UpdateContentWithRevision(
	ctx context.Context, id uint, fromRevision int64, content model.Content,
) (bool, error) {
	// updated_at 由 GORM Updates 依据模型 UpdatedAt 字段自动刷新
	res := r.withContext(ctx).
		Model(&model.TenantWorkbench{}).
		Where("id = ? AND revision = ?", id, fromRevision).
		Updates(map[string]interface{}{
			"content":  content,
			"revision": gorm.Expr("revision + 1"),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *workbenchRepository) Migrate() error {
	return r.db.AutoMigrate(&model.TenantWorkbench{})
}
