package numeric

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 跨端契约测试（设计 §52/§53）：前端 vitest 与本测试共读
// docs/contracts/numeric-test-vectors.json，保证 JS Runtime == Go Runtime
type vector struct {
	Operation   string  `json:"operation"`
	Args        []any   `json:"args"` // string / nil
	RoundScale  *int    `json:"roundScale"`
	RoundMode   string  `json:"roundMode"`
	Expected    *string `json:"expected"` // nil 表空值结果
	ExpectError string  `json:"expectError"`
}

func loadVectors(t *testing.T) []vector {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	// 本文件位于 apps/evolyn-core/internal/platform/numeric/ → 仓库根上溯 5 级
	path := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..", "docs", "contracts", "numeric-test-vectors.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "契约向量文件应位于 docs/contracts/")
	var doc struct {
		Vectors []vector `json:"vectors"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	require.NotEmpty(t, doc.Vectors)
	return doc.Vectors
}

// tryArg 参数规约：nil → 空值 Numeric；字符串走 Parse。
// 解析失败原样上抛——错误类向量的期望码可能正是解析错误本身
func tryArg(arg any) (Numeric, error) {
	if arg == nil {
		return Numeric{null: true}, nil
	}
	text, ok := arg.(string)
	if !ok {
		return Numeric{}, newErr(CodeInvalidArgument, fmt.Sprintf("契约向量参数仅允许 string/null，got %T", arg))
	}
	return Parse(text)
}

func serializeResult(t *testing.T, n Numeric) *string {
	t.Helper()
	text, err := n.Serialize()
	require.NoError(t, err, "结果序列化不应报错")
	if n.null {
		return nil
	}
	return &text
}

func applyRound(n Numeric, v vector) (Numeric, error) {
	if v.RoundScale == nil {
		return n, nil
	}
	mode := RoundingMode(v.RoundMode)
	if mode == "" {
		mode = ModeHalfUp
	}
	return n.Round(int32(*v.RoundScale), mode)
}

func TestNumericContractVectors(t *testing.T) {
	for i, v := range loadVectors(t) {
		v := v
		name := fmt.Sprintf("#%d %s(%v)", i, v.Operation, v.Args)
		t.Run(name, func(t *testing.T) {
			result, err := dispatch(t, v)
			if v.ExpectError != "" {
				require.Error(t, err, "期望稳定错误码 %s", v.ExpectError)
				var ne *NumericError
				require.ErrorAs(t, err, &ne)
				assert.Equal(t, NumericErrorCode(v.ExpectError), ne.Code)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, deref(v.Expected), deref(serializeResult(t, result)))
		})
	}
}

func deref(s *string) string {
	if s == nil {
		return "<null>"
	}
	return *s
}

func dispatch(t *testing.T, v vector) (Numeric, error) {
	t.Helper()
	args := v.Args

	binary := func(fn func(a, b Numeric) (Numeric, error)) (Numeric, error) {
		a, err := tryArg(args[0])
		if err != nil {
			return a, err
		}
		b, err := tryArg(args[1])
		if err != nil {
			return b, err
		}
		return fn(a, b)
	}
	unary := func(fn func(a Numeric) (Numeric, error)) (Numeric, error) {
		a, err := tryArg(args[0])
		if err != nil {
			return a, err
		}
		return fn(a)
	}

	switch v.Operation {
	case "parse", "serialize":
		return unary(func(a Numeric) (Numeric, error) { return a, nil })
	case "add":
		return binary(func(a, b Numeric) (Numeric, error) { return a.Add(b) })
	case "subtract":
		return binary(func(a, b Numeric) (Numeric, error) { return a.Subtract(b) })
	case "multiply":
		return binary(func(a, b Numeric) (Numeric, error) { return a.Multiply(b) })
	case "divide":
		return binary(func(a, b Numeric) (Numeric, error) {
			n, err := a.Divide(b)
			if err != nil {
				return n, err
			}
			return applyRound(n, v)
		})
	case "mod":
		return binary(func(a, b Numeric) (Numeric, error) { return a.Mod(b) })
	case "abs":
		return unary(func(a Numeric) (Numeric, error) { return a.Abs() })
	case "negate":
		return unary(func(a Numeric) (Numeric, error) { return a.Negate() })
	case "sqrt":
		return unary(func(a Numeric) (Numeric, error) {
			n, err := a.Sqrt()
			if err != nil {
				return n, err
			}
			return applyRound(n, v)
		})
	case "round":
		scale := 0
		if v.RoundScale != nil {
			scale = *v.RoundScale
		}
		mode := RoundingMode(v.RoundMode)
		if mode == "" {
			mode = ModeHalfUp
		}
		final := int32(scale)
		return unary(func(a Numeric) (Numeric, error) { return a.Round(final, mode) })
	case "ceil":
		return unary(func(a Numeric) (Numeric, error) { return a.Round(0, ModeCeil) })
	case "floor":
		return unary(func(a Numeric) (Numeric, error) { return a.Round(0, ModeFloor) })
	case "compare":
		return binary(func(a, b Numeric) (Numeric, error) {
			cmp, err := a.Compare(b)
			if err != nil {
				return Numeric{}, err
			}
			return ParseMust(fmtInt(cmp)), nil
		})
	case "sum":
		return aggregate(args, Sum)
	case "average":
		return aggregate(args, func(values []Numeric) (Numeric, error) {
			n, err := Average(values)
			if err != nil {
				return n, err
			}
			return applyRound(n, v)
		})
	case "min":
		return aggregate(args, Min)
	case "max":
		return aggregate(args, Max)
	case "product":
		return aggregate(args, Product)
	default:
		t.Fatalf("未知契约操作: %s", v.Operation)
		return Numeric{}, nil
	}
}

func aggregate(args []any, fn func([]Numeric) (Numeric, error)) (Numeric, error) {
	values := make([]Numeric, 0, len(args))
	for _, a := range args {
		n, err := tryArg(a)
		if err != nil {
			return Numeric{}, err
		}
		values = append(values, n)
	}
	return fn(values)
}
