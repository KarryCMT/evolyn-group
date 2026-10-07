package service

import (
	"context"
	"fmt"

	queryengine "evolyn/internal/engine/query"
	dashboarderrors "evolyn/internal/platform/dashboard"
	"evolyn/internal/platform/dashboard/model"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
)

func (s *dashboardService) ListFormDataSources(ctx context.Context, member *iammodel.User, code string) ([]model.FormDataSource, error) {
	dashboard, err := s.loadForDataDesign(ctx, member, code)
	if err != nil {
		return nil, err
	}
	if s.forms == nil {
		return nil, httpx.Wrap(dashboarderrors.ErrDataSourceUnavailable, fmt.Errorf("form data catalog not configured"))
	}
	views, err := s.forms.ListForms(ctx, member, dashboard.AppID)
	if err != nil {
		return nil, httpx.Wrap(dashboarderrors.ErrDataSourceUnavailable, err)
	}
	items := make([]model.FormDataSource, 0, len(views))
	for _, view := range views {
		if view.AppID != dashboard.AppID {
			continue
		}
		items = append(items, model.FormDataSource{
			Code: view.Code, Name: view.Name, PublishedVersion: view.PublishedVersion,
			SchemaRevision: view.SchemaRevision,
		})
	}
	return items, nil
}

func (s *dashboardService) GetFormFieldCatalog(ctx context.Context, member *iammodel.User, code, formCode string) (*model.FormFieldCatalog, error) {
	dashboard, err := s.loadForDataDesign(ctx, member, code)
	if err != nil {
		return nil, err
	}
	if s.forms == nil {
		return nil, httpx.Wrap(dashboarderrors.ErrDataSourceUnavailable, fmt.Errorf("form data catalog not configured"))
	}
	view, err := s.forms.GetFields(ctx, member, formCode)
	if err != nil || view == nil || view.AppID != dashboard.AppID {
		if err == nil {
			err = fmt.Errorf("form %s is outside dashboard app", formCode)
		}
		return nil, httpx.Wrap(dashboarderrors.ErrDataSourceUnavailable, err)
	}
	fields := make([]model.DataSourceField, 0, len(view.Fields))
	for _, field := range view.Fields {
		aggregates := make([]queryengine.AggregateOperator, 0, len(field.Aggregates))
		for _, aggregate := range field.Aggregates {
			aggregates = append(aggregates, queryengine.AggregateOperator(aggregate))
		}
		fields = append(fields, model.DataSourceField{
			FieldID: field.FieldID, FieldCode: field.FieldCode, Label: field.Label,
			Type: queryengine.FieldType(field.Type), Filterable: field.Filterable,
			Sortable: field.Sortable, Projectable: field.Projectable,
			Groupable: field.Groupable, Aggregates: aggregates,
		})
	}
	return &model.FormFieldCatalog{
		FormCode: view.FormCode, FormName: view.FormName,
		PublishedVersion: view.PublishedVersion, SchemaRevision: view.SchemaRevision,
		Fields: fields,
	}, nil
}

func (s *dashboardService) loadForDataDesign(ctx context.Context, member *iammodel.User, code string) (*model.Dashboard, error) {
	permissions := s.access.Permissions(ctx, member)
	if !permissions["dashboards:get"] || !permissions["dashboard-actions:design"] {
		return nil, httpx.Wrap(dashboarderrors.ErrForbidden, fmt.Errorf("member cannot design dashboard data"))
	}
	return s.load(ctx, code)
}
