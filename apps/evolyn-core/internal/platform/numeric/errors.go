// Package numeric Go 侧高精度数值域（设计 §29/§30，Phase 2 契约对端）：
// 与前端 @evolyn.do/numeric 共享 docs/contracts/numeric-test-vectors.json
// 契约向量，保证 JS Runtime == Go Runtime。核心业务金额禁止 float64。
package numeric

import "fmt"

// NumericErrorCode 稳定错误码（与前端 errors/NumericErrorCode.ts 逐字对齐）
type NumericErrorCode string

const (
	CodeInvalidNumber     NumericErrorCode = "INVALID_NUMBER"
	CodeDivisionByZero    NumericErrorCode = "DIVISION_BY_ZERO"
	CodeNonFiniteNumber   NumericErrorCode = "NON_FINITE_NUMBER"
	CodePrecisionExceeded NumericErrorCode = "PRECISION_EXCEEDED"
	CodeScaleExceeded     NumericErrorCode = "SCALE_EXCEEDED"
	CodeInvalidArgument   NumericErrorCode = "INVALID_ARGUMENT"
)

// NumericError 数值域统一错误：code 即对外稳定标识
type NumericError struct {
	Code NumericErrorCode
	Msg  string
}

func (e *NumericError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Msg)
}

func newErr(code NumericErrorCode, msg string) *NumericError {
	return &NumericError{Code: code, Msg: msg}
}
