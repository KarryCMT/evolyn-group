package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"evolyn/internal/contextx"
	queryengine "evolyn/internal/engine/query"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/form/repository"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
)

const dashboardMaxAggregateGroups = 5000

type dashboardQueryField struct {
	fieldID string
	query   recordQueryField
	typeOf  queryengine.FieldType
}

// ExecuteDashboardQuery 在 form 域安全边界内完成 fieldId→发布字段→存储表达式
// 编译。dashboard 域永远拿不到物理表名、列名或权限 SQL。
func (s *formService) ExecuteDashboardQuery(ctx context.Context, member *iammodel.User, formCode string, plan queryengine.LogicalPlan) (*DashboardQueryResult, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID || !s.access.Permissions(ctx, member)["form-records:get"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot query form records"))
	}
	form, err := s.loadByCode(ctx, formCode)
	if err != nil {
		return nil, err
	}
	if form.LatestVersionID == nil {
		return nil, httpx.Wrap(apperrors.ErrNotPublished, fmt.Errorf("form %s not published", formCode))
	}
	version, err := s.versions.GetByID(ctx, *form.LatestVersionID)
	if err != nil {
		return nil, err
	}
	content := make(map[string]any)
	if err := json.Unmarshal(version.Content, &content); err != nil {
		return nil, fmt.Errorf("published schema decode: %w", err)
	}
	fieldList, err := buildPermissionFieldList(content)
	if err != nil {
		return nil, err
	}
	mappings, err := snapshotFieldMappings(version)
	if err != nil {
		return nil, err
	}
	fieldsByCode, err := queryFields(mappings, fieldList)
	if err != nil {
		return nil, err
	}
	resolved, err := s.evaluatePermissions(ctx, member, form.ID)
	if err != nil {
		return nil, err
	}
	if resolved != nil && !resolved.AllowsViewRecords() {
		return nil, httpx.Wrap(apperrors.ErrPermissionDenied, fmt.Errorf("member cannot view form %s", formCode))
	}
	visible := map[string]FieldPermission(nil)
	if resolved != nil {
		visible = resolved.FieldsForNew(model.PermissionOpView)
	}
	byID := make(map[string]dashboardQueryField, len(mappings))
	catalog := make(queryengine.FieldCatalog, len(mappings))
	for _, mapping := range mappings {
		field, exists := fieldsByCode[mapping.WidgetName]
		if !exists || mapping.FieldID == "" || (visible != nil && !visible[mapping.WidgetName].Visible) {
			continue
		}
		fieldType, supported := dashboardFieldType(field.meta)
		if !supported {
			continue
		}
		byID[mapping.FieldID] = dashboardQueryField{fieldID: mapping.FieldID, query: field, typeOf: fieldType}
		catalog[mapping.FieldID] = queryengine.FieldCapability{
			Type: fieldType, Filterable: true, Sortable: true, Projectable: true,
			Groupable: fieldType != queryengine.FieldDepartment, Aggregates: queryAggregatesOf(fieldType),
		}
	}
	document := queryengine.Document{
		Version: queryengine.Version, Filter: plan.Filter, Sorts: plan.Sorts,
		Paging: plan.Paging, Projection: plan.Projection, GroupBy: plan.GroupBy, Aggregates: plan.Aggregates,
	}
	validated := queryengine.Validate(document, catalog, queryengine.DefaultBudget())
	if len(validated.Issues) > 0 || validated.Plan == nil {
		if len(validated.Issues) > 0 {
			return nil, fmt.Errorf("%s at %s", validated.Issues[0].Code, validated.Issues[0].Path)
		}
		return nil, fmt.Errorf("dashboard query plan invalid")
	}
	plan = *validated.Plan
	if !plan.Aggregate && len(plan.Projection) == 0 {
		return nil, fmt.Errorf("detail query projection is empty")
	}

	opts, binding, err := s.physicalListBinding(ctx, form)
	if err != nil {
		return nil, err
	}
	translatedFilter, err := translateDashboardExpression(plan.Filter, byID)
	if err != nil {
		return nil, err
	}
	filter, err := CompileRecordListQuery(model.RecordQueryDocument{
		Version: queryengine.Version, Filter: translatedFilter,
		Paging: model.RecordQueryPaging{Page: plan.Paging.Page, PageSize: plan.Paging.PageSize},
	}, mappings, fieldList, opts)
	if err != nil {
		return nil, err
	}
	predicates := []CompiledRecordQuery{filter}
	if resolved != nil && !resolved.Admin && !resolved.Baseline {
		scopes := make([]CompiledRecordQuery, 0, len(resolved.Matched))
		for _, group := range resolved.Matched {
			if !group.Operations[model.PermissionOpView] {
				continue
			}
			scope, compileErr := CompilePermissionScopeSQL(group.DataScope, mappings, fieldList, opts)
			if compileErr != nil {
				return nil, compileErr
			}
			scopes = append(scopes, scope)
		}
		if len(scopes) == 0 {
			return &DashboardQueryResult{Columns: []DashboardQueryColumn{}, Rows: []map[string]any{}, Page: plan.Paging.Page, PageSize: plan.Paging.PageSize}, nil
		}
		predicates = append(predicates, joinCompiled(scopes, " OR "))
	}

	compiled, columns, outputTypes, err := compileDashboardProjection(plan, byID, opts)
	if err != nil {
		return nil, err
	}
	compiled.TenantID, compiled.FormID = tenantID, form.ID
	predicate := joinCompiled(predicates, " AND ")
	compiled.Where, compiled.WhereArgs = predicate.Where, predicate.Args
	if binding != nil {
		compiled.PhysicalTable = binding.TableName
	}

	var rows *repository.DashboardQueryRows
	queryFn := func(execCtx context.Context) error {
		var queryErr error
		rows, queryErr = s.records.QueryDashboard(execCtx, compiled)
		return queryErr
	}
	if binding != nil {
		err = s.tx.WithinTransaction(ctx, queryFn)
	} else {
		err = queryFn(ctx)
	}
	if err != nil {
		return nil, err
	}
	if plan.Aggregate && len(rows.Rows) > dashboardMaxAggregateGroups {
		return nil, fmt.Errorf("dashboard aggregate groups exceed %d", dashboardMaxAggregateGroups)
	}
	for _, row := range rows.Rows {
		for key, value := range row {
			row[key] = normalizeDashboardQueryValue(value, outputTypes[key])
		}
	}
	return &DashboardQueryResult{
		Columns: columns, Rows: rows.Rows, Total: rows.Total,
		Page: plan.Paging.Page, PageSize: plan.Paging.PageSize,
	}, nil
}

func translateDashboardExpression(expression *queryengine.Expression, fields map[string]dashboardQueryField) (*queryengine.Expression, error) {
	if expression == nil {
		return nil, nil
	}
	copy := *expression
	if copy.Type == "condition" {
		field, ok := fields[copy.Field]
		if !ok {
			return nil, fmt.Errorf("dashboard field %q unavailable", copy.Field)
		}
		copy.Field = field.query.mapping.WidgetName
		return &copy, nil
	}
	copy.Children = make([]queryengine.Expression, len(expression.Children))
	for i := range expression.Children {
		child, err := translateDashboardExpression(&expression.Children[i], fields)
		if err != nil {
			return nil, err
		}
		copy.Children[i] = *child
	}
	return &copy, nil
}

func compileDashboardProjection(plan queryengine.LogicalPlan, fields map[string]dashboardQueryField, opts RecordQueryCompileOptions) (repository.DashboardQueryParams, []DashboardQueryColumn, map[string]queryengine.FieldType, error) {
	params := repository.DashboardQueryParams{Aggregate: plan.Aggregate}
	columns := make([]DashboardQueryColumn, 0, len(plan.Projection)+len(plan.Aggregates))
	types := make(map[string]queryengine.FieldType)
	aliases := make(map[string]string)
	addField := func(fieldID, key, label string) error {
		field, ok := fields[fieldID]
		if !ok {
			return fmt.Errorf("dashboard field %q unavailable", fieldID)
		}
		expression, args := opts.compiler()(field.query)
		if field.typeOf == queryengine.FieldDecimal || field.typeOf == queryengine.FieldNumber {
			expression = "(" + expression + ")::numeric::text"
		}
		alias := fmt.Sprintf("c%d", len(params.Selects))
		params.Selects = append(params.Selects, expression+" AS "+alias)
		params.SelectArgs = append(params.SelectArgs, args...)
		params.Keys = append(params.Keys, key)
		aliases[fieldID] = alias
		columns = append(columns, DashboardQueryColumn{Key: key, Label: label, Type: field.typeOf})
		types[key] = field.typeOf
		return nil
	}

	if plan.Aggregate {
		for _, fieldID := range plan.GroupBy {
			field := fields[fieldID]
			if err := addField(fieldID, fieldID, field.query.meta.Label); err != nil {
				return params, nil, nil, err
			}
		}
		params.GroupByCount = len(plan.GroupBy)
		for _, aggregate := range plan.Aggregates {
			field, ok := fields[aggregate.Field]
			if !ok {
				return params, nil, nil, fmt.Errorf("aggregate field %q unavailable", aggregate.Field)
			}
			expression, args := opts.compiler()(field.query)
			function := strings.ToUpper(string(aggregate.Operator))
			if aggregate.Operator != queryengine.AggregateCount && (field.typeOf == queryengine.FieldDecimal || field.typeOf == queryengine.FieldNumber) {
				expression = function + "((" + expression + ")::numeric)::text"
			} else {
				expression = function + "(" + expression + ")"
			}
			alias := fmt.Sprintf("c%d", len(params.Selects))
			outputType := field.typeOf
			if aggregate.Operator == queryengine.AggregateCount {
				outputType = queryengine.FieldNumber
			}
			params.Selects = append(params.Selects, expression+" AS "+alias)
			params.SelectArgs = append(params.SelectArgs, args...)
			params.Keys = append(params.Keys, aggregate.Alias)
			aliases[aggregate.Alias] = alias
			columns = append(columns, DashboardQueryColumn{Key: aggregate.Alias, Label: aggregate.Alias, Type: outputType})
			types[aggregate.Alias] = outputType
		}
		params.Limit = dashboardMaxAggregateGroups + 1
		for _, sort := range plan.Sorts {
			if alias := aliases[sort.Field]; alias != "" {
				params.OrderBy += orderSeparator(params.OrderBy) + alias + " " + strings.ToUpper(sort.Direction)
			}
		}
		if params.OrderBy == "" && params.GroupByCount > 0 {
			params.OrderBy = "c0 ASC"
		}
		return params, columns, types, nil
	}

	for _, fieldID := range plan.Projection {
		field := fields[fieldID]
		if err := addField(fieldID, fieldID, field.query.meta.Label); err != nil {
			return params, nil, nil, err
		}
	}
	params.Limit = plan.Paging.PageSize
	params.Offset = (plan.Paging.Page - 1) * plan.Paging.PageSize
	for _, sort := range plan.Sorts {
		alias := aliases[sort.Field]
		if alias == "" {
			return params, nil, nil, fmt.Errorf("sort field %q must be projected", sort.Field)
		}
		params.OrderBy += orderSeparator(params.OrderBy) + alias + " " + strings.ToUpper(sort.Direction)
	}
	params.OrderBy += orderSeparator(params.OrderBy) + "r.id DESC"
	return params, columns, types, nil
}

func orderSeparator(value string) string {
	if value == "" {
		return ""
	}
	return ", "
}

func normalizeDashboardQueryValue(value any, fieldType queryengine.FieldType) any {
	if value == nil {
		return nil
	}
	if bytes, ok := value.([]byte); ok {
		return string(bytes)
	}
	if timestamp, ok := value.(time.Time); ok {
		return timestamp.Format("2006-01-02 15:04:05")
	}
	if fieldType == queryengine.FieldDecimal || fieldType == queryengine.FieldNumber {
		return fmt.Sprint(value)
	}
	return value
}
