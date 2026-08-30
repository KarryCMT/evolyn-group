package repository

import (
	"context"
	"fmt"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/form/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (r *formRecordRepository) CreateIdempotent(ctx context.Context, record *model.FormRecord) (*model.FormRecord, bool, error) {
	if record.DataOpID == nil || *record.DataOpID == "" {
		return nil, false, fmt.Errorf("form record data operation id is required")
	}
	db := infrastructure.ResolveDB(ctx, r.db)
	result := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "data_op_id"}},
		DoNothing: true,
	}).Create(record)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected > 0 {
		return record, true, nil
	}
	existing := new(model.FormRecord)
	if err := db.Where("tenant_id = ? AND data_op_id = ?", record.TenantID, *record.DataOpID).First(existing).Error; err != nil {
		return nil, false, err
	}
	return existing, false, nil
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
	if err := r.db.AutoMigrate(&model.FormRecord{}); err != nil {
		return err
	}
	return r.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uk_form_records_tenant_data_op
		ON form_records (tenant_id, data_op_id)`).Error
}
