package service

import (
	"context"
	"time"

	queryengine "evolyn/internal/engine/query"
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

// FormDataCatalog 是 dashboard 域拥有的消费端窄口。生产适配器从 form 域读取
// 已发布表单和权限裁剪字段，dashboard 域不直接依赖表单模型或仓储。
type FormDataCatalog interface {
	ListForms(ctx context.Context, member *iammodel.User, appID uint) ([]FormDataSourceView, error)
	GetFields(ctx context.Context, member *iammodel.User, formCode string) (*FormFieldCatalogView, error)
}

type FormDataSourceView struct {
	AppID            uint
	Code             string
	Name             string
	PublishedVersion int
	SchemaRevision   string
}

type FormFieldView struct {
	FieldID, FieldCode, Label string
	Type                      string
	Filterable                bool
	Sortable                  bool
	Projectable               bool
	Groupable                 bool
	Aggregates                []string
}

type FormFieldCatalogView struct {
	AppID              uint
	FormCode, FormName string
	PublishedVersion   int
	SchemaRevision     string
	Fields             []FormFieldView
}

type FormDataCatalogInjector interface {
	UseFormDataCatalog(catalog FormDataCatalog)
}

type DashboardQueryExecutor interface {
	Execute(ctx context.Context, member *iammodel.User, formCode string, plan queryengine.LogicalPlan) (*DashboardQueryResultView, error)
}

type DashboardQueryResultColumnView struct{ Key, Label, Type string }

type DashboardQueryResultView struct {
	Columns  []DashboardQueryResultColumnView
	Rows     []map[string]any
	Total    int64
	Page     int
	PageSize int
}

type DashboardQueryExecutorInjector interface {
	UseDashboardQueryExecutor(executor DashboardQueryExecutor)
}

type QueryRuntimeConfig struct {
	Timeout       time.Duration
	MaxConcurrent int
}

type QueryRuntimeConfigInjector interface {
	UseQueryRuntimeConfig(config QueryRuntimeConfig)
}

type DashboardService interface {
	Precreate(ctx context.Context, member *iammodel.User, appCode string, req *model.PrecreateRequest) (*model.Detail, error)
	Get(ctx context.Context, member *iammodel.User, code string) (*model.Detail, error)
	GetRuntime(ctx context.Context, member *iammodel.User, code string) (*model.RuntimeBootstrap, error)
	Update(ctx context.Context, member *iammodel.User, code string, req *model.UpdateRequest) (*model.Detail, error)
	SaveDraft(ctx context.Context, member *iammodel.User, code string, req *model.SaveDraftRequest) (*model.SaveDraftResult, error)
	ListFormDataSources(ctx context.Context, member *iammodel.User, code string) ([]model.FormDataSource, error)
	GetFormFieldCatalog(ctx context.Context, member *iammodel.User, code, formCode string) (*model.FormFieldCatalog, error)
	PreviewWidgetQuery(ctx context.Context, member *iammodel.User, code, widgetID string, req *model.PreviewQueryRequest) (*model.PreviewQueryResult, error)
	RuntimeWidgetQuery(ctx context.Context, member *iammodel.User, code, widgetID string, req *model.RuntimeQueryRequest) (*model.PreviewQueryResult, error)
	Delete(ctx context.Context, member *iammodel.User, code string) error
}
