package service

import (
	"context"
	"errors"
	"time"

	"evolyn/internal/platform/edition/model"
	tenantmodel "evolyn/internal/platform/tenant/model"

	"gorm.io/gorm"
)

// resolvedEdition 读时解析结果：订阅行 + 生效套餐版本快照 + 投影状态。
// fallback=true 表示「订阅已到期、降级任务未落库」的窗口期，资源与功能
// 权益按免费快照 + 仅有效 manual 覆盖解析（设计 4.3.1/4.4.1）
type resolvedEdition struct {
	sub      *model.TenantSubscription
	version  *model.EditionPlanVersion
	status   string // 出网投影状态：active / expired / legacy_pending_review
	fallback bool
}

// resolveCurrent 解析租户当前生效的订阅与快照：
//   - 无订阅记录（迁移前数据异常兜底）：按 pf_tenants.plan 合成 active 视图；
//   - legacy_pending_review：按订阅快照展示「有效期待确认」，不参与降级；
//   - active 且已过 ends_at：投影 expired，快照切到免费版（fallback）。
func (s *editionService) resolveCurrent(ctx context.Context, tenantID uint, now time.Time) (*resolvedEdition, error) {
	sub, err := s.repo.GetCurrentSubscription(ctx, tenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.synthesizeFromTenant(ctx, tenantID, now)
		}
		return nil, err
	}

	if sub.Status == model.SubscriptionLegacyPendingReview {
		version, err := s.loadVersion(ctx, sub.PlanVersionID)
		if err != nil {
			return nil, err
		}
		return &resolvedEdition{
			sub: sub, version: version,
			status: model.SubscriptionLegacyPendingReview,
		}, nil
	}

	// active：到期即时投影（读时判定，不等降级任务）
	if sub.EndsAt != nil && !sub.EndsAt.Time().After(now) {
		freeVersion, err := s.repo.GetLatestPublishedByCompat(ctx, tenantmodel.PlanFree)
		if err != nil {
			return nil, err
		}
		return &resolvedEdition{
			sub: sub, version: freeVersion,
			status:   model.SubscriptionExpired,
			fallback: true,
		}, nil
	}

	version, err := s.loadVersion(ctx, sub.PlanVersionID)
	if err != nil {
		return nil, err
	}
	return &resolvedEdition{sub: sub, version: version, status: model.SubscriptionActive}, nil
}

// loadVersion 加载套餐版本（缺行属数据异常：订阅引用了被物理删除的版本）
func (s *editionService) loadVersion(ctx context.Context, versionID uint) (*model.EditionPlanVersion, error) {
	version, _, err := s.repo.GetPlanVersionWithPlan(ctx, versionID)
	if err != nil {
		return nil, err
	}
	return version, nil
}

// synthesizeFromTenant 无订阅记录时的兜底视图：按 pf_tenants.plan 反查最新
// 已发布版本合成 active 订阅（不落库）。仅在迁移回填缺漏等异常场景触发，
// 保证页面可用；plan 非法时回落免费版
func (s *editionService) synthesizeFromTenant(ctx context.Context, tenantID uint, now time.Time) (*resolvedEdition, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	planCode := tenant.Plan
	if !tenantmodel.IsValidPlan(planCode) {
		planCode = tenantmodel.PlanFree
	}
	version, err := s.repo.GetLatestPublishedByCompat(ctx, planCode)
	if err != nil {
		return nil, err
	}
	return &resolvedEdition{
		sub: &model.TenantSubscription{
			TenantID:  tenantID,
			Status:    model.SubscriptionActive,
			GrantType: model.GrantSystem,
			StartsAt:  tenant.CreatedAt,
		},
		version: version,
		status:  model.SubscriptionActive,
	}, nil
}

// effectiveLimits 解析各资源键的生效上限与来源：快照规则为底，覆盖按
// manual > trial > legacy 同键优先级取先命中（manual 是最新运营意图）。
// manualOnly=true 用于到期窗口（GuardLimit 与页面 fallback 同口径）：
// 仅保留有效 manual 覆盖，trial 已随订阅失效、legacy 残留不得放大上限
func effectiveLimits(
	version *model.EditionPlanVersion,
	overrides []model.TenantEntitlementOverride,
	manualOnly bool,
) (map[string]int64, map[string]string) {
	limits := make(map[string]int64, len(version.Entitlements.Resources))
	sources := make(map[string]string, len(version.Entitlements.Resources))
	for _, r := range version.Entitlements.Resources {
		limits[r.Key] = r.Limit
		sources[r.Key] = model.LimitSourcePlanVersion
	}

	// 逐键选出优先级最高的覆盖后再统一应用，避免遍历顺序影响结果
	best := make(map[string]*model.TenantEntitlementOverride)
	for i := range overrides {
		o := &overrides[i]
		if _, known := limits[o.EntitlementKey]; !known {
			continue // 快照未声明的资源不放开
		}
		if manualOnly && o.Source != model.OverrideSourceManual {
			continue
		}
		if cur := best[o.EntitlementKey]; cur == nil ||
			overridePriority(o.Source) >= overridePriority(cur.Source) {
			best[o.EntitlementKey] = o
		}
	}
	for key, o := range best {
		limits[key] = o.Value
		if o.Source == model.OverrideSourceLegacy {
			sources[key] = model.LimitSourceLegacyQuota
		} else {
			sources[key] = model.LimitSourceTenantOverride
		}
	}
	return limits, sources
}

// overridePriority 同键覆盖取舍优先级：manual > trial > legacy
func overridePriority(source string) int {
	switch source {
	case model.OverrideSourceManual:
		return 3
	case model.OverrideSourceTrial:
		return 2
	default:
		return 1
	}
}

// projectCompatQuotas 计算旧 pf_tenants.quotas 投影（设计 4.4.1）：effective
// 为新键空间的生效值；逐键换算到旧键后，仅保留与套餐默认不同的项，缺省
// 键交由 DefaultQuotas 兜底——由此 QuotaService 的
// Quotas.Get(plan, key, 0) 读取结果与页面解析恒一致。storage_bytes 按
// 整 GiB 精确除法换算（发布/授予入口已拒绝非整 GiB 值，禁止取整）
func projectCompatQuotas(plan string, effective map[string]int64) tenantmodel.Quotas {
	projected := tenantmodel.Quotas{}
	for key, value := range effective {
		oldKey, oldValue, ok := compatKey(key, value)
		if !ok {
			continue // 未接线的旧键空间资源不投影
		}
		if def, exists := tenantmodel.DefaultQuotas(plan)[oldKey]; exists && def == oldValue {
			continue // 与套餐默认一致：不落覆盖
		}
		projected[oldKey] = oldValue
	}
	return projected
}

// compatKey 新权益键 → 旧配额键换算；仅五个已接线键参与兼容投影
func compatKey(key string, value int64) (string, int64, bool) {
	switch key {
	case model.ResourceStorage:
		return tenantmodel.QuotaStorageGB, model.StorageBytesToGB(value), true
	case model.ResourceApps:
		return tenantmodel.QuotaApps, value, true
	case model.ResourceMembers:
		return tenantmodel.QuotaMembers, value, true
	case model.ResourceForms:
		return tenantmodel.QuotaForms, value, true
	case model.ResourceWorkflowMo:
		return tenantmodel.QuotaWorkflowRunsMonth, value, true
	default:
		return "", 0, false
	}
}
