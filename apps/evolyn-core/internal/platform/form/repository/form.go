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

func (r *formRepository) GetByCode(ctx context.Context, code string) (*model.Form, error) {
	form := new(model.Form)
	if err := r.withContext(ctx).Where("code = ?", code).First(form).Error; err != nil {
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

// UpdateFormType 切换表单类型（ADR-011）：服务层已校验枚举与「类型确有
// 变化」，仓储只做白名单字段写入。
func (r *formRepository) UpdateFormType(ctx context.Context, id uint, formType model.FormType) error {
	return r.withContext(ctx).Model(&model.Form{}).
		Where("id = ?", id).Update("form_type", string(formType)).Error
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

// ExistingFormTargets 批量查询菜单内部目标 ID 对应的公开编码与表单类型。
func (r *formRepository) ExistingFormTargets(ctx context.Context, ids []uint) (map[uint]FormMenuTarget, error) {
	existing := make(map[uint]FormMenuTarget, len(ids))
	if len(ids) == 0 {
		return existing, nil
	}
	rows := make([]struct {
		ID       uint
		Code     string
		FormType model.FormType
	}, 0, len(ids))
	if err := r.withContext(ctx).Model(&model.Form{}).
		Where("id IN ?", ids).
		Select("id", "code", "form_type").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		existing[row.ID] = FormMenuTarget{Code: row.Code, FormType: row.FormType}
	}
	return existing, nil
}

// Migrate 开发/测试路径：AutoMigrate 建表（索引与迁移链同构部分）
func (r *formRepository) Migrate() error {
	if err := r.db.AutoMigrate(&model.Form{}); err != nil {
		return err
	}
	return r.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uk_forms_tenant_code
		ON forms (tenant_id, code) WHERE deleted_at IS NULL`).Error
}
