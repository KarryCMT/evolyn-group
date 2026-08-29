package repository

import (
	"context"
	"errors"
	"strconv"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/workflow/model"

	"gorm.io/gorm"
)

// definitionRepository 流程定义 GORM 仓储。
type definitionRepository struct {
	base *gorm.DB
}

// NewDefinitionRepository 构造流程定义仓储。
func NewDefinitionRepository(base *gorm.DB) DefinitionRepository {
	return &definitionRepository{base: base}
}

func (r *definitionRepository) Create(ctx context.Context, def *model.WfDefinition) (*model.WfDefinition, error) {
	if err := infrastructure.ResolveDB(ctx, r.base).Create(def).Error; err != nil {
		return nil, err
	}
	return def, nil
}

func (r *definitionRepository) GetByCode(ctx context.Context, code string) (*model.WfDefinition, error) {
	var def model.WfDefinition
	err := infrastructure.ResolveDB(ctx, r.base).Where("code = ?", code).First(&def).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &def, err
}

func (r *definitionRepository) List(ctx context.Context, params ListParams) ([]model.WfDefinition, bool, error) {
	query := infrastructure.ResolveDB(ctx, r.base).Model(&model.WfDefinition{}).Order("id DESC")
	if params.HasCursor {
		query = query.Where("id < ?", params.AfterID)
	}
	// limit+1 探测下一页
	rows := make([]model.WfDefinition, 0, params.Limit+1)
	if err := query.Limit(params.Limit + 1).Find(&rows).Error; err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > params.Limit
	if hasMore {
		rows = rows[:params.Limit]
	}
	return rows, hasMore, nil
}

func (r *definitionRepository) UpdateMeta(ctx context.Context, id uint, name, description string) error {
	return infrastructure.ResolveDB(ctx, r.base).Model(&model.WfDefinition{}).
		Where("id = ?", id).
		Updates(map[string]any{"name": name, "description": description}).Error
}

// SaveDraft 乐观锁条件更新：draft_revision 匹配才写入并将口令 +1；
// 0 行影响返回 false（口令过期），由 Service 转稳定错误。
func (r *definitionRepository) SaveDraft(ctx context.Context, id uint, fromRevision int64, content model.DSLContent) (bool, error) {
	result := infrastructure.ResolveDB(ctx, r.base).Model(&model.WfDefinition{}).
		Where("id = ? AND draft_revision = ?", id, fromRevision).
		Updates(map[string]any{"draft_content": content, "draft_revision": fromRevision + 1})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *definitionRepository) MarkPublished(ctx context.Context, id uint, versionID uint, versionNo int) error {
	return infrastructure.ResolveDB(ctx, r.base).Model(&model.WfDefinition{}).
		Where("id = ?", id).
		Updates(map[string]any{"latest_version_id": versionID, "published_version": versionNo}).Error
}

func (r *definitionRepository) SoftDelete(ctx context.Context, def *model.WfDefinition) error {
	return infrastructure.ResolveDB(ctx, r.base).Delete(def).Error
}

func (r *definitionRepository) Migrate() error {
	return r.base.AutoMigrate(&model.WfDefinition{})
}

// versionRepository 发布快照 GORM 仓储（追加写）。
type versionRepository struct {
	base *gorm.DB
}

// NewVersionRepository 构造发布快照仓储。
func NewVersionRepository(base *gorm.DB) VersionRepository {
	return &versionRepository{base: base}
}

func (r *versionRepository) MaxVersionNo(ctx context.Context, definitionID uint) (int, error) {
	var maxNo int
	err := infrastructure.ResolveDB(ctx, r.base).Model(&model.WfDefinitionVersion{}).
		Where("definition_id = ?", definitionID).
		Select("COALESCE(MAX(version_no), 0)").
		Scan(&maxNo).Error
	return maxNo, err
}

func (r *versionRepository) Create(ctx context.Context, version *model.WfDefinitionVersion) (*model.WfDefinitionVersion, error) {
	if err := infrastructure.ResolveDB(ctx, r.base).Create(version).Error; err != nil {
		return nil, err
	}
	return version, nil
}

func (r *versionRepository) GetByDefinitionAndVersionNo(ctx context.Context, definitionID uint, versionNo int) (*model.WfDefinitionVersion, error) {
	var version model.WfDefinitionVersion
	err := infrastructure.ResolveDB(ctx, r.base).
		Where("definition_id = ? AND version_no = ?", definitionID, versionNo).
		First(&version).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &version, err
}

func (r *versionRepository) ListByDefinition(ctx context.Context, definitionID uint) ([]model.WfDefinitionVersion, error) {
	rows := make([]model.WfDefinitionVersion, 0)
	err := infrastructure.ResolveDB(ctx, r.base).
		Where("definition_id = ?", definitionID).
		Order("version_no DESC").
		Find(&rows).Error
	return rows, err
}

func (r *versionRepository) Migrate() error {
	return r.base.AutoMigrate(&model.WfDefinitionVersion{})
}

// ParseCursor 解析不透明游标（上一页最后一条的 id）。
func ParseCursor(cursor string) (uint, bool, error) {
	if cursor == "" {
		return 0, false, nil
	}
	id, err := strconv.ParseUint(cursor, 10, 64)
	if err != nil || id == 0 {
		return 0, false, err
	}
	return uint(id), true, nil
}
