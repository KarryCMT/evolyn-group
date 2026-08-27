package repository

import (
	"context"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/form/model"

	"gorm.io/gorm"
)

type formRepository struct {
	db *gorm.DB
}

// NewRepository 表单域仓储工厂（ADR-007 域模块化）
func NewRepository(db *gorm.DB) FormRepository {
	return &formRepository{db: db}
}

// withContext 以请求 ctx 打开新会话：GORM 租户 Callback 自动注入过滤/回填；
// ctx 携带事务 session 时加入外层事务（FIX-020/021 统一事务边界）
func (r *formRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *formRepository) Create(ctx context.Context, form *model.Form) (*model.Form, error) {
	if err := r.withContext(ctx).Create(form).Error; err != nil {
		return nil, err
	}
	return form, nil
}

func (r *formRepository) GetByID(ctx context.Context, id uint) (*model.Form, error) {
	form := new(model.Form)
	if err := r.withContext(ctx).First(form, id).Error; err != nil {
		return nil, err
	}
	return form, nil
}

// List 应用内游标分页：固定序 id DESC，游标行之后取 limit 条，多取一条探测 hasMore
func (r *formRepository) List(ctx context.Context, params ListParams) ([]model.Form, bool, error) {
	query := r.withContext(ctx).Model(&model.Form{}).
		Where("application_id = ?", params.ApplicationID)
	if params.HasCursor {
		query = query.Where("id < ?", params.AfterID)
	}
	forms := make([]model.Form, 0)
	if err := query.Order("id DESC").Limit(params.Limit + 1).Find(&forms).Error; err != nil {
		return nil, false, err
	}
	if len(forms) > params.Limit {
		return forms[:params.Limit], true, nil
	}
	return forms, false, nil
}

func (r *formRepository) UpdateName(ctx context.Context, id uint, name string) error {
	return r.withContext(ctx).Model(&model.Form{}).
		Where("id = ?", id).Update("name", name).Error
}

// UpdateDraft 草稿乐观锁保存：条件递增避免读改写竞态（口径同 tenantproduct revision）
func (r *formRepository) UpdateDraft(
	ctx context.Context, id uint, fromRevision int64, content model.JSONContent,
) (bool, error) {
	result := r.withContext(ctx).Model(&model.Form{}).
		Where("id = ? AND draft_revision = ?", id, fromRevision).
		Updates(map[string]interface{}{
			"draft_content":  content,
			"draft_revision": gorm.Expr("draft_revision + 1"),
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// MarkPublished 发布事务内回写最新发布指针：只移动指针，不触碰草稿/历史快照。
func (r *formRepository) MarkPublished(ctx context.Context, id uint, versionID uint, versionNo int) error {
	return r.withContext(ctx).Model(&model.Form{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"latest_version_id": versionID,
			"published_version": versionNo,
		}).Error
}

func (r *formRepository) SoftDelete(ctx context.Context, form *model.Form) error {
	return r.withContext(ctx).Delete(form).Error
}

// CountBillableFormsByTenant 显式 Scope（配额执行路径可能在无上下文的事务内）
func (r *formRepository) CountBillableFormsByTenant(ctx context.Context, tenantID uint) (int64, error) {
	var count int64
	err := r.withContext(ctx).Model(&model.Form{}).
		Scopes(infrastructure.TenantScope(tenantID)).Count(&count).Error
	return count, err
}

// ExistingFormIDs 批量存在性查询：IN 命中 + 软删排除 + ctx 租户过滤。
func (r *formRepository) ExistingFormIDs(ctx context.Context, ids []uint) (map[uint]bool, error) {
	existing := make(map[uint]bool, len(ids))
	if len(ids) == 0 {
		return existing, nil
	}
	rows := make([]uint, 0, len(ids))
	if err := r.withContext(ctx).Model(&model.Form{}).
		Where("id IN ?", ids).
		Pluck("id", &rows).Error; err != nil {
		return nil, err
	}
	for _, id := range rows {
		existing[id] = true
	}
	return existing, nil
}

// Migrate 开发/测试路径：AutoMigrate 建表（索引与迁移链同构部分）
func (r *formRepository) Migrate() error {
	return r.db.AutoMigrate(&model.Form{})
}
