package service

import (
	"context"
	"errors"
	"testing"
	"time"

	auditservice "evolyn/internal/platform/audit/service"
	apperrors "evolyn/internal/platform/edition"
	"evolyn/internal/platform/edition/model"
	tenantmodel "evolyn/internal/platform/tenant/model"

	kernel "evolyn/internal/model"

	"evolyn/internal/platform/httpx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---- 版本信息服务单元测试（真实 PostgreSQL 链路可后续按 SEC-* 模式补充）----

// passThroughTx 不携带事务语义、直接执行 fn
type passThroughTx struct{}

func (passThroughTx) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// fakeTenantRepo 租户仓储替身（edition 域窄接口 tenantAccess 的最小实现）
type fakeTenantRepo struct {
	tenants map[uint]*tenantmodel.Tenant
	locked  []uint
}

func newFakeTenantRepo(tenants ...*tenantmodel.Tenant) *fakeTenantRepo {
	m := map[uint]*tenantmodel.Tenant{}
	for _, t := range tenants {
		m[t.ID] = t
	}
	return &fakeTenantRepo{tenants: m}
}

func (f *fakeTenantRepo) GetByID(ctx context.Context, id uint) (*tenantmodel.Tenant, error) {
	if t, ok := f.tenants[id]; ok {
		clone := *t
		return &clone, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeTenantRepo) LockByID(ctx context.Context, id uint) error {
	f.locked = append(f.locked, id)
	return nil
}

func (f *fakeTenantRepo) Update(ctx context.Context, tenant *tenantmodel.Tenant) (*tenantmodel.Tenant, error) {
	clone := *tenant
	f.tenants[tenant.ID] = &clone
	return &clone, nil
}

// fakeRepo 版本信息仓储替身：内存态模拟部分唯一索引与条件迁移语义
type fakeRepo struct {
	versions  map[uint]*model.EditionPlanVersion
	plans     map[uint]*model.EditionPlan
	subs      []*model.TenantSubscription
	overrides []*model.TenantEntitlementOverride
	nextID    uint
}

func (f *fakeRepo) versionByCompat(compat string) *model.EditionPlanVersion {
	var best *model.EditionPlanVersion
	for _, v := range f.versions {
		if v.CompatibilityPlanCode == compat && v.RetiredAt == nil {
			if best == nil || v.Version > best.Version {
				best = v
			}
		}
	}
	return best
}

func (f *fakeRepo) currentSub(tenantID uint) *model.TenantSubscription {
	var active, legacy *model.TenantSubscription
	for i := range f.subs {
		s := f.subs[i]
		if s.TenantID != tenantID {
			continue
		}
		switch s.Status {
		case model.SubscriptionActive:
			active = s
		case model.SubscriptionLegacyPendingReview:
			legacy = s
		}
	}
	if active != nil {
		return active
	}
	return legacy
}

func (f *fakeRepo) GetCurrentSubscription(ctx context.Context, tenantID uint) (*model.TenantSubscription, error) {
	if s := f.currentSub(tenantID); s != nil {
		clone := *s
		return &clone, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeRepo) ListSubscriptions(ctx context.Context, tenantID uint) ([]model.TenantSubscription, error) {
	out := make([]model.TenantSubscription, 0)
	for i := len(f.subs) - 1; i >= 0; i-- {
		if f.subs[i].TenantID == tenantID {
			out = append(out, *f.subs[i])
		}
	}
	return out, nil
}

func (f *fakeRepo) ListExpiredActive(ctx context.Context, now time.Time) ([]model.TenantSubscription, error) {
	out := make([]model.TenantSubscription, 0)
	for i := range f.subs {
		s := f.subs[i]
		if s.Status == model.SubscriptionActive && s.EndsAt != nil && !s.EndsAt.Time().After(now) {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (f *fakeRepo) LockSubscription(ctx context.Context, id uint) (*model.TenantSubscription, error) {
	for i := range f.subs {
		if f.subs[i].ID == id {
			clone := *f.subs[i]
			return &clone, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeRepo) CloseSubscription(ctx context.Context, id uint, fromStatus, toStatus string) error {
	for i := range f.subs {
		if f.subs[i].ID == id && f.subs[i].Status == fromStatus {
			f.subs[i].Status = toStatus
			return nil
		}
	}
	return nil // 0 行影响：语义上视为已被并发处理
}

func (f *fakeRepo) CreateSubscription(ctx context.Context, sub *model.TenantSubscription) error {
	// 模拟「同租户最多一条 active」部分唯一索引
	if sub.Status == model.SubscriptionActive && f.currentSub(sub.TenantID) != nil &&
		f.currentSub(sub.TenantID).Status == model.SubscriptionActive {
		return errors.New("duplicate active subscription")
	}
	f.nextID++
	sub.ID = f.nextID
	clone := *sub
	f.subs = append(f.subs, &clone)
	return nil
}

func (f *fakeRepo) GetPlanVersionWithPlan(ctx context.Context, id uint) (*model.EditionPlanVersion, *model.EditionPlan, error) {
	v, ok := f.versions[id]
	if !ok {
		return nil, nil, gorm.ErrRecordNotFound
	}
	plan := f.plans[v.PlanID]
	vc, pc := *v, *plan
	return &vc, &pc, nil
}

func (f *fakeRepo) GetLatestPublishedByCompat(ctx context.Context, compatCode string) (*model.EditionPlanVersion, error) {
	if v := f.versionByCompat(compatCode); v != nil {
		clone := *v
		return &clone, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeRepo) ListPublishedBaseVersions(ctx context.Context) ([]model.EditionPlanVersion, []model.EditionPlan, error) {
	vs, ps := make([]model.EditionPlanVersion, 0), make([]model.EditionPlan, 0)
	for _, v := range f.versions {
		plan := f.plans[v.PlanID]
		if plan.Kind == "base" && plan.Status == "active" && v.RetiredAt == nil {
			vs, ps = append(vs, *v), append(ps, *plan)
		}
	}
	return vs, ps, nil
}

func (f *fakeRepo) ListValidOverrides(ctx context.Context, tenantID uint, now time.Time) ([]model.TenantEntitlementOverride, error) {
	out := make([]model.TenantEntitlementOverride, 0)
	for i := range f.overrides {
		o := f.overrides[i]
		if o.TenantID != tenantID {
			continue
		}
		if o.StartsAt.Time().After(now) {
			continue
		}
		if o.EndsAt != nil && !o.EndsAt.Time().After(now) {
			continue
		}
		out = append(out, *o)
	}
	return out, nil
}

func (f *fakeRepo) ListAllOverrides(ctx context.Context, tenantID uint) ([]model.TenantEntitlementOverride, error) {
	out := make([]model.TenantEntitlementOverride, 0)
	for i := len(f.overrides) - 1; i >= 0; i-- {
		if f.overrides[i].TenantID == tenantID {
			out = append(out, *f.overrides[i])
		}
	}
	return out, nil
}

func (f *fakeRepo) ReplaceActiveOverrides(ctx context.Context, tenantID uint, items []model.TenantEntitlementOverride) error {
	kept := f.overrides[:0]
	for _, o := range f.overrides {
		if o.TenantID == tenantID &&
			(o.Source == model.OverrideSourceManual || o.Source == model.OverrideSourceTrial) {
			continue
		}
		kept = append(kept, o)
	}
	f.overrides = kept
	for i := range items {
		f.nextID++
		items[i].ID = f.nextID
		items[i].TenantID = tenantID
		clone := items[i]
		f.overrides = append(f.overrides, &clone)
	}
	return nil
}

func (f *fakeRepo) DeleteStaleOverrides(ctx context.Context, tenantID uint, now time.Time) error {
	kept := f.overrides[:0]
	for _, o := range f.overrides {
		if o.TenantID == tenantID &&
			(o.Source == model.OverrideSourceTrial || (o.EndsAt != nil && !o.EndsAt.Time().After(now))) {
			continue
		}
		kept = append(kept, o)
	}
	f.overrides = kept
	return nil
}

func (f *fakeRepo) Migrate() error { return nil }

// fakeCounter 计量替身（members/apps/storage 同构）
type fakeCounter struct {
	value int64
	err   error
}

func (f fakeCounter) CountByTenant(ctx context.Context, tenantID uint) (int64, error) {
	return f.value, f.err
}

func (f fakeCounter) CountBillableByTenant(ctx context.Context, tenantID uint) (int64, error) {
	return f.value, f.err
}

func (f fakeCounter) CountStorageBytes(ctx context.Context, tenantID uint) (int64, error) {
	return f.value, f.err
}

// fakeAudit 审计替身：捕获提交后写入的事件
type fakeAudit struct{ entries []auditservice.Entry }

func (f *fakeAudit) Record(ctx context.Context, e auditservice.Entry) {
	f.entries = append(f.entries, e)
}

// ---- 测试脚手架 ----

// seedCatalog 按 000030 迁移口径构造三档目录与 v1 快照（数值对齐 DefaultQuotas）
func seedCatalog(repo *fakeRepo) {
	repo.plans = map[uint]*model.EditionPlan{}
	repo.versions = map[uint]*model.EditionPlanVersion{}
	quotaOf := func(apps, members, forms, storage, workflow int64) []model.ResourceRule {
		return []model.ResourceRule{
			{Key: model.ResourceApps, Category: model.CategoryStock, Limit: apps, Unit: "count"},
			{Key: model.ResourceMembers, Category: model.CategoryStock, Limit: members, Unit: "person"},
			{Key: model.ResourceForms, Category: model.CategoryStock, Limit: forms, Unit: "count"},
			{Key: model.ResourceStorage, Category: model.CategoryStock, Limit: storage, Unit: "byte"},
			{Key: model.ResourceWorkflowMo, Category: model.CategoryPeriodic, Limit: workflow, Unit: "count", ResetCycle: "monthly"},
		}
	}
	for i, spec := range []struct {
		code, name string
		rules      []model.ResourceRule
	}{
		{"free", "免费版", quotaOf(3, 5, 10, 1*model.GiB, 100)},
		{"trial", "试用版", quotaOf(10, 30, 50, 5*model.GiB, 10000)},
		{"pro", "专业版", quotaOf(-1, -1, -1, -1, -1)},
	} {
		planID := uint(i + 1)
		repo.plans[planID] = &model.EditionPlan{ID: planID, Code: spec.code, Name: spec.name, Status: "active", Kind: "base"}
		repo.versions[uint(i+100)] = &model.EditionPlanVersion{
			ID: uint(i + 100), PlanID: planID, Version: 1, DisplayName: spec.name,
			CompatibilityPlanCode: spec.code,
			Entitlements:          model.EntitlementSet{Resources: spec.rules},
		}
	}
	repo.nextID = 500
}

func newTestService(repo *fakeRepo, tenantRepo *fakeTenantRepo, audit *fakeAudit) EditionService {
	return NewEditionService(passThroughTx{}, repo, tenantRepo, audit,
		fakeCounter{value: 2}, fakeCounter{value: 1}, fakeCounter{value: 2 * model.GiB})
}

func mustTime(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, kernel.CSTLocation())
	if err != nil {
		panic(err)
	}
	return t
}

// ---- 存储双键换算（设计 4.3.2/4.4.1）----

func TestStorageLimitValidation(t *testing.T) {
	assert.True(t, model.ValidStorageLimit(-1), "-1 不限量")
	assert.True(t, model.ValidStorageLimit(0), "0 禁用")
	assert.True(t, model.ValidStorageLimit(5*model.GiB), "整 GiB")
	assert.False(t, model.ValidStorageLimit(500*1024*1024), "500 MiB 拒绝")
	assert.False(t, model.ValidStorageLimit(-2), "非 -1 负数拒绝")
	assert.False(t, model.ValidStorageLimit(model.GiB-1), "差一字节也拒绝")
}

func TestStorageBytesConversion(t *testing.T) {
	assert.Equal(t, int64(5), model.StorageBytesToGB(5*model.GiB))
	assert.Equal(t, int64(-1), model.StorageBytesToGB(-1), "-1 不限量原样透传，不得被整数除法折成 0")
	assert.Equal(t, int64(0), model.StorageBytesToGB(0))

	back, err := model.GBToStorageBytes(5)
	require.NoError(t, err)
	assert.Equal(t, 5*model.GiB, back)

	_, err = model.GBToStorageBytes(int64(1) << 62)
	assert.Error(t, err, "乘法溢出必须报错")

	neg, err := model.GBToStorageBytes(-1)
	require.NoError(t, err)
	assert.Equal(t, int64(-1), neg)
}

// ---- 兼容投影（设计 4.4.1：仅保留与套餐默认不同的键）----

func TestProjectCompatQuotas(t *testing.T) {
	effective := map[string]int64{
		model.ResourceMembers:    30,            // 与 trial 默认一致 → 省略
		model.ResourceApps:       20,            // 与 trial 默认 10 不同 → 写入
		model.ResourceStorage:    5 * model.GiB, // 与 trial 默认一致 → 省略
		model.ResourceWorkflowMo: 10000,         // 与 trial 默认一致 → 省略
		model.ResourceForms:      0,             // 与默认 50 不同 → 写入 0
	}
	projected := projectCompatQuotas(tenantmodel.PlanTrial, effective)
	assert.Equal(t, tenantmodel.Quotas{
		tenantmodel.QuotaApps:  20,
		tenantmodel.QuotaForms: 0,
	}, projected, "与套餐默认一致的键不得落覆盖（交 DefaultQuotas 兜底）")

	free := projectCompatQuotas(tenantmodel.PlanFree, map[string]int64{
		model.ResourceStorage: -1,
	})
	assert.Equal(t, tenantmodel.Quotas{tenantmodel.QuotaStorageGB: -1}, free, "-1 经字节键投影回 -1")
}

// ---- 覆盖解析优先级（设计 4.4.4）----

func trialVersionRules() []model.ResourceRule {
	return []model.ResourceRule{
		{Key: model.ResourceMembers, Category: model.CategoryStock, Limit: 30, Unit: "person"},
		{Key: model.ResourceStorage, Category: model.CategoryStock, Limit: 5 * model.GiB, Unit: "byte"},
	}
}

func TestEffectiveLimitsOverridePriority(t *testing.T) {
	version := &model.EditionPlanVersion{Entitlements: model.EntitlementSet{Resources: trialVersionRules()}}
	overrides := []model.TenantEntitlementOverride{
		{EntitlementKey: model.ResourceMembers, Value: 100, Source: model.OverrideSourceLegacy},
		{EntitlementKey: model.ResourceMembers, Value: 40, Source: model.OverrideSourceTrial},
		{EntitlementKey: model.ResourceMembers, Value: 50, Source: model.OverrideSourceManual},
	}
	limits, sources := effectiveLimits(version, overrides, false)
	assert.Equal(t, int64(50), limits[model.ResourceMembers], "manual 是最新运营意图，优先级最高")
	assert.Equal(t, model.LimitSourceTenantOverride, sources[model.ResourceMembers])
	assert.Equal(t, 5*model.GiB, limits[model.ResourceStorage], "无覆盖键回落快照值")
	assert.Equal(t, model.LimitSourcePlanVersion, sources[model.ResourceStorage])

	// legacy 单独存在时标注 legacy_quota
	limits, sources = effectiveLimits(version, overrides[:1], false)
	assert.Equal(t, int64(100), limits[model.ResourceMembers])
	assert.Equal(t, model.LimitSourceLegacyQuota, sources[model.ResourceMembers])

	// manualOnly（到期窗口）：trial/legacy 均不得放大上限，仅 manual 仍生效
	limits, _ = effectiveLimits(version, overrides, true)
	assert.Equal(t, int64(50), limits[model.ResourceMembers], "到期窗口 manual 覆盖仍生效")
	limits, _ = effectiveLimits(version, []model.TenantEntitlementOverride{
		{EntitlementKey: model.ResourceMembers, Value: 100, Source: model.OverrideSourceLegacy},
		{EntitlementKey: model.ResourceMembers, Value: 40, Source: model.OverrideSourceTrial},
	}, true)
	assert.Equal(t, int64(30), limits[model.ResourceMembers], "trial/legacy 在到期窗口被剔除，回落快照底值")
}

// ---- 读取投影：到期即时 expired + 免费快照回退（设计 4.3.1/4.5.1）----

func TestGetCurrentExpiryFallback(t *testing.T) {
	repo, tenantRepo, audit := &fakeRepo{}, newFakeTenantRepo(), &fakeAudit{}
	seedCatalog(repo)
	trialVer := repo.versionByCompat(tenantmodel.PlanTrial)

	// 已到期的试用订阅：ends_at 在过去
	past := kernel.JSONTime(mustTime("2026-08-01 00:00:00"))
	require.NoError(t, repo.CreateSubscription(context.Background(), &model.TenantSubscription{
		TenantID: 7, PlanVersionID: trialVer.ID, Status: model.SubscriptionActive,
		GrantType: model.GrantTrial, StartsAt: past, EndsAt: &past,
	}))
	// 残留 trial 覆盖：页面必须剔除（不得放大）
	repo.overrides = append(repo.overrides, &model.TenantEntitlementOverride{
		TenantID: 7, EntitlementKey: model.ResourceMembers, Value: 99,
		Source: model.OverrideSourceTrial, StartsAt: past, EndsAt: &past,
	})

	svc := newTestService(repo, tenantRepo, audit)
	out, err := svc.GetCurrent(context.Background(), 7)
	require.NoError(t, err)

	assert.Equal(t, model.SubscriptionExpired, out.Subscription.Status, "到期即时投影 expired")
	assert.Equal(t, tenantmodel.PlanFree, out.Subscription.PlanCode, "权益按免费版解析")
	assert.Equal(t, "downgrade_to_free", out.Subscription.ExpiresAction)

	byKey := map[string]model.QuotaView{}
	for _, q := range out.Quotas {
		byKey[q.Key] = q
	}
	assert.Equal(t, int64(5), byKey[model.ResourceMembers].Limit, "免费快照 members=5")
	assert.Equal(t, model.LimitSourceExpiryFallback, byKey[model.ResourceMembers].LimitSource)
	assert.Equal(t, model.GiB, byKey[model.ResourceStorage].Limit, "免费快照存储 1GiB")
	assert.Equal(t, "ready", byKey[model.ResourceMembers].MeteringStatus)
	assert.NotNil(t, byKey[model.ResourceMembers].Usage)
	assert.Nil(t, byKey[model.ResourceForms].Usage, "待计量键不返回伪零值")
	assert.Equal(t, "pending", byKey[model.ResourceForms].MeteringStatus)
	assert.Equal(t, "monthly", byKey[model.ResourceWorkflowMo].ResetCycle, "仅周期额度带 resetCycle")
	assert.Empty(t, byKey[model.ResourceMembers].ResetCycle)
}

func TestGetCurrentActiveAndLegacy(t *testing.T) {
	repo, tenantRepo, audit := &fakeRepo{}, newFakeTenantRepo(), &fakeAudit{}
	seedCatalog(repo)
	freeVer := repo.versionByCompat(tenantmodel.PlanFree)
	svc := newTestService(repo, tenantRepo, audit)

	// 正常活动订阅
	require.NoError(t, repo.CreateSubscription(context.Background(), &model.TenantSubscription{
		TenantID: 8, PlanVersionID: freeVer.ID, Status: model.SubscriptionActive,
		GrantType: model.GrantSystem, StartsAt: kernel.JSONTime(time.Now()),
	}))
	out, err := svc.GetCurrent(context.Background(), 8)
	require.NoError(t, err)
	assert.Equal(t, model.SubscriptionActive, out.Subscription.Status)
	assert.Equal(t, "none", out.Subscription.ExpiresAction)
	assert.Equal(t, int64(3), quotaLimit(out, model.ResourceApps))

	// 存量试用待补录：显示订阅快照但状态为待确认，不参与降级
	repo2 := &fakeRepo{}
	seedCatalog(repo2)
	trialVer := repo2.versionByCompat(tenantmodel.PlanTrial)
	require.NoError(t, repo2.CreateSubscription(context.Background(), &model.TenantSubscription{
		TenantID: 9, PlanVersionID: trialVer.ID, Status: model.SubscriptionLegacyPendingReview,
		GrantType: model.GrantTrial, StartsAt: kernel.JSONTime(time.Now()),
	}))
	svc2 := newTestService(repo2, tenantRepo, audit)
	out2, err := svc2.GetCurrent(context.Background(), 9)
	require.NoError(t, err)
	assert.Equal(t, model.SubscriptionLegacyPendingReview, out2.Subscription.Status)
	assert.Equal(t, tenantmodel.PlanTrial, out2.Subscription.PlanCode)
	assert.Nil(t, out2.Subscription.EndsAt)
}

func quotaLimit(out *model.CurrentEdition, key string) int64 {
	for _, q := range out.Quotas {
		if q.Key == key {
			return q.Limit
		}
	}
	return 0
}

// ---- 人工授予与投影同步（设计 4.5.2）----

func TestGrantReplacesSubscriptionAndSyncsProjection(t *testing.T) {
	repo, audit := &fakeRepo{}, &fakeAudit{}
	seedCatalog(repo)
	tenant := &tenantmodel.Tenant{ID: 12, Code: "t-12", Name: "租户12", Plan: tenantmodel.PlanFree,
		Quotas: tenantmodel.Quotas{}, Status: tenantmodel.TenantActive}
	tenantRepo := newFakeTenantRepo(tenant)
	svc := newTestService(repo, tenantRepo, audit)

	trialVer := repo.versionByCompat(tenantmodel.PlanTrial)
	ends := kernel.JSONTime(mustTime("2026-09-30 23:59:59"))
	err := svc.Grant(context.Background(), 12, 42, &model.GrantRequest{
		PlanVersionID: trialVer.ID,
		GrantType:     model.GrantTrial,
		EndsAt:        &ends,
		Remark:        "补录试用",
		Overrides: &[]model.OverrideInput{
			{Key: model.ResourceMembers, Value: 40, Reason: "临时扩容"},
		},
	})
	require.NoError(t, err)

	sub := repo.currentSub(12)
	require.NotNil(t, sub)
	assert.Equal(t, model.SubscriptionActive, sub.Status)
	assert.Equal(t, model.GrantTrial, sub.GrantType)
	require.NotNil(t, sub.EndsAt)
	assert.True(t, sub.EndsAt.Time().Equal(ends.Time()))

	// 兼容投影：plan=trial；members 覆盖 40 与默认 30 不同 → 落旧键
	updated, err := tenantRepo.GetByID(context.Background(), 12)
	require.NoError(t, err)
	assert.Equal(t, tenantmodel.PlanTrial, updated.Plan)
	assert.Equal(t, int64(40), updated.Quotas[tenantmodel.QuotaMembers])
	assert.NotContains(t, updated.Quotas, tenantmodel.QuotaStorageGB, "与默认一致的键不落覆盖")

	// trial 覆盖 ends 与订阅同日到期
	for _, o := range repo.overrides {
		if o.EntitlementKey == model.ResourceMembers {
			assert.Equal(t, model.OverrideSourceTrial, o.Source)
			require.NotNil(t, o.EndsAt)
			assert.True(t, o.EndsAt.Time().Equal(ends.Time()))
		}
	}

	// 再次授予免费版：旧订阅被替换、覆盖清空、投影回落
	freeVer := repo.versionByCompat(tenantmodel.PlanFree)
	err = svc.Grant(context.Background(), 12, 42, &model.GrantRequest{
		PlanVersionID: freeVer.ID, GrantType: model.GrantManual, Overrides: &[]model.OverrideInput{},
	})
	require.NoError(t, err)
	assert.Len(t, repo.subs, 2, "两次授予形成替换链（初始无订阅）")
	assert.Equal(t, model.SubscriptionReplaced, repo.subs[0].Status)
	assert.Equal(t, tenantmodel.PlanFree, tenantRepo.tenants[12].Plan)
	assert.Empty(t, tenantRepo.tenants[12].Quotas)

	require.Len(t, audit.entries, 2, "每次授予提交后写审计")
	assert.Equal(t, "grant", audit.entries[0].Action)
}

func TestGrantValidation(t *testing.T) {
	repo := &fakeRepo{}
	seedCatalog(repo)
	trialVer := repo.versionByCompat(tenantmodel.PlanTrial)
	tenant := &tenantmodel.Tenant{ID: 13, Code: "t-13", Name: "租户13", Status: tenantmodel.TenantActive}
	svc := newTestService(repo, newFakeTenantRepo(tenant), &fakeAudit{})

	// 试用缺到期日
	err := svc.Grant(context.Background(), 13, 1, &model.GrantRequest{
		PlanVersionID: trialVer.ID, GrantType: model.GrantTrial,
	})
	assertBizCode(t, err, apperrors.ErrGrantInvalid)

	// 非整 GiB 存储
	err = svc.Grant(context.Background(), 13, 1, &model.GrantRequest{
		PlanVersionID: trialVer.ID, GrantType: model.GrantTrial,
		EndsAt: ptrTime(mustTime("2026-09-30 23:59:59")),
		Overrides: &[]model.OverrideInput{
			{Key: model.ResourceStorage, Value: 500 * 1024 * 1024},
		},
	})
	assertBizCode(t, err, apperrors.ErrStorageLimitInvalid)

	// 未知覆盖键
	err = svc.Grant(context.Background(), 13, 1, &model.GrantRequest{
		PlanVersionID: trialVer.ID, GrantType: model.GrantTrial,
		EndsAt:    ptrTime(mustTime("2026-09-30 23:59:59")),
		Overrides: &[]model.OverrideInput{{Key: "unknown_key", Value: 1}},
	})
	assertBizCode(t, err, apperrors.ErrOverrideInvalid)

	// 版本不存在
	err = svc.Grant(context.Background(), 13, 1, &model.GrantRequest{
		PlanVersionID: 9999, GrantType: model.GrantManual,
	})
	assertBizCode(t, err, apperrors.ErrPlanVersionNotFound)

	// 非法 action
	err = svc.Grant(context.Background(), 13, 1, &model.GrantRequest{Action: "hack"})
	assertBizCode(t, err, apperrors.ErrGrantInvalid)
}

func TestGrantCancelDowngradesToFree(t *testing.T) {
	repo, audit := &fakeRepo{}, &fakeAudit{}
	seedCatalog(repo)
	tenant := &tenantmodel.Tenant{ID: 14, Code: "t-14", Name: "租户14", Plan: tenantmodel.PlanPro,
		Quotas: tenantmodel.Quotas{}, Status: tenantmodel.TenantActive}
	tenantRepo := newFakeTenantRepo(tenant)
	svc := newTestService(repo, tenantRepo, audit)

	proVer := repo.versionByCompat(tenantmodel.PlanPro)
	require.NoError(t, repo.CreateSubscription(context.Background(), &model.TenantSubscription{
		TenantID: 14, PlanVersionID: proVer.ID, Status: model.SubscriptionActive,
		GrantType: model.GrantManual, StartsAt: kernel.JSONTime(time.Now()),
	}))

	err := svc.Grant(context.Background(), 14, 7, &model.GrantRequest{Action: model.GrantActionCancel})
	require.NoError(t, err)

	assert.Equal(t, tenantmodel.PlanFree, tenantRepo.tenants[14].Plan, "取消后投影回落免费版")
	assert.Equal(t, model.SubscriptionCancelled, repo.subs[0].Status)
	assert.Equal(t, model.SubscriptionActive, repo.subs[1].Status)
	assert.Equal(t, model.GrantSystem, repo.subs[1].GrantType)

	// 无订阅时取消 → 明确错误
	err = svc.Grant(context.Background(), 999, 7, &model.GrantRequest{Action: model.GrantActionCancel})
	assert.NotNil(t, err) // 租户不存在路径
}

// ---- 到期降级幂等（设计 4.3.1）----

func TestDowngradeExpiredOnceIdempotent(t *testing.T) {
	repo, audit := &fakeRepo{}, &fakeAudit{}
	seedCatalog(repo)
	tenant := &tenantmodel.Tenant{ID: 20, Code: "t-20", Name: "租户20", Plan: tenantmodel.PlanTrial,
		Quotas: tenantmodel.Quotas{tenantmodel.QuotaMembers: 40}, Status: tenantmodel.TenantActive}
	tenantRepo := newFakeTenantRepo(tenant)

	trialVer := repo.versionByCompat(tenantmodel.PlanTrial)
	past := kernel.JSONTime(mustTime("2026-08-01 00:00:00"))
	require.NoError(t, repo.CreateSubscription(context.Background(), &model.TenantSubscription{
		TenantID: 20, PlanVersionID: trialVer.ID, Status: model.SubscriptionActive,
		GrantType: model.GrantTrial, StartsAt: past, EndsAt: &past,
	}))
	// 覆盖：trial 临时（应清除）、manual 长期（应保留）
	repo.overrides = append(repo.overrides,
		&model.TenantEntitlementOverride{TenantID: 20, EntitlementKey: model.ResourceMembers,
			Value: 99, Source: model.OverrideSourceTrial, StartsAt: past, EndsAt: &past},
		&model.TenantEntitlementOverride{TenantID: 20, EntitlementKey: model.ResourceApps,
			Value: 8, Source: model.OverrideSourceManual, StartsAt: past},
	)

	svc := newTestService(repo, tenantRepo, audit)
	n, err := svc.DowngradeExpiredOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	// 终态校验：旧订阅 expired、新免费订阅 active、投影回落、覆盖清理
	assert.Equal(t, model.SubscriptionExpired, repo.subs[0].Status)
	assert.Equal(t, model.SubscriptionActive, repo.subs[1].Status)
	assert.Equal(t, model.GrantSystem, repo.subs[1].GrantType)
	assert.Equal(t, tenantmodel.PlanFree, tenantRepo.tenants[20].Plan)
	assert.Equal(t, int64(8), tenantRepo.tenants[20].Quotas[tenantmodel.QuotaApps], "manual 覆盖投影保留")
	assert.NotContains(t, tenantRepo.tenants[20].Quotas, tenantmodel.QuotaMembers, "trial 覆盖随降级清除")
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "downgrade", audit.entries[0].Action)

	// 幂等：重复执行不再产生新订阅/审计
	n, err = svc.DowngradeExpiredOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Len(t, repo.subs, 2)
	assert.Len(t, audit.entries, 1, "重复执行不得重复写审计")
}

// ---- 到期守卫（设计 4.4.1：写路径与页面同档位降级）----

func TestGuardLimit(t *testing.T) {
	repo := &fakeRepo{}
	seedCatalog(repo)
	tenantRepo := newFakeTenantRepo()
	trialVer := repo.versionByCompat(tenantmodel.PlanTrial)
	svc := newTestService(repo, tenantRepo, &fakeAudit{}).(*editionService)

	// 未到期：decided=false，走旧路径
	future := kernel.JSONTime(mustTime("2099-01-01 00:00:00"))
	require.NoError(t, repo.CreateSubscription(context.Background(), &model.TenantSubscription{
		TenantID: 30, PlanVersionID: trialVer.ID, Status: model.SubscriptionActive,
		GrantType: model.GrantTrial, StartsAt: kernel.JSONTime(time.Now().Add(-time.Hour)), EndsAt: &future,
	}))
	_, decided, err := svc.GuardLimit(context.Background(), 30, model.ResourceMembers)
	require.NoError(t, err)
	assert.False(t, decided, "未到期不干预")

	// 已到期：返回免费快照值，且旧 quotas 中的 trial 残留不得放大
	past := kernel.JSONTime(mustTime("2026-08-01 00:00:00"))
	require.NoError(t, repo.CloseSubscription(context.Background(), repo.currentSub(30).ID,
		model.SubscriptionActive, model.SubscriptionExpired))
	require.NoError(t, repo.CreateSubscription(context.Background(), &model.TenantSubscription{
		TenantID: 31, PlanVersionID: trialVer.ID, Status: model.SubscriptionActive,
		GrantType: model.GrantTrial, StartsAt: past, EndsAt: &past,
	}))
	repo.overrides = append(repo.overrides,
		&model.TenantEntitlementOverride{TenantID: 31, EntitlementKey: model.ResourceMembers,
			Value: 99, Source: model.OverrideSourceTrial, StartsAt: past, EndsAt: &past},
		&model.TenantEntitlementOverride{TenantID: 31, EntitlementKey: model.ResourceApps,
			Value: 6, Source: model.OverrideSourceManual, StartsAt: past},
	)
	limit, decided, err := svc.GuardLimit(context.Background(), 31, model.ResourceMembers)
	require.NoError(t, err)
	assert.True(t, decided)
	assert.Equal(t, int64(5), limit, "免费快照 members=5；trial 残留不得放大")

	limit, decided, err = svc.GuardLimit(context.Background(), 31, model.ResourceApps)
	require.NoError(t, err)
	assert.True(t, decided)
	assert.Equal(t, int64(6), limit, "有效 manual 覆盖在到期窗口仍生效")

	limit, decided, err = svc.GuardLimit(context.Background(), 31, model.ResourceStorage)
	require.NoError(t, err)
	assert.True(t, decided)
	assert.Equal(t, model.GiB, limit)

	// 非存量键 / 无订阅
	_, decided, err = svc.GuardLimit(context.Background(), 31, model.ResourceForms)
	require.NoError(t, err)
	assert.False(t, decided, "非存量键交回旧路径")
	_, decided, err = svc.GuardLimit(context.Background(), 404, model.ResourceMembers)
	require.NoError(t, err)
	assert.False(t, decided, "无订阅记录不干预")
}

// ---- 开通种子订阅 ----

func TestSeedInitial(t *testing.T) {
	repo := &fakeRepo{}
	seedCatalog(repo)
	svc := newTestService(repo, newFakeTenantRepo(), &fakeAudit{})

	require.NoError(t, svc.SeedInitial(context.Background(), 50, tenantmodel.PlanFree))
	sub := repo.currentSub(50)
	require.NotNil(t, sub)
	assert.Equal(t, model.SubscriptionActive, sub.Status)
	assert.Nil(t, sub.EndsAt, "免费初始订阅长期有效")

	// trial 无到期信息：落待补录态，不产生违反「试用必须非空」的 active 记录
	require.NoError(t, svc.SeedInitial(context.Background(), 51, tenantmodel.PlanTrial))
	sub = repo.currentSub(51)
	require.NotNil(t, sub)
	assert.Equal(t, model.SubscriptionLegacyPendingReview, sub.Status)
	assert.Equal(t, model.GrantTrial, sub.GrantType)
}

// ---- 辅助 ----

func ptrTime(t time.Time) *kernel.JSONTime {
	v := kernel.JSONTime(t)
	return &v
}

// assertBizCode 断言错误链上携带目标 BizError 稳定码（ADR-008：按码分支）
func assertBizCode(t *testing.T, err error, target *httpx.BizError) {
	t.Helper()
	require.Error(t, err)
	var biz *httpx.BizError
	require.ErrorAs(t, err, &biz)
	assert.Equal(t, target.Code, biz.Code)
}
