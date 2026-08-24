package service

import (
	"context"
	"fmt"
	"net/http"

	"evolyn/internal/platform/httpx"
	tenantmodel "evolyn/internal/platform/tenant/model"
)

// ErrQuotaExceeded 配额超限稳定业务错误码（FIX-011，ADR-008 起承载于 BizError）：
// 对外只见「配额已用尽」，用量/上限数值经 Wrap 只入日志（避免内部数据泄漏）；
// 应用/表单/存储/流程次数配额复用同一错误
var ErrQuotaExceeded = httpx.NewBiz("QUOTA_EXCEEDED", "配额已用尽，请升级套餐或联系管理员", http.StatusForbidden)

// quota 依赖的最小仓储面（窄接口便于单测与解耦，Go 惯例）
type (
	// tenantReader 按需加载租户（含套餐与配额覆盖）
	tenantReader interface {
		GetByID(ctx context.Context, id uint) (*tenantmodel.Tenant, error)
	}
	// tenantLocker 配额并发校验用的租户行锁（由 TenantRepository 实现）
	tenantLocker interface {
		LockByID(ctx context.Context, id uint) error
	}
	// memberCounter 租户有效成员数
	memberCounter interface {
		CountByTenant(ctx context.Context, tenantID uint) (int64, error)
	}
	// applicationCounter 计费应用数（由应用域仓储实现，装配期注入；
	// 计数口径见应用域 CountBillableByTenant：软删行除外全量计数）
	applicationCounter interface {
		CountBillableByTenant(ctx context.Context, tenantID uint) (int64, error)
	}
	// storageCounter 存储用量字节数（上传会话预留 + 已确认对象）；由文件域
	// 仓储实现。计量单位为字节，但套餐配置仍保持 storage_gb 的 GB 语义。
	storageCounter interface {
		CountStorageBytes(ctx context.Context, tenantID uint) (int64, error)
	}
)

// QuotaService 配额执行服务（FIX-011）：统一入口校验「当前用量是否仍在上限内」。
// 生效值 = 租户覆盖 > 套餐默认（-1 不限量、0 禁用）；members/apps 键各自
// 依赖对应计量仓储，未注入的键 Check 返回未支持错误
type QuotaService interface {
	// Check 校验指定配额键还能否新增一个单位，超限返回 ErrQuotaExceeded
	Check(ctx context.Context, tenantID uint, key string) error
	// Usage 当前用量（members/apps）
	Usage(ctx context.Context, tenantID uint, key string) (int64, error)
	// CheckAndReserve 并发安全的「校验+占位」（应用域 §10）：调用方必须已
	// 开启事务（TxManager.WithinTransaction）——先 FOR UPDATE 锁定租户行
	// 串行化同租户并发创建，再判限额，通过后 fn 在同一事务内完成资源写入，
	// 任一步失败随外层事务整体回滚。禁止绕过本方法做「事务外预检再插入」
	// （存在并发穿透），也不能在 Service 外自行加锁
	CheckAndReserve(ctx context.Context, tenantID uint, key string, fn func(ctx context.Context) error) error
}

// StorageQuotaService 是文件域使用的附加配额能力。它与 QuotaService 分开，
// 以免既有业务服务的测试桩被无关方法强制扩展。
type StorageQuotaService interface {
	CheckAndReserveStorage(ctx context.Context, tenantID uint, bytes int64, fn func(ctx context.Context) error) error
}

type quotaService struct {
	tenants tenantReader
	locker  tenantLocker
	members memberCounter
	apps    applicationCounter
	storage storageCounter
}

// NewQuotaService 构造配额服务。locker/apps 为应用域落地后的扩展依赖
// （locker 由租户仓储自身实现），未接入时传 nil：apps 键不可用、
// CheckAndReserve 拒绝执行
func NewQuotaService(tenants tenantReader, locker tenantLocker, members memberCounter, apps applicationCounter, storages ...storageCounter) QuotaService {
	svc := &quotaService{
		tenants: tenants,
		locker:  locker,
		members: members,
		apps:    apps,
	}
	if len(storages) > 0 {
		svc.storage = storages[0]
	}
	return svc
}

func (s *quotaService) Check(ctx context.Context, tenantID uint, key string) error {
	limit, usage, err := s.limitAndUsage(ctx, tenantID, key)
	if err != nil {
		return err
	}

	// -1 不限量；0 禁用（一个都不允许）；正数即上限
	if limit < 0 {
		return nil
	}
	if usage >= limit {
		return httpx.Wrap(ErrQuotaExceeded, fmt.Errorf("tenant %d %s limit %d, used %d", tenantID, key, limit, usage))
	}
	return nil
}

func (s *quotaService) Usage(ctx context.Context, tenantID uint, key string) (int64, error) {
	_, usage, err := s.limitAndUsage(ctx, tenantID, key)
	return usage, err
}

// CheckAndReserve 实现见接口注释；fn 只在锁定+限额通过后执行，
// 锁与计数都在调用方事务 session 内（仓储经 ResolveDB 加入同一事务）
func (s *quotaService) CheckAndReserve(ctx context.Context, tenantID uint, key string, fn func(ctx context.Context) error) error {
	if s.locker == nil {
		return fmt.Errorf("quota locker not configured (CheckAndReserve unavailable)")
	}
	if err := s.locker.LockByID(ctx, tenantID); err != nil {
		return err
	}
	if err := s.Check(ctx, tenantID, key); err != nil {
		return err
	}
	return fn(ctx)
}

func (s *quotaService) CheckAndReserveStorage(ctx context.Context, tenantID uint, bytes int64, fn func(ctx context.Context) error) error {
	if bytes <= 0 {
		return fmt.Errorf("storage reservation bytes must be positive")
	}
	if s.locker == nil || s.storage == nil {
		return fmt.Errorf("storage quota counter not configured")
	}
	if err := s.locker.LockByID(ctx, tenantID); err != nil {
		return err
	}
	limit, usage, err := s.storageLimitAndUsage(ctx, tenantID)
	if err != nil {
		return err
	}
	if limit >= 0 && (usage > limit || bytes > limit-usage) {
		return httpx.Wrap(ErrQuotaExceeded, fmt.Errorf("tenant %d storage limit=%d used=%d requested=%d", tenantID, limit, usage, bytes))
	}
	return fn(ctx)
}

// limitAndUsage 取生效上限与当前用量。租户上下文可能不存在（运营/后台路径），
// 租户读取走仓储默认行为
func (s *quotaService) limitAndUsage(ctx context.Context, tenantID uint, key string) (int64, int64, error) {
	var usage int64
	var err error
	switch key {
	case tenantmodel.QuotaMembers:
		if s.members == nil {
			return 0, 0, fmt.Errorf("quota counter not configured: %s", key)
		}
		usage, err = s.members.CountByTenant(ctx, tenantID)
	case tenantmodel.QuotaApps:
		if s.apps == nil {
			return 0, 0, fmt.Errorf("quota counter not configured: %s", key)
		}
		usage, err = s.apps.CountBillableByTenant(ctx, tenantID)
	default:
		return 0, 0, fmt.Errorf("quota key not supported yet: %s", key)
	}
	if err != nil {
		return 0, 0, err
	}

	tenant, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return 0, 0, err
	}

	// 缺省回落 0（禁用）：无套餐默认的键不允许新增，显式配置才放开
	limit := tenant.Quotas.Get(tenant.Plan, key, 0)
	return limit, usage, nil
}

// storageLimitAndUsage 将 storage_gb 的配置转换为字节上限；-1 仍表示不限量。
func (s *quotaService) storageLimitAndUsage(ctx context.Context, tenantID uint) (int64, int64, error) {
	if s.storage == nil {
		return 0, 0, fmt.Errorf("quota counter not configured: %s", tenantmodel.QuotaStorageGB)
	}
	usage, err := s.storage.CountStorageBytes(ctx, tenantID)
	if err != nil {
		return 0, 0, err
	}
	tenant, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return 0, 0, err
	}
	limitGB := tenant.Quotas.Get(tenant.Plan, tenantmodel.QuotaStorageGB, 0)
	if limitGB < 0 {
		return -1, usage, nil
	}
	const gib = int64(1024 * 1024 * 1024)
	if limitGB > (int64(^uint64(0)>>1) / gib) {
		return 0, 0, fmt.Errorf("storage quota overflow: %d GB", limitGB)
	}
	return limitGB * gib, usage, nil
}
