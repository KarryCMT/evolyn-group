package model

// Variable 流程上下文变量（表达式白名单 variables.* 的数据源）。
// 值形态由类型字段约束，禁止自由嵌套结构进入表达式环境。
type Variable struct {
	// InstanceID 归属实例
	InstanceID uint
	// Key 变量名（表达式引用名，命名规则与 Node key 一致）
	Key string
	// ValueType 值类型（string/number/boolean；V1 冻结三类标量）
	ValueType VariableType
	// Value 标量值（持久化形态由仓储适配层决定）
	Value any
}

// VariableType 变量值类型（V1 冻结标量集合）。
type VariableType string

const (
	VariableTypeString  VariableType = "string"
	VariableTypeNumber  VariableType = "number"
	VariableTypeBoolean VariableType = "boolean"
)
