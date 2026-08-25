package service

import (
	"context"
	"time"

	"evolyn/internal/platform/auth/security/repository"

	"github.com/sirupsen/logrus"
)

// 会话清理参数（ADR-009 SEC-3-4）
const (
	defaultSessionCleanupInterval = time.Hour
	// sessionRetention 撤销/过期会话的保留期：删除前保留 7 天供安全审计回溯
	sessionRetention = 7 * 24 * time.Hour
)

// sessionCleanerRepo 清理所需的最小仓储面
type sessionCleanerRepo interface {
	DeleteExpiredBefore(ctx context.Context, before time.Time) (int64, error)
}

// SessionCleanupWorker 会话清理任务：周期删除「已过期」或「已撤销且过保留期」
// 的历史会话行。活跃集合（revoked_at IS NULL 且未过期）永不触碰
type SessionCleanupWorker struct {
	sessions sessionCleanerRepo
	interval time.Duration
	logger   *logrus.Logger
}

func NewSessionCleanupWorker(sessions repository.SessionRepository, interval time.Duration, logger *logrus.Logger) *SessionCleanupWorker {
	if interval <= 0 {
		interval = defaultSessionCleanupInterval
	}
	return &SessionCleanupWorker{sessions: sessions, interval: interval, logger: logger}
}

// Run 随服务启动，ctx 取消即退出（与租户注销清理/文件清理同模式）
func (w *SessionCleanupWorker) Run(ctx context.Context) {
	if w == nil || w.sessions == nil {
		// 防御性退出：装配遗漏不应让服务在后台 goroutine 中 panic；正常路径由
		// server.New 显式创建 worker，见 server.go。
		return
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 删除口径：过期超过保留期，或撤销超过保留期
			deleted, err := w.sessions.DeleteExpiredBefore(ctx, time.Now().Add(-sessionRetention))
			if err != nil {
				w.logger.Warnf("session cleanup: %v", err)
				continue
			}
			if deleted > 0 {
				w.logger.Infof("session cleanup: removed %d stale sessions", deleted)
			}
		}
	}
}
