package repository

import (
	"context"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/dashboard/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type dashboardRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) DashboardRepository { return &dashboardRepository{db: db} }

func (r *dashboardRepository) dbFor(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *dashboardRepository) Create(ctx context.Context, dashboard *model.Dashboard) (*model.Dashboard, error) {
	if err := r.dbFor(ctx).Create(dashboard).Error; err != nil {
		return nil, err
	}
	return dashboard, nil
}

func (r *dashboardRepository) GetByCode(ctx context.Context, code string) (*model.Dashboard, error) {
	var dashboard model.Dashboard
	if err := r.dbFor(ctx).Where("code = ?", code).First(&dashboard).Error; err != nil {
		return nil, err
	}
	return &dashboard, nil
}

func (r *dashboardRepository) GetByID(ctx context.Context, id uint) (*model.Dashboard, error) {
	var dashboard model.Dashboard
	if err := r.dbFor(ctx).Where("id = ?", id).First(&dashboard).Error; err != nil {
		return nil, err
	}
	return &dashboard, nil
}

// ExistingDashboardTargets 为应用菜单提供内部 ID 到稳定公开编码的最小
// 投影。租户隔离由 GORM 租户 Callback 从 ctx 注入，软删记录默认排除。
func (r *dashboardRepository) ExistingDashboardTargets(ctx context.Context, ids []uint) (map[uint]string, error) {
	targets := make(map[uint]string)
	if len(ids) == 0 {
		return targets, nil
	}
	var dashboards []model.Dashboard
	if err := r.dbFor(ctx).Select("id", "code").Where("id IN ?", ids).Find(&dashboards).Error; err != nil {
		return nil, err
	}
	for i := range dashboards {
		targets[dashboards[i].ID] = dashboards[i].Code
	}
	return targets, nil
}

func (r *dashboardRepository) UpdateFields(ctx context.Context, id uint, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return r.dbFor(ctx).Model(&model.Dashboard{}).Where("id = ?", id).Updates(fields).Error
}

func (r *dashboardRepository) UpdateDraft(ctx context.Context, id uint, expectedRevision int64, protocolVersion int, content model.JSONContent) (bool, error) {
	result := r.dbFor(ctx).Model(&model.Dashboard{}).
		Where("id = ? AND draft_revision = ?", id, expectedRevision).
		Updates(map[string]any{
			"draft_content": content, "protocol_version": protocolVersion,
			"draft_revision": gorm.Expr("draft_revision + 1"),
		})
	return result.RowsAffected == 1, result.Error
}

func (r *dashboardRepository) SoftDelete(ctx context.Context, dashboard *model.Dashboard) error {
	return r.dbFor(ctx).Delete(dashboard).Error
}

func (r *dashboardRepository) CountBillableDashboardsByTenant(ctx context.Context, tenantID uint) (int64, error) {
	var count int64
	err := r.dbFor(ctx).Model(&model.Dashboard{}).Scopes(infrastructure.TenantScope(tenantID)).Count(&count).Error
	return count, err
}

func (r *dashboardRepository) PublishedSummary(ctx context.Context, dashboard *model.Dashboard) (*model.PublishedSummary, error) {
	summary := &model.PublishedSummary{Version: dashboard.PublishedVersion}
	if dashboard.LatestVersionID == nil {
		return summary, nil
	}
	var version model.DashboardVersion
	if err := r.dbFor(ctx).Where("id = ? AND dashboard_id = ?", *dashboard.LatestVersionID, dashboard.ID).First(&version).Error; err != nil {
		return nil, err
	}
	summary.PublishedAt = &version.PublishedAt
	return summary, nil
}

func (r *dashboardRepository) ClaimCreateBinding(ctx context.Context, binding *model.CreateBinding) (bool, error) {
	result := r.dbFor(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(binding)
	return result.RowsAffected == 1, result.Error
}

func (r *dashboardRepository) GetCreateBinding(ctx context.Context, tenantID, memberID uint, requestID string) (*model.CreateBinding, error) {
	var binding model.CreateBinding
	err := r.dbFor(ctx).Where("tenant_id = ? AND member_id = ? AND request_id = ?", tenantID, memberID, requestID).First(&binding).Error
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

func (r *dashboardRepository) CompleteCreateBinding(ctx context.Context, bindingID, dashboardID uint) error {
	result := r.dbFor(ctx).Model(&model.CreateBinding{}).
		Where("id = ? AND dashboard_id IS NULL", bindingID).
		Updates(map[string]any{"dashboard_id": dashboardID, "updated_at": gorm.Expr("NOW()")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *dashboardRepository) Migrate() error {
	if err := r.db.AutoMigrate(&model.Dashboard{}, &model.DashboardVersion{}, &model.DashboardVersionSubject{}, &model.CreateBinding{}); err != nil {
		return err
	}
	return r.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_dashboards_tenant_code
		ON tn_dashboards (tenant_id, code) WHERE deleted_at IS NULL`).Error
}
