package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	storagepkg "evolyn/internal/engine/data/storage"
	"evolyn/internal/platform/form/model"
)

// RecordQueryCondition 是 Query DSL 的单个条件。字段只能是发布快照中冻结的
// widgetName；JSONPath、物理列名和 SQL 文本不属于该协议。
type RecordQueryCondition struct {
	Field    string          `json:"field"`
	Operator string          `json:"operator"`
	Value    json.RawMessage `json:"value"`
}

// CompiledRecordQuery 是仓储层可追加到固定 tenant_id/form_id 条件的安全片段。
// Where 始终由本包生成，Args 中只有数据库绑定参数。
type CompiledRecordQuery struct {
	Where string
	Args  []any
}

// RecordQueryCompileOptions 查询编译模式（方案 §11）：legacy（JSONB 记录）
// 与 physical（物理值表 JOIN）共用同一份操作符矩阵与模板，仅值表达式解析
// 不同——physical 列名来自已应用存储模型（服务端唯一事实源），绝不经
// field_mappings 的意图字段直达 SQL。
type RecordQueryCompileOptions struct {
	// Physical 为 true 时用户字段编译为物理列引用（d.<column>）。
	Physical bool
	// PhysicalColumns widgetName → 物理列名（来自已应用 StorageModel.Columns）。
	PhysicalColumns map[string]string
	// PhysicalArrayColumns 标识采用 PostgreSQL BIGINT[] 的多部门列。它们不能
	// 复用 legacy JSONB 数组函数，须按数组成员关系编译为 ANY/cardinality。
	PhysicalArrayColumns map[string]bool
}

// compileOptionsOf 归一可选编译参数（缺省 = legacy JSONB 模式）。
func compileOptionsOf(opts ...RecordQueryCompileOptions) RecordQueryCompileOptions {
	if len(opts) == 0 {
		return RecordQueryCompileOptions{}
	}
	return opts[0]
}

// physicalAlias 物理值表 JOIN 别名。
const physicalAlias = "d."

// recordEnvelopeAlias 物理模式下记录信封表别名。
const recordEnvelopeAlias = "r."

// valueCompiler 用户字段值表达式工厂（物理表存储方案 §11：逻辑字段 →
// 物理表达式由存储解析器决定）。
type valueCompiler func(field recordQueryField) (string, []any)

// compiler 按模式返回值表达式工厂。
func (o RecordQueryCompileOptions) compiler() valueCompiler {
	if !o.Physical {
		return normalizedValueSQL
	}
	return func(field recordQueryField) (string, []any) {
		return physicalAlias + o.PhysicalColumns[field.mapping.WidgetName], nil
	}
}

// validatePhysicalColumns 编译入口预检：所有可过滤字段的物理列必须存在且
// 过白名单（模型与冻结映射不一致时以查询非法拒绝，绝不带病生成 SQL）。
func (o RecordQueryCompileOptions) validatePhysicalColumns(fields map[string]recordQueryField) error {
	if !o.Physical {
		return nil
	}
	for name := range fields {
		column := o.PhysicalColumns[name]
		if column == "" {
			return fmt.Errorf("physical column for %q is not present in applied storage model", name)
		}
		if err := storagepkg.ValidateColumnName(column); err != nil {
			return fmt.Errorf("physical column for %q: %w", name, err)
		}
	}
	return nil
}

// systemFieldPrefix 系统字段表前缀（按字段分派，物理表存储方案 §5.3/§11）：
// legacy 单表无前缀；physical 模式下流程三字段（单号/状态/更新时间）在物理
// 表预置同名列与复合索引（高频筛选由物理侧承载），编译为 d. 前缀以命中
// 索引，其余系统字段（提交人/提交时间/更新时间为信封物理属性，物理表无
// 同名列）恒挂信封表 r.。
func (o RecordQueryCompileOptions) systemFieldPrefix(field string) string {
	if !o.Physical {
		return ""
	}
	switch field {
	case SysFieldWorkflowInstanceNo, SysFieldWorkflowStatus, SysFieldWorkflowUpdatedAt:
		return physicalAlias
	default:
		return recordEnvelopeAlias
	}
}

func CompileRecordQueryCondition(mappings []SnapshotFieldMapping, condition RecordQueryCondition) (CompiledRecordQuery, error) {
	field, ok := fieldFromMappings(mappings, strings.TrimSpace(condition.Field))
	if !ok {
		return CompiledRecordQuery{}, fmt.Errorf("query field %q is not present in published field mappings", condition.Field)
	}
	var value any
	if err := json.Unmarshal(condition.Value, &value); err != nil {
		return CompiledRecordQuery{}, fmt.Errorf("query value for %q: %w", condition.Field, err)
	}
	return compileUserCondition(field, condition.Operator, value, normalizedValueSQL, false)
}

// CompilePermissionScopeSQL 将已通过配置期校验的数据范围编译为与
// permissionScopeMatches 等价的 PostgreSQL JSONB 谓词。fieldList 保留 datetime
// format 等快照信息，但字段白名单的唯一事实源仍是 mappings。
func CompilePermissionScopeSQL(scope model.PermissionDataScopeSpec, mappings []SnapshotFieldMapping, fieldList []permissionFieldMeta, opts ...RecordQueryCompileOptions) (CompiledRecordQuery, error) {
	options := compileOptionsOf(opts...)
	scope.Normalize()
	if len(scope.Conditions) == 0 {
		return CompiledRecordQuery{Where: "TRUE"}, nil
	}
	fields, err := queryFields(mappings, fieldList)
	if err != nil {
		return CompiledRecordQuery{}, err
	}
	if err := options.validatePhysicalColumns(fields); err != nil {
		return CompiledRecordQuery{}, err
	}
	parts := make([]CompiledRecordQuery, 0, len(scope.Conditions))
	for _, condition := range scope.Conditions {
		field, ok := fields[condition.Field]
		if !ok {
			return CompiledRecordQuery{}, fmt.Errorf("permission scope field %q is not present in published field mappings", condition.Field)
		}
		part, err := compileScopeCondition(field, condition, options.compiler(), options.PhysicalArrayColumns[field.mapping.WidgetName])
		if err != nil {
			return CompiledRecordQuery{}, err
		}
		parts = append(parts, part)
	}
	joiner := " AND "
	if scope.Match == model.PermissionScopeMatchAny {
		joiner = " OR "
	}
	return joinCompiled(parts, joiner), nil
}

type recordQueryField struct {
	mapping SnapshotFieldMapping
	meta    permissionFieldMeta
	class   permissionFieldClass
}

func queryFields(mappings []SnapshotFieldMapping, fieldList []permissionFieldMeta) (map[string]recordQueryField, error) {
	metaByKey := permissionFieldIndex(fieldList)
	fields := make(map[string]recordQueryField, len(mappings))
	for _, mapping := range mappings {
		if mapping.WidgetName == "" || mapping.JSONBKey == "" || mapping.WidgetName != mapping.JSONBKey {
			return nil, fmt.Errorf("invalid frozen field mapping for %q", mapping.WidgetName)
		}
		meta, ok := metaByKey[mapping.WidgetName]
		if !ok || meta.WidgetType != mapping.WidgetType {
			return nil, fmt.Errorf("frozen field mapping %q does not match published schema", mapping.WidgetName)
		}
		class := permissionClassOfWidget(mapping.WidgetType)
		if class == "" {
			continue
		}
		if _, duplicate := fields[mapping.WidgetName]; duplicate {
			return nil, fmt.Errorf("duplicate frozen field mapping %q", mapping.WidgetName)
		}
		fields[mapping.WidgetName] = recordQueryField{mapping: mapping, meta: meta, class: class}
	}
	return fields, nil
}

func fieldFromMappings(mappings []SnapshotFieldMapping, name string) (recordQueryField, bool) {
	for _, mapping := range mappings {
		if mapping.WidgetName == name && mapping.WidgetName == mapping.JSONBKey && mapping.WidgetName != "" {
			class := permissionClassOfWidget(mapping.WidgetType)
			// 000065 前的测试夹具可能缺少 widgetType；真实冻结映射始终携带
			// 类型。空类型只回落为文本，未知真实控件仍拒绝过滤。
			if class == "" && mapping.WidgetType == "" {
				class = permFieldClassText
			}
			return recordQueryField{mapping: mapping, meta: permissionFieldMeta{Key: name, WidgetType: mapping.WidgetType}, class: class}, true
		}
	}
	return recordQueryField{}, false
}

func compileScopeCondition(field recordQueryField, condition model.PermissionDataCondition, vc valueCompiler, physicalArray bool) (CompiledRecordQuery, error) {
	if !permissionClassOperators[field.class][condition.Operator] {
		return CompiledRecordQuery{}, fmt.Errorf("permission scope operator %q is not applicable to %q", condition.Operator, field.mapping.WidgetName)
	}
	var value any
	switch condition.Operator {
	case "empty", "not_empty":
	case "in", "not_in":
		value = condition.Value
	default:
		if len(condition.Value) != 1 {
			return CompiledRecordQuery{}, fmt.Errorf("permission scope condition %q requires one value", condition.Operator)
		}
		value = condition.Value[0]
	}
	return compileCondition(field, condition.Operator, value, vc, physicalArray)
}

func compileUserCondition(field recordQueryField, operator string, value any, vc valueCompiler, physicalArray bool) (CompiledRecordQuery, error) {
	if field.class == "" {
		return CompiledRecordQuery{}, fmt.Errorf("query field %q does not support filtering", field.mapping.WidgetName)
	}
	return compileCondition(field, operator, value, vc, physicalArray)
}

// compileCondition is the sole SQL-template factory. No user-controlled string can
// reach Where: field keys and values are always bound through Args.
func compileCondition(field recordQueryField, operator string, value any, vc valueCompiler, physicalArray bool) (CompiledRecordQuery, error) {
	vSQL, vArgs := vc(field)
	copyArgs := func(times int) []any {
		args := make([]any, 0, len(vArgs)*times)
		for range times {
			args = append(args, vArgs...)
		}
		return args
	}
	where2 := func(template string, args []any) CompiledRecordQuery {
		return CompiledRecordQuery{Where: fmt.Sprintf(template, vSQL, vSQL), Args: args}
	}
	textValue := func() (string, error) {
		text, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("query value for %q must be a string", field.mapping.WidgetName)
		}
		return text, nil
	}
	numberValue := func() (float64, error) {
		number, ok := jsonFloat(value)
		if !ok {
			return 0, fmt.Errorf("query value for %q must be a number", field.mapping.WidgetName)
		}
		return number, nil
	}
	setValues := func() ([]any, error) {
		values, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("query value for %q must be an array", field.mapping.WidgetName)
		}
		out := make([]any, 0, len(values))
		for _, raw := range values {
			if field.class == permFieldClassNumber {
				number, ok := jsonFloat(raw)
				if !ok {
					return nil, fmt.Errorf("query set value for %q must be a number", field.mapping.WidgetName)
				}
				out = append(out, number)
				continue
			}
			text, ok := raw.(string)
			if !ok {
				return nil, fmt.Errorf("query set value for %q must be a string", field.mapping.WidgetName)
			}
			out = append(out, text)
		}
		return out, nil
	}
	isArray := field.class == permFieldClassMultiOption
	departmentID := func(raw any) (int64, error) {
		text, ok := raw.(string)
		if !ok {
			return 0, fmt.Errorf("query value for %q must be a department ID string", field.mapping.WidgetName)
		}
		id, err := strconv.ParseInt(text, 10, 64)
		if err != nil || id <= 0 {
			return 0, fmt.Errorf("query value for %q must be a positive department ID", field.mapping.WidgetName)
		}
		return id, nil
	}
	physicalArrayText := func() (int64, error) { return departmentID(value) }
	physicalArraySet := func() ([]int64, error) {
		values, err := setValues()
		if err != nil {
			return nil, err
		}
		ids := make([]int64, 0, len(values))
		for _, raw := range values {
			id, err := departmentID(raw)
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
		return ids, nil
	}

	switch operator {
	case "eq":
		// Query DSL 的 enum=多选控件时，eq 表示“包含该选项”；权限范围
		// 配置并不开放 multiOption 的 eq，因此不会改变 permissionScopeMatches。
		if physicalArray {
			id, err := physicalArrayText()
			if err != nil {
				return CompiledRecordQuery{}, err
			}
			return CompiledRecordQuery{Where: "(" + vSQL + ") IS NOT NULL AND ? = ANY(" + vSQL + ")", Args: append(copyArgs(2), id)}, nil
		}
		if isArray {
			text, err := textValue()
			if err != nil {
				return CompiledRecordQuery{}, err
			}
			// jsonb `?` 操作符与驱动占位符冲突，经 jsonb_exists 函数形式表达
			return where2("(%s) IS NOT NULL AND jsonb_exists(%s, ?)", append(copyArgs(2), text)), nil
		}
		if field.class == permFieldClassNumber {
			n, err := numberValue()
			if err != nil {
				return CompiledRecordQuery{}, err
			}
			return where2("(%s) IS NOT NULL AND (%s)::numeric = ?", append(copyArgs(2), n)), nil
		}
		text, err := textValue()
		if err != nil {
			return CompiledRecordQuery{}, err
		}
		return where2("(%s) IS NOT NULL AND (%s) = ?", append(copyArgs(2), text)), nil
	case "ne", "neq":
		if physicalArray {
			id, err := physicalArrayText()
			if err != nil {
				return CompiledRecordQuery{}, err
			}
			return CompiledRecordQuery{Where: "(" + vSQL + ") IS NOT NULL AND NOT (? = ANY(" + vSQL + "))", Args: append(copyArgs(2), id)}, nil
		}
		if isArray {
			text, err := textValue()
			if err != nil {
				return CompiledRecordQuery{}, err
			}
			return where2("(%s) IS NOT NULL AND NOT jsonb_exists(%s, ?)", append(copyArgs(2), text)), nil
		}
		if field.class == permFieldClassNumber {
			n, err := numberValue()
			if err != nil {
				return CompiledRecordQuery{}, err
			}
			return where2("(%s) IS NOT NULL AND (%s)::numeric <> ?", append(copyArgs(2), n)), nil
		}
		text, err := textValue()
		if err != nil {
			return CompiledRecordQuery{}, err
		}
		return where2("(%s) IS NOT NULL AND (%s) <> ?", append(copyArgs(2), text)), nil
	case "gt", "gte", "lt", "lte":
		if field.class != permFieldClassNumber && field.class != permFieldClassDateTime {
			return CompiledRecordQuery{}, fmt.Errorf("query operator %q is not applicable to %q", operator, field.mapping.WidgetName)
		}
		comparison := map[string]string{"gt": ">", "gte": ">=", "lt": "<", "lte": "<="}[operator]
		var rhs any
		var err error
		if field.class == permFieldClassNumber {
			rhs, err = numberValue()
		} else {
			rhs, err = textValue()
		}
		if err != nil {
			return CompiledRecordQuery{}, err
		}
		if field.class == permFieldClassNumber {
			return where2("(%s) IS NOT NULL AND (%s)::numeric "+comparison+" ?", append(copyArgs(2), rhs)), nil
		}
		return where2("(%s) IS NOT NULL AND (%s) "+comparison+" ?", append(copyArgs(2), rhs)), nil
	case "contains":
		if physicalArray {
			id, err := physicalArrayText()
			if err != nil {
				return CompiledRecordQuery{}, err
			}
			return CompiledRecordQuery{Where: "(" + vSQL + ") IS NOT NULL AND ? = ANY(" + vSQL + ")", Args: append(copyArgs(2), id)}, nil
		}
		text, err := textValue()
		if err != nil {
			return CompiledRecordQuery{}, err
		}
		if isArray {
			return where2("(%s) IS NOT NULL AND jsonb_exists(%s, ?)", append(copyArgs(2), text)), nil
		}
		return where2("(%s) IS NOT NULL AND position(? in (%s)) > 0", append(copyArgs(2), text)), nil
	case "notContains":
		if physicalArray {
			id, err := physicalArrayText()
			if err != nil {
				return CompiledRecordQuery{}, err
			}
			return CompiledRecordQuery{Where: "(" + vSQL + ") IS NULL OR NOT (? = ANY(" + vSQL + "))", Args: append(copyArgs(2), id)}, nil
		}
		text, err := textValue()
		if err != nil {
			return CompiledRecordQuery{}, err
		}
		if isArray {
			return where2("(%s) IS NULL OR NOT jsonb_exists(%s, ?)", append(copyArgs(2), text)), nil
		}
		return where2("(%s) IS NULL OR position(? in (%s)) = 0", append(copyArgs(2), text)), nil
	case "startsWith", "endsWith":
		if isArray {
			return CompiledRecordQuery{}, fmt.Errorf("query operator %q is not applicable to %q", operator, field.mapping.WidgetName)
		}
		text, err := textValue()
		if err != nil {
			return CompiledRecordQuery{}, err
		}
		pattern := escapeLike(text)
		if operator == "startsWith" {
			pattern += "%"
		} else {
			pattern = "%" + pattern
		}
		return where2("(%s) IS NOT NULL AND (%s) LIKE ? ESCAPE '\\\\'", append(copyArgs(2), pattern)), nil
	case "in", "not_in", "notIn":
		if physicalArray {
			ids, err := physicalArraySet()
			if err != nil {
				return CompiledRecordQuery{}, err
			}
			if len(ids) == 0 {
				if operator == "not_in" || operator == "notIn" {
					return CompiledRecordQuery{Where: "TRUE"}, nil
				}
				return CompiledRecordQuery{Where: "FALSE"}, nil
			}
			atoms := make([]string, 0, len(ids))
			args := make([]any, 0, len(ids)*2+1)
			for _, id := range ids {
				atoms = append(atoms, "? = ANY("+vSQL+")")
				args = append(args, id)
				args = append(args, vArgs...)
			}
			membership := strings.Join(atoms, " OR ")
			if operator == "not_in" || operator == "notIn" {
				return CompiledRecordQuery{Where: "(" + vSQL + ") IS NULL OR NOT (" + membership + ")", Args: append(append([]any{}, vArgs...), args...)}, nil
			}
			return CompiledRecordQuery{Where: "(" + vSQL + ") IS NOT NULL AND (" + membership + ")", Args: append(append([]any{}, vArgs...), args...)}, nil
		}
		values, err := setValues()
		if err != nil {
			return CompiledRecordQuery{}, err
		}
		if len(values) == 0 {
			if operator == "not_in" || operator == "notIn" {
				return CompiledRecordQuery{Where: "TRUE"}, nil
			}
			return CompiledRecordQuery{Where: "FALSE"}, nil
		}
		atoms := make([]string, 0, len(values))
		args := append([]any{}, vArgs...)
		for _, candidate := range values {
			if isArray {
				atoms = append(atoms, "jsonb_exists("+vSQL+", ?)")
			} else if field.class == permFieldClassNumber {
				atoms = append(atoms, "("+vSQL+")::numeric = ?")
			} else {
				atoms = append(atoms, "("+vSQL+") = ?")
			}
			args = append(args, vArgs...)
			args = append(args, candidate)
		}
		membership := strings.Join(atoms, " OR ")
		if operator == "not_in" || operator == "notIn" {
			return CompiledRecordQuery{Where: "(" + vSQL + ") IS NULL OR NOT (" + membership + ")", Args: args}, nil
		}
		return CompiledRecordQuery{Where: "(" + vSQL + ") IS NOT NULL AND (" + membership + ")", Args: args}, nil
	case "empty":
		if physicalArray {
			return CompiledRecordQuery{Where: "(" + vSQL + ") IS NULL OR cardinality(" + vSQL + ") = 0", Args: copyArgs(2)}, nil
		}
		if isArray {
			return where2("(%s) IS NULL OR (%s) = '[]'::jsonb", copyArgs(2)), nil
		}
		return where2("(%s) IS NULL OR (%s) = ''", copyArgs(2)), nil
	case "not_empty":
		if physicalArray {
			return CompiledRecordQuery{Where: "(" + vSQL + ") IS NOT NULL AND cardinality(" + vSQL + ") > 0", Args: copyArgs(2)}, nil
		}
		if isArray {
			return where2("(%s) IS NOT NULL AND (%s) <> '[]'::jsonb", copyArgs(2)), nil
		}
		return where2("(%s) IS NOT NULL AND (%s) <> ''", copyArgs(2)), nil
	case "isNull":
		return CompiledRecordQuery{Where: "(" + vSQL + ") IS NULL", Args: copyArgs(1)}, nil
	case "isNotNull":
		return CompiledRecordQuery{Where: "(" + vSQL + ") IS NOT NULL", Args: copyArgs(1)}, nil
	case "between":
		values, ok := value.([]any)
		if !ok || len(values) != 2 {
			return CompiledRecordQuery{}, fmt.Errorf("query range for %q must contain two values", field.mapping.WidgetName)
		}
		var lo, hi any
		var err error
		if field.class == permFieldClassNumber {
			var ok bool
			lo, ok = jsonFloat(values[0])
			if !ok {
				err = fmt.Errorf("query range for %q must contain numbers", field.mapping.WidgetName)
			} else {
				hi, ok = jsonFloat(values[1])
				if !ok {
					err = fmt.Errorf("query range for %q must contain numbers", field.mapping.WidgetName)
				}
			}
		} else if field.class == permFieldClassDateTime {
			lo, _ = values[0].(string)
			hi, _ = values[1].(string)
			if lo == "" || hi == "" {
				err = fmt.Errorf("query range for %q must contain strings", field.mapping.WidgetName)
			}
		} else {
			err = fmt.Errorf("query operator between is not applicable to %q", field.mapping.WidgetName)
		}
		if err != nil {
			return CompiledRecordQuery{}, err
		}
		if field.class == permFieldClassNumber {
			return where2("(%s) IS NOT NULL AND (%s)::numeric BETWEEN ? AND ?", append(copyArgs(2), lo, hi)), nil
		}
		return where2("(%s) IS NOT NULL AND (%s) BETWEEN ? AND ?", append(copyArgs(2), lo, hi)), nil
	default:
		return CompiledRecordQuery{}, fmt.Errorf("query operator %q is not supported for field %q", operator, field.mapping.WidgetName)
	}
}

// normalizedValueSQL mirrors normalizeConditionValue. Malformed JSON becomes SQL
// NULL before any cast/comparison, so corrupt historical records cannot leak.
func normalizedValueSQL(field recordQueryField) (string, []any) {
	key := field.mapping.JSONBKey
	switch field.class {
	case permFieldClassMultiOption:
		return "CASE WHEN jsonb_typeof(values -> ?) = 'array' AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements(values -> ?) AS item WHERE jsonb_typeof(item) <> 'string') THEN values -> ? END", []any{key, key, key}
	case permFieldClassNumber:
		return "CASE WHEN jsonb_typeof(values -> ?) = 'number' THEN values ->> ? WHEN jsonb_typeof(values -> ?) = 'string' AND values ->> ? ~ ? THEN values ->> ? END", []any{key, key, key, key, numericGuardPattern.String(), key}
	case permFieldClassDateTime:
		format := field.meta.Format
		if format == "" {
			format = "datetime"
		}
		pattern := map[string]string{"date": "^[0-9]{4}-[0-9]{2}-[0-9]{2}$", "datetime": "^[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}$", "month": "^[0-9]{4}-[0-9]{2}$", "time": "^[0-9]{2}:[0-9]{2}$"}[format]
		return "CASE WHEN jsonb_typeof(values -> ?) = 'string' AND values ->> ? ~ ? THEN values ->> ? END", []any{key, key, pattern, key}
	default:
		return "CASE WHEN jsonb_typeof(values -> ?) = 'string' THEN values ->> ? WHEN jsonb_typeof(values -> ?) = 'number' THEN values ->> ? END", []any{key, key, key, key}
	}
}

func joinCompiled(parts []CompiledRecordQuery, joiner string) CompiledRecordQuery {
	where := make([]string, 0, len(parts))
	args := make([]any, 0)
	for _, part := range parts {
		where = append(where, "("+part.Where+")")
		args = append(args, part.Args...)
	}
	return CompiledRecordQuery{Where: strings.Join(where, joiner), Args: args}
}

func escapeLike(value string) string {
	return strings.NewReplacer(`\\`, `\\\\`, `%`, `\\%`, `_`, `\\_`).Replace(value)
}
