package query

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var aggregateAliasPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,63}$`)

var valueOperators = map[Operator]bool{
	OperatorEQ: true, OperatorNEQ: true, OperatorContains: true, OperatorNotContains: true,
	OperatorStartsWith: true, OperatorEndsWith: true, OperatorGT: true, OperatorGTE: true,
	OperatorLT: true, OperatorLTE: true, OperatorIn: true, OperatorNotIn: true, OperatorBetween: true,
}

var valuelessOperators = map[Operator]bool{OperatorIsNull: true, OperatorIsNotNull: true}

var operatorsByType = map[FieldType]map[Operator]bool{
	FieldText:       setOperators(OperatorEQ, OperatorNEQ, OperatorContains, OperatorNotContains, OperatorStartsWith, OperatorEndsWith, OperatorIn, OperatorNotIn, OperatorIsNull, OperatorIsNotNull),
	FieldDecimal:    setOperators(OperatorEQ, OperatorNEQ, OperatorGT, OperatorGTE, OperatorLT, OperatorLTE, OperatorIn, OperatorNotIn, OperatorBetween, OperatorIsNull, OperatorIsNotNull),
	FieldNumber:     setOperators(OperatorEQ, OperatorNEQ, OperatorGT, OperatorGTE, OperatorLT, OperatorLTE, OperatorIn, OperatorNotIn, OperatorBetween, OperatorIsNull, OperatorIsNotNull),
	FieldBoolean:    setOperators(OperatorEQ, OperatorNEQ, OperatorIsNull, OperatorIsNotNull),
	FieldDate:       setOperators(OperatorEQ, OperatorNEQ, OperatorGT, OperatorGTE, OperatorLT, OperatorLTE, OperatorBetween, OperatorIsNull, OperatorIsNotNull),
	FieldDateTime:   setOperators(OperatorEQ, OperatorNEQ, OperatorGT, OperatorGTE, OperatorLT, OperatorLTE, OperatorBetween, OperatorIsNull, OperatorIsNotNull),
	FieldEnum:       setOperators(OperatorEQ, OperatorNEQ, OperatorIn, OperatorNotIn, OperatorIsNull, OperatorIsNotNull),
	FieldMember:     setOperators(OperatorEQ, OperatorNEQ, OperatorIn, OperatorNotIn, OperatorIsNull, OperatorIsNotNull),
	FieldDepartment: setOperators(OperatorEQ, OperatorNEQ, OperatorIn, OperatorNotIn, OperatorIsNull, OperatorIsNotNull),
}

// NormalizeJSON 严格解析并规范化 Query DSL。root 级未知键会保留精确问题路径，
// 嵌套未知键由 DisallowUnknownFields 拒绝，避免 SQL 或存储私有配置进入事实源。
func NormalizeJSON(raw []byte, fields FieldCatalog, budget Budget) Result {
	issues := make([]Issue, 0)
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil || root == nil {
		return Result{Issues: []Issue{{Path: "$", Code: "QUERY_DOCUMENT_INVALID", Message: "查询文档必须是对象。"}}}
	}
	allowed := map[string]bool{"version": true, "filter": true, "sorts": true, "paging": true, "projection": true, "groupBy": true, "aggregates": true}
	for key := range root {
		if !allowed[key] {
			issues = append(issues, Issue{Path: key, Code: "QUERY_DOCUMENT_INVALID", Message: "查询文档包含未受支持的配置。"})
		}
	}

	var document Document
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		if len(issues) == 0 {
			issues = append(issues, Issue{Path: "$", Code: "QUERY_DOCUMENT_INVALID", Message: "查询文档必须是合法且受支持的 JSON 对象。"})
		}
		return Result{Issues: issues}
	}
	if err := ensureEOF(decoder); err != nil {
		return Result{Issues: append(issues, Issue{Path: "$", Code: "QUERY_DOCUMENT_INVALID", Message: "查询文档只能包含一个 JSON 对象。"})}
	}
	validated := Validate(document, fields, budget)
	validated.Issues = append(issues, validated.Issues...)
	if len(validated.Issues) > 0 {
		validated.Document = nil
		validated.Plan = nil
	}
	return validated
}

// Validate 校验已解码 AST 并在成功时生成规范化文档和存储无关逻辑计划。
func Validate(document Document, fields FieldCatalog, budget Budget) Result {
	budget = normalizeBudget(budget)
	issues := make([]Issue, 0)
	if document.Version != Version {
		addIssue(&issues, "version", "QUERY_INVALID_VERSION", fmt.Sprintf("仅支持 Query DSL v%d。", Version))
	}
	counter := Complexity{}
	if document.Filter != nil {
		validateExpression(document.Filter, "filter", 1, fields, budget, &counter, &issues)
	}

	groupSet := validateFields(document.GroupBy, "groupBy", "QUERY_INVALID_GROUP_BY", budget.MaxGroupBy, fields, "group", &issues)
	aliases := validateAggregates(document.Aggregates, fields, budget, &issues)
	outputFields := make(map[string]bool, len(groupSet)+len(aliases))
	for field := range groupSet {
		outputFields[field] = true
	}
	for alias := range aliases {
		outputFields[alias] = true
	}
	validateSorts(document.Sorts, fields, budget, outputFields, &issues)
	validateFields(document.Projection, "projection", "QUERY_INVALID_PROJECTION", budget.MaxProjection, fields, "project", &issues)
	validatePaging(document.Paging, budget, &issues)
	if counter.Conditions > budget.MaxConditions {
		addIssue(&issues, "filter", "QUERY_COMPLEXITY_EXCEEDED", fmt.Sprintf("筛选条件不能超过 %d 个。", budget.MaxConditions))
	}
	if len(issues) > 0 {
		return Result{Issues: issues}
	}

	normalizeDocument(&document)
	plan := BuildLogicalPlan(document)
	return Result{Document: &document, Plan: &plan, Issues: []Issue{}}
}

func validateExpression(expression *Expression, path string, depth int, fields FieldCatalog, budget Budget, counter *Complexity, issues *[]Issue) {
	if depth > counter.Depth {
		counter.Depth = depth
	}
	if depth > budget.MaxDepth {
		addIssue(issues, path, "QUERY_COMPLEXITY_EXCEEDED", fmt.Sprintf("筛选嵌套不能超过 %d 层。", budget.MaxDepth))
		return
	}
	switch expression.Type {
	case "group":
		if expression.Conjunction != "and" && expression.Conjunction != "or" {
			addIssue(issues, path+".conjunction", "QUERY_INVALID_EXPRESSION", "条件组连接词必须是 and 或 or。")
		}
		if len(expression.Children) == 0 {
			addIssue(issues, path, "QUERY_EMPTY_GROUP", "条件组至少需要一个子条件。")
			return
		}
		for i := range expression.Children {
			validateExpression(&expression.Children[i], fmt.Sprintf("%s.children[%d]", path, i), depth+1, fields, budget, counter, issues)
		}
	case "condition":
		counter.Conditions++
		field := strings.TrimSpace(expression.Field)
		if field == "" {
			addIssue(issues, path+".field", "QUERY_EMPTY_FIELD", "筛选字段不能为空。")
		}
		if !valueOperators[expression.Operator] && !valuelessOperators[expression.Operator] {
			addIssue(issues, path+".operator", "QUERY_INVALID_OPERATOR", "筛选操作符无效。")
			return
		}
		if field != "" && fields != nil {
			capability, ok := fields[field]
			if !ok {
				addIssue(issues, path+".field", "QUERY_UNKNOWN_FIELD", fmt.Sprintf("字段 %s 不在字段目录中。", field))
			} else if !capability.Filterable || !operatorsByType[capability.Type][expression.Operator] {
				addIssue(issues, path+".operator", "QUERY_OPERATOR_NOT_ALLOWED", fmt.Sprintf("字段类型 %s 不支持操作符 %s。", capability.Type, expression.Operator))
			}
		}
		validateValue(expression.Operator, expression.Value, path+".value", issues)
	default:
		addIssue(issues, path+".type", "QUERY_INVALID_EXPRESSION", "未知的筛选表达式类型。")
	}
}

func validateValue(operator Operator, value any, path string, issues *[]Issue) {
	if valuelessOperators[operator] {
		if value != nil {
			addIssue(issues, path, "QUERY_INVALID_VALUE", "空值操作符不得携带值。")
		}
		return
	}
	if value == nil || !isQueryValue(value) {
		addIssue(issues, path, "QUERY_INVALID_VALUE", "该筛选操作符必须提供合法值。")
		return
	}
	items, array := value.([]any)
	if (operator == OperatorIn || operator == OperatorNotIn) && (!array || len(items) == 0) {
		addIssue(issues, path, "QUERY_INVALID_VALUE", "集合筛选至少需要一个值。")
	} else if operator == OperatorBetween && (!array || len(items) != 2) {
		addIssue(issues, path, "QUERY_INVALID_VALUE", "区间筛选必须提供两个值。")
	}
}

func validateFields(values []string, path, invalidCode string, maximum int, fields FieldCatalog, operation string, issues *[]Issue) map[string]bool {
	seen := make(map[string]bool, len(values))
	if len(values) > maximum {
		addIssue(issues, path, "QUERY_COMPLEXITY_EXCEEDED", fmt.Sprintf("%s 不能超过 %d 项。", path, maximum))
	}
	for i, raw := range values {
		field := strings.TrimSpace(raw)
		itemPath := fmt.Sprintf("%s[%d]", path, i)
		if field == "" || seen[field] {
			addIssue(issues, itemPath, invalidCode, "字段不能为空或重复。")
			continue
		}
		seen[field] = true
		validateFieldCapability(field, itemPath, operation, fields, issues)
	}
	return seen
}

func validateAggregates(values []Aggregate, fields FieldCatalog, budget Budget, issues *[]Issue) map[string]bool {
	aliases := make(map[string]bool, len(values))
	if len(values) > budget.MaxAggregates {
		addIssue(issues, "aggregates", "QUERY_COMPLEXITY_EXCEEDED", fmt.Sprintf("聚合指标不能超过 %d 个。", budget.MaxAggregates))
	}
	for i := range values {
		value := values[i]
		path := fmt.Sprintf("aggregates[%d]", i)
		field, alias := strings.TrimSpace(value.Field), strings.TrimSpace(value.Alias)
		if field == "" || alias == "" || !aggregateAliasPattern.MatchString(alias) || !validAggregate(value.Operator) {
			addIssue(issues, path, "QUERY_INVALID_AGGREGATE", "聚合字段、函数或别名无效。")
			continue
		}
		if aliases[alias] {
			addIssue(issues, path+".alias", "QUERY_DUPLICATE_ALIAS", "聚合别名不能重复。")
		} else {
			aliases[alias] = true
		}
		if fields != nil {
			capability, ok := fields[field]
			if !ok {
				addIssue(issues, path+".field", "QUERY_UNKNOWN_FIELD", fmt.Sprintf("字段 %s 不在字段目录中。", field))
			} else if !containsAggregate(capability.Aggregates, value.Operator) {
				addIssue(issues, path+".operator", "QUERY_AGGREGATE_NOT_ALLOWED", fmt.Sprintf("字段 %s 不支持聚合 %s。", field, value.Operator))
			}
		}
	}
	return aliases
}

func validateSorts(values []Sort, fields FieldCatalog, budget Budget, outputFields map[string]bool, issues *[]Issue) {
	if len(values) > budget.MaxSorts {
		addIssue(issues, "sorts", "QUERY_COMPLEXITY_EXCEEDED", fmt.Sprintf("排序字段不能超过 %d 个。", budget.MaxSorts))
	}
	for i := range values {
		field := strings.TrimSpace(values[i].Field)
		path := fmt.Sprintf("sorts[%d]", i)
		if field == "" || (values[i].Direction != "asc" && values[i].Direction != "desc") {
			addIssue(issues, path, "QUERY_INVALID_SORT", "排序字段或方向无效。")
			continue
		}
		if !outputFields[field] {
			validateFieldCapability(field, path+".field", "sort", fields, issues)
		}
	}
}

func validatePaging(value Paging, budget Budget, issues *[]Issue) {
	if value.Page <= 0 {
		addIssue(issues, "paging.page", "QUERY_INVALID_PAGING", "页码必须是正整数。")
	}
	if value.PageSize <= 0 {
		addIssue(issues, "paging.pageSize", "QUERY_INVALID_PAGING", "每页数量必须是正整数。")
	} else if value.PageSize > budget.MaxPageSize {
		addIssue(issues, "paging.pageSize", "QUERY_COMPLEXITY_EXCEEDED", fmt.Sprintf("每页数量不能超过 %d。", budget.MaxPageSize))
	}
}

func validateFieldCapability(field, path, operation string, fields FieldCatalog, issues *[]Issue) {
	if fields == nil {
		return
	}
	capability, ok := fields[field]
	if !ok {
		addIssue(issues, path, "QUERY_UNKNOWN_FIELD", fmt.Sprintf("字段 %s 不在字段目录中。", field))
		return
	}
	allowed, code := true, ""
	switch operation {
	case "project":
		allowed, code = capability.Projectable, "QUERY_FIELD_NOT_PROJECTABLE"
	case "group":
		allowed, code = capability.Groupable, "QUERY_FIELD_NOT_GROUPABLE"
	case "sort":
		allowed, code = capability.Sortable, "QUERY_FIELD_NOT_SORTABLE"
	}
	if !allowed {
		addIssue(issues, path, code, fmt.Sprintf("字段 %s 不支持该查询能力。", field))
	}
}

func normalizeDocument(document *Document) {
	document.Version = Version
	if document.Sorts == nil {
		document.Sorts = []Sort{}
	}
	for i := range document.Sorts {
		document.Sorts[i].Field = strings.TrimSpace(document.Sorts[i].Field)
	}
	document.Projection = normalizeStrings(document.Projection)
	document.GroupBy = normalizeStrings(document.GroupBy)
	for i := range document.Aggregates {
		document.Aggregates[i].Field = strings.TrimSpace(document.Aggregates[i].Field)
		document.Aggregates[i].Alias = strings.TrimSpace(document.Aggregates[i].Alias)
	}
	if document.Filter != nil {
		normalizeExpression(document.Filter)
	}
}

func normalizeExpression(expression *Expression) {
	if expression.Type == "condition" {
		expression.Field = strings.TrimSpace(expression.Field)
		return
	}
	for i := range expression.Children {
		normalizeExpression(&expression.Children[i])
	}
}

func normalizeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func normalizeBudget(value Budget) Budget {
	defaults := DefaultBudget()
	if value.MaxDepth <= 0 {
		value.MaxDepth = defaults.MaxDepth
	}
	if value.MaxConditions <= 0 {
		value.MaxConditions = defaults.MaxConditions
	}
	if value.MaxSorts <= 0 {
		value.MaxSorts = defaults.MaxSorts
	}
	if value.MaxProjection <= 0 {
		value.MaxProjection = defaults.MaxProjection
	}
	if value.MaxGroupBy <= 0 {
		value.MaxGroupBy = defaults.MaxGroupBy
	}
	if value.MaxAggregates <= 0 {
		value.MaxAggregates = defaults.MaxAggregates
	}
	if value.MaxPageSize <= 0 {
		value.MaxPageSize = defaults.MaxPageSize
	}
	return value
}

func isQueryValue(value any) bool {
	switch typed := value.(type) {
	case string, bool, float64:
		return true
	case []any:
		for _, item := range typed {
			if item != nil && !isScalar(item) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func isScalar(value any) bool {
	switch value.(type) {
	case string, bool, float64:
		return true
	}
	return false
}
func validAggregate(value AggregateOperator) bool {
	return value == AggregateCount || value == AggregateSum || value == AggregateAvg || value == AggregateMin || value == AggregateMax
}
func containsAggregate(values []AggregateOperator, target AggregateOperator) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
func setOperators(values ...Operator) map[Operator]bool {
	out := make(map[Operator]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}
func addIssue(issues *[]Issue, path, code, message string) {
	*issues = append(*issues, Issue{Path: path, Code: code, Message: message})
}
func ensureEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing json value")
		}
		return err
	}
	return nil
}
