// Package worker 表单域后台 Worker（物理表存储方案 §8/§13）。
//
// DDL Job Worker：FOR UPDATE SKIP LOCKED 小批领取（DDL 串行，每轮一条），
// claim+执行+回写同事务——crash 时整体回滚为 PENDING（天然 crash recovery，
// 与 wf_job Worker 同口径）；失败重试在独立事务记账退避回队，超限转终态
// FAILED 并把 SchemaVersion/Storage 同步落败。advisory lock(form) 防同表单
// 并发 DDL；checksum 在领取后复核，模型被篡改即拒绝执行。
package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"evolyn/internal/engine/data/storage"
	"evolyn/internal/infrastructure/dynamicddl"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/form/repository"
	"evolyn/internal/platform/httpx"

	"github.com/sirupsen/logrus"
)

// DDL Worker 调度常量：DDL 是低频重操作——小批串行 + 分钟级退避即可，
// 队列长度经监控指标观察后再调整。
const (
	DefaultDDLPollInterval = 5 * time.Second
	DefaultDDLMaxRetries   = 3
	DefaultDDLBaseBackoff  = 30 * time.Second
)

// TxManager 事务窄端口（装配层由 infrastructure.TxManager 适配，与
// workflow JobWorker 同模式——worker 依赖接口便于单测桩）。
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// DDExecutor DDL 执行窄端口（装配层由 dynamicddl.Executor 适配）。
type DDExecutor interface {
	AcquireAdvisoryLock(ctx context.Context, tenantID, storageID uint) error
	ExecutePlan(ctx context.Context, plan storage.Plan, model *storage.StorageModel, commentCtx dynamicddl.TableCommentContext) error
}

// DDLJobWorker 物理表 DDL 执行 Worker。
type DDLJobWorker struct {
	pollInterval time.Duration
	maxRetries   int
	baseBackoff  time.Duration
	logger       *logrus.Logger

	tx             TxManager
	jobs           repository.FormDDLJobRepository
	schemaVersions repository.StorageSchemaVersionRepository
	storages       repository.FormStorageRepository
	versions       repository.FormVersionRepository
	forms          repository.FormRepository
	executor       DDExecutor
}

// NewDDLJobWorker 构造 Worker（interval<=0 / logger=nil 取默认）。
func NewDDLJobWorker(
	tx TxManager,
	jobs repository.FormDDLJobRepository,
	schemaVersions repository.StorageSchemaVersionRepository,
	storages repository.FormStorageRepository,
	versions repository.FormVersionRepository,
	forms repository.FormRepository,
	executor DDExecutor,
	logger *logrus.Logger,
) *DDLJobWorker {
	w := &DDLJobWorker{
		pollInterval:   DefaultDDLPollInterval,
		maxRetries:     DefaultDDLMaxRetries,
		baseBackoff:    DefaultDDLBaseBackoff,
		logger:         logrus.StandardLogger(),
		tx:             tx,
		jobs:           jobs,
		schemaVersions: schemaVersions,
		storages:       storages,
		versions:       versions,
		forms:          forms,
		executor:       executor,
	}
	if w.pollInterval <= 0 {
		w.pollInterval = DefaultDDLPollInterval
	}
	if w.maxRetries <= 0 {
		w.maxRetries = DefaultDDLMaxRetries
	}
	if w.baseBackoff <= 0 {
		w.baseBackoff = DefaultDDLBaseBackoff
	}
	if logger != nil {
		w.logger = logger
	}
	return w
}

// Run 主循环：ticker 轮询到期 Job；ctx 取消即退出。
func (w *DDLJobWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processRound(ctx)
		}
	}
}

// ProcessOnce 立即执行一轮领取与处理（导出入口：集成测试与运维手动触发
// 使用；返回是否处理了 Job）。
func (w *DDLJobWorker) ProcessOnce(ctx context.Context) (bool, error) {
	return w.processOne(ctx)
}

func (w *DDLJobWorker) processRound(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		processed, err := w.processOne(ctx)
		if err != nil {
			w.logger.WithError(err).Warn("form ddl worker: process job failed")
			return
		}
		if !processed {
			return
		}
	}
}

// processOne 领取并执行一条 Job；返回是否处理了 Job（false=队列空）。
func (w *DDLJobWorker) processOne(ctx context.Context) (bool, error) {
	var jobID uint
	execErr := w.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		claimed, err := w.jobs.ClaimNextDue(tctx, time.Now(), 1)
		if err != nil {
			return err
		}
		if len(claimed) == 0 {
			return nil
		}
		job := claimed[0]
		jobID = job.ID
		if err := w.executeClaimedJob(tctx, &job); err != nil {
			return err
		}
		return nil
	})
	if execErr != nil {
		if jobID == 0 {
			return false, execErr // 领取本身失败
		}
		// 执行事务已回滚（Job 回到 PENDING）：独立事务记账重试或终态失败
		if retryErr := w.recordFailure(ctx, jobID, execErr); retryErr != nil {
			return true, retryErr
		}
		return true, nil
	}
	return jobID != 0, nil
}

// executeClaimedJob 执行已领取（PROCESSING）的 Job；任何错误都使整个事务
// 回滚——DDL 语句与元数据推进同进退，无半应用状态。
func (w *DDLJobWorker) executeClaimedJob(ctx context.Context, job *model.FormDDLJob) error {
	schemaVersion, err := w.schemaVersions.GetByID(ctx, job.StorageSchemaVersionID)
	if err != nil {
		return err
	}
	if schemaVersion.State != model.SchemaVersionPending {
		return fmt.Errorf("storage schema version %d state %s is not pending", schemaVersion.ID, schemaVersion.State)
	}
	var plan storage.Plan
	if err := json.Unmarshal([]byte(schemaVersion.Plan), &plan); err != nil {
		return fmt.Errorf("storage plan decode: %w", err)
	}
	var parsedModel storage.StorageModel
	if err := json.Unmarshal([]byte(schemaVersion.Model), &parsedModel); err != nil {
		return fmt.Errorf("storage model decode: %w", err)
	}
	if err := parsedModel.Validate(); err != nil {
		return fmt.Errorf("storage model invalid: %w", err)
	}
	// checksum 复核：持久化模型与计划必须自洽，防篡改/防半写
	checksum, err := storage.ModelChecksum(&parsedModel)
	if err != nil {
		return err
	}
	if checksum != schemaVersion.Checksum {
		return fmt.Errorf("storage model checksum mismatch: %s != %s", checksum, schemaVersion.Checksum)
	}

	binding, err := w.storages.LockByID(ctx, schemaVersion.StorageID)
	if err != nil {
		return err
	}
	// advisory lock：同表单（租户+存储复合键）串行化 DDL，事务结束自动释放
	if err := w.executor.AcquireAdvisoryLock(ctx, job.TenantID, binding.ID); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	formVersion, err := w.versions.GetByID(ctx, schemaVersion.FormVersionID)
	if err != nil {
		return err
	}
	form, err := w.forms.GetByID(ctx, formVersion.FormID)
	if err != nil {
		return err
	}
	if err := w.executor.ExecutePlan(ctx, plan, &parsedModel, dynamicddl.TableCommentContext{
		TenantID:  job.TenantID,
		StorageID: binding.ID,
		FormCode:  form.Code,
	}); err != nil {
		return err
	}
	// 同事务推进：模型版本 APPLIED → 存储 READY → 表单发布指针（运行时
	// 自此刻起可使用新快照；方案 §8 时序）。
	if err := w.schemaVersions.MarkApplied(ctx, schemaVersion.ID); err != nil {
		return err
	}
	if err := w.storages.MarkReady(ctx, binding.ID, schemaVersion.ID); err != nil {
		return err
	}
	if err := w.forms.MarkPublished(ctx, form.ID, formVersion.ID, formVersion.VersionNo); err != nil {
		return err
	}
	finished := model.JSONTimeOf(time.Now())
	job.Status = model.DDLJobSucceeded
	job.FinishedAt = &finished
	return w.jobs.SaveJob(ctx, job)
}

// recordFailure 独立事务记账：未超限退避回 PENDING（重试幂等——DDL 语句
// 全部 IF NOT EXISTS），超限转 FAILED 并把版本/存储同步落败（旧发布版本
// 继续可运行，新版本不成为 latest——方案 §16.2）。
func (w *DDLJobWorker) recordFailure(ctx context.Context, jobID uint, execErr error) error {
	return w.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		job, err := w.jobs.GetByID(tctx, jobID)
		if err != nil {
			return err
		}
		// claim 事务已回滚，Job 此刻应为 PENDING
		if job.Status != model.DDLJobPending {
			return fmt.Errorf("job %d status %s after rollback, expected PENDING", jobID, job.Status)
		}
		job.RetryCount++
		job.LastErrorCode = ddlErrorCode(execErr)
		job.LastErrorDetail = execErr.Error()
		if job.RetryCount > w.maxRetries {
			job.Status = model.DDLJobFailed
			finished := model.JSONTimeOf(time.Now())
			job.FinishedAt = &finished
			if err := w.jobs.SaveJob(tctx, job); err != nil {
				return err
			}
			if err := w.schemaVersions.MarkFailed(tctx, job.StorageSchemaVersionID, job.LastErrorCode, job.LastErrorDetail); err != nil {
				return err
			}
			schemaVersion, err := w.schemaVersions.GetByID(tctx, job.StorageSchemaVersionID)
			if err != nil {
				return err
			}
			return w.storages.MarkFailed(tctx, schemaVersion.StorageID)
		}
		job.NextAttemptAt = model.JSONTimeOf(time.Now().Add(w.baseBackoff << uint(job.RetryCount)))
		return w.jobs.SaveJob(tctx, job)
	})
}

// ddlErrorCode 从执行错误提取稳定错误码（BizError 直取，其余归为执行失败）。
func ddlErrorCode(err error) string {
	var biz *httpx.BizError
	if errors.As(err, &biz) {
		return biz.Code
	}
	return "FORM_STORAGE_DDL_FAILED"
}
