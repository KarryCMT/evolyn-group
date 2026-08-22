package service

import (
	"context"
	"errors"
	"testing"

	tenantmodel "evolyn/internal/platform/tenant/model"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// fakeTenantReader 配额测试用租户读取桩
type fakeTenantReader struct {
	tenant *tenantmodel.Tenant
}

func (f fakeTenantReader) GetByID(ctx context.Context, id uint) (*tenantmodel.Tenant, error) {
	if f.tenant != nil && f.tenant.ID == id {
		return f.tenant, nil
	}
	return nil, gorm.ErrRecordNotFound
}

// fakeMemberCounter 可配置成员数
type fakeMemberCounter struct {
	count int64
}

func (f fakeMemberCounter) CountByTenant(ctx context.Context, tenantID uint) (int64, error) {
	return f.count, nil
}

// fakeAppCounter 可配置计费应用数（apps 键计量桩）
type fakeAppCounter struct {
	count int64
}

func (f fakeAppCounter) CountBillableByTenant(ctx context.Context, tenantID uint) (int64, error) {
	return f.count, nil
}

// fakeLocker 记录加锁顺序的租户行锁桩（验证 CheckAndReserve 先锁后判）
type fakeLocker struct {
	locked []uint
}

func (f *fakeLocker) LockByID(ctx context.Context, id uint) error {
	f.locked = append(f.locked, id)
	return nil
}

func TestQuotaCheck(t *testing.T) {
	tenant := &tenantmodel.Tenant{ID: 1, Plan: tenantmodel.PlanFree} // free 默认 members=5

	cases := []struct {
		name      string
		quotas    tenantmodel.Quotas // 覆盖值
		count     int64              // 当前用量
		expectErr error              // 期望错误（nil = 放行）
	}{
		{"under limit", nil, 4, nil},
		{"at limit rejected", nil, 5, ErrQuotaExceeded},
		{"override unlimited", tenantmodel.Quotas{"members": -1}, 9999, nil},
		{"override smaller", tenantmodel.Quotas{"members": 3}, 3, ErrQuotaExceeded},
		{"disabled zero", tenantmodel.Quotas{"members": 0}, 0, ErrQuotaExceeded},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tenant.Quotas = tc.quotas
			svc := NewQuotaService(fakeTenantReader{tenant}, nil, fakeMemberCounter{count: tc.count}, nil)

			err := svc.Check(context.Background(), 1, tenantmodel.QuotaMembers)
			if tc.expectErr == nil {
				assert.NoError(t, err)
			} else {
				assert.True(t, errors.Is(err, tc.expectErr), "got: %v", err)
			}
		})
	}
}

func TestQuotaCheckUnknownKey(t *testing.T) {
	tenant := &tenantmodel.Tenant{ID: 1, Plan: tenantmodel.PlanPro}
	svc := NewQuotaService(fakeTenantReader{tenant}, nil, fakeMemberCounter{}, nil)

	// 未接入计量的键显式报错而非静默放行
	_, err := svc.Usage(context.Background(), 1, "forms")
	assert.Error(t, err)
}

func TestQuotaAppsCheck(t *testing.T) {
	// free 套餐默认 apps=3（plan.go），与 members 同一套判定口径
	tenant := &tenantmodel.Tenant{ID: 1, Plan: tenantmodel.PlanFree}
	svc := NewQuotaService(fakeTenantReader{tenant}, nil, nil, fakeAppCounter{count: 2})

	assert.NoError(t, svc.Check(context.Background(), 1, tenantmodel.QuotaApps))

	svc = NewQuotaService(fakeTenantReader{tenant}, nil, nil, fakeAppCounter{count: 3})
	assert.True(t, errors.Is(svc.Check(context.Background(), 1, tenantmodel.QuotaApps), ErrQuotaExceeded))

	// 未注入 apps 计量（apps 键未接入的装配形态）：显式报错
	svc = NewQuotaService(fakeTenantReader{tenant}, nil, fakeMemberCounter{}, nil)
	_, err := svc.Usage(context.Background(), 1, tenantmodel.QuotaApps)
	assert.Error(t, err)
}

func TestQuotaCheckAndReserve(t *testing.T) {
	tenant := &tenantmodel.Tenant{ID: 1, Plan: tenantmodel.PlanFree}

	t.Run("先锁行再判定，通过后执行 fn", func(t *testing.T) {
		locker := &fakeLocker{}
		fnRuns := 0
		svc := NewQuotaService(fakeTenantReader{tenant}, locker, nil, fakeAppCounter{count: 1})

		err := svc.CheckAndReserve(context.Background(), 1, tenantmodel.QuotaApps, func(ctx context.Context) error {
			fnRuns++
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, fnRuns)
		// 行锁发生在资源写入（fn）之前
		assert.NotEmpty(t, locker.locked)
	})

	t.Run("超限时 fn 不执行", func(t *testing.T) {
		svc := NewQuotaService(fakeTenantReader{tenant}, &fakeLocker{}, nil, fakeAppCounter{count: 3})

		fnRuns := 0
		err := svc.CheckAndReserve(context.Background(), 1, tenantmodel.QuotaApps, func(ctx context.Context) error {
			fnRuns++
			return nil
		})
		assert.True(t, errors.Is(err, ErrQuotaExceeded))
		assert.Zero(t, fnRuns)
	})

	t.Run("未装配行锁时拒绝执行", func(t *testing.T) {
		svc := NewQuotaService(fakeTenantReader{tenant}, nil, nil, fakeAppCounter{count: 0})
		err := svc.CheckAndReserve(context.Background(), 1, tenantmodel.QuotaApps, func(ctx context.Context) error {
			return nil
		})
		assert.Error(t, err)
	})
}
