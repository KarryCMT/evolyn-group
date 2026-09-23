// Package worker 将标签域批量任务接入 Asynq；Handler 只负责可靠投递、
// 租户上下文恢复与重试，业务状态机仍由 label service 维护。
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"evolyn/internal/config"
	"evolyn/internal/contextx"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
)

const batchTaskType = "label:render:batch"

type BatchProcessor interface {
	ProcessBatch(ctx context.Context, taskCode string) error
	FailBatch(ctx context.Context, taskCode string, cause error) error
}

type batchPayload struct {
	TenantID uint   `json:"tenantId"`
	TaskCode string `json:"taskId"`
}

type Runtime struct {
	client    *asynq.Client
	server    *asynq.Server
	processor BatchProcessor
	logger    *logrus.Logger
}

func (r *Runtime) Available() bool { return r != nil && r.client != nil && r.server != nil }

func NewRuntime(conf config.RedisConfig, processor BatchProcessor, logger *logrus.Logger) *Runtime {
	if !conf.Enable || processor == nil {
		return nil
	}
	redis := asynq.RedisClientOpt{Addr: fmt.Sprintf("%s:%d", conf.Host, conf.Port), Password: conf.Password}
	return &Runtime{
		client: asynq.NewClient(redis),
		server: asynq.NewServer(redis, asynq.Config{
			Concurrency: 2,
			Queues:      map[string]int{"label": 1},
		}),
		processor: processor,
		logger:    logger,
	}
}

func (r *Runtime) Enqueue(ctx context.Context, tenantID uint, taskCode string) error {
	if r == nil || r.client == nil || tenantID == 0 || taskCode == "" {
		return fmt.Errorf("label batch queue unavailable")
	}
	payload, err := json.Marshal(batchPayload{TenantID: tenantID, TaskCode: taskCode})
	if err != nil {
		return err
	}
	_, err = r.client.EnqueueContext(ctx, asynq.NewTask(batchTaskType, payload),
		asynq.Queue("label"), asynq.MaxRetry(3), asynq.Unique(10*time.Minute))
	return err
}

func (r *Runtime) Run(ctx context.Context) {
	if r == nil || r.server == nil {
		return
	}
	mux := asynq.NewServeMux()
	mux.HandleFunc(batchTaskType, r.handleBatch)
	if err := r.server.Start(mux); err != nil {
		r.logger.Errorf("label asynq worker start failed: %v", err)
		return
	}
	<-ctx.Done()
	r.server.Shutdown()
}

func (r *Runtime) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

func (r *Runtime) handleBatch(ctx context.Context, task *asynq.Task) error {
	var payload batchPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("decode label batch payload: %w", asynq.SkipRetry)
	}
	if payload.TenantID == 0 || payload.TaskCode == "" {
		return fmt.Errorf("invalid label batch payload: %w", asynq.SkipRetry)
	}
	tenantCtx := contextx.NewTenantContext(ctx, payload.TenantID)
	err := r.processor.ProcessBatch(tenantCtx, payload.TaskCode)
	if err == nil {
		return nil
	}
	retryCount, retryOK := asynq.GetRetryCount(ctx)
	maxRetry, maxOK := asynq.GetMaxRetry(ctx)
	if retryOK && maxOK && retryCount >= maxRetry {
		if finishErr := r.processor.FailBatch(tenantCtx, payload.TaskCode, err); finishErr != nil {
			return fmt.Errorf("finish exhausted label batch: %v (original: %w)", finishErr, err)
		}
	}
	return err
}
