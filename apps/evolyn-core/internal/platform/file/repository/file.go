package repository

import (
	"context"
	"time"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/file/model"

	"gorm.io/gorm"
)

type fileRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) FileRepository { return &fileRepository{db: db} }

func (r *fileRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *fileRepository) Create(ctx context.Context, file *model.File) error {
	return r.withContext(ctx).Create(file).Error
}

func (r *fileRepository) GetByCode(ctx context.Context, code string) (*model.File, error) {
	file := new(model.File)
	if err := r.withContext(ctx).Where("code = ?", code).First(file).Error; err != nil {
		return nil, err
	}
	return file, nil
}

func (r *fileRepository) MarkReady(ctx context.Context, code string, actualSize int64, contentType string) (bool, error) {
	result := r.withContext(ctx).Model(&model.File{}).
		Where("code = ? AND state = ?", code, model.FileStateUploading).
		Updates(map[string]interface{}{
			"state":        model.FileStateReady,
			"actual_size":  actualSize,
			"content_type": contentType,
			"expires_at":   nil,
		})
	return result.RowsAffected == 1, result.Error
}

func (r *fileRepository) SoftDelete(ctx context.Context, file *model.File) error {
	return r.withContext(ctx).Delete(file).Error
}

func (r *fileRepository) ListExpiredUploads(ctx context.Context, now time.Time) ([]model.File, error) {
	files := make([]model.File, 0)
	err := r.withContext(ctx).Where("state = ? AND expires_at IS NOT NULL AND expires_at <= ?", model.FileStateUploading, now).
		Order("id").Find(&files).Error
	return files, err
}

func (r *fileRepository) CountStorageBytes(ctx context.Context, tenantID uint) (int64, error) {
	var total int64
	// uploading 使用客户端事先声明的字节数作为预留；ready 使用 RustFS
	// 确认过的实际字节数。该查询在租户行锁保护下避免并发配额穿透。
	err := r.withContext(ctx).Model(&model.File{}).
		Scopes(infrastructure.TenantScope(tenantID)).
		Select(`COALESCE(SUM(CASE WHEN state = 'uploading' THEN declared_size ELSE actual_size END), 0)`).
		Scan(&total).Error
	return total, err
}

func (r *fileRepository) Migrate() error {
	if err := r.db.AutoMigrate(&model.File{}); err != nil {
		return err
	}
	return r.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_files_tenant_code
		ON files (tenant_id, code) WHERE deleted_at IS NULL`).Error
}
