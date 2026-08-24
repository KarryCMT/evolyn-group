package service

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// UploadCleanupWorker 周期删除到期上传会话的残留对象和元数据，防止预留存储
// 配额永久占用。它只依赖 FileService，便于后续替换为队列或分布式定时任务。
type UploadCleanupWorker struct {
	service  FileService
	interval time.Duration
	logger   *logrus.Logger
}

func NewUploadCleanupWorker(service FileService, interval time.Duration, logger *logrus.Logger) *UploadCleanupWorker {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &UploadCleanupWorker{service: service, interval: interval, logger: logger}
}

func (w *UploadCleanupWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.service.CleanupExpired(ctx); err != nil {
				w.logger.Warnf("expired file upload cleanup failed: %v", err)
			}
		}
	}
}
