// WorkflowJobWorker（Phase 5，第 19 章）：wf_job 持久化延时任务的 DB Worker。
//
// 竞争纪律：领取走 FOR UPDATE SKIP LOCKED 小批量，禁止裸 UPDATE 抢占；
// 原子边界：claim + 执行 + 回写结果在同一事务内，crash 时整体回滚为
// PENDING（天然 crash recovery，无孤儿 PROCESSING）；失败重试在独立事务
// 记账（retry_count 递增 → 未超限回队 PENDING + 退避，超限 FAILED）。
//
// 自动动作边界（第 19.4 章）：超时自动 approve/reject 必须经 Runtime →
// Task Engine 正常执行路径（AutoTimeout：Actor 语义替代校验、状态机校验、
// TIMEOUT 操作流水、事件发布），Worker 不得直接修改 wf_task/wf_node_instance
// /wf_instance 状态。
package worker

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"evolyn/internal/contextx"
	"evolyn/internal/engine/workflow/event"
	"evolyn/internal/engine/workflow/model"
	"evolyn/internal/engine/workflow/provider"
	"evolyn/internal/engine/workflow/repository"
	engineruntime "evolyn/internal/engine/workflow/runtime"
	"evolyn/internal/engine/workflow/task"
	"evolyn/internal/infrastructure"

	"github.com/sirupsen/logrus"
)

const (
	// DefaultPollInterval 轮询周期（分钟级足够：超时/提醒非秒级实时场景）
	DefaultPollInterval = 30 * time.Second
	// defaultBatchSize 单轮最多处理 Job 数（防止单轮长事务占用过久）
	defaultBatchSize = 16
	// defaultRetryBackoff 失败重试回队退避
	defaultRetryBackoff = time.Minute
	// timeoutAutoComment 超时自动动作的操作流水意见（operator=0 系统触发）
	timeoutAutoComment = "超时自动处理"
	// reminderOperator 提醒流水操作人（0=系统）
	reminderOperator uint = 0
)

// JobWorker wf_job 轮询 Worker（随服务生命周期启停，风格同 EditionWorker）。
type JobWorker struct {
	tx         TxManager
	jobs       repository.JobRepository
	tasks      repository.TaskRepository
	operations repository.OperationRepository
	runtime    *engineruntime.Runtime
	// publisher 领域事件发布窄端口（Phase 6）：催办到点经适配器桥接
	// notification 域事务 Outbox，与 REMINDER 流水同事务落库
	publisher provider.EventPublisher
	interval  time.Duration
	batchSize int
	logger    *logrus.Logger
}

// TxManager 事务窄端口（装配层由 infrastructure.TxManager 适配）。
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// txManager 适配 infrastructure.TxManager（避免 worker 依赖具体实现类型）。
type txManager struct{ inner *infrastructure.TxManager }

func (t txManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return t.inner.WithinTransaction(ctx, fn)
}

// NewJobWorker 构造 Job Worker（interval<=0 取默认轮询周期）。
func NewJobWorker(
	inner *infrastructure.TxManager,
	jobs repository.JobRepository,
	tasks repository.TaskRepository,
	operations repository.OperationRepository,
	runtime *engineruntime.Runtime,
	publisher provider.EventPublisher,
	interval time.Duration,
	logger *logrus.Logger,
) *JobWorker {
	if interval <= 0 {
		interval = DefaultPollInterval
	}
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &JobWorker{
		tx:         txManager{inner: inner},
		jobs:       jobs,
		tasks:      tasks,
		operations: operations,
		runtime:    runtime,
		publisher:  publisher,
		interval:   interval,
		batchSize:  defaultBatchSize,
		logger:     logger,
	}
}

// Run 启动轮询循环，ctx 取消即退出（随服务优雅停机）。
func (w *JobWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.logger.Infof("workflow job worker started, interval: %s", w.interval)
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("workflow job worker stopped")
			return
		case <-ticker.C:
			w.processRound(ctx)
		}
	}
}

// processRound 单轮处理：逐 Job 独立事务（claim + 执行 + 回写同事务），
// 单个 Job 失败不影响其余（下一轮重试由重试记账承担）。
func (w *JobWorker) processRound(ctx context.Context) {
	processed := 0
	for i := 0; i < w.batchSize; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		handled, err := w.processOne(ctx)
		if err != nil {
			w.logger.Warnf("workflow job round error: %v", err)
			return
		}
		if !handled {
			return
		}
		processed++
	}
	if processed > 0 {
		w.logger.Debugf("workflow job round processed %d job(s)", processed)
	}
}

// processOne 领取并执行单个 Job；无到期 Job 返回 false。
// 执行事务失败整体回滚（claim 一并回滚），随后在独立事务做重试记账。
func (w *JobWorker) processOne(ctx context.Context) (bool, error) {
	var jobID uint
	execErr := w.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		claims, err := w.jobs.ClaimDueJobs(tctx, time.Now(), 1)
		if err != nil {
			return err
		}
		if len(claims) == 0 {
			return nil
		}
		job := claims[0]
		jobID = job.ID
		// 租户上下文修复：Worker 全租户轮询路径 ctx 原本无租户，租户
		// Callback（Create 注入/查询过滤）对执行期内嵌套的平台写入
		//（变量/Outbox/表单写回等）原本无副作用。按 Job 租户上下文化后，
		// 与请求路径同口径（Phase 7 service 执行引入平台写入后成为必须）
		tctx = contextx.NewTenantContext(tctx, job.TenantID)
		return w.handle(tctx, &job)
	})
	if execErr == nil {
		return jobID != 0, nil
	}
	if jobID == 0 {
		// 领取阶段失败（无 Job 被锁定）：直接上抛入日志
		return false, execErr
	}
	// 执行失败：重试记账（未超限回队 + 退避，超限 FAILED 终态）
	if err := w.recordRetryFailure(ctx, jobID, execErr); err != nil {
		return false, err
	}
	return true, nil
}

// handle 执行单个已领取 Job 并回写终态（同事务）。
func (w *JobWorker) handle(ctx context.Context, job *model.Job) error {
	switch job.Type {
	case model.JobTypeTaskTimeout:
		if err := w.handleTimeout(ctx, job); err != nil {
			return err
		}
	case model.JobTypeTaskReminder:
		if err := w.handleReminder(ctx, job); err != nil {
			return err
		}
	case model.JobTypeServiceInvoke:
		// 服务节点异步调用（Phase 7）：经 Runtime 正常推进路径执行
		//（行锁校验/变量写入/续跑推进/操作流水与人工动作同口径），
		// Worker 不得直改流程状态；失败由重试记账退避回队。实例已终态
		//（终止与领取竞态）幂等空跑，与超时动作「终态空跑」同口径
		if _, err := w.runtime.InvokeServiceNode(ctx, engineruntime.ServiceInvokeInput{
			TenantID:       job.TenantID,
			InstanceID:     job.InstanceID,
			NodeInstanceID: job.NodeInstanceID,
		}); err != nil {
			if errors.Is(err, task.ErrInstanceNotRunning) {
				w.logger.Infof("service invoke job %d: instance %d not running, skip", job.ID, job.InstanceID)
				return nil
			}
			return err
		}
	default:
		// 未知类型直接失败终态（校验器/版本演进防御）
		job.Status = model.JobStatusFAILED
		job.LastError = "unknown job type: " + string(job.Type)
		return w.jobs.SaveJob(ctx, job)
	}
	job.Status = model.JobStatusSUCCEEDED
	job.LastError = ""
	return w.jobs.SaveJob(ctx, job)
}

// handleTimeout 超时自动动作（第 19.4 章）：任务已终态 → 幂等空跑；
// 仍 PENDING → 经 Runtime.Approve/Reject（AutoTimeout）走 Task Engine
// 正常执行路径，禁止直改状态。
func (w *JobWorker) handleTimeout(ctx context.Context, job *model.Job) error {
	task, err := w.tasks.FindTaskByIDForUpdate(ctx, job.TenantID, job.TaskID)
	if err != nil {
		// 任务不存在（实例被清理等）：幂等空跑，避免无意义重试
		w.logger.Warnf("timeout job %d: task %d not found, skip", job.ID, job.TaskID)
		return nil //nolint:nilerr // 有意幂等：任务已消失的 Job 不重试（见上注释）
	}
	if task.Status != model.TaskStatusPENDING {
		// 任务已处理/取消（完成联动取消 Job 失效兜底）：幂等空跑
		return nil
	}
	action, _ := job.Payload["action"].(string)
	switch model.TimeoutAction(action) {
	case model.TimeoutActionApprove:
		_, err = w.runtime.Approve(ctx, engineruntime.ApproveInput{
			TenantID: job.TenantID, TaskID: job.TaskID,
			OperatorMemberID: 0, Comment: timeoutAutoComment, AutoTimeout: true,
		})
		return err
	case model.TimeoutActionReject:
		_, err = w.runtime.Reject(ctx, engineruntime.RejectInput{
			TenantID: job.TenantID, TaskID: job.TaskID,
			OperatorMemberID: 0, Comment: timeoutAutoComment, AutoTimeout: true,
		})
		return err
	default:
		return &unknownTimeoutActionError{Action: action}
	}
}

// handleReminder 待办提醒：任务仍 PENDING 时落 REMINDER 操作流水（时间线
// 可见）并发布 workflow.task.reminder 事件（Phase 6：经适配器同事务写通知
// 域 Outbox，由 Dispatcher 扇出站内信给任务参与人）；已终态则空跑。
// V1 单次提醒，不循环。
func (w *JobWorker) handleReminder(ctx context.Context, job *model.Job) error {
	task, err := w.tasks.FindTaskByIDForUpdate(ctx, job.TenantID, job.TaskID)
	if err != nil {
		w.logger.Warnf("reminder job %d: task %d not found, skip", job.ID, job.TaskID)
		return nil //nolint:nilerr // 有意幂等：任务已消失的 Job 不重试（见上注释）
	}
	if task.Status != model.TaskStatusPENDING {
		return nil
	}
	if err := w.operations.AppendOperation(ctx, &model.Operation{
		TenantID:         job.TenantID,
		InstanceID:       job.InstanceID,
		TaskID:           job.TaskID,
		OperatorMemberID: reminderOperator,
		Type:             model.OperationTypeReminder,
		Payload:          jobPayloadSnapshot(job),
	}); err != nil {
		return err
	}
	if w.publisher != nil {
		_ = w.publisher.PublishInTx(ctx, provider.Event{
			EventName:     event.TaskReminder,
			TenantID:      job.TenantID,
			InstanceID:    job.InstanceID,
			TaskID:        job.TaskID,
			ActorMemberID: reminderOperator,
		})
	}
	return nil
}

// recordRetryFailure 失败重试记账（独立事务：执行事务已回滚）：
// retry_count 递增 → 未超限回队 PENDING + 退避，超限 FAILED + last_error；
// service.invoke 追加 SERVICE/FAILED 操作流水（时间线可见，第 19 章诊断）。
func (w *JobWorker) recordRetryFailure(ctx context.Context, jobID uint, cause error) error {
	lastError := truncateError(cause)
	return w.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		job, err := w.jobs.FindJobByID(tctx, 0, jobID)
		if err != nil {
			return err
		}
		// 记账路径同样上下文化 Job 租户（流水/回写走租户 Callback 过滤）
		tctx = contextx.NewTenantContext(tctx, job.TenantID)
		job.RetryCount++
		job.LastError = lastError
		if job.RetryCount >= job.MaxRetryCount {
			job.Status = model.JobStatusFAILED
		} else {
			job.Status = model.JobStatusPENDING
			job.ExecuteAt = time.Now().Add(defaultRetryBackoff)
		}
		if err := w.jobs.SaveJob(tctx, job); err != nil {
			return err
		}
		if job.Type != model.JobTypeServiceInvoke {
			return nil
		}
		return w.operations.AppendOperation(tctx, &model.Operation{
			TenantID:   job.TenantID,
			InstanceID: job.InstanceID,
			TaskID:     0,
			Type:       model.OperationTypeService,
			Payload: map[string]any{
				"status": "FAILED", "retryCount": job.RetryCount, "error": lastError,
			},
		})
	})
}

// jobPayloadSnapshot 提醒流水载荷（脱敏快照：仅保留诊断所需字段）。
func jobPayloadSnapshot(job *model.Job) map[string]any {
	nodeKey, _ := job.Payload["nodeKey"].(string)
	return map[string]any{"nodeKey": nodeKey, "jobId": job.ID}
}

// truncateError 错误摘要截断（last_error 只入诊断，防超长文本入库）。
func truncateError(err error) string {
	text := err.Error()
	if len(text) > 512 {
		text = text[:512]
	}
	return text
}

// unknownTimeoutActionError 未知超时动作（防御：校验器已拦截）。
type unknownTimeoutActionError struct{ Action string }

func (e *unknownTimeoutActionError) Error() string {
	raw, _ := json.Marshal(e.Action)
	return "unknown timeout action: " + string(raw)
}
