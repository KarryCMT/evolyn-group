// Package query 定义与 @evolyn.do/query 镜像的纯查询 AST、能力矩阵和逻辑计划。
// 本包不依赖 Gin、GORM、HTTP 或具体存储，平台适配层只能把已校验计划编译为
// 参数化查询，不能把客户端字符串直接解释为 SQL。
package query

const Version = 1

type Operator = string

const (
	OperatorEQ          Operator = "eq"
	OperatorNEQ         Operator = "neq"
	OperatorContains    Operator = "contains"
	OperatorNotContains Operator = "notContains"
	OperatorStartsWith  Operator = "startsWith"
	OperatorEndsWith    Operator = "endsWith"
	OperatorGT          Operator = "gt"
	OperatorGTE         Operator = "gte"
	OperatorLT          Operator = "lt"
	OperatorLTE         Operator = "lte"
	OperatorIn          Operator = "in"
	OperatorNotIn       Operator = "notIn"
	OperatorBetween     Operator = "between"
	OperatorIsNull      Operator = "isNull"
	OperatorIsNotNull   Operator = "isNotNull"
)

type AggregateOperator = string

const (
	AggregateCount AggregateOperator = "count"
	AggregateSum   AggregateOperator = "sum"
	AggregateAvg   AggregateOperator = "avg"
	AggregateMin   AggregateOperator = "min"
	AggregateMax   AggregateOperator = "max"
)

type Expression struct {
	Type        string       `json:"type"`
	Conjunction string       `json:"conjunction,omitempty"`
	Children    []Expression `json:"children,omitempty"`
	Field       string       `json:"field,omitempty"`
	Operator    Operator     `json:"operator,omitempty"`
	Value       any          `json:"value,omitempty"`
}

type Sort struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type Paging struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

type Aggregate struct {
	Field    string            `json:"field"`
	Operator AggregateOperator `json:"operator"`
	Alias    string            `json:"alias"`
}

type Document struct {
	Version    int         `json:"version"`
	Filter     *Expression `json:"filter,omitempty"`
	Sorts      []Sort      `json:"sorts"`
	Paging     Paging      `json:"paging"`
	Projection []string    `json:"projection,omitempty"`
	GroupBy    []string    `json:"groupBy,omitempty"`
	Aggregates []Aggregate `json:"aggregates,omitempty"`
}

type FieldType string

const (
	FieldText       FieldType = "text"
	FieldDecimal    FieldType = "decimal"
	FieldNumber     FieldType = "number"
	FieldBoolean    FieldType = "boolean"
	FieldDate       FieldType = "date"
	FieldDateTime   FieldType = "datetime"
	FieldEnum       FieldType = "enum"
	FieldMember     FieldType = "member"
	FieldDepartment FieldType = "department"
)

type FieldCapability struct {
	Type        FieldType           `json:"type"`
	Filterable  bool                `json:"filterable"`
	Sortable    bool                `json:"sortable"`
	Projectable bool                `json:"projectable"`
	Groupable   bool                `json:"groupable"`
	Aggregates  []AggregateOperator `json:"aggregates"`
}

type FieldCatalog map[string]FieldCapability

type Budget struct {
	MaxDepth      int `json:"maxDepth"`
	MaxConditions int `json:"maxConditions"`
	MaxSorts      int `json:"maxSorts"`
	MaxProjection int `json:"maxProjection"`
	MaxGroupBy    int `json:"maxGroupBy"`
	MaxAggregates int `json:"maxAggregates"`
	MaxPageSize   int `json:"maxPageSize"`
}

func DefaultBudget() Budget {
	return Budget{MaxDepth: 12, MaxConditions: 50, MaxSorts: 8, MaxProjection: 100, MaxGroupBy: 8, MaxAggregates: 16, MaxPageSize: 100}
}

type Issue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

type Complexity struct {
	Depth      int `json:"depth"`
	Conditions int `json:"conditions"`
}

type LogicalPlan struct {
	Filter     *Expression `json:"filter,omitempty"`
	Projection []string    `json:"projection"`
	GroupBy    []string    `json:"groupBy"`
	Aggregates []Aggregate `json:"aggregates"`
	Sorts      []Sort      `json:"sorts"`
	Paging     Paging      `json:"paging"`
	Aggregate  bool        `json:"aggregate"`
	Complexity Complexity  `json:"complexity"`
}

type Result struct {
	Document *Document    `json:"document"`
	Plan     *LogicalPlan `json:"plan"`
	Issues   []Issue      `json:"issues"`
}
