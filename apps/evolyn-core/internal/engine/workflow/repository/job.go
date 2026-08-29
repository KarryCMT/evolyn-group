package repository

import (
	"context"
	"time"

	"evolyn/internal/engine/workflow/model"
)

// JobRepository 延时任务仓储（Worker 竞争纪律：领取必须走
// FOR UPDATE SKIP LOCKED 小批量，禁止裸 UPDATE 抢占，第 19.2 章）。
type JobRepository interface {
	// CreateJob 创建延时任务（如任务创建时排 task.timeout/task.reminder）
	CreateJob(ctx context.Context, job *model.Job) error
	// ClaimDueJobs 领取到期任务：status=PENDING AND execute_at<=now，
	// SKIP LOCKED 置为 PROCESSING 并返回；crash 恢复由 Worker 侧超时回收承担
	ClaimDueJobs(ctx context.Context, now time.Time, batch int) ([]model.Job, error)
	// SaveJob 回写执行结果（SUCCEEDED / FAILED / PENDING 重试回队）
	SaveJob(ctx context.Context, job *model.Job) error
	// FindJobByID 按行 ID 加载（重试记账使用；租户过滤）
	FindJobByID(ctx context.Context, tenantID, jobID uint) (*model.Job, error)
	// CancelJobsByTask 任务终态时联动取消未执行 Job（第 19 章）
	CancelJobsByTask(ctx context.Context, taskID uint) error
	// CancelJobsByNodeInstance 节点完成联动取消（或签淘汰/会签达标后剩余
	// 任务的排期 Job 随任务取消而失效）
	CancelJobsByNodeInstance(ctx context.Context, nodeInstanceID uint) error
	// CancelJobsByInstance 实例终态联动取消（驳回/撤回/终止）
	CancelJobsByInstance(ctx context.Context, instanceID uint) error
	// ListJobsByInstance 实例 Job 列表（诊断/联动取消）
	ListJobsByInstance(ctx context.Context, instanceID uint) ([]model.Job, error)
}
