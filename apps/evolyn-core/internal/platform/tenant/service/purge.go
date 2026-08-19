package service

import (
	"context"
	"time"

	"evolyn/internal/platform/tenant/repository"

	"github.com/sirupsen/logrus"
)

// DefaultPurgeInterval 注销数据清理扫描默认周期
const DefaultPurgeInterval = time.Hour

// PurgeWorker 注销租户数据清理任务（FIX-012）：
// 周期扫描「deleted 且保留期已过且未清理」的租户，物理清理其业务数据
// 并落墓碑标记。一期只清 IAM 域四表与关系表，后续模块（应用/表单数据）
// 落地后在 PurgeTenantData 中扩展
type PurgeWorker struct {
	tenantRepo repository.TenantRepository
	interval   time.Duration
	logger     *logrus.Logger
}

func NewPurgeWorker(tenantRepo repository.TenantRepository, interval time.Duration, logger *logrus.Logger) *PurgeWorker {
	if interval <= 0 {
		interval = DefaultPurgeInterval
	}
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &PurgeWorker{tenantRepo: tenantRepo, interval: interval, logger: logger}
}

// Run 启动清理循环，ctx 取消时退出（随服务优雅停机）
func (w *PurgeWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.logger.Infof("tenant purge worker started, interval: %s", w.interval)
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("tenant purge worker stopped")
			return
		case <-ticker.C:
			// 扫描周期到即执行；单次失败仅记日志，等待下轮重试
			if err := w.PurgeOnce(ctx); err != nil {
				w.logger.Warnf("tenant purge round failed: %v", err)
			}
		}
	}
}

// PurgeOnce 执行一轮清理：扫描到期租户逐个清理，单个失败不阻断其余
func (w *PurgeWorker) PurgeOnce(ctx context.Context) error {
	tenants, err := w.tenantRepo.ListPurgeable(ctx, time.Now())
	if err != nil {
		return err
	}
	if len(tenants) == 0 {
		return nil
	}

	for i := range tenants {
		tenant := tenants[i]
		if err := w.tenantRepo.PurgeTenantData(ctx, tenant.ID); err != nil {
			// 单租户失败继续处理其余；事务内已回滚，下一轮重试
			w.logger.Warnf("purge tenant %s(%d) failed: %v", tenant.Code, tenant.ID, err)
			continue
		}
		w.logger.Infof("tenant %s(%d) purged (retention ended)", tenant.Code, tenant.ID)
	}
	return nil
}
