package service

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// DefaultDowngradeInterval 到期降级扫描默认周期。页面与配额守卫已做读时
// 到期投影，任务只负责把投影物化为免费订阅与旧字段，分钟级延迟即可
const DefaultDowngradeInterval = 5 * time.Minute

// EditionWorker 订阅到期降级任务：周期扫描「活动且已过 ends_at」的订阅，
// 单事务内降级为免费订阅并同步兼容投影。幂等由「订阅行 FOR UPDATE 重检 +
// 条件状态迁移 + active 部分唯一索引」三层保证（设计 4.3.1）
type EditionWorker struct {
	service  EditionService
	interval time.Duration
	logger   *logrus.Logger
}

func NewEditionWorker(service EditionService, interval time.Duration, logger *logrus.Logger) *EditionWorker {
	if interval <= 0 {
		interval = DefaultDowngradeInterval
	}
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &EditionWorker{service: service, interval: interval, logger: logger}
}

// Run 启动降级循环，ctx 取消时退出（随服务优雅停机）
func (w *EditionWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.logger.Infof("edition downgrade worker started, interval: %s", w.interval)
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("edition downgrade worker stopped")
			return
		case <-ticker.C:
			downgraded, err := w.service.DowngradeExpiredOnce(ctx)
			if err != nil {
				// 部分租户失败仅记日志，已成功者不回滚，下一轮重试剩余
				w.logger.Warnf("edition downgrade round finished with error: %v", err)
			}
			if downgraded > 0 {
				w.logger.Infof("edition downgrade round downgraded %d subscription(s)", downgraded)
			}
		}
	}
}
