package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"evolyn/internal/contextx"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/form/repository"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
)

const (
	defaultRecordListPageSize = 20
	maxRecordListPageSize     = 100
)

// ListRecords applies the user Query DSL and the member's record-level view
// scopes as one database predicate before Count/Offset pagination. The later
// field projection is intentionally a second, per-record permission decision:
// data scopes decide which rows exist; field matrices decide what each row shows.
func (s *formService) ListRecords(ctx context.Context, member *iammodel.User, code string, query model.RecordQueryDocument) (*model.FormRecordPage, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member not in tenant %d", tenantID))
	}
	if !s.access.Permissions(ctx, member)["form-records:get"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot list form records"))
	}
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if form.LatestVersionID == nil {
		return nil, httpx.Wrap(apperrors.ErrNotPublished, fmt.Errorf("form %s not published", code))
	}
	version, err := s.versions.GetByID(ctx, *form.LatestVersionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.Wrap(apperrors.ErrNotPublished, err)
		}
		return nil, err
	}
	// 000065 之前已存在的不可变发布快照不会被回写 field_mappings；这类
	// 快照只能从其自身的 Content 推导一次只读映射，绝不能根据草稿或记录
	// values 猜字段。新快照始终以冻结列为唯一事实源。
	mappings, err := snapshotFieldMappings(version)
	if err != nil {
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

	page, pageSize := normalizeRecordListPaging(query.Paging)
	userFilter, err := CompileRecordListQuery(query, mappings, fieldList)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrRecordQueryInvalid, err)
	}
	keywordFilter, err := CompileRecordKeyword(query.Keyword, mappings, fieldList)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrRecordQueryInvalid, err)
	}
	predicates := []CompiledRecordQuery{userFilter, keywordFilter}
	resolved, err := s.evaluatePermissions(ctx, member, form.ID)
	if err != nil {
		return nil, err
	}
	if resolved != nil && !resolved.Admin && !resolved.Baseline {
		scopes := make([]CompiledRecordQuery, 0, len(resolved.Matched))
		for _, group := range resolved.Matched {
			if !group.Operations[model.PermissionOpView] {
				continue
			}
			scope, cerr := CompilePermissionScopeSQL(group.DataScope, mappings, fieldList)
			if cerr != nil {
				return nil, fmt.Errorf("compile view scope for group %s: %w", group.Code, cerr)
			}
			scopes = append(scopes, scope)
		}
		if len(scopes) == 0 {
			return &model.FormRecordPage{Items: []model.FormRecordDTO{}, Page: page, PageSize: pageSize}, nil
		}
		predicates = append(predicates, joinCompiled(scopes, " OR "))
	}
	predicate := joinCompiled(predicates, " AND ")
	// 排序先行编译：仅系统字段可排序（CompileRecordListSorts 白名单），非法
	// 排序与非法筛选同样以 FORM_RECORD_QUERY_INVALID 拒绝。
	orderBy, err := CompileRecordListSorts(query.Sorts)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrRecordQueryInvalid, err)
	}
	records, total, err := s.records.ListControlled(ctx, repository.RecordListParams{FormID: form.ID, Page: page, PageSize: pageSize, Where: predicate.Where, Args: predicate.Args, OrderBy: orderBy})
	if err != nil {
		return nil, err
	}
	items := make([]model.FormRecordDTO, 0, len(records))
	allowed := make(map[string]bool, len(mappings))
	for _, mapping := range mappings {
		allowed[mapping.WidgetName] = true
	}
	for _, record := range records {
		values := make(map[string]any)
		if err := json.Unmarshal(record.Values, &values); err != nil {
			return nil, fmt.Errorf("record %d values decode: %w", record.ID, err)
		}
		if resolved != nil {
			fields := resolved.FieldsFor(model.PermissionOpView, values)
			for key := range allowed {
				permission, exists := fields[key]
				if !exists || !permission.Visible {
					delete(values, key)
				}
			}
		}
		for key := range values {
			if !allowed[key] {
				delete(values, key)
			}
		}
		// 系统字段不参与字段矩阵裁剪：行级可见即可见；提交人快照为空（昵称
		// 未设置的边缘态或 000067 前未回填命中）回落固定文案保证可读。
		submittedByName := record.SubmittedByName
		if strings.TrimSpace(submittedByName) == "" {
			submittedByName = "成员"
		}
		items = append(items, model.FormRecordDTO{ID: record.ID, Values: values, SubmittedByMemberID: record.SubmittedByMemberID, SubmittedByName: submittedByName, SubmittedAt: record.SubmittedAt, UpdatedAt: record.UpdatedAt})
	}
	return &model.FormRecordPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func snapshotFieldMappings(version *model.FormVersion) ([]SnapshotFieldMapping, error) {
	mappings := make([]SnapshotFieldMapping, 0)
	if len(version.FieldMappings) > 0 && string(version.FieldMappings) != "[]" {
		if err := json.Unmarshal(version.FieldMappings, &mappings); err != nil {
			return nil, fmt.Errorf("published field mappings decode: %w", err)
		}
		return mappings, nil
	}
	content := make(map[string]any)
	if err := json.Unmarshal(version.Content, &content); err != nil {
		return nil, fmt.Errorf("legacy published schema decode: %w", err)
	}
	return ExtractSnapshotFieldMappings(content), nil
}

func normalizeRecordListPaging(paging model.RecordQueryPaging) (int, int) {
	page, pageSize := paging.Page, paging.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultRecordListPageSize
	}
	if pageSize > maxRecordListPageSize {
		pageSize = maxRecordListPageSize
	}
	return page, pageSize
}
