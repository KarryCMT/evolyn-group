package repository

import (
	"context"
	"time"

	"evolyn/internal/platform/file/model"
)

// FileRepository 仅负责文件元数据持久化；对象 I/O 由 infrastructure/objectstore
// 处理，跨系统调用不可置于数据库事务中。
type FileRepository interface {
	Create(ctx context.Context, file *model.File) error
	GetByCode(ctx context.Context, code string) (*model.File, error)
	MarkReady(ctx context.Context, code string, actualSize int64, contentType string) (bool, error)
	SoftDelete(ctx context.Context, file *model.File) error
	ListExpiredUploads(ctx context.Context, now time.Time) ([]model.File, error)
	// CountStorageBytes 计入上传预留和已完成文件；软删文件自动排除，供
	// QuotaService 在持有租户行锁时执行并发安全的存储配额校验。
	CountStorageBytes(ctx context.Context, tenantID uint) (int64, error)
	Migrate() error
}
