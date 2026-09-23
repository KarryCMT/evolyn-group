// Package repository 是标签模板域的租户隔离持久化层。
package repository

import (
	"context"
	"strconv"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/label/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ListParams struct {
	Limit     int
	AfterID   uint
	HasCursor bool
	Keyword   string
	Status    string
	FormCode  string
}

type TemplateRepository interface {
	Create(ctx context.Context, template *model.Template) (*model.Template, error)
	GetByCode(ctx context.Context, code string) (*model.Template, error)
	GetByCodeForUpdate(ctx context.Context, code string) (*model.Template, error)
	List(ctx context.Context, params ListParams) ([]model.Template, bool, error)
	SaveDraft(ctx context.Context, id uint, revision int64, schema model.SchemaContent, width, height float64, unit string, dpi int) (bool, error)
	MarkPreviewed(ctx context.Context, id uint, revision int64) (bool, error)
	MarkPublished(ctx context.Context, id, versionID uint, versionNo int, draftRevision int64) error
	SoftDelete(ctx context.Context, template *model.Template) error
	Migrate() error
}

type VersionRepository interface {
	MaxVersionNo(ctx context.Context, templateID uint) (int, error)
	Create(ctx context.Context, version *model.TemplateVersion) (*model.TemplateVersion, error)
	GetByID(ctx context.Context, id uint) (*model.TemplateVersion, error)
	Migrate() error
}

type templateRepository struct{ db *gorm.DB }
type versionRepository struct{ db *gorm.DB }

func NewTemplateRepository(db *gorm.DB) TemplateRepository { return &templateRepository{db: db} }
func NewVersionRepository(db *gorm.DB) VersionRepository   { return &versionRepository{db: db} }

func (r *templateRepository) session(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *templateRepository) Create(ctx context.Context, template *model.Template) (*model.Template, error) {
	if err := r.session(ctx).Create(template).Error; err != nil {
		return nil, err
	}
	return template, nil
}

func (r *templateRepository) GetByCode(ctx context.Context, code string) (*model.Template, error) {
	var template model.Template
	err := r.session(ctx).Where("code = ?", code).First(&template).Error
	return &template, err
}

// GetByCodeForUpdate 串行化同一模板的发布，保证版本号与草稿快照来自同一锁定行。
func (r *templateRepository) GetByCodeForUpdate(ctx context.Context, code string) (*model.Template, error) {
	var template model.Template
	err := r.session(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ?", code).First(&template).Error
	return &template, err
}

func (r *templateRepository) List(ctx context.Context, params ListParams) ([]model.Template, bool, error) {
	query := r.session(ctx).Model(&model.Template{}).Order("id DESC")
	if params.HasCursor {
		query = query.Where("id < ?", params.AfterID)
	}
	if params.FormCode != "" {
		query = query.Where("form_code = ?", params.FormCode)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.Keyword != "" {
		query = query.Where("name ILIKE ?", "%"+params.Keyword+"%")
	}
	rows := make([]model.Template, 0, params.Limit+1)
	if err := query.Limit(params.Limit + 1).Find(&rows).Error; err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > params.Limit
	if hasMore {
		rows = rows[:params.Limit]
	}
	return rows, hasMore, nil
}

func (r *templateRepository) SaveDraft(ctx context.Context, id uint, revision int64, schema model.SchemaContent, width, height float64, unit string, dpi int) (bool, error) {
	result := r.session(ctx).Model(&model.Template{}).
		Where("id = ? AND draft_revision = ?", id, revision).
		Updates(map[string]any{
			"draft_schema": schema, "draft_revision": revision + 1,
			"width": width, "height": height, "unit": unit, "dpi": dpi,
		})
	return result.RowsAffected > 0, result.Error
}

func (r *templateRepository) MarkPreviewed(ctx context.Context, id uint, revision int64) (bool, error) {
	result := r.session(ctx).Model(&model.Template{}).
		Where("id = ? AND draft_revision = ?", id, revision).
		Update("previewed_draft_revision", revision)
	return result.RowsAffected > 0, result.Error
}

func (r *templateRepository) MarkPublished(ctx context.Context, id, versionID uint, versionNo int, draftRevision int64) error {
	return r.session(ctx).Model(&model.Template{}).Where("id = ?", id).Updates(map[string]any{
		"latest_version_id": versionID, "published_version": versionNo,
		"published_draft_revision": draftRevision, "status": model.StatusPublished,
	}).Error
}

func (r *templateRepository) SoftDelete(ctx context.Context, template *model.Template) error {
	return r.session(ctx).Delete(template).Error
}

func (r *templateRepository) Migrate() error {
	if err := r.db.AutoMigrate(&model.Template{}); err != nil {
		return err
	}
	// AutoMigrate 无法表达软删除条件唯一索引，开发库需与迁移链保持一致。
	for _, statement := range []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_label_templates_tenant_code ON tn_label_templates (tenant_id, code) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_label_templates_form ON tn_label_templates (tenant_id, form_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_tn_label_templates_tenant ON tn_label_templates (tenant_id, id DESC) WHERE deleted_at IS NULL`,
	} {
		if err := r.db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *versionRepository) MaxVersionNo(ctx context.Context, templateID uint) (int, error) {
	var maxVersion int
	err := infrastructure.ResolveDB(ctx, r.db).Model(&model.TemplateVersion{}).
		Where("template_id = ?", templateID).Select("COALESCE(MAX(version_no), 0)").Scan(&maxVersion).Error
	return maxVersion, err
}

func (r *versionRepository) Create(ctx context.Context, version *model.TemplateVersion) (*model.TemplateVersion, error) {
	if err := infrastructure.ResolveDB(ctx, r.db).Create(version).Error; err != nil {
		return nil, err
	}
	return version, nil
}

func (r *versionRepository) GetByID(ctx context.Context, id uint) (*model.TemplateVersion, error) {
	var version model.TemplateVersion
	err := infrastructure.ResolveDB(ctx, r.db).Where("id = ?", id).First(&version).Error
	return &version, err
}

func (r *versionRepository) Migrate() error {
	return r.db.AutoMigrate(&model.TemplateVersion{})
}

func ParseCursor(cursor string) (uint, bool, error) {
	if cursor == "" {
		return 0, false, nil
	}
	parsed, err := strconv.ParseUint(cursor, 10, 64)
	if err != nil || parsed == 0 {
		return 0, false, err
	}
	return uint(parsed), true, nil
}
