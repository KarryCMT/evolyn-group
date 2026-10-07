package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"evolyn/internal/contextx"
	queryengine "evolyn/internal/engine/query"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/form/repository"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
)

// ListDashboardDataSources 返回同应用内已发布且成员可查看记录的表单。权限组
// 判定批量执行，避免数据源面板按表单逐项查询形成 N+1。
func (s *formService) ListDashboardDataSources(ctx context.Context, member *iammodel.User, appID uint) ([]model.DashboardDataSource, error) {
	if err := s.validateDashboardCatalogMember(ctx, member); err != nil {
		return nil, err
	}
	if appID == 0 {
		return nil, httpx.Wrap(apperrors.ErrFormAppInvalid, fmt.Errorf("app id required"))
	}

	forms := make([]model.Form, 0)
	params := repository.ListParams{AppID: appID, Limit: maxListLimit}
	for {
		page, hasMore, err := s.repo.List(ctx, params)
		if err != nil {
			return nil, err
		}
		forms = append(forms, page...)
		if !hasMore || len(page) == 0 {
			break
		}
		params.HasCursor = true
		params.AfterID = page[len(page)-1].ID
	}

	formIDs := make([]uint, 0, len(forms))
	for i := range forms {
		if forms[i].LatestVersionID != nil {
			formIDs = append(formIDs, forms[i].ID)
		}
	}
	permissions := map[uint]*ResolvedFormPermission{}
	if s.permissions != nil && len(formIDs) > 0 {
		var err error
		permissions, err = s.permissions.EvaluateForForms(ctx, member, formIDs)
		if err != nil {
			return nil, err
		}
	}

	items := make([]model.DashboardDataSource, 0, len(formIDs))
	for i := range forms {
		form := &forms[i]
		if form.LatestVersionID == nil {
			continue
		}
		if s.permissions != nil {
			resolved, exists := permissions[form.ID]
			if !exists || resolved == nil || !resolved.AllowsViewRecords() {
				continue
			}
		}
		version, err := s.versions.GetByID(ctx, *form.LatestVersionID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return nil, err
		}
		items = append(items, model.DashboardDataSource{
			AppID: form.AppID, Code: form.Code, Name: form.Name,
			PublishedVersion: version.VersionNo,
			SchemaRevision:   strconv.FormatInt(version.SchemaRevision, 10),
		})
	}
	return items, nil
}

// GetDashboardFieldCatalog 从不可变发布快照构造稳定字段目录，并按 view 字段
// 矩阵裁剪。v8 前没有 fieldId 的旧快照不暴露给仪表盘，避免持久化易变字段名。
func (s *formService) GetDashboardFieldCatalog(ctx context.Context, member *iammodel.User, formCode string) (*model.DashboardFieldCatalog, error) {
	if err := s.validateDashboardCatalogMember(ctx, member); err != nil {
		return nil, err
	}
	form, err := s.loadByCode(ctx, formCode)
	if err != nil {
		return nil, err
	}
	if form.LatestVersionID == nil {
		return nil, httpx.Wrap(apperrors.ErrNotPublished, fmt.Errorf("form %s not published", formCode))
	}
	resolved, err := s.evaluatePermissions(ctx, member, form.ID)
	if err != nil {
		return nil, err
	}
	if resolved != nil && !resolved.AllowsViewRecords() {
		return nil, httpx.Wrap(apperrors.ErrPermissionDenied, fmt.Errorf("member cannot view form %s", formCode))
	}

	version, err := s.versions.GetByID(ctx, *form.LatestVersionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.Wrap(apperrors.ErrNotPublished, err)
		}
		return nil, err
	}
	content := make(map[string]any)
	if err := json.Unmarshal(version.Content, &content); err != nil {
		return nil, fmt.Errorf("published schema decode: %w", err)
	}
	fieldList, err := buildPermissionFieldList(content)
	if err != nil {
		return nil, fmt.Errorf("published field list: %w", err)
	}
	metaByCode := permissionFieldIndex(fieldList)
	mappings, err := snapshotFieldMappings(version)
	if err != nil {
		return nil, err
	}
	visible := map[string]FieldPermission(nil)
	if resolved != nil {
		visible = resolved.FieldsForNew(model.PermissionOpView)
	}

	fields := make([]model.DashboardFieldCapability, 0, len(mappings))
	for _, mapping := range mappings {
		meta, ok := metaByCode[mapping.WidgetName]
		if !ok || mapping.FieldID == "" || (visible != nil && !visible[mapping.WidgetName].Visible) {
			continue
		}
		fieldType, ok := dashboardFieldType(meta)
		if !ok {
			continue
		}
		fields = append(fields, model.DashboardFieldCapability{
			FieldID: mapping.FieldID, FieldCode: mapping.WidgetName, Label: meta.Label,
			Type: fieldType, Filterable: true, Sortable: true, Projectable: true,
			Groupable:  fieldType != queryengine.FieldDepartment,
			Aggregates: queryAggregatesOf(fieldType),
		})
	}
	return &model.DashboardFieldCatalog{
		AppID: form.AppID, FormCode: form.Code, FormName: form.Name,
		PublishedVersion: version.VersionNo,
		SchemaRevision:   strconv.FormatInt(version.SchemaRevision, 10),
		Fields:           fields,
	}, nil
}

func (s *formService) validateDashboardCatalogMember(ctx context.Context, member *iammodel.User) error {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID || !s.access.Permissions(ctx, member)["form-records:get"] {
		return httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot read form data catalog"))
	}
	return nil
}

func dashboardFieldType(field permissionFieldMeta) (queryengine.FieldType, bool) {
	switch field.WidgetType {
	case "text", "textarea", "sn":
		return queryengine.FieldText, true
	case "number":
		return queryengine.FieldNumber, true
	case "decimal", "money", "percent":
		return queryengine.FieldDecimal, true
	case "datetime":
		if field.Format == "date" {
			return queryengine.FieldDate, true
		}
		return queryengine.FieldDateTime, true
	case "radiogroup", "combo", "checkboxgroup", "combocheck":
		return queryengine.FieldEnum, true
	case "user", "usergroup":
		return queryengine.FieldMember, true
	case "dept", "deptgroup":
		return queryengine.FieldDepartment, true
	default:
		return "", false
	}
}
