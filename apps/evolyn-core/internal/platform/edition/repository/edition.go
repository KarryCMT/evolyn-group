package repository

import (
	"context"
	"errors"
	"time"

	"evolyn/internal/contextx"
	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/edition/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type editionRepository struct {
	db *gorm.DB
}

// NewRepository 版本信息域仓储工厂（ADR-007 域模块化）
func NewRepository(db *gorm.DB) EditionRepository {
	return &editionRepository{db: db}
}

// withContext 打开会话并剥离请求租户上下文：本域表由平台侧/worker/租户侧
// 三方读取，定位一律用显式 tenant_id 条件，避免运营者会话携带的租户过滤
// 污染跨租户操作；ctx 携带事务 session 时仍加入外层事务（FIX-020/021）
func (r *editionRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(contextx.DetachTenant(ctx), r.db)
}

func (r *editionRepository) GetCurrentSubscription(ctx context.Context, tenantID uint) (*model.TenantSubscription, error) {
	sub := new(model.TenantSubscription)
	// active 由部分唯一索引保证唯一；legacy_pending_review 兜底取最新一条
	err := r.withContext(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, model.SubscriptionActive).
		First(sub).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = r.withContext(ctx).
			Where("tenant_id = ? AND status = ?", tenantID, model.SubscriptionLegacyPendingReview).
			Order("id DESC").
			First(sub).Error
	}
	if err != nil {
		return nil, err
	}
	return sub, nil
}

func (r *editionRepository) ListSubscriptions(ctx context.Context, tenantID uint) ([]model.TenantSubscription, error) {
	subs := make([]model.TenantSubscription, 0)
	err := r.withContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("id DESC").
		Find(&subs).Error
	return subs, err
}

func (r *editionRepository) ListExpiredActive(ctx context.Context, now time.Time) ([]model.TenantSubscription, error) {
	subs := make([]model.TenantSubscription, 0)
	err := r.withContext(ctx).
		Where("status = ? AND ends_at IS NOT NULL AND ends_at <= ?", model.SubscriptionActive, now).
		Order("ends_at ASC").
		Find(&subs).Error
	return subs, err
}

func (r *editionRepository) LockSubscription(ctx context.Context, id uint) (*model.TenantSubscription, error) {
	sub := new(model.TenantSubscription)
	err := r.withContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(sub, id).Error
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// CloseSubscription 条件状态迁移：仅当行仍处于 fromStatus 时更新，
// 0 行影响视为「已被并发处理」，调用方据此跳过后续降级写入
func (r *editionRepository) CloseSubscription(ctx context.Context, id uint, fromStatus, toStatus string) error {
	res := r.withContext(ctx).
		Model(&model.TenantSubscription{}).
		Where("id = ? AND status = ?", id, fromStatus).
		Updates(map[string]interface{}{"status": toStatus, "updated_at": time.Now()})
	return res.Error
}

func (r *editionRepository) CreateSubscription(ctx context.Context, sub *model.TenantSubscription) error {
	return r.withContext(ctx).Create(sub).Error
}

// VersionWithPlan 版本 + 目录套餐的联表读取结果
type VersionWithPlan struct {
	model.EditionPlanVersion
	PlanCode string `gorm:"column:plan_code"`
	PlanName string `gorm:"column:plan_name"`
	PlanKind string `gorm:"column:plan_kind"`
	PlanStat string `gorm:"column:plan_status"`
}

func (r *editionRepository) GetPlanVersionWithPlan(ctx context.Context, id uint) (*model.EditionPlanVersion, *model.EditionPlan, error) {
	var row VersionWithPlan
	err := r.withContext(ctx).
		Table("edition_plan_versions AS v").
		Select("v.*, p.code AS plan_code, p.name AS plan_name, p.kind AS plan_kind, p.status AS plan_status").
		Joins("JOIN edition_plans p ON p.id = v.plan_id").
		Where("v.id = ?", id).
		Take(&row).Error
	if err != nil {
		return nil, nil, err
	}
	version := row.EditionPlanVersion
	plan := &model.EditionPlan{
		ID:     version.PlanID,
		Code:   row.PlanCode,
		Name:   row.PlanName,
		Kind:   row.PlanKind,
		Status: row.PlanStat,
	}
	return &version, plan, nil
}

func (r *editionRepository) GetLatestPublishedByCompat(ctx context.Context, compatCode string) (*model.EditionPlanVersion, error) {
	version := new(model.EditionPlanVersion)
	err := r.withContext(ctx).
		Where("compatibility_plan_code = ? AND retired_at IS NULL", compatCode).
		Order("version DESC").
		First(version).Error
	if err != nil {
		return nil, err
	}
	return version, nil
}

func (r *editionRepository) ListPublishedBaseVersions(ctx context.Context) ([]model.EditionPlanVersion, []model.EditionPlan, error) {
	var rows []VersionWithPlan
	err := r.withContext(ctx).
		Table("edition_plan_versions AS v").
		Select("v.*, p.code AS plan_code, p.name AS plan_name, p.kind AS plan_kind, p.status AS plan_status").
		Joins("JOIN edition_plans p ON p.id = v.plan_id").
		Where("p.kind = ? AND p.status = ? AND v.retired_at IS NULL", "base", "active").
		Order("p.code ASC, v.version DESC").
		Find(&rows).Error
	if err != nil {
		return nil, nil, err
	}
	versions := make([]model.EditionPlanVersion, 0, len(rows))
	plans := make([]model.EditionPlan, 0, len(rows))
	for i := range rows {
		versions = append(versions, rows[i].EditionPlanVersion)
		plans = append(plans, model.EditionPlan{
			ID:     rows[i].PlanID,
			Code:   rows[i].PlanCode,
			Name:   rows[i].PlanName,
			Kind:   rows[i].PlanKind,
			Status: rows[i].PlanStat,
		})
	}
	return versions, plans, nil
}

func (r *editionRepository) ListValidOverrides(ctx context.Context, tenantID uint, now time.Time) ([]model.TenantEntitlementOverride, error) {
	overrides := make([]model.TenantEntitlementOverride, 0)
	err := r.withContext(ctx).
		Where("tenant_id = ? AND starts_at <= ? AND (ends_at IS NULL OR ends_at > ?)", tenantID, now, now).
		Find(&overrides).Error
	return overrides, err
}

func (r *editionRepository) ListAllOverrides(ctx context.Context, tenantID uint) ([]model.TenantEntitlementOverride, error) {
	overrides := make([]model.TenantEntitlementOverride, 0)
	err := r.withContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("id DESC").
		Find(&overrides).Error
	return overrides, err
}

// ReplaceActiveOverrides 全量替换 manual+trial 覆盖：legacy 行保留。
// 调用方必须已开启事务（授予/取消主流程内），先删后插保证覆盖集与订阅一致
func (r *editionRepository) ReplaceActiveOverrides(ctx context.Context, tenantID uint, items []model.TenantEntitlementOverride) error {
	if err := r.withContext(ctx).
		Where("tenant_id = ? AND source IN ?", tenantID,
			[]string{model.OverrideSourceManual, model.OverrideSourceTrial}).
		Delete(&model.TenantEntitlementOverride{}).Error; err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	return r.withContext(ctx).Create(&items).Error
}

func (r *editionRepository) DeleteStaleOverrides(ctx context.Context, tenantID uint, now time.Time) error {
	return r.withContext(ctx).
		Where("tenant_id = ? AND (source = ? OR (ends_at IS NOT NULL AND ends_at <= ?))",
			tenantID, model.OverrideSourceTrial, now).
		Delete(&model.TenantEntitlementOverride{}).Error
}

func (r *editionRepository) Migrate() error {
	return r.db.AutoMigrate(
		&model.EditionPlan{},
		&model.EditionPlanVersion{},
		&model.TenantSubscription{},
		&model.TenantEntitlementOverride{},
	)
}
