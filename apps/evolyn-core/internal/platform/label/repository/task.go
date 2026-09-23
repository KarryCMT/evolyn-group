package repository

import (
	"context"
	"time"

	"evolyn/internal/infrastructure"
	kernel "evolyn/internal/model"
	"evolyn/internal/platform/label/model"

	"gorm.io/gorm"
)

type RenderTaskRepository interface {
	Create(ctx context.Context, task *model.RenderTask, items []model.RenderTaskItem) error
	GetByCode(ctx context.Context, code string) (*model.RenderTask, error)
	ListItems(ctx context.Context, taskID uint) ([]model.RenderTaskItem, error)
	Claim(ctx context.Context, code string) (bool, error)
	ResetPending(ctx context.Context, taskID uint, errorCode, errorMessage string) error
	MarkItemRunning(ctx context.Context, itemID uint) error
	MarkItemFinished(ctx context.Context, itemID uint, status, errorCode, errorMessage string) error
	UpdateProgress(ctx context.Context, taskID uint, successCount, failedCount, progress int) error
	Finish(ctx context.Context, taskID uint, status, fileCode, errorCode, errorMessage string, successCount, failedCount int) error
	Migrate() error
}

type renderTaskRepository struct{ db *gorm.DB }

func NewRenderTaskRepository(db *gorm.DB) RenderTaskRepository {
	return &renderTaskRepository{db: db}
}

func (r *renderTaskRepository) session(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *renderTaskRepository) Create(ctx context.Context, task *model.RenderTask, items []model.RenderTaskItem) error {
	if err := r.session(ctx).Create(task).Error; err != nil {
		return err
	}
	for index := range items {
		items[index].TaskID = task.ID
		items[index].TenantID = task.TenantID
	}
	return r.session(ctx).Create(&items).Error
}

func (r *renderTaskRepository) GetByCode(ctx context.Context, code string) (*model.RenderTask, error) {
	var task model.RenderTask
	err := r.session(ctx).Where("code = ?", code).First(&task).Error
	return &task, err
}

func (r *renderTaskRepository) ListItems(ctx context.Context, taskID uint) ([]model.RenderTaskItem, error) {
	items := make([]model.RenderTaskItem, 0)
	err := r.session(ctx).Where("task_id = ?", taskID).Order("sequence_no ASC").Find(&items).Error
	return items, err
}

func (r *renderTaskRepository) Claim(ctx context.Context, code string) (bool, error) {
	now := kernel.JSONTime(time.Now())
	result := r.session(ctx).Model(&model.RenderTask{}).
		Where("code = ? AND status = ?", code, model.RenderTaskPending).
		Updates(map[string]any{"status": model.RenderTaskRunning, "started_at": &now, "error_code": "", "error_message": ""})
	return result.RowsAffected > 0, result.Error
}

func (r *renderTaskRepository) ResetPending(ctx context.Context, taskID uint, errorCode, errorMessage string) error {
	return r.session(ctx).Model(&model.RenderTask{}).Where("id = ? AND status = ?", taskID, model.RenderTaskRunning).
		Updates(map[string]any{"status": model.RenderTaskPending, "error_code": errorCode, "error_message": errorMessage}).Error
}

func (r *renderTaskRepository) MarkItemRunning(ctx context.Context, itemID uint) error {
	now := kernel.JSONTime(time.Now())
	return r.session(ctx).Model(&model.RenderTaskItem{}).Where("id = ?", itemID).
		Updates(map[string]any{"status": model.RenderItemRunning, "started_at": &now, "error_code": "", "error_message": ""}).Error
}

func (r *renderTaskRepository) MarkItemFinished(ctx context.Context, itemID uint, status, errorCode, errorMessage string) error {
	now := kernel.JSONTime(time.Now())
	return r.session(ctx).Model(&model.RenderTaskItem{}).Where("id = ?", itemID).
		Updates(map[string]any{"status": status, "finished_at": &now, "error_code": errorCode, "error_message": errorMessage}).Error
}

func (r *renderTaskRepository) UpdateProgress(ctx context.Context, taskID uint, successCount, failedCount, progress int) error {
	return r.session(ctx).Model(&model.RenderTask{}).Where("id = ?", taskID).
		Updates(map[string]any{"success_count": successCount, "failed_count": failedCount, "progress": progress}).Error
}

func (r *renderTaskRepository) Finish(ctx context.Context, taskID uint, status, fileCode, errorCode, errorMessage string, successCount, failedCount int) error {
	now := kernel.JSONTime(time.Now())
	return r.session(ctx).Model(&model.RenderTask{}).Where("id = ?", taskID).Updates(map[string]any{
		"status": status, "file_code": fileCode, "error_code": errorCode, "error_message": errorMessage,
		"success_count": successCount, "failed_count": failedCount, "progress": 100, "finished_at": &now,
	}).Error
}

func (r *renderTaskRepository) Migrate() error {
	return r.db.AutoMigrate(&model.RenderTask{}, &model.RenderTaskItem{})
}
