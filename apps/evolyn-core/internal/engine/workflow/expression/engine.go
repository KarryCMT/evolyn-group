// Package expression 表达式引擎 SPI 与 Expr 实现（第 16 章）。
// V1 统一使用 expr-lang/expr 承载条件分支、动态审批人条件等场景；
// 安全要求：仅暴露白名单变量（form/starter/variables），禁止任意
// Go 函数调用与数据库访问，发布期预编译，运行期错误映射稳定业务错误码。
package expression

import "fmt"

// MaxExpressionLength 表达式最大长度（冻结复杂度上限，第 16.2 章）。
const MaxExpressionLength = 512

// Program 预编译后的可执行表达式。
type Program interface {
	// Eval 在白名单环境变量上求值；env 键仅接受 form/starter/variables，
	// 多余键忽略，缺失键以零值参与求值。
	Eval(env map[string]any) (any, error)
}

// Engine 表达式引擎 SPI：发布期校验与预编译、运行期求值的唯一入口。
// 内核其它部分不得直接 import 表达式库。
type Engine interface {
	// Compile 编译表达式；语法错误、超出长度限制、引用非白名单变量
	// 或尝试函数调用时返回错误（发布校验据此拒绝发布）。
	Compile(source string) (Program, error)
}

// ErrExpressionInvalid 表达式非法（运行期求值错误统一包装为本类型，
// 出网映射稳定业务错误码 WORKFLOW_EXPRESSION_INVALID）。
type ErrExpressionInvalid struct{ Cause error }

func (e *ErrExpressionInvalid) Error() string {
	return fmt.Sprintf("表达式非法: %v", e.Cause)
}

func (e *ErrExpressionInvalid) Unwrap() error { return e.Cause }
