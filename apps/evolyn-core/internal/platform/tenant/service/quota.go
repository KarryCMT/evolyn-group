package service

import (
	"context"
	"errors"
	"fmt"

	tenantmodel "evolyn/internal/platform/tenant/model"
)

// ErrQuotaExceeded 配额超限稳定业务错误码（FIX-011）：错误文本即对外错误码，
// 后续应用/表单/存储/流程次数配额复用同一错误
var ErrQuotaExceeded = errors.New("QUOTA_EXCEEDED")

// quota 依赖的最小仓储面（窄接口便于单测与解耦，Go 惯例）
type (
	// tenantReader 按需加载租户（含套餐与配额覆盖）
	tenantReader interface {
		GetByID(ctx context.Context, id uint) (*tenantmodel.Tenant, error)
	}
	// memberCounter 租户有效成员数
	memberCounter interface {
		CountByTenant(ctx context.Context, tenantID uint) (int64, error)
	}
)

// QuotaService 配额执行服务（FIX-011）：统一入口校验「当前用量是否仍在上限内」。
// 一期仅落地 members 配额验证执行架构；生效值 = 租户覆盖 > 套餐默认（-1 不限量、0 禁用）
type QuotaService interface {
	// Check 校验指定配额键还能否新增一个单位，超限返回 ErrQuotaExceeded
	Check(ctx context.Context, tenantID uint, key string) error
	// Usage 当前用量（一期仅 members）
	Usage(ctx context.Context, tenantID uint, key string) (int64, error)
}

type quotaService struct {
	tenants tenantReader
	members memberCounter
}

func NewQuotaService(tenants tenantReader, members memberCounter) QuotaService {
	return &quotaService{tenants: tenants, members: members}
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
		return fmt.Errorf("%w: tenant %d %s limit %d, used %d", ErrQuotaExceeded, tenantID, key, limit, usage)
	}
	return nil
}

func (s *quotaService) Usage(ctx context.Context, tenantID uint, key string) (int64, error) {
	_, usage, err := s.limitAndUsage(ctx, tenantID, key)
	return usage, err
}

// limitAndUsage 取生效上限与当前用量。租户上下文可能不存在（运营/后台路径），
// 租户读取走仓储默认行为
func (s *quotaService) limitAndUsage(ctx context.Context, tenantID uint, key string) (int64, int64, error) {
	if key != tenantmodel.QuotaMembers {
		return 0, 0, fmt.Errorf("quota key not supported yet: %s", key)
	}

	tenant, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return 0, 0, err
	}

	usage, err := s.members.CountByTenant(ctx, tenantID)
	if err != nil {
		return 0, 0, err
	}

	// 缺省回落 0（禁用）：无套餐默认的键不允许新增，显式配置才放开
	limit := tenant.Quotas.Get(tenant.Plan, key, 0)
	return limit, usage, nil
}
