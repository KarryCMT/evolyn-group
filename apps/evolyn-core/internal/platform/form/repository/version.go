package repository

import (
	"context"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/form/model"

	"gorm.io/gorm"
)

type formVersionRepository struct {
	db *gorm.DB
}

// NewVersionRepository 发布版本仓储工厂
func NewVersionRepository(db *gorm.DB) FormVersionRepository {
	return &formVersionRepository{db: db}
}

func (r *formVersionRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *formVersionRepository) Create(ctx context.Context, version *model.FormVersion) (*model.FormVersion, error) {
	if err := r.withContext(ctx).Create(version).Error; err != nil {
		return nil, err
	}
	return version, nil
}

// SetSchemaRevision 发布事务内一次性回填修订口令（= 行 id）：此后版本行不再有任何
// 更新路径（不可变快照，阶段交接纪律）
func (r *formVersionRepository) SetSchemaRevision(ctx context.Context, id uint, revision int64) error {
	return r.withContext(ctx).Model(&model.FormVersion{}).
		Where("id = ?", id).Update("schema_revision", revision).Error
}

func (r *formVersionRepository) GetByID(ctx context.Context, id uint) (*model.FormVersion, error) {
	version := new(model.FormVersion)
	if err := r.withContext(ctx).First(version, id).Error; err != nil {
		return nil, err
	}
	return version, nil
}

func (r *formVersionRepository) MaxVersionNo(ctx context.Context, formID uint) (int, error) {
	var maxNo int
	err := r.withContext(ctx).Model(&model.FormVersion{}).
		Where("form_id = ?", formID).
		Select("COALESCE(MAX(version_no), 0)").Scan(&maxNo).Error
	return maxNo, err
}

func (r *formVersionRepository) GetByFormAndVersionNo(ctx context.Context, formID uint, versionNo int) (*model.FormVersion, error) {
	version := new(model.FormVersion)
	err := r.withContext(ctx).
		Where("form_id = ? AND version_no = ?", formID, versionNo).
		First(version).Error
	if err != nil {
		return nil, err
	}
	return version, nil
}

func (r *formVersionRepository) Migrate() error {
	return r.db.AutoMigrate(&model.FormVersion{})
}

type formRecordRepository struct {
	db *gorm.DB
}

// NewRecordRepository 记录仓储工厂
func NewRecordRepository(db *gorm.DB) FormRecordRepository {
	return &formRecordRepository{db: db}
}

func (r *formRecordRepository) Create(ctx context.Context, record *model.FormRecord) (*model.FormRecord, error) {
	if err := infrastructure.ResolveDB(ctx, r.db).Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (r *formRecordRepository) GetByID(ctx context.Context, id uint) (*model.FormRecord, error) {
	record := &model.FormRecord{}
	if err := infrastructure.ResolveDB(ctx, r.db).Where("id = ?", id).First(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (r *formRecordRepository) UpdateValues(ctx context.Context, id uint, values model.JSONContent) error {
	return infrastructure.ResolveDB(ctx, r.db).Model(&model.FormRecord{}).
		Where("id = ?", id).Update("values", values).Error
}

func (r *formRecordRepository) Migrate() error {
	return r.db.AutoMigrate(&model.FormRecord{})
}
