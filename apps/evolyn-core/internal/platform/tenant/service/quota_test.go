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
			svc := NewQuotaService(fakeTenantReader{tenant}, fakeMemberCounter{count: tc.count})

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
	svc := NewQuotaService(fakeTenantReader{tenant}, fakeMemberCounter{})

	// 一期仅支持 members，其余键显式报错而非静默放行
	_, err := svc.Usage(context.Background(), 1, "apps")
	assert.Error(t, err)
}
