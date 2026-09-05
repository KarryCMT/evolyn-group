package service

import (
	"fmt"
	"strings"

	"evolyn/internal/platform/form/model"
)

const recordQueryDSLVersion = 1

// CompileRecordListQuery validates the public Query DSL against the immutable
// field mappings and returns one parameterized predicate. Sorting is compiled
// separately by CompileRecordListSorts (system fields only); projection,
// grouping and aggregates stay rejected until those result shapes have an API.
func CompileRecordListQuery(document model.RecordQueryDocument, mappings []SnapshotFieldMapping, fieldList []permissionFieldMeta) (CompiledRecordQuery, error) {
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
	if document.Filter == nil {
		return CompiledRecordQuery{Where: "TRUE"}, nil
	}
	return compileRecordExpression(*document.Filter, fields, 0)
}

func compileRecordExpression(expression model.RecordQueryExpression, fields map[string]recordQueryField, depth int) (CompiledRecordQuery, error) {
	if depth > 12 {
		return CompiledRecordQuery{}, fmt.Errorf("query filter nesting exceeds 12 levels")
	}
	switch expression.Type {
	case "condition":
		trimmed := strings.TrimSpace(expression.Field)
		// 系统字段（sys.*）走物理列编译，与 widgetName 白名单互斥
		if IsRecordSystemField(trimmed) {
			return compileSystemRecordCondition(trimmed, expression.Operator, expression.Value)
		}
		field, ok := fields[trimmed]
		if !ok {
			return CompiledRecordQuery{}, fmt.Errorf("query field %q is not present in published field mappings", expression.Field)
		}
		return compileUserCondition(field, expression.Operator, expression.Value)
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
			part, err := compileRecordExpression(child, fields, depth+1)
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
// fields. The user can provide only the bound keyword, never a JSONB key/path.
func CompileRecordKeyword(keyword string, mappings []SnapshotFieldMapping, fieldList []permissionFieldMeta) (CompiledRecordQuery, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return CompiledRecordQuery{Where: "TRUE"}, nil
	}
	fields, err := queryFields(mappings, fieldList)
	if err != nil {
		return CompiledRecordQuery{}, err
	}
	parts := make([]CompiledRecordQuery, 0, len(fields))
	for _, field := range fields {
		if field.class == permFieldClassMultiOption {
			continue
		}
		valueSQL, valueArgs := normalizedValueSQL(field)
		parts = append(parts, CompiledRecordQuery{Where: "(" + valueSQL + ") IS NOT NULL AND (" + valueSQL + ") ILIKE ? ESCAPE '\\\\'", Args: append(append([]any{}, valueArgs...), append(valueArgs, "%"+escapeLike(keyword)+"%")...)})
	}
	if len(parts) == 0 {
		return CompiledRecordQuery{Where: "FALSE"}, nil
	}
	return joinCompiled(parts, " OR "), nil
}
