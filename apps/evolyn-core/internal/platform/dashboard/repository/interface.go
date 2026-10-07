package repository

import (
	"context"

	"evolyn/internal/platform/dashboard/model"
)

type DashboardRepository interface {
	Create(ctx context.Context, dashboard *model.Dashboard) (*model.Dashboard, error)
	GetByCode(ctx context.Context, code string) (*model.Dashboard, error)
	GetByID(ctx context.Context, id uint) (*model.Dashboard, error)
	ExistingDashboardTargets(ctx context.Context, ids []uint) (map[uint]string, error)
	UpdateFields(ctx context.Context, id uint, fields map[string]any) error
	UpdateDraft(ctx context.Context, id uint, expectedRevision int64, protocolVersion int, content model.JSONContent) (bool, error)
	SoftDelete(ctx context.Context, dashboard *model.Dashboard) error
	CountBillableDashboardsByTenant(ctx context.Context, tenantID uint) (int64, error)
	PublishedSummary(ctx context.Context, dashboard *model.Dashboard) (*model.PublishedSummary, error)
	ClaimCreateBinding(ctx context.Context, binding *model.CreateBinding) (bool, error)
	GetCreateBinding(ctx context.Context, tenantID, memberID uint, requestID string) (*model.CreateBinding, error)
	CompleteCreateBinding(ctx context.Context, bindingID, dashboardID uint) error
	Migrate() error
}
