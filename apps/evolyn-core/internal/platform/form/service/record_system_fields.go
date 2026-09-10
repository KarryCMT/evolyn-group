package service

import (
	"fmt"
	"regexp"
	"strings"

	"evolyn/internal/platform/form/model"
)

// 记录系统字段（000067）：提交人/提交时间/更新时间是记录行的物理属性，
// 不属于任何发布快照字段矩阵（键=widgetName），因此走物理列直映射、不经
// values JSONB。Query DSL 的 field 命中以下命名空间键时进入系统列编译，
// 与 widgetName 白名单互斥；`sys.` 前缀保证永不与 `_widget_` 键冲突。
const (
	SysFieldSubmittedBy = "sys.submittedBy"
	SysFieldSubmittedAt = "sys.submittedAt"
	SysFieldUpdatedAt   = "sys.updatedAt"
	// SysFieldUpdatedBy 最后写人人（000072）：提交时=提交人，审批编辑/发起人
	// 修改写回后=操作人快照；信封物理属性（physical 表无同名列，恒挂 r.）。
	SysFieldUpdatedBy = "sys.updatedBy"
	// 流程查询投影系统字段（物理表存储方案 §10.1）：三者同存于记录信封
	//（tn_form_records）；physical 表额外预置同名列作不依赖 JSONB 解包的
	// 高频筛选/导出投影。状态直接复制流程实例状态枚举，普通表单为 NONE。
	SysFieldWorkflowInstanceNo = "sys.workflowInstanceNo"
	SysFieldWorkflowStatus     = "sys.workflowStatus"
	SysFieldWorkflowUpdatedAt  = "sys.workflowUpdatedAt"
)

// systemFieldColumns 系统字段 → 物理列映射。列名是服务端固定枚举，不是
// 用户输入；与 record_system_fields_test.go 的白名单用例共同冻结。
var systemFieldColumns = map[string]string{
	SysFieldSubmittedBy:        "submitted_by_member_id",
	SysFieldSubmittedAt:        "submitted_at",
	SysFieldUpdatedAt:          "updated_at",
	SysFieldUpdatedBy:          "updated_by_member_id",
	SysFieldWorkflowInstanceNo: "workflow_instance_no",
	SysFieldWorkflowStatus:     "workflow_status",
	SysFieldWorkflowUpdatedAt:  "workflow_updated_at",
}

// workflowStatusValuePattern 流程状态值白名单：与引擎实例状态枚举
// （DRAFT/RUNNING/COMPLETED/REJECTED/CANCELLED）及普通表单 NONE 冻结一致。
var workflowStatusValuePattern = regexp.MustCompile(`^(NONE|DRAFT|RUNNING|COMPLETED|REJECTED|CANCELLED)$`)

// systemFieldOperators 系统字段允许的操作符，与前端 @evolyn.do/query 的
// 字段类型操作符字典镜像：提交人=enum、时间=datetime。
var systemFieldOperators = map[string]map[string]bool{
	SysFieldSubmittedBy: {"eq": true, "neq": true, "in": true, "notIn": true, "isNull": true, "isNotNull": true},
	SysFieldSubmittedAt: {"eq": true, "neq": true, "gt": true, "gte": true, "lt": true, "lte": true, "between": true, "isNull": true, "isNotNull": true},
	SysFieldUpdatedAt:   {"eq": true, "neq": true, "gt": true, "gte": true, "lt": true, "lte": true, "between": true, "isNull": true, "isNotNull": true},
	// 最后写人人=enum（与提交人同口径：系统路径可能无写人，isNull 有意义）
	SysFieldUpdatedBy: {"eq": true, "neq": true, "in": true, "notIn": true, "isNull": true, "isNotNull": true},
	// 单号=enum 文本；状态=enum（低基数，索引挂复合 (tenant,status,updated)，
	// 单列不建索引）；更新时间=datetime。
	SysFieldWorkflowInstanceNo: {"eq": true, "neq": true, "in": true, "notIn": true, "isNull": true, "isNotNull": true, "contains": true, "notContains": true, "startsWith": true},
	SysFieldWorkflowStatus:     {"eq": true, "neq": true, "in": true, "notIn": true},
	SysFieldWorkflowUpdatedAt:  {"eq": true, "neq": true, "gt": true, "gte": true, "lt": true, "lte": true, "between": true, "isNull": true, "isNotNull": true},
}

// systemTimestampPattern 系统时间字段的值格式：秒级 datetime（与 JSONTime
// 出网「2006-01-02 15:04:05」一致）或 date-only（Postgres 解释为当日零点）。
// 服务端先行校验，避免非法字符串落进 timestamptz 比较引发数据库错误。
var systemTimestampPattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}([ T][0-9]{2}:[0-9]{2}(:[0-9]{2})?)?$`)

// IsRecordSystemField 报告 field 是否为系统字段命名空间键。
func IsRecordSystemField(field string) bool {
	_, ok := systemFieldColumns[strings.TrimSpace(field)]
	return ok
}

// compileSystemRecordCondition 将系统字段条件编译为参数化谓词。operator
// 与值形态严格按 systemFieldOperators 校验；与 widget 条件一样，用户输入
// 只能进入 Args 绑定参数。value 已由控制器 BindJSON 解码为 any。
func compileSystemRecordCondition(field, operator string, value any, prefix string) (CompiledRecordQuery, error) {
	trimmedField := strings.TrimSpace(field)
	column := systemFieldColumns[trimmedField]
	allowed := systemFieldOperators[trimmedField]
	if column == "" || allowed == nil {
		return CompiledRecordQuery{}, fmt.Errorf("query field %q is not a known system field", field)
	}
	operator = strings.TrimSpace(operator)
	if !allowed[operator] {
		return CompiledRecordQuery{}, fmt.Errorf("query operator %q is not applicable to system field %q", operator, field)
	}
	// prefix 为 physical 模式的表别名（按字段分派：流程三字段挂物理表 d，
	// 其余挂信封表 r；legacy 为空串），列名仍是服务端固定枚举。
	switch trimmedField {
	case SysFieldSubmittedBy:
		return compileSystemMemberCondition(prefix+column, operator, value, false)
	case SysFieldUpdatedBy:
		// updated_by 可空（系统自动写回路径无操作人）：isNull/isNotNull 是
		// 真列检查，不是 submittedBy 那样的 NOT NULL 恒真恒假常量
		return compileSystemMemberCondition(prefix+column, operator, value, true)
	case SysFieldWorkflowInstanceNo:
		return compileSystemInstanceNoCondition(prefix+column, operator, value)
	case SysFieldWorkflowStatus:
		return compileSystemEnumCondition(prefix+column, operator, value)
	case SysFieldWorkflowUpdatedAt:
		return compileSystemTimeCondition(prefix+column, operator, value)
	default:
		return compileSystemTimeCondition(prefix+column, operator, value)
	}
}

// compileSystemInstanceNoCondition 流程单号条件：文本等值/集合/前缀匹配
// （isNull 语义=普通表单无流程，与空串同义）。
func compileSystemInstanceNoCondition(column, operator string, value any) (CompiledRecordQuery, error) {
	text := func(raw any) (string, error) {
		text, ok := raw.(string)
		if !ok || strings.TrimSpace(text) == "" {
			return "", fmt.Errorf("query value for %q must be a non-empty string", SysFieldWorkflowInstanceNo)
		}
		return text, nil
	}
	switch operator {
	case "isNull":
		return CompiledRecordQuery{Where: column + " = ''"}, nil
	case "isNotNull":
		return CompiledRecordQuery{Where: column + " <> ''"}, nil
	case "eq", "neq":
		v, err := text(value)
		if err != nil {
			return CompiledRecordQuery{}, err
		}
		comparison := "="
		if operator == "neq" {
			comparison = "<>"
		}
		return CompiledRecordQuery{Where: column + " " + comparison + " ?", Args: []any{v}}, nil
	case "in", "notIn":
		values, ok := value.([]any)
		if !ok {
			return CompiledRecordQuery{}, fmt.Errorf("query value for %q must be an array", SysFieldWorkflowInstanceNo)
		}
		items := make([]any, 0, len(values))
		for _, raw := range values {
			v, err := text(raw)
			if err != nil {
				return CompiledRecordQuery{}, err
			}
			items = append(items, v)
		}
		if len(items) == 0 {
			if operator == "notIn" {
				return CompiledRecordQuery{Where: "TRUE"}, nil
			}
			return CompiledRecordQuery{Where: "FALSE"}, nil
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(items)), ", ")
		if operator == "notIn" {
			return CompiledRecordQuery{Where: column + " NOT IN (" + placeholders + ")", Args: items}, nil
		}
		return CompiledRecordQuery{Where: column + " IN (" + placeholders + ")", Args: items}, nil
	case "contains", "notContains", "startsWith":
		v, err := text(value)
		if err != nil {
			return CompiledRecordQuery{}, err
		}
		pattern := "%" + escapeLike(v) + "%"
		if operator == "startsWith" {
			pattern = escapeLike(v) + "%"
		}
		predicate := column + " LIKE ? ESCAPE '\\'"
		if operator == "notContains" {
			predicate = "(" + column + " = '' OR " + column + " NOT LIKE ? ESCAPE '\\')"
		}
		return CompiledRecordQuery{Where: predicate, Args: []any{pattern}}, nil
	default:
		return CompiledRecordQuery{}, fmt.Errorf("query operator %q is not applicable to system field %q", operator, SysFieldWorkflowInstanceNo)
	}
}

// compileSystemEnumCondition 流程状态条件：值白名单冻结为实例状态枚举+NONE，
// 仅 eq/neq/in/notIn（isNull/isNotNull 无意义——列 NOT NULL DEFAULT NONE）。
func compileSystemEnumCondition(column, operator string, value any) (CompiledRecordQuery, error) {
	status := func(raw any) (string, error) {
		text, ok := raw.(string)
		if !ok || !workflowStatusValuePattern.MatchString(text) {
			return "", fmt.Errorf("query value for %q must be one of NONE/DRAFT/RUNNING/COMPLETED/REJECTED/CANCELLED", SysFieldWorkflowStatus)
		}
		return text, nil
	}
	switch operator {
	case "eq", "neq":
		v, err := status(value)
		if err != nil {
			return CompiledRecordQuery{}, err
		}
		comparison := "="
		if operator == "neq" {
			comparison = "<>"
		}
		return CompiledRecordQuery{Where: column + " " + comparison + " ?", Args: []any{v}}, nil
	case "in", "notIn":
		values, ok := value.([]any)
		if !ok {
			return CompiledRecordQuery{}, fmt.Errorf("query value for %q must be an array", SysFieldWorkflowStatus)
		}
		items := make([]any, 0, len(values))
		for _, raw := range values {
			v, err := status(raw)
			if err != nil {
				return CompiledRecordQuery{}, err
			}
			items = append(items, v)
		}
		if len(items) == 0 {
			if operator == "notIn" {
				return CompiledRecordQuery{Where: "TRUE"}, nil
			}
			return CompiledRecordQuery{Where: "FALSE"}, nil
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(items)), ", ")
		if operator == "notIn" {
			return CompiledRecordQuery{Where: column + " NOT IN (" + placeholders + ")", Args: items}, nil
		}
		return CompiledRecordQuery{Where: column + " IN (" + placeholders + ")", Args: items}, nil
	default:
		return CompiledRecordQuery{}, fmt.Errorf("query operator %q is not applicable to system field %q", operator, SysFieldWorkflowStatus)
	}
}

// compileSystemMemberCondition 成员（成员 ID）条件：eq/neq/in/notIn 绑定
// 数字；nullable=false（submittedBy，列 NOT NULL）时 isNull/isNotNull 编译为
// 常量语义（恒假/恒真），保持 enum 操作符集完整而不是在协议层挖特例；
// nullable=true（updatedBy，000072 列可空——系统自动写回无操作人）时
// isNull/isNotNull 是真实列检查。
func compileSystemMemberCondition(column, operator string, value any, nullable bool) (CompiledRecordQuery, error) {
	memberID := func(raw any) (int64, error) {
		number, ok := jsonFloat(raw)
		if !ok || number != float64(int64(number)) || number <= 0 {
			return 0, fmt.Errorf("query value for %q must be a positive member id", SysFieldSubmittedBy)
		}
		return int64(number), nil
	}
	switch operator {
	case "isNull":
		if nullable {
			return CompiledRecordQuery{Where: column + " IS NULL"}, nil
		}
		return CompiledRecordQuery{Where: "FALSE"}, nil
	case "isNotNull":
		if nullable {
			return CompiledRecordQuery{Where: column + " IS NOT NULL"}, nil
		}
		return CompiledRecordQuery{Where: "TRUE"}, nil
	case "eq", "neq":
		id, err := memberID(value)
		if err != nil {
			return CompiledRecordQuery{}, err
		}
		comparison := "="
		if operator == "neq" {
			comparison = "<>"
		}
		return CompiledRecordQuery{Where: column + " " + comparison + " ?", Args: []any{id}}, nil
	case "in", "notIn":
		values, ok := value.([]any)
		if !ok {
			return CompiledRecordQuery{}, fmt.Errorf("query value for %q must be an array", SysFieldSubmittedBy)
		}
		ids := make([]any, 0, len(values))
		for _, raw := range values {
			id, err := memberID(raw)
			if err != nil {
				return CompiledRecordQuery{}, err
			}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			if operator == "notIn" {
				return CompiledRecordQuery{Where: "TRUE"}, nil
			}
			return CompiledRecordQuery{Where: "FALSE"}, nil
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(ids)), ", ")
		if operator == "notIn" {
			return CompiledRecordQuery{Where: column + " NOT IN (" + placeholders + ")", Args: ids}, nil
		}
		return CompiledRecordQuery{Where: column + " IN (" + placeholders + ")", Args: ids}, nil
	default:
		return CompiledRecordQuery{}, fmt.Errorf("query operator %q is not applicable to system field %q", operator, SysFieldSubmittedBy)
	}
}

// compileSystemTimeCondition 时间条件：值先行格式校验（systemTimestampPattern），
// 再以字符串绑定进 timestamptz 比较，由参数类型推断完成转换。
func compileSystemTimeCondition(column, operator string, value any) (CompiledRecordQuery, error) {
	timestamp := func(raw any) (string, error) {
		text, ok := raw.(string)
		if !ok || !systemTimestampPattern.MatchString(text) {
			return "", fmt.Errorf("query value for system time field must be YYYY-MM-DD or YYYY-MM-DD HH:MM:SS")
		}
		return text, nil
	}
	switch operator {
	case "isNull":
		return CompiledRecordQuery{Where: column + " IS NULL"}, nil
	case "isNotNull":
		return CompiledRecordQuery{Where: column + " IS NOT NULL"}, nil
	case "eq", "neq", "gt", "gte", "lt", "lte":
		text, err := timestamp(value)
		if err != nil {
			return CompiledRecordQuery{}, err
		}
		comparison := map[string]string{"eq": "=", "neq": "<>", "gt": ">", "gte": ">=", "lt": "<", "lte": "<="}[operator]
		return CompiledRecordQuery{Where: column + " " + comparison + " ?", Args: []any{text}}, nil
	case "between":
		values, ok := value.([]any)
		if !ok || len(values) != 2 {
			return CompiledRecordQuery{}, fmt.Errorf("query range for system time field must contain two values")
		}
		lo, err := timestamp(values[0])
		if err != nil {
			return CompiledRecordQuery{}, err
		}
		hi, err := timestamp(values[1])
		if err != nil {
			return CompiledRecordQuery{}, err
		}
		return CompiledRecordQuery{Where: column + " BETWEEN ? AND ?", Args: []any{lo, hi}}, nil
	default:
		return CompiledRecordQuery{}, fmt.Errorf("query operator %q is not applicable to system time fields", operator)
	}
}

// CompileRecordListSorts 将 sorts 编译为 ORDER BY 片段。一期只开放系统字段
// 排序（物理列直映射、成本与语义都确定）；表单字段排序依赖 JSONB 路径
// 定序，待列存/表达式索引方案定板后另行开放。方向白名单 asc/desc，
// id DESC 稳定尾排序由仓储层恒定追加。
func CompileRecordListSorts(sorts []model.RecordQuerySort, opts ...RecordQueryCompileOptions) (string, error) {
	options := compileOptionsOf(opts...)
	if len(sorts) == 0 {
		return "", nil
	}
	if len(sorts) > 3 {
		return "", fmt.Errorf("record list supports at most 3 sort fields")
	}
	parts := make([]string, 0, len(sorts))
	for _, sort := range sorts {
		trimmed := strings.TrimSpace(sort.Field)
		column, ok := systemFieldColumns[trimmed]
		if !ok {
			return "", fmt.Errorf("sorting is only supported for system fields (sys.*), not %q", sort.Field)
		}
		// 前缀按字段分派（physical 流程三字段挂物理表 d 命中预置索引，
		// 其余挂信封表 r；legacy 恒为空串）
		prefix := options.systemFieldPrefix(trimmed)
		switch strings.ToLower(strings.TrimSpace(sort.Direction)) {
		case "asc":
			parts = append(parts, prefix+column+" ASC")
		case "desc":
			parts = append(parts, prefix+column+" DESC")
		default:
			return "", fmt.Errorf("sort direction %q is invalid", sort.Direction)
		}
	}
	return strings.Join(parts, ", "), nil
}
