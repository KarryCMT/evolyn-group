package service

import (
	"fmt"
	"strings"

	queryengine "evolyn/internal/engine/query"
	"evolyn/internal/platform/form/model"
)

const recordQueryDSLVersion = 1

// CompileRecordListQuery validates the public Query DSL against the immutable
// field mappings and returns one parameterized predicate. Sorting is compiled
// separately by CompileRecordListSorts (system fields only); projection,
// grouping and aggregates stay rejected until those result shapes have an API.
// opts 选择值表达式解析模式（legacy JSONB / physical 物理列）。
func CompileRecordListQuery(document model.RecordQueryDocument, mappings []SnapshotFieldMapping, fieldList []permissionFieldMeta, opts ...RecordQueryCompileOptions) (CompiledRecordQuery, error) {
	options := compileOptionsOf(opts...)
	if document.Version != 0 && document.Version != recordQueryDSLVersion {
		return CompiledRecordQuery{}, fmt.Errorf("unsupported query DSL version %d", document.Version)
	}
	if len(document.Projection) > 0 || len(document.GroupBy) > 0 || len(document.Aggregates) > 0 {
		return CompiledRecordQuery{}, fmt.Errorf("projection, grouping and aggregates are not supported by record list")
	}
	fields, err := queryFields(mappings, fieldList)
	if err != nil {
		return CompiledRecordQuery{}, err
	}
	if err := options.validatePhysicalColumns(fields); err != nil {
		return CompiledRecordQuery{}, err
	}
	page, pageSize := normalizeRecordListPaging(document.Paging)
	version := document.Version
	if version == 0 {
		version = queryengine.Version
	}
	validated := queryengine.Validate(queryengine.Document{
		Version: version,
		Filter:  document.Filter,
		Sorts:   document.Sorts,
		Paging:  queryengine.Paging{Page: page, PageSize: pageSize},
	}, recordQueryCapabilities(fields), queryengine.Budget{MaxSorts: 3})
	if len(validated.Issues) > 0 {
		issue := validated.Issues[0]
		return CompiledRecordQuery{}, fmt.Errorf("%s at %s", issue.Code, issue.Path)
	}
	if validated.Document == nil || validated.Document.Filter == nil {
		return CompiledRecordQuery{Where: "TRUE"}, nil
	}
	return compileRecordExpression(*validated.Document.Filter, fields, 0, options)
}

// recordQueryCapabilities 把发布字段解析结果投影为纯查询内核能力目录。目录是
// SQL 编译前的共同白名单，dashboard 聚合适配器也复用同一字段解析结论。
func recordQueryCapabilities(fields map[string]recordQueryField) queryengine.FieldCatalog {
	catalog := make(queryengine.FieldCatalog, len(fields)+len(systemFieldColumns))
	for name, field := range fields {
		fieldType := queryFieldTypeOf(field)
		catalog[name] = queryengine.FieldCapability{
			Type: fieldType, Filterable: true, Projectable: true,
			Groupable: fieldType != queryengine.FieldDepartment,
			Sortable:  false, Aggregates: queryAggregatesOf(fieldType),
		}
	}
	for name := range systemFieldColumns {
		fieldType := queryengine.FieldText
		switch name {
		case SysFieldSubmittedAt, SysFieldUpdatedAt, SysFieldWorkflowUpdatedAt:
			fieldType = queryengine.FieldDateTime
		case SysFieldSubmittedBy, SysFieldUpdatedBy:
			fieldType = queryengine.FieldMember
		}
		catalog[name] = queryengine.FieldCapability{
			Type: fieldType, Filterable: true, Sortable: true, Projectable: true,
			Groupable: true, Aggregates: queryAggregatesOf(fieldType),
		}
	}
	return catalog
}

func queryFieldTypeOf(field recordQueryField) queryengine.FieldType {
	switch field.class {
	case permFieldClassNumber:
		return queryengine.FieldDecimal
	case permFieldClassDateTime:
		return queryengine.FieldDateTime
	case permFieldClassSingleOption, permFieldClassMultiOption:
		switch field.mapping.WidgetType {
		case "user", "usergroup":
			return queryengine.FieldMember
		case "dept", "deptgroup":
			return queryengine.FieldDepartment
		}
		return queryengine.FieldEnum
	default:
		return queryengine.FieldText
	}
}

func queryAggregatesOf(fieldType queryengine.FieldType) []queryengine.AggregateOperator {
	if fieldType == queryengine.FieldNumber || fieldType == queryengine.FieldDecimal {
		return []queryengine.AggregateOperator{queryengine.AggregateCount, queryengine.AggregateSum, queryengine.AggregateAvg, queryengine.AggregateMin, queryengine.AggregateMax}
	}
	return []queryengine.AggregateOperator{queryengine.AggregateCount, queryengine.AggregateMin, queryengine.AggregateMax}
}

func compileRecordExpression(expression model.RecordQueryExpression, fields map[string]recordQueryField, depth int, options RecordQueryCompileOptions) (CompiledRecordQuery, error) {
	if depth > 12 {
		return CompiledRecordQuery{}, fmt.Errorf("query filter nesting exceeds 12 levels")
	}
	switch expression.Type {
	case "condition":
		trimmed := strings.TrimSpace(expression.Field)
		// 系统字段（sys.*）走物理列编译，与 widgetName 白名单互斥；physical
		// 模式下前缀按字段分派（流程三字段挂物理表 d，其余挂信封表 r，
		// JOIN 后消除歧义并命中物理侧预置索引）
		if IsRecordSystemField(trimmed) {
			return compileSystemRecordCondition(trimmed, expression.Operator, expression.Value, options.systemFieldPrefix(trimmed))
		}
		field, ok := fields[trimmed]
		if !ok {
			return CompiledRecordQuery{}, fmt.Errorf("query field %q is not present in published field mappings", expression.Field)
		}
		return compileUserCondition(field, expression.Operator, expression.Value, options.compiler(), options.PhysicalArrayColumns[field.mapping.WidgetName])
	case "group":
		if len(expression.Children) == 0 || len(expression.Children) > 50 {
			return CompiledRecordQuery{}, fmt.Errorf("query group must contain 1 to 50 children")
		}
		joiner := " AND "
		if expression.Conjunction == "or" {
			joiner = " OR "
		} else if expression.Conjunction != "and" {
			return CompiledRecordQuery{}, fmt.Errorf("query conjunction %q is invalid", expression.Conjunction)
		}
		parts := make([]CompiledRecordQuery, 0, len(expression.Children))
		for _, child := range expression.Children {
			part, err := compileRecordExpression(child, fields, depth+1, options)
			if err != nil {
				return CompiledRecordQuery{}, err
			}
			parts = append(parts, part)
		}
		return joinCompiled(parts, joiner), nil
	default:
		return CompiledRecordQuery{}, fmt.Errorf("query expression type %q is invalid", expression.Type)
	}
}

// CompileRecordKeyword builds a controlled OR predicate across frozen searchable
// fields. The user can provide only the bound keyword, never a JSONB key/path
// or physical column name.
func CompileRecordKeyword(keyword string, mappings []SnapshotFieldMapping, fieldList []permissionFieldMeta, opts ...RecordQueryCompileOptions) (CompiledRecordQuery, error) {
	options := compileOptionsOf(opts...)
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return CompiledRecordQuery{Where: "TRUE"}, nil
	}
	fields, err := queryFields(mappings, fieldList)
	if err != nil {
		return CompiledRecordQuery{}, err
	}
	if err := options.validatePhysicalColumns(fields); err != nil {
		return CompiledRecordQuery{}, err
	}
	vc := options.compiler()
	parts := make([]CompiledRecordQuery, 0, len(fields))
	for _, field := range fields {
		if field.class == permFieldClassMultiOption {
			continue
		}
		valueSQL, valueArgs := vc(field)
		parts = append(parts, CompiledRecordQuery{Where: "(" + valueSQL + ") IS NOT NULL AND (" + valueSQL + ") ILIKE ? ESCAPE '\\\\'", Args: append(append([]any{}, valueArgs...), append(valueArgs, "%"+escapeLike(keyword)+"%")...)})
	}
	if len(parts) == 0 {
		return CompiledRecordQuery{Where: "FALSE"}, nil
	}
	return joinCompiled(parts, " OR "), nil
}
