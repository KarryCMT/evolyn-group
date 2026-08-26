package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	auditservice "evolyn/internal/platform/audit/service"
	apperrors "evolyn/internal/platform/edition"
	"evolyn/internal/platform/edition/model"
	"evolyn/internal/platform/edition/repository"
	"evolyn/internal/platform/httpx"
	tenantmodel "evolyn/internal/platform/tenant/model"

	kernel "evolyn/internal/model"

	"gorm.io/gorm"
)

// 用量计量窄接口：与 QuotaService 同形，装配期由 iam/application/file 仓储
// 注入；未注入的键读取时按 pending 处理（单测/分域装配场景）
type (
	memberCounter interface {
		CountByTenant(ctx context.Context, tenantID uint) (int64, error)
	}
	appCounter interface {
		CountBillableByTenant(ctx context.Context, tenantID uint) (int64, error)
	}
	storageCounter interface {
		CountStorageBytes(ctx context.Context, tenantID uint) (int64, error)
	}
)

// tenantAccess edition 域所需的租户仓储能力子集（消费者侧窄接口）：
// 投影同步读取/更新 + 与配额校验同一把租户行锁；生产侧由租户域仓储
// 结构性满足，测试侧可用轻量替身
type tenantAccess interface {
	GetByID(ctx context.Context, id uint) (*tenantmodel.Tenant, error)
	LockByID(ctx context.Context, id uint) error
	Update(ctx context.Context, tenant *tenantmodel.Tenant) (*tenantmodel.Tenant, error)
}

// editionService 版本信息服务实现。依赖方向 edition→tenant（窄接口 +
// 模型）单向；QuotaService 经消费者侧窄接口 GuardLimit 集成本服务，
// tenant 域经 SeedInitial 在开通事务内补种订阅，均不产生反向 import
type editionService struct {
	tx         TxManager
	repo       repository.EditionRepository
	tenantRepo tenantAccess
	audit      auditservice.Recorder
	members    memberCounter
	apps       appCounter
	storage    storageCounter
}

func NewEditionService(
	tx TxManager,
	repo repository.EditionRepository,
	tenantRepo tenantAccess,
	audit auditservice.Recorder,
	members memberCounter,
	apps appCounter,
	storage storageCounter,
) EditionService {
	return &editionService{
		tx:         tx,
		repo:       repo,
		tenantRepo: tenantRepo,
		audit:      audit,
		members:    members,
		apps:       apps,
		storage:    storage,
	}
}

// ---- 租户侧读取 ----

// GetCurrent 版本信息概览：读时投影订阅状态（到期未降级立即 expired），
// 配额按「快照 + 有效覆盖」解析并叠加真实用量（设计 4.5.1）
func (s *editionService) GetCurrent(ctx context.Context, tenantID uint) (*model.CurrentEdition, error) {
	now := time.Now()

	state, err := s.resolveCurrent(ctx, tenantID, now)
	if err != nil {
		return nil, err
	}
	overrides, err := s.repo.ListValidOverrides(ctx, tenantID, now)
	if err != nil {
		return nil, err
	}
	limits, sources := effectiveLimits(state.version, overrides, state.fallback)

	quotas, err := s.buildQuotaViews(ctx, tenantID, state.version, limits, sources, state.fallback, now)
	if err != nil {
		return nil, err
	}

	features := make([]model.FeatureView, 0, len(state.version.Entitlements.Features))
	for _, f := range state.version.Entitlements.Features {
		features = append(features, model.FeatureView{
			Group:       f.Group,
			Key:         f.Key,
			Name:        f.Name,
			Available:   f.Available,
			Parameters:  f.Parameters,
			Description: f.Description,
		})
	}

	return &model.CurrentEdition{
		Subscription: model.SubscriptionView{
			PlanCode:      state.version.CompatibilityPlanCode,
			PlanName:      state.version.DisplayName,
			Status:        state.status,
			GrantType:     state.sub.GrantType,
			StartsAt:      state.sub.StartsAt,
			EndsAt:        state.sub.EndsAt,
			ExpiresAction: expiresAction(state),
		},
		Quotas:   quotas,
		Features: features,
		AsOf:     kernel.JSONTime(now),
	}, nil
}

// expiresAction 到期处理规则：非免费套餐且有到期日 → 到期降级免费版。
// 到期未降级窗口（fallback）本身就是降级进行中，同样返回 downgrade_to_free
func expiresAction(state *resolvedEdition) string {
	if state.fallback {
		return "downgrade_to_free"
	}
	if state.version.CompatibilityPlanCode == tenantmodel.PlanFree {
		return "none"
	}
	if state.sub.EndsAt != nil {
		return "downgrade_to_free"
	}
	return "none"
}

// ---- 平台运营面 ----

// GetTenantEdition 平台侧详情：当前概览 + 历史订阅（含运营备注）+ 全量覆盖
func (s *editionService) GetTenantEdition(ctx context.Context, tenantID uint) (*model.TenantEditionDetail, error) {
	current, err := s.GetCurrent(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	subs, err := s.repo.ListSubscriptions(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	history := make([]model.SubscriptionRec, 0, len(subs))
	for i := range subs {
		sub := &subs[i]
		rec := model.SubscriptionRec{
			ID:                sub.ID,
			Status:            sub.Status,
			GrantType:         sub.GrantType,
			StartsAt:          sub.StartsAt,
			EndsAt:            sub.EndsAt,
			OperatorAccountID: sub.OperatorAccountID,
			Remark:            sub.Remark,
			CreatedAt:         sub.CreatedAt,
		}
		if ver, _, err := s.repo.GetPlanVersionWithPlan(ctx, sub.PlanVersionID); err == nil {
			rec.PlanCode = ver.CompatibilityPlanCode
			rec.PlanName = ver.DisplayName
		}
		history = append(history, rec)
	}

	rows, err := s.repo.ListAllOverrides(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	overrideRecs := make([]model.OverrideRec, 0, len(rows))
	for i := range rows {
		o := &rows[i]
		overrideRecs = append(overrideRecs, model.OverrideRec{
			ID:                o.ID,
			EntitlementKey:    o.EntitlementKey,
			Value:             o.Value,
			Reason:            o.Reason,
			Source:            o.Source,
			StartsAt:          o.StartsAt,
			EndsAt:            o.EndsAt,
			OperatorAccountID: o.OperatorAccountID,
		})
	}

	return &model.TenantEditionDetail{
		TenantID:  tenantID,
		Current:   current,
		History:   history,
		Overrides: overrideRecs,
	}, nil
}

// ListGrantableVersions 可授予版本：已发布、未下架的基础套餐；试用兼容
// 版本同时开放 trial / manual 两种授予方式（补录场景），其余仅 manual
func (s *editionService) ListGrantableVersions(ctx context.Context) ([]model.GrantableVersion, error) {
	versions, plans, err := s.repo.ListPublishedBaseVersions(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]model.GrantableVersion, 0, len(versions))
	for i := range versions {
		grantTypes := []string{tenantGrantManual}
		if versions[i].CompatibilityPlanCode == tenantmodel.PlanTrial {
			grantTypes = append(grantTypes, tenantGrantTrial)
		}
		result = append(result, model.GrantableVersion{
			ID:                    versions[i].ID,
			PlanCode:              plans[i].Code,
			PlanName:              plans[i].Name,
			DisplayName:           versions[i].DisplayName,
			Version:               versions[i].Version,
			BillingCycle:          versions[i].BillingCycle,
			CompatibilityPlanCode: versions[i].CompatibilityPlanCode,
			Entitlements:          versions[i].Entitlements,
			GrantTypes:            grantTypes,
		})
	}
	return result, nil
}

// 人工授予可用的授予方式（system/self_service 不开放运营入口）
const (
	tenantGrantManual = model.GrantManual
	tenantGrantTrial  = model.GrantTrial
)

// Grant 平台侧订阅写入主流程（设计 4.5.2）：入参校验 → 事务内
// 锁租户 → 关闭旧订阅 → 创建新订阅/降级 → 覆盖替换 → 兼容投影同步，
// 提交后 best-effort 写审计
func (s *editionService) Grant(ctx context.Context, tenantID, operatorAccountID uint, req *model.GrantRequest) error {
	if req.Action == "" {
		req.Action = model.GrantActionGrant
	}

	// 入参校验（事务外快速失败，不占用连接）
	startsAt, endsAt, overrideItems, err := validateGrant(req)
	if err != nil {
		return err
	}

	var (
		beforeSnap map[string]any
		afterSnap  map[string]any
	)
	err = s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		// 租户行锁：与 QuotaService.CheckAndReserve 同一把锁，串行化
		// 同租户的授予/降级/资源创建并发
		if err := s.tenantRepo.LockByID(tctx, tenantID); err != nil {
			return err
		}
		tenant, err := s.tenantRepo.GetByID(tctx, tenantID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return httpx.Wrap(apperrors.ErrTenantNotFound, err)
			}
			return err
		}
		beforeSnap = map[string]any{"plan": tenant.Plan, "quotas": tenant.Quotas}

		old, err := s.repo.GetCurrentSubscription(tctx, tenantID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		switch req.Action {
		case model.GrantActionGrant:
			afterSnap, err = s.grantInTx(tctx, tenant, old, operatorAccountID, req, startsAt, endsAt, overrideItems)
		case model.GrantActionCancel:
			afterSnap, err = s.cancelInTx(tctx, tenant, old)
		default:
			err = httpx.Wrap(apperrors.ErrGrantInvalid, fmt.Errorf("unknown grant action %q", req.Action))
		}
		return err
	})
	if err != nil {
		return err
	}

	// 审计在事务提交成功后独立写入（best-effort，失败不回滚业务）
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "edition", Action: req.Action, ResourceType: "tenant_subscription",
			ResourceID: strconv.FormatUint(uint64(tenantID), 10),
			TenantID:   tenantID, AccountID: operatorAccountID,
			Before: beforeSnap, After: afterSnap,
		})
	}
	return nil
}

// grantInTx 事务内授予/替换：关旧 → 建新 → 覆盖替换 → 投影同步
func (s *editionService) grantInTx(
	tctx context.Context,
	tenant *tenantmodel.Tenant,
	old *model.TenantSubscription,
	operatorAccountID uint,
	req *model.GrantRequest,
	startsAt, endsAt time.Time,
	overrideItems []model.TenantEntitlementOverride,
) (map[string]any, error) {
	version, plan, err := s.repo.GetPlanVersionWithPlan(tctx, req.PlanVersionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(apperrors.ErrPlanVersionNotFound, err)
		}
		return nil, err
	}
	if plan.Kind != "base" || plan.Status != "active" || version.RetiredAt != nil {
		return nil, httpx.Wrap(apperrors.ErrPlanVersionNotGrantable,
			fmt.Errorf("plan version %d not grantable (kind=%s status=%s retired=%v)",
				req.PlanVersionID, plan.Kind, plan.Status, version.RetiredAt != nil))
	}

	if old != nil {
		if err := s.repo.CloseSubscription(tctx, old.ID, old.Status, model.SubscriptionReplaced); err != nil {
			return nil, err
		}
	}

	sub := &model.TenantSubscription{
		TenantID:      tenant.ID,
		PlanVersionID: version.ID,
		Status:        model.SubscriptionActive,
		GrantType:     req.GrantType,
		StartsAt:      kernel.JSONTime(startsAt),
		Remark:        req.Remark,
	}
	if !endsAt.IsZero() {
		endsCopy := kernel.JSONTime(endsAt)
		sub.EndsAt = &endsCopy
		// 试用临时覆盖与订阅同日到期（设计 4.4.4）
		for i := range overrideItems {
			if overrideItems[i].Source == model.OverrideSourceTrial {
				endsCopy := kernel.JSONTime(endsAt)
				overrideItems[i].EndsAt = &endsCopy
			}
		}
	}
	if operatorAccountID != 0 {
		sub.OperatorAccountID = &operatorAccountID
	}
	if err := s.repo.CreateSubscription(tctx, sub); err != nil {
		return nil, err
	}

	// 覆盖三态：nil 保持不变（跳过）；空数组/非空 → 全量替换 manual+trial
	if req.Overrides != nil {
		if err := s.repo.ReplaceActiveOverrides(tctx, tenant.ID, overrideItems); err != nil {
			return nil, err
		}
	}

	// 兼容投影：tenants.plan + tenants.quotas 与活动订阅同事务同步
	after, err := s.syncCompatProjection(tctx, tenant, version)
	if err != nil {
		return nil, err
	}
	return after, nil
}

// cancelInTx 事务内取消订阅：关闭当前订阅并降级免费版（复用到期降级口径）
func (s *editionService) cancelInTx(tctx context.Context, tenant *tenantmodel.Tenant, old *model.TenantSubscription) (map[string]any, error) {
	if old == nil {
		return nil, httpx.Wrap(apperrors.ErrSubscriptionNotFound, errors.New("no active subscription"))
	}
	if err := s.repo.CloseSubscription(tctx, old.ID, old.Status, model.SubscriptionCancelled); err != nil {
		return nil, err
	}
	return s.downgradeToFree(tctx, tenant, time.Now(), "人工取消订阅，降级免费版")
}

// downgradeToFree 事务内降级免费版：清理降级覆盖 → 建免费订阅 → 投影同步。
// 人工取消与到期任务共用，保证两条路径的终态一致
func (s *editionService) downgradeToFree(tctx context.Context, tenant *tenantmodel.Tenant, now time.Time, remark string) (map[string]any, error) {
	if err := s.repo.DeleteStaleOverrides(tctx, tenant.ID, now); err != nil {
		return nil, err
	}
	freeVersion, err := s.repo.GetLatestPublishedByCompat(tctx, tenantmodel.PlanFree)
	if err != nil {
		return nil, fmt.Errorf("free plan version missing: %w", err)
	}
	if err := s.repo.CreateSubscription(tctx, &model.TenantSubscription{
		TenantID:      tenant.ID,
		PlanVersionID: freeVersion.ID,
		Status:        model.SubscriptionActive,
		GrantType:     model.GrantSystem,
		StartsAt:      kernel.JSONTime(now),
		Remark:        remark,
	}); err != nil {
		return nil, err
	}
	return s.syncCompatProjection(tctx, tenant, freeVersion)
}

// syncCompatProjection 同步旧字段投影：tenants.plan = 版本兼容代码；
// tenants.quotas = 生效值中与套餐默认不同的键（缺省键交 DefaultQuotas
// 兜底），保证 QuotaService 旧读取路径与页面解析结果一致（设计 4.4.1）
func (s *editionService) syncCompatProjection(tctx context.Context, tenant *tenantmodel.Tenant, version *model.EditionPlanVersion) (map[string]any, error) {
	now := time.Now()
	overrides, err := s.repo.ListValidOverrides(tctx, tenant.ID, now)
	if err != nil {
		return nil, err
	}
	limits, _ := effectiveLimits(version, overrides, false)

	tenant.Plan = version.CompatibilityPlanCode
	tenant.Quotas = projectCompatQuotas(version.CompatibilityPlanCode, limits)
	if _, err := s.tenantRepo.Update(tctx, tenant); err != nil {
		return nil, err
	}
	return map[string]any{"plan": tenant.Plan, "quotas": tenant.Quotas}, nil
}

// ---- 入参校验 ----

// validateGrant 授予请求校验：动作/授予方式/起止时间/覆盖项，返回规整后
// 的时间与覆盖实体（source 按授予方式定标；数值合法性含整 GiB 约束）
func validateGrant(req *model.GrantRequest) (time.Time, time.Time, []model.TenantEntitlementOverride, error) {
	if req.Action != model.GrantActionGrant && req.Action != model.GrantActionCancel {
		return time.Time{}, time.Time{}, nil,
			httpx.Wrap(apperrors.ErrGrantInvalid, fmt.Errorf("unknown action %q", req.Action))
	}
	if req.Action == model.GrantActionCancel {
		return time.Time{}, time.Time{}, nil, nil
	}

	if req.GrantType != model.GrantManual && req.GrantType != model.GrantTrial {
		return time.Time{}, time.Time{}, nil,
			httpx.Wrap(apperrors.ErrGrantInvalid, fmt.Errorf("grant type %q not allowed", req.GrantType))
	}
	if req.PlanVersionID == 0 {
		return time.Time{}, time.Time{}, nil,
			httpx.Wrap(apperrors.ErrGrantInvalid, errors.New("planVersionId required"))
	}

	startsAt := time.Now()
	if req.StartsAt != nil && !req.StartsAt.IsZero() {
		startsAt = req.StartsAt.Time()
	}
	endsAt := time.Time{}
	if req.EndsAt != nil && !req.EndsAt.IsZero() {
		endsAt = req.EndsAt.Time()
		if !endsAt.After(startsAt) {
			return time.Time{}, time.Time{}, nil,
				httpx.Wrap(apperrors.ErrGrantInvalid, errors.New("endsAt must be after startsAt"))
		}
	}
	// 试用必须有到期时间（设计 4.3.1）
	if req.GrantType == model.GrantTrial && endsAt.IsZero() {
		return time.Time{}, time.Time{}, nil,
			httpx.Wrap(apperrors.ErrGrantInvalid, errors.New("trial subscription requires endsAt"))
	}

	items := make([]model.TenantEntitlementOverride, 0)
	if req.Overrides != nil {
		source := model.OverrideSourceManual
		if req.GrantType == model.GrantTrial {
			source = model.OverrideSourceTrial
		}
		seen := map[string]bool{}
		for _, in := range *req.Overrides {
			if !isKnownResourceKey(in.Key) {
				return time.Time{}, time.Time{}, nil,
					httpx.Wrap(apperrors.ErrOverrideInvalid, fmt.Errorf("unknown entitlement key %q", in.Key))
			}
			if seen[in.Key] {
				return time.Time{}, time.Time{}, nil,
					httpx.Wrap(apperrors.ErrOverrideInvalid, fmt.Errorf("duplicate key %q", in.Key))
			}
			seen[in.Key] = true
			if in.Value < -1 {
				return time.Time{}, time.Time{}, nil,
					httpx.Wrap(apperrors.ErrOverrideInvalid, fmt.Errorf("invalid value %d for %s", in.Value, in.Key))
			}
			if in.Key == model.ResourceStorage && !model.ValidStorageLimit(in.Value) {
				return time.Time{}, time.Time{}, nil,
					httpx.Wrap(apperrors.ErrStorageLimitInvalid, fmt.Errorf("storage %d not integral GiB", in.Value))
			}
			items = append(items, model.TenantEntitlementOverride{
				EntitlementKey: in.Key,
				Value:          in.Value,
				Reason:         in.Reason,
				Source:         source,
				StartsAt:       kernel.JSONTime(startsAt),
			})
		}
	}
	return startsAt, endsAt, items, nil
}

// isKnownResourceKey 可特批的权益资源键（一期五个已接线键）
func isKnownResourceKey(key string) bool {
	switch key {
	case model.ResourceApps, model.ResourceMembers, model.ResourceForms,
		model.ResourceStorage, model.ResourceWorkflowMo:
		return true
	}
	return false
}

// ---- 容量视图组装 ----

// 资源展示元数据：显示名与单位（API 出网口径，快照缺省时兜底）
var resourceDisplay = map[string]struct{ Name, Unit string }{
	model.ResourceMembers:    {"可用人数", "person"},
	model.ResourceApps:       {"应用数", "count"},
	model.ResourceForms:      {"表单数", "count"},
	model.ResourceStorage:    {"附件存储容量", "byte"},
	model.ResourceWorkflowMo: {"月度流程发起量", "count"},
}

// 页面展示顺序：已接入资源在前，未知键（后续领域扩展）追加在尾部
var quotaDisplayOrder = []string{
	model.ResourceMembers, model.ResourceApps, model.ResourceStorage,
	model.ResourceForms, model.ResourceWorkflowMo,
}

// buildQuotaViews 组装配额视图：快照资源规则 × 生效上限 × 真实用量。
// 已接入键（members/apps/storage_bytes）计量器未注入或读取失败按错误上抛；
// 未接线键一律 pending，不返回伪零值
func (s *editionService) buildQuotaViews(
	ctx context.Context, tenantID uint,
	version *model.EditionPlanVersion,
	limits map[string]int64, sources map[string]string,
	fallback bool, now time.Time,
) ([]model.QuotaView, error) {
	rules := map[string]model.ResourceRule{}
	for _, r := range version.Entitlements.Resources {
		rules[r.Key] = r
	}
	ordered := make([]string, 0, len(rules))
	for _, key := range quotaDisplayOrder {
		if _, ok := rules[key]; ok {
			ordered = append(ordered, key)
		}
	}
	for _, r := range version.Entitlements.Resources {
		known := false
		for _, key := range quotaDisplayOrder {
			if r.Key == key {
				known = true
				break
			}
		}
		if !known {
			ordered = append(ordered, r.Key)
		}
	}

	views := make([]model.QuotaView, 0, len(ordered))
	for _, key := range ordered {
		rule := rules[key]
		display, ok := resourceDisplay[key]
		if !ok {
			display = struct{ Name, Unit string }{Name: key, Unit: rule.Unit}
		}

		view := model.QuotaView{
			Key:         key,
			Category:    rule.Category,
			Name:        display.Name,
			Unit:        display.Unit,
			Limit:       limits[key],
			ResetCycle:  rule.ResetCycle,
			LimitSource: sources[key],
		}
		// 到期未降级窗口：全部资源标记 expiry_fallback（设计 4.5.1）
		if fallback {
			view.LimitSource = model.LimitSourceExpiryFallback
		}

		if usage, metered, err := s.readUsage(ctx, tenantID, key); err != nil {
			return nil, err
		} else if metered {
			view.MeteringStatus = "ready"
			view.Usage = &usage
			asOf := kernel.JSONTime(now)
			view.AsOf = &asOf
			if view.Limit > 0 {
				percent := math.Round(float64(usage)/float64(view.Limit)*10000) / 100
				view.UsagePercent = &percent
			}
		} else {
			view.MeteringStatus = "pending"
		}
		views = append(views, view)
	}
	return views, nil
}

// readUsage 读取真实用量：仅已接入键返回 metered=true；计量器未装配的
// 已接入键视为基础设施缺失直接报错，不允许静默降级为伪 0
func (s *editionService) readUsage(ctx context.Context, tenantID uint, key string) (int64, bool, error) {
	switch key {
	case model.ResourceMembers:
		if s.members == nil {
			return 0, false, fmt.Errorf("member usage counter not configured")
		}
		n, err := s.members.CountByTenant(ctx, tenantID)
		return n, true, err
	case model.ResourceApps:
		if s.apps == nil {
			return 0, false, fmt.Errorf("app usage counter not configured")
		}
		n, err := s.apps.CountBillableByTenant(ctx, tenantID)
		return n, true, err
	case model.ResourceStorage:
		if s.storage == nil {
			return 0, false, fmt.Errorf("storage usage counter not configured")
		}
		n, err := s.storage.CountStorageBytes(ctx, tenantID)
		return n, true, err
	default:
		return 0, false, nil // 未接线领域：待计量
	}
}

// ---- 集成钩子（tenant 域消费）----

// GuardLimit 到期守卫（设计 4.4.1）：活动订阅已到期时以「免费快照 + 仅
// 有效 manual 覆盖」替代旧路径读取——不复用 Quotas.Get(plan=free)，避免
// 旧 tenants.quotas 中残留的试用投影把上限放大回旧档位
func (s *editionService) GuardLimit(ctx context.Context, tenantID uint, resourceKey string) (int64, bool, error) {
	switch resourceKey {
	case model.ResourceMembers, model.ResourceApps, model.ResourceStorage:
	default:
		return 0, false, nil // 非存量键：继续走旧路径
	}

	sub, err := s.repo.GetCurrentSubscription(ctx, tenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, nil // 无订阅记录：迁移前语义兜底
		}
		return 0, false, err
	}
	now := time.Now()
	if sub.Status != model.SubscriptionActive ||
		sub.EndsAt == nil || sub.EndsAt.Time().After(now) {
		return 0, false, nil // 未到期 / 待补录：旧投影仍有效
	}

	// 到期窗口：与页面同一解析器（免费快照 + 仅有效 manual 覆盖）
	freeVersion, err := s.repo.GetLatestPublishedByCompat(ctx, tenantmodel.PlanFree)
	if err != nil {
		return 0, false, err
	}
	overrides, err := s.repo.ListValidOverrides(ctx, tenantID, now)
	if err != nil {
		return 0, false, err
	}
	limits, _ := effectiveLimits(freeVersion, overrides, true)
	if v, ok := limits[resourceKey]; ok {
		return v, true, nil
	}
	// 快照缺键 = 不可用（与 QuotaService「缺省回落 0 禁用」语义一致）
	return 0, true, nil
}

// SeedInitial 租户开通事务内补种初始订阅：free/pro 落 active 长期订阅；
// trial 无到期信息落 legacy_pending_review 待补录（不产生违反
// 「试用必须有到期时间」的 active 记录）
func (s *editionService) SeedInitial(ctx context.Context, tenantID uint, planCode string) error {
	if !tenantmodel.IsValidPlan(planCode) {
		planCode = tenantmodel.PlanFree
	}
	version, err := s.repo.GetLatestPublishedByCompat(ctx, planCode)
	if err != nil {
		return fmt.Errorf("seed initial subscription: plan %s version missing: %w", planCode, err)
	}
	sub := &model.TenantSubscription{
		TenantID:      tenantID,
		PlanVersionID: version.ID,
		GrantType:     model.GrantSystem,
		StartsAt:      kernel.JSONTime(time.Now()),
		Remark:        "租户开通初始订阅",
	}
	if planCode == tenantmodel.PlanTrial {
		sub.Status = model.SubscriptionLegacyPendingReview
		sub.GrantType = model.GrantTrial
		sub.Remark = "开通时指定试用：无到期信息，待运营补录"
	} else {
		sub.Status = model.SubscriptionActive
	}
	return s.repo.CreateSubscription(ctx, sub)
}

// ---- 到期降级任务 ----

// DowngradeExpiredOnce 到期降级单轮扫描（设计 4.3.1）：逐租户事务内
// 「锁租户 → 锁订阅重检 → 关闭过期 → 建免费订阅 → 清理覆盖 → 投影同步」。
// 订阅行锁内重检状态，重复执行/并发竞争不会产生第二条活动订阅
func (s *editionService) DowngradeExpiredOnce(ctx context.Context) (int, error) {
	now := time.Now()
	subs, err := s.repo.ListExpiredActive(ctx, now)
	if err != nil {
		return 0, err
	}
	downgraded := 0
	var lastErr error
	for i := range subs {
		if err := s.downgradeOne(ctx, &subs[i], now); err != nil {
			lastErr = err
			continue // 单租户失败不阻断其余；事务已回滚，下一轮重试
		}
		downgraded++
	}
	return downgraded, lastErr
}

// downgradeOne 单租户到期降级：返回 nil 表示成功或已被并发处理（幂等跳过）
func (s *editionService) downgradeOne(ctx context.Context, scan *model.TenantSubscription, now time.Time) error {
	var (
		tenant    *tenantmodel.Tenant
		handled   bool
		auditSnap map[string]any
	)
	err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		// 租户行锁与人工授予同一把，串行化同租户订阅变更
		if err := s.tenantRepo.LockByID(tctx, scan.TenantID); err != nil {
			return err
		}
		// 订阅行 FOR UPDATE 内重检：已被处理/续期则整体跳过
		fresh, err := s.repo.LockSubscription(tctx, scan.ID)
		if err != nil {
			return err
		}
		if fresh.Status != model.SubscriptionActive ||
			fresh.EndsAt == nil || fresh.EndsAt.Time().After(now) {
			return nil
		}

		if err := s.repo.CloseSubscription(tctx, fresh.ID, model.SubscriptionActive, model.SubscriptionExpired); err != nil {
			return err
		}
		tenant, err = s.tenantRepo.GetByID(tctx, fresh.TenantID)
		if err != nil {
			return err
		}
		after, err := s.downgradeToFree(tctx, tenant, now, "订阅到期自动降级免费版")
		if err != nil {
			return err
		}
		handled = true
		auditSnap = after
		return nil
	})
	if err != nil || !handled {
		return err
	}

	// 审计在事务提交后独立写入（best-effort）
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "edition", Action: "downgrade", ResourceType: "tenant_subscription",
			ResourceID: strconv.FormatUint(uint64(scan.TenantID), 10),
			TenantID:   scan.TenantID,
			After:      auditSnap,
		})
	}
	return nil
}
