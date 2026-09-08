// 存储元数据仓储（物理表存储方案 §5.2）：绑定/模型版本/DDL Job 的持久化。
// 全部经 infrastructure.ResolveDB 加入 ctx 传播事务；DDL Job 领取用
// FOR UPDATE SKIP LOCKED 与 wf_job 同口径。
package repository

import (
	"context"
	"errors"
	"time"

	"evolyn/internal/infrastructure"
	kernel "evolyn/internal/model"
	"evolyn/internal/platform/form/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FormStorageRepository 存储绑定仓储。
type FormStorageRepository interface {
	// Create 创建绑定；table_name 唯一冲突返回 isUniqueConflict=true（调用方
	// 换表名重试，至多五次，绝不先查 catalog 再建表——方案 §7）。
	Create(ctx context.Context, storage *model.FormStorage) (created bool, isUniqueConflict bool, err error)
	// GetByFormID 按表单加载绑定（无绑定=存量 JSONB 表单，返回 NotFound）。
	GetByFormID(ctx context.Context, formID uint) (*model.FormStorage, error)
	// GetByID 按行 ID 加载（DDL Worker Job 链路）。
	GetByID(ctx context.Context, id uint) (*model.FormStorage, error)
	// LockByID 行锁加载（DDL 执行事务内防并发状态推进）。
	LockByID(ctx context.Context, id uint) (*model.FormStorage, error)
	// LockByForm 发布事务内行锁（SELECT ... FOR UPDATE），防同表单并发发布。
	LockByForm(ctx context.Context, formID uint) (*model.FormStorage, error)
	// MarkPublishing 置为发布中（绑定 DDL Job 创建之后）。
	MarkPublishing(ctx context.Context, id uint) error
	// MarkReady Job 成功：恢复可用并推进已应用模型版本。
	MarkReady(ctx context.Context, id uint, appliedSchemaVersionID uint) error
	// MarkFailed Job 终态失败。
	MarkFailed(ctx context.Context, id uint) error
	// Migrate 开发/测试 AutoMigrate 路径（生产只走 SQL 迁移）。
	Migrate() error
}

// StorageSchemaVersionRepository 物理模型版本仓储（不可变：创建后仅状态推进）。
type StorageSchemaVersionRepository interface {
	Create(ctx context.Context, version *model.FormStorageSchemaVersion) (*model.FormStorageSchemaVersion, error)
	GetByID(ctx context.Context, id uint) (*model.FormStorageSchemaVersion, error)
	// GetApplied 最新已应用模型（无发布返回 NotFound）。
	GetApplied(ctx context.Context, storageID uint) (*model.FormStorageSchemaVersion, error)
	MarkApplied(ctx context.Context, id uint) error
	MarkFailed(ctx context.Context, id uint, errorCode, errorDetail string) error
	// ResetForRetry 管理员重试：FAILED→PENDING 复位（模型/计划不可变，仅状态回退）。
	ResetForRetry(ctx context.Context, id uint) error
	// Migrate 开发/测试 AutoMigrate 路径。
	Migrate() error
}

// StorageChildRepository 子表单物理子表映射仓储（Phase 3，000071）。
type StorageChildRepository interface {
	// Create 分配子表映射；table_name 唯一冲突返回 isUniqueConflict=true
	//（调用方换名重试，与父表名防重同口径）。
	Create(ctx context.Context, child *model.FormStorageChild) (created bool, isUniqueConflict bool, err error)
	// ListByStorage 存储绑定的全部子表映射（发布 Diff 继承既有表名）。
	ListByStorage(ctx context.Context, storageID uint) ([]model.FormStorageChild, error)
	// Migrate 开发/测试 AutoMigrate 路径。
	Migrate() error
}

// FormDDLJobRepository DDL Job 仓储。
type FormDDLJobRepository interface {
	Create(ctx context.Context, job *model.FormDDLJob) (*model.FormDDLJob, error)
	GetByID(ctx context.Context, id uint) (*model.FormDDLJob, error)
	// HasActiveByStorage 存储是否存在未完成 Job（PENDING/PROCESSING）：
	// 并发发布的 FORM_STORAGE_BUSY 判定。
	HasActiveByStorage(ctx context.Context, storageID uint) (bool, error)
	// ClaimNextDue FOR UPDATE SKIP LOCKED 领取一批到期 Job（状态置 PROCESSING）。
	ClaimNextDue(ctx context.Context, now time.Time, limit int) ([]model.FormDDLJob, error)
	// SaveJob 执行事务内回写（成功/失败路径共用，含状态与错误记账）。
	SaveJob(ctx context.Context, job *model.FormDDLJob) error
	// ResetForRetry 管理员重试失败 Job：清空错误并立即回队（不接受新模型）。
	ResetForRetry(ctx context.Context, jobID uint, nextAttemptAt time.Time) error
	// Migrate 开发/测试 AutoMigrate 路径。
	Migrate() error
}

// ---- 实现 ----

type formStorageRepository struct{ db *gorm.DB }

// NewStorageRepository 构造存储绑定仓储。
func NewStorageRepository(db *gorm.DB) FormStorageRepository {
	return &formStorageRepository{db: db}
}

func (r *formStorageRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *formStorageRepository) Create(ctx context.Context, storage *model.FormStorage) (bool, bool, error) {
	err := r.withContext(ctx).Create(storage).Error
	if err != nil {
		if isUniqueViolation(err) {
			return false, true, nil
		}
		return false, false, err
	}
	return true, false, nil
}

func (r *formStorageRepository) GetByFormID(ctx context.Context, formID uint) (*model.FormStorage, error) {
	var storage model.FormStorage
	if err := r.withContext(ctx).Where("form_id = ?", formID).First(&storage).Error; err != nil {
		return nil, err
	}
	return &storage, nil
}

func (r *formStorageRepository) GetByID(ctx context.Context, id uint) (*model.FormStorage, error) {
	var storage model.FormStorage
	if err := infrastructure.ResolveDB(ctx, r.db).First(&storage, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &storage, nil
}

func (r *formStorageRepository) LockByID(ctx context.Context, id uint) (*model.FormStorage, error) {
	var storage model.FormStorage
	if err := infrastructure.ResolveDB(ctx, r.db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&storage, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &storage, nil
}

func (r *formStorageRepository) LockByForm(ctx context.Context, formID uint) (*model.FormStorage, error) {
	var storage model.FormStorage
	if err := r.withContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("form_id = ?", formID).
		First(&storage).Error; err != nil {
		return nil, err
	}
	return &storage, nil
}

func (r *formStorageRepository) MarkPublishing(ctx context.Context, id uint) error {
	return r.withContext(ctx).Model(&model.FormStorage{}).
		Where("id = ?", id).
		Updates(map[string]any{"state": model.StorageStatePublishing, "updated_at": time.Now()}).Error
}

func (r *formStorageRepository) MarkReady(ctx context.Context, id uint, appliedSchemaVersionID uint) error {
	return r.withContext(ctx).Model(&model.FormStorage{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"state":                     model.StorageStateReady,
			"applied_schema_version_id": appliedSchemaVersionID,
			"updated_at":                time.Now(),
		}).Error
}

func (r *formStorageRepository) MarkFailed(ctx context.Context, id uint) error {
	return r.withContext(ctx).Model(&model.FormStorage{}).
		Where("id = ?", id).
		Updates(map[string]any{"state": model.StorageStateFailed, "updated_at": time.Now()}).Error
}

func (r *formStorageRepository) Migrate() error {
	return r.db.AutoMigrate(&model.FormStorage{})
}

type storageSchemaVersionRepository struct{ db *gorm.DB }

// NewStorageSchemaVersionRepository 构造物理模型版本仓储。
func NewStorageSchemaVersionRepository(db *gorm.DB) StorageSchemaVersionRepository {
	return &storageSchemaVersionRepository{db: db}
}

func (r *storageSchemaVersionRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *storageSchemaVersionRepository) Create(ctx context.Context, version *model.FormStorageSchemaVersion) (*model.FormStorageSchemaVersion, error) {
	if err := r.withContext(ctx).Create(version).Error; err != nil {
		return nil, err
	}
	return version, nil
}

func (r *storageSchemaVersionRepository) GetByID(ctx context.Context, id uint) (*model.FormStorageSchemaVersion, error) {
	var version model.FormStorageSchemaVersion
	if err := r.withContext(ctx).First(&version, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

func (r *storageSchemaVersionRepository) GetApplied(ctx context.Context, storageID uint) (*model.FormStorageSchemaVersion, error) {
	var version model.FormStorageSchemaVersion
	if err := r.withContext(ctx).
		Where("storage_id = ? AND state = ?", storageID, model.SchemaVersionApplied).
		Order("id DESC").
		First(&version).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

func (r *storageSchemaVersionRepository) MarkApplied(ctx context.Context, id uint) error {
	return r.withContext(ctx).Model(&model.FormStorageSchemaVersion{}).
		Where("id = ? AND state = ?", id, model.SchemaVersionPending).
		Updates(map[string]any{
			"state":        model.SchemaVersionApplied,
			"applied_at":   time.Now(),
			"error_code":   "",
			"error_detail": "",
		}).Error
}

func (r *storageSchemaVersionRepository) MarkFailed(ctx context.Context, id uint, errorCode, errorDetail string) error {
	return r.withContext(ctx).Model(&model.FormStorageSchemaVersion{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"state":        model.SchemaVersionFailed,
			"error_code":   errorCode,
			"error_detail": truncate(errorDetail, 2000),
		}).Error
}

// ResetForRetry FAILED 版本复位回 PENDING（模型/计划不可变，仅状态回退）。
func (r *storageSchemaVersionRepository) ResetForRetry(ctx context.Context, id uint) error {
	return infrastructure.ResolveDB(ctx, r.db).Model(&model.FormStorageSchemaVersion{}).
		Where("id = ? AND state = ?", id, model.SchemaVersionFailed).
		Updates(map[string]any{"state": model.SchemaVersionPending, "error_code": "", "error_detail": ""}).Error
}

func (r *storageSchemaVersionRepository) Migrate() error {
	return r.db.AutoMigrate(&model.FormStorageSchemaVersion{})
}

type storageChildRepository struct{ db *gorm.DB }

// NewStorageChildRepository 构造子表映射仓储。
func NewStorageChildRepository(db *gorm.DB) StorageChildRepository {
	return &storageChildRepository{db: db}
}

func (r *storageChildRepository) Create(ctx context.Context, child *model.FormStorageChild) (bool, bool, error) {
	if err := infrastructure.ResolveDB(ctx, r.db).Create(child).Error; err != nil {
		if isUniqueViolation(err) {
			return false, true, nil
		}
		return false, false, err
	}
	return true, false, nil
}

func (r *storageChildRepository) ListByStorage(ctx context.Context, storageID uint) ([]model.FormStorageChild, error) {
	children := make([]model.FormStorageChild, 0)
	if err := infrastructure.ResolveDB(ctx, r.db).
		Where("storage_id = ?", storageID).
		Order("id ASC").
		Find(&children).Error; err != nil {
		return nil, err
	}
	return children, nil
}

func (r *storageChildRepository) Migrate() error {
	return r.db.AutoMigrate(&model.FormStorageChild{})
}

type formDDLJobRepository struct{ db *gorm.DB }

// NewDDLJobRepository 构造 DDL Job 仓储。
func NewDDLJobRepository(db *gorm.DB) FormDDLJobRepository {
	return &formDDLJobRepository{db: db}
}

func (r *formDDLJobRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *formDDLJobRepository) Create(ctx context.Context, job *model.FormDDLJob) (*model.FormDDLJob, error) {
	if err := r.withContext(ctx).Create(job).Error; err != nil {
		return nil, err
	}
	return job, nil
}

func (r *formDDLJobRepository) GetByID(ctx context.Context, id uint) (*model.FormDDLJob, error) {
	var job model.FormDDLJob
	if err := r.withContext(ctx).First(&job, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *formDDLJobRepository) HasActiveByStorage(ctx context.Context, storageID uint) (bool, error) {
	var count int64
	err := r.withContext(ctx).Model(&model.FormDDLJob{}).
		Joins("JOIN tn_form_storage_schema_versions v ON v.id = tn_form_ddl_jobs.storage_schema_version_id").
		Where("v.storage_id = ? AND tn_form_ddl_jobs.status IN ?", storageID,
			[]model.DDLJobStatus{model.DDLJobPending, model.DDLJobProcessing}).
		Count(&count).Error
	return count > 0, err
}

func (r *formDDLJobRepository) ClaimNextDue(ctx context.Context, now time.Time, limit int) ([]model.FormDDLJob, error) {
	var jobs []model.FormDDLJob
	err := r.withContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("status = ? AND next_attempt_at <= ?", model.DDLJobPending, now).
		Order("id ASC").
		Limit(limit).
		Find(&jobs).Error
	if err != nil {
		return nil, err
	}
	for i := range jobs {
		jobs[i].Status = model.DDLJobProcessing
		started := kernel.JSONTime(now)
		jobs[i].StartedAt = &started
		if err := infrastructure.ResolveDB(ctx, r.db).Model(&model.FormDDLJob{}).
			Where("id = ?", jobs[i].ID).
			Updates(map[string]any{"status": model.DDLJobProcessing, "started_at": now}).Error; err != nil {
			return nil, err
		}
	}
	return jobs, nil
}

func (r *formDDLJobRepository) SaveJob(ctx context.Context, job *model.FormDDLJob) error {
	// finished_at：终态（SUCCEEDED/FAILED）写时间，退避回队保持 NULL。
	var finishedAt any
	if job.Status == model.DDLJobSucceeded || job.Status == model.DDLJobFailed {
		finishedAt = time.Time(derefTime(job.FinishedAt))
	}
	return infrastructure.ResolveDB(ctx, r.db).Model(&model.FormDDLJob{}).
		Where("id = ?", job.ID).
		Updates(map[string]any{
			"status":            job.Status,
			"retry_count":       job.RetryCount,
			"next_attempt_at":   time.Time(job.NextAttemptAt),
			"last_error_code":   job.LastErrorCode,
			"last_error_detail": truncate(job.LastErrorDetail, 512),
			"finished_at":       finishedAt,
		}).Error
}

func (r *formDDLJobRepository) ResetForRetry(ctx context.Context, jobID uint, nextAttemptAt time.Time) error {
	result := r.withContext(ctx).Model(&model.FormDDLJob{}).
		Where("id = ? AND status = ?", jobID, model.DDLJobFailed).
		Updates(map[string]any{
			"status":          model.DDLJobPending,
			"next_attempt_at": nextAttemptAt,
			"finished_at":     nil,
		}).Error
	if result != nil {
		return result
	}
	return nil
}

func (r *formDDLJobRepository) Migrate() error {
	return r.db.AutoMigrate(&model.FormDDLJob{})
}

// ---- 助手 ----

// isUniqueViolation 判定 PostgreSQL 唯一约束冲突（表名防重重试依赖）。
func isUniqueViolation(err error) bool {
	var pgErr interface{ GetSQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.GetSQLState() == "23505"
	}
	return false
}

func truncate(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}

func derefTime(t *kernel.JSONTime) kernel.JSONTime {
	if t == nil {
		return kernel.JSONTime{}
	}
	return *t
}
