package server

import (
	"context"

	queryengine "evolyn/internal/engine/query"
	dashboardservice "evolyn/internal/platform/dashboard/service"
	formservice "evolyn/internal/platform/form/service"
	iammodel "evolyn/internal/platform/iam/model"
)

// dashboardFormDataCatalog 是 dashboard 消费端窄口的装配适配器。表单的
// 发布状态、租户边界、行/字段权限均由 form 域裁决，适配层只做模型投影。
type dashboardFormDataCatalog struct{ forms formservice.FormDataCatalog }

func (a dashboardFormDataCatalog) ListForms(ctx context.Context, member *iammodel.User, appID uint) ([]dashboardservice.FormDataSourceView, error) {
	items, err := a.forms.ListDashboardDataSources(ctx, member, appID)
	if err != nil {
		return nil, err
	}
	views := make([]dashboardservice.FormDataSourceView, 0, len(items))
	for _, item := range items {
		views = append(views, dashboardservice.FormDataSourceView{
			AppID: item.AppID, Code: item.Code, Name: item.Name,
			PublishedVersion: item.PublishedVersion, SchemaRevision: item.SchemaRevision,
		})
	}
	return views, nil
}

// dashboardFormQueryExecutor 保持查询执行在 form 域内，dashboard 域只持有
// 逻辑计划和稳定结果视图。
type dashboardFormQueryExecutor struct {
	forms formservice.DashboardQueryExecutor
}

func (a dashboardFormQueryExecutor) Execute(ctx context.Context, member *iammodel.User, formCode string, plan queryengine.LogicalPlan) (*dashboardservice.DashboardQueryResultView, error) {
	result, err := a.forms.ExecuteDashboardQuery(ctx, member, formCode, plan)
	if err != nil {
		return nil, err
	}
	columns := make([]dashboardservice.DashboardQueryResultColumnView, 0, len(result.Columns))
	for _, column := range result.Columns {
		columns = append(columns, dashboardservice.DashboardQueryResultColumnView{
			Key: column.Key, Label: column.Label, Type: string(column.Type),
		})
	}
	return &dashboardservice.DashboardQueryResultView{
		Columns: columns, Rows: result.Rows, Total: result.Total,
		Page: result.Page, PageSize: result.PageSize,
	}, nil
}

func (a dashboardFormDataCatalog) GetFields(ctx context.Context, member *iammodel.User, formCode string) (*dashboardservice.FormFieldCatalogView, error) {
	catalog, err := a.forms.GetDashboardFieldCatalog(ctx, member, formCode)
	if err != nil {
		return nil, err
	}
	fields := make([]dashboardservice.FormFieldView, 0, len(catalog.Fields))
	for _, field := range catalog.Fields {
		aggregates := make([]string, 0, len(field.Aggregates))
		for _, aggregate := range field.Aggregates {
			aggregates = append(aggregates, string(aggregate))
		}
		fields = append(fields, dashboardservice.FormFieldView{
			FieldID: field.FieldID, FieldCode: field.FieldCode, Label: field.Label,
			Type: string(field.Type), Filterable: field.Filterable, Sortable: field.Sortable,
			Projectable: field.Projectable, Groupable: field.Groupable, Aggregates: aggregates,
		})
	}
	return &dashboardservice.FormFieldCatalogView{
		AppID: catalog.AppID, FormCode: catalog.FormCode, FormName: catalog.FormName,
		PublishedVersion: catalog.PublishedVersion, SchemaRevision: catalog.SchemaRevision,
		Fields: fields,
	}, nil
}
