package service

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// Worker 轮询间隔默认值（装配层可经 config.notification 覆盖）
const (
	DefaultOutboxInterval    = 5 * time.Second
	DefaultRetentionInterval = 6 * time.Hour
)

// OutboxWorker Outbox 物化任务：周期领取待处理事件并扇出站内信；单轮失败
// 仅告警，下轮自然重试（幂等），不中断 Worker。
type OutboxWorker struct {
	dispatcher *Dispatcher
	interval   time.Duration
	logger     *logrus.Logger
}

func NewOutboxWorker(dispatcher *Dispatcher, interval time.Duration, logger *logrus.Logger) *OutboxWorker {
	if interval <= 0 {
		interval = DefaultOutboxInterval
	}
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &OutboxWorker{dispatcher: dispatcher, interval: interval, logger: logger}
}

func (w *OutboxWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	w.logger.Infof("notification outbox worker started, interval: %s", w.interval)
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("notification outbox worker stopped")
			return
		case <-ticker.C:
			// Worker 自起 context：无租户上下文（跨租户领取），随服务生命周期取消
			processed, err := w.dispatcher.DispatchBatch(ctx)
			if err != nil {
				w.logger.WithError(err).Warn("notification outbox dispatch batch failed")
				continue
			}
			if processed > 0 {
				w.logger.Debugf("notification outbox dispatched %d events", processed)
			}
		}
	}
}

// RetentionWorker 过期消息清理任务：先删收件箱行再删无引用消息，分批受控。
type RetentionWorker struct {
	dispatcher *Dispatcher
	interval   time.Duration
	logger     *logrus.Logger
}

func NewRetentionWorker(dispatcher *Dispatcher, interval time.Duration, logger *logrus.Logger) *RetentionWorker {
	if interval <= 0 {
		interval = DefaultRetentionInterval
	}
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &RetentionWorker{dispatcher: dispatcher, interval: interval, logger: logger}
}

func (w *RetentionWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	w.logger.Infof("notification retention worker started, interval: %s", w.interval)
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("notification retention worker stopped")
			return
		case <-ticker.C:
			if err := w.dispatcher.CleanupExpiredOnce(ctx); err != nil {
				w.logger.WithError(err).Warn("notification retention cleanup failed")
			}
		}
	}
}
