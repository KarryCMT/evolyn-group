package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	enginedashboard "evolyn/internal/engine/dashboard"
	queryengine "evolyn/internal/engine/query"
	dashboarderrors "evolyn/internal/platform/dashboard"
	"evolyn/internal/platform/dashboard/model"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
)

// PreviewWidgetQuery 只从服务端已保存草稿恢复 Dataset。请求不能提交 Query
// AST、字段、聚合或数据源，避免设计器通过调试接口扩大访问范围。
func (s *dashboardService) PreviewWidgetQuery(ctx context.Context, member *iammodel.User, code, widgetID string, req *model.PreviewQueryRequest) (*model.PreviewQueryResult, error) {
	permissions := s.access.Permissions(ctx, member)
	if !permissions["dashboards:get"] || !permissions["dashboard-actions:design"] || !permissions["dashboard-actions:preview"] {
		return nil, httpx.Wrap(dashboarderrors.ErrForbidden, fmt.Errorf("member cannot preview dashboard"))
	}
	if req == nil || req.DraftRevision <= 0 || strings.TrimSpace(widgetID) == "" {
		return nil, httpx.Wrap(dashboarderrors.ErrQueryInvalid, fmt.Errorf("invalid preview query request"))
	}
	return s.querySavedDashboardWidget(ctx, member, code, widgetID, req.DraftRevision, req.Page, req.FilterValues)
}

// RuntimeWidgetQuery 与设计预览共用服务端已保存文档和资源护栏，但只接受
// dashboard-actions:view。浏览器不能提交 Dataset 或 Query AST。
func (s *dashboardService) RuntimeWidgetQuery(ctx context.Context, member *iammodel.User, code, widgetID string, req *model.RuntimeQueryRequest) (*model.PreviewQueryResult, error) {
	if !s.access.Permissions(ctx, member)["dashboard-actions:view"] {
		return nil, httpx.Wrap(dashboarderrors.ErrForbidden, fmt.Errorf("member cannot query dashboard runtime"))
	}
	if req == nil || req.Version <= 0 || strings.TrimSpace(widgetID) == "" {
		return nil, httpx.Wrap(dashboarderrors.ErrQueryInvalid, fmt.Errorf("invalid runtime query request"))
	}
	return s.querySavedDashboardWidget(ctx, member, code, widgetID, req.Version, req.Page, req.FilterValues)
}

// querySavedDashboardWidget 从服务端文档恢复可信查询。version 既防止资产切换
// 的迟到请求，也确保定义与组件结果来自同一个已保存修订。
func (s *dashboardService) querySavedDashboardWidget(ctx context.Context, member *iammodel.User, code, widgetID string, version int64, page int, filterValues map[string]any) (*model.PreviewQueryResult, error) {
	if len(filterValues) > 0 {
		return nil, httpx.Wrap(dashboarderrors.ErrQueryInvalid, fmt.Errorf("runtime filters are not declared in dashboard v1"))
	}
	dashboard, err := s.load(ctx, code)
	if err != nil {
		return nil, err
	}
	if dashboard.DraftRevision != version {
		return nil, httpx.Wrap(dashboarderrors.ErrDraftConflict, fmt.Errorf("dashboard revision stale"))
	}
	normalized := enginedashboard.Normalize(dashboard.DraftContent)
	if len(normalized.Issues) > 0 {
		return nil, httpx.Wrap(dashboarderrors.ErrSchemaInvalid.WithData(map[string]any{"issues": normalized.Issues}), fmt.Errorf("saved dashboard draft invalid"))
	}
	widget, dataset := previewWidgetAndDataset(&normalized.Document, widgetID)
	if widget == nil || dataset == nil {
		return nil, httpx.Wrap(dashboarderrors.ErrQueryInvalid, fmt.Errorf("widget or dataset unavailable"))
	}
	document := dataset.Query
	if widget.Type == "table" {
		var settings enginedashboard.TableSettings
		if err := json.Unmarshal(widget.Settings, &settings); err != nil {
			return nil, httpx.Wrap(dashboarderrors.ErrQueryInvalid, err)
		}
		document.Projection = make([]string, 0, len(settings.Columns))
		for _, column := range settings.Columns {
			document.Projection = append(document.Projection, column.Field.FieldID)
		}
		document.Sorts = make([]queryengine.Sort, 0, len(settings.Sorts))
		for _, sort := range settings.Sorts {
			document.Sorts = append(document.Sorts, queryengine.Sort{Field: sort.Field.FieldID, Direction: sort.Direction})
		}
		if page < 1 {
			page = 1
		}
		document.Paging = queryengine.Paging{Page: page, PageSize: settings.Pagination.PageSize}
	}
	result, err := s.executePreviewPlan(ctx, member, dataset.Source.FormCode, queryengine.BuildLogicalPlan(document))
	if err != nil {
		return nil, err
	}
	columns := make([]model.QueryResultColumn, 0, len(result.Columns))
	for _, column := range result.Columns {
		columns = append(columns, model.QueryResultColumn{Key: column.Key, Label: column.Label, Type: column.Type})
	}
	return &model.PreviewQueryResult{
		DatasetID: dataset.ID, Columns: columns, Rows: result.Rows, Total: result.Total,
		Page: result.Page, PageSize: result.PageSize,
	}, nil
}

// executePreviewPlan 统一执行超时、并发槽位和稳定错误分类。单独收口后，设计
// 预览与后续发布态查询可以复用完全相同的资源护栏。
func (s *dashboardService) executePreviewPlan(ctx context.Context, member *iammodel.User, formCode string, plan queryengine.LogicalPlan) (*DashboardQueryResultView, error) {
	if s.queries == nil {
		return nil, httpx.Wrap(dashboarderrors.ErrQueryInvalid, fmt.Errorf("dashboard query executor not configured"))
	}
	select {
	case s.querySlots <- struct{}{}:
		defer func() { <-s.querySlots }()
	default:
		return nil, httpx.Wrap(dashboarderrors.ErrQueryLimitExceeded, fmt.Errorf("dashboard query concurrency limit reached"))
	}
	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	result, err := s.queries.Execute(queryCtx, member, formCode, plan)
	if err == nil {
		return result, nil
	}
	if queryCtx.Err() != nil {
		return nil, httpx.Wrap(dashboarderrors.ErrQueryLimitExceeded, queryCtx.Err())
	}
	if strings.Contains(err.Error(), "exceed") {
		return nil, httpx.Wrap(dashboarderrors.ErrQueryLimitExceeded, err)
	}
	return nil, httpx.Wrap(dashboarderrors.ErrQueryInvalid, err)
}

func previewWidgetAndDataset(document *enginedashboard.Document, widgetID string) (*enginedashboard.Widget, *enginedashboard.Dataset) {
	var widget *enginedashboard.Widget
	for i := range document.Widgets {
		if document.Widgets[i].ID == widgetID {
			widget = &document.Widgets[i]
			break
		}
	}
	if widget == nil || widget.DatasetID == "" {
		return nil, nil
	}
	for i := range document.Datasets {
		if document.Datasets[i].ID == widget.DatasetID {
			return widget, &document.Datasets[i]
		}
	}
	return nil, nil
}
