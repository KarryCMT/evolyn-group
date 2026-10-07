package service

import (
	"context"

	"evolyn/internal/platform/dashboard/model"
	iammodel "evolyn/internal/platform/iam/model"
)

type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type AccessEvaluator interface {
	Permissions(ctx context.Context, member *iammodel.User) map[string]bool
}

type AppView struct {
	ID, TenantID uint
	Code, Name   string
	Status       string
}

type AppDirectory interface {
	AppByCode(ctx context.Context, code string) (AppView, bool, error)
	AppByID(ctx context.Context, id uint) (AppView, bool, error)
}

// MenuMaintenance 是 dashboard → app 的窄端口。一次调用完成节点字段变更和
// 单次 menu_revision 推进，避免名称、外观和移动组合更新时多次递增。
type MenuMaintenance interface {
	AttachDashboardNode(ctx context.Context, appID, dashboardID uint, name, icon, color, parentMenuCode string) error
	SyncDashboardNode(ctx context.Context, appID, dashboardID uint, name, icon, color *string, parentMenuCode *string) error
	DetachDashboardNode(ctx context.Context, appID, dashboardID uint) error
}

type DashboardService interface {
	Precreate(ctx context.Context, member *iammodel.User, appCode string, req *model.PrecreateRequest) (*model.Detail, error)
	Get(ctx context.Context, member *iammodel.User, code string) (*model.Detail, error)
	Update(ctx context.Context, member *iammodel.User, code string, req *model.UpdateRequest) (*model.Detail, error)
	SaveDraft(ctx context.Context, member *iammodel.User, code string, req *model.SaveDraftRequest) (*model.SaveDraftResult, error)
	Delete(ctx context.Context, member *iammodel.User, code string) error
}
