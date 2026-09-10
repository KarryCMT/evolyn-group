package numeric

import (
	"math"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

// 平台默认口径（设计 §10.1，与前端 DEFAULT_NUMERIC_CONTEXT 对齐）
const (
	DefaultPrecision = 40
	DefaultMaxScale  = 18
)

// RoundingMode 平台舍入枚举（字符串形态与前端逐字对齐，跨端契约载体）
type RoundingMode string

const (
	ModeUp       RoundingMode = "UP"
	ModeDown     RoundingMode = "DOWN"
	ModeCeil     RoundingMode = "CEIL"
	ModeFloor    RoundingMode = "FLOOR"
	ModeHalfUp   RoundingMode = "HALF_UP"
	ModeHalfDown RoundingMode = "HALF_DOWN"
	ModeHalfEven RoundingMode = "HALF_EVEN"
)

// Numeric 数值载体：null 布尔承载 SQL 空值语义并沿运算传播
// （对齐前端 EmptyValuePolicy.NULL 平台默认）
type Numeric struct {
	d    decimal.Decimal
	null bool
}

// Null 空值判定（SQL NULL 语义）
func (n Numeric) Null() bool { return n.null }

// Parse 字符串解析：拒 NaN/Infinity/垃圾文本，超精度抛 PRECISION_EXCEEDED
func Parse(input string) (Numeric, error) {
	text := strings.TrimSpace(input)
	if regexpNaN.MatchString(text) {
		return Numeric{}, newErr(CodeNonFiniteNumber, "non-finite number literal: "+input)
	}
	if !regexpDecimal.MatchString(text) {
		return Numeric{}, newErr(CodeInvalidNumber, "invalid decimal string: "+input)
	}
	d, err := decimal.NewFromString(text)
	if err != nil {
		return Numeric{}, newErr(CodeInvalidNumber, "invalid decimal string: "+input)
	}
	return guardPrecision(d)
}

var (
	regexpNaN     = regexp.MustCompile(`^(?i)(nan|infinity|-infinity|\+?inf)$`)
	regexpDecimal = regexp.MustCompile(`^[+-]?(\d+(\.\d*)?|\.\d+)([eE][+-]?\d+)?$`)
)

func init() {
	// 除法展开精度对齐平台：40 位小数中间态（canonical 前按有效数字截断）
	decimal.DivisionPrecision = DefaultPrecision
}

// guardPrecision 有效数字护栏（零值豁免）
func guardPrecision(d decimal.Decimal) (Numeric, error) {
	if !d.IsZero() && d.NumDigits() > DefaultPrecision {
		return Numeric{}, newErr(CodePrecisionExceeded, "significant digits exceed precision")
	}
	return Numeric{d: d}, nil
}

// binary 二元运算：任一空值结果为空值（SQL 语义），运算后再过精度护栏
func binary(a, b Numeric, fn func(x, y decimal.Decimal) (decimal.Decimal, error)) (Numeric, error) {
	if a.null || b.null {
		return Numeric{null: true}, nil
	}
	result, err := fn(a.d, b.d)
	if err != nil {
		return Numeric{}, err
	}
	return guardPrecision(result)
}

func (n Numeric) Add(other Numeric) (Numeric, error) {
	return binary(n, other, func(x, y decimal.Decimal) (decimal.Decimal, error) { return x.Add(y), nil })
}

func (n Numeric) Subtract(other Numeric) (Numeric, error) {
	return binary(n, other, func(x, y decimal.Decimal) (decimal.Decimal, error) { return x.Sub(y), nil })
}

func (n Numeric) Multiply(other Numeric) (Numeric, error) {
	return binary(n, other, func(x, y decimal.Decimal) (decimal.Decimal, error) { return x.Mul(y), nil })
}

func (n Numeric) Divide(other Numeric) (Numeric, error) {
	return binary(n, other, func(x, y decimal.Decimal) (decimal.Decimal, error) {
		if y.IsZero() {
			return decimal.Decimal{}, newErr(CodeDivisionByZero, "division by zero")
		}
		// 除法按全局 DivisionPrecision 位小数展开，再截断到平台有效数字
		// （DivRound 的精度参数是小数位而非有效数字，直接用会虚增系数位数）
		return truncateToPrecision(x.Div(y)), nil
	})
}

func (n Numeric) Mod(other Numeric) (Numeric, error) {
	return binary(n, other, func(x, y decimal.Decimal) (decimal.Decimal, error) {
		if y.IsZero() {
			return decimal.Decimal{}, newErr(CodeDivisionByZero, "modulo by zero")
		}
		return x.Mod(y), nil
	})
}

func (n Numeric) unary(fn func(x decimal.Decimal) decimal.Decimal) (Numeric, error) {
	if n.null {
		return n, nil
	}
	return guardPrecision(fn(n.d))
}

func (n Numeric) Abs() (Numeric, error) {
	return n.unary(func(x decimal.Decimal) decimal.Decimal { return x.Abs() })
}

func (n Numeric) Negate() (Numeric, error) {
	return n.unary(func(x decimal.Decimal) decimal.Decimal { return x.Neg() })
}

// Sqrt 负数开方抛 INVALID_ARGUMENT；结果按平台精度舍入（无理数截断）
func (n Numeric) Sqrt() (Numeric, error) {
	if n.null {
		return n, nil
	}
	if n.d.IsNegative() {
		return Numeric{}, newErr(CodeInvalidArgument, "sqrt of negative number")
	}
	// shopspring v1.4 无 Sqrt：牛顿迭代 x' = (x + v/x) / 2，收敛到平台精度
	return n.unary(decSqrt)
}

var decTwo = decimal.NewFromInt(2)

func decSqrt(x decimal.Decimal) decimal.Decimal {
	if x.IsZero() {
		return x
	}
	// 初值取 float64 平方根（约 16 位有效数字，牛顿迭代每步倍增精度）；
	// 超出 float64 范围时退化为 x 自身（大数迭代收敛稍慢但可达）
	seed := decimal.NewFromFloat(math.Sqrt(x.InexactFloat64()))
	if seed.IsZero() || seed.IsNegative() || !seed.IsPositive() { // NaN/Inf 防御（IsPositive 对非有限值恒假）
		seed = x
	}
	guess := seed
	for i := 0; i < 80; i++ {
		next := guess.Add(x.DivRound(guess, DefaultPrecision)).DivRound(decTwo, DefaultPrecision)
		if next.Equal(guess) {
			break
		}
		guess = next
	}
	return truncateToPrecision(guess)
}

// truncateToPrecision 截断到平台有效数字位数（无理数截断，不额外引入误差放大）
func truncateToPrecision(d decimal.Decimal) decimal.Decimal {
	places := int32(DefaultPrecision) - int32(d.NumDigits()) - d.Exponent()
	if places < 0 {
		places = 0
	}
	return d.Round(places)
}

// roundHalfDown 半值向零：正好 0.5 舍位（DOWN），否则常规四舍五入（HALF_UP）
func roundHalfDown(d decimal.Decimal, places int32) decimal.Decimal {
	scaled := d.Shift(places)
	frac := scaled.Sub(scaled.Truncate(0)).Abs()
	if frac.Equal(decimal.New(5, -1)) {
		return d.RoundDown(places)
	}
	return d.Round(places)
}

// Compare 三态比较：空值参与抛 INVALID_ARGUMENT
func (n Numeric) Compare(other Numeric) (int, error) {
	if n.null || other.null {
		return 0, newErr(CodeInvalidArgument, "compare with null operand")
	}
	switch {
	case n.d.LessThan(other.d):
		return -1, nil
	case n.d.GreaterThan(other.d):
		return 1, nil
	default:
		return 0, nil
	}
}

// Round 按 scale 舍入：七种平台模式映射 shopspring 对应实现
func (n Numeric) Round(scale int32, mode RoundingMode) (Numeric, error) {
	if scale < 0 || scale > DefaultMaxScale {
		return Numeric{}, newErr(CodeInvalidArgument, "scale out of range")
	}
	if n.null {
		return n, nil
	}
	var d decimal.Decimal
	switch mode {
	case ModeUp:
		d = n.d.RoundUp(scale)
	case ModeDown:
		d = n.d.RoundDown(scale)
	case ModeCeil:
		d = n.d.RoundCeil(scale)
	case ModeFloor:
		d = n.d.RoundFloor(scale)
	case ModeHalfUp:
		d = n.d.Round(scale) // shopspring Round 即 HALF_UP（半值远离零）
	case ModeHalfDown:
		d = roundHalfDown(n.d, scale)
	case ModeHalfEven:
		d = n.d.RoundBank(scale)
	default:
		return Numeric{}, newErr(CodeInvalidArgument, "unknown rounding mode: "+string(mode))
	}
	return Numeric{d: d}, nil
}

// Serialize canonical 序列化（设计 §48/§49）：scale 护栏后输出最短精确
// 表示（去无意义尾零），空值返回空串由调用方转 null
func (n Numeric) Serialize() (string, error) {
	if n.null {
		return "", nil
	}
	// 先取 canonical（去无意义尾零：除法恒展开到 40 位小数，整除商
	// 2.000...0 的真实 scale 为 0），再按 canonical 实际小数位数做护栏
	text := canonical(n.d)
	if idx := strings.IndexByte(text, '.'); idx >= 0 && len(text)-idx-1 > DefaultMaxScale {
		return "", newErr(CodeScaleExceeded, "decimal places exceed maxScale; round explicitly before serialize")
	}
	return text, nil
}

// canonical 最短精确表示：去尾零（shopspring String 不主动去除无意义尾零，
// 与前端 decimal.js 行为对齐须显式裁剪），无指数形态
func canonical(d decimal.Decimal) string {
	text := d.String()
	if strings.Contains(text, ".") {
		text = strings.TrimRight(text, "0")
		text = strings.TrimSuffix(text, ".")
	}
	if text == "" || text == "-" {
		return "0"
	}
	return text
}

// Sum/Min/Max/Product/Aggregate 聚合：空值元素按 SQL 语义跳过，全空为空值
func Sum(values []Numeric) (Numeric, error) {
	items := nonNull(values)
	if len(items) == 0 {
		return Numeric{null: true}, nil
	}
	acc := items[0]
	for _, v := range items[1:] {
		var err error
		if acc, err = acc.Add(v); err != nil {
			return Numeric{}, err
		}
	}
	return acc, nil
}

func Average(values []Numeric) (Numeric, error) {
	items := nonNull(values)
	if len(items) == 0 {
		return Numeric{null: true}, nil
	}
	total, err := Sum(items)
	if err != nil {
		return Numeric{}, err
	}
	return total.Divide(ParseMust(fmtInt(len(items))))
}

func Min(values []Numeric) (Numeric, error) {
	items := nonNull(values)
	if len(items) == 0 {
		return Numeric{null: true}, nil
	}
	best := items[0]
	for _, v := range items[1:] {
		cmp, err := v.Compare(best)
		if err != nil {
			return Numeric{}, err
		}
		if cmp < 0 {
			best = v
		}
	}
	return best, nil
}

func Max(values []Numeric) (Numeric, error) {
	items := nonNull(values)
	if len(items) == 0 {
		return Numeric{null: true}, nil
	}
	best := items[0]
	for _, v := range items[1:] {
		cmp, err := v.Compare(best)
		if err != nil {
			return Numeric{}, err
		}
		if cmp > 0 {
			best = v
		}
	}
	return best, nil
}

func Product(values []Numeric) (Numeric, error) {
	items := nonNull(values)
	if len(items) == 0 {
		return Numeric{null: true}, nil
	}
	acc := items[0]
	for _, v := range items[1:] {
		var err error
		if acc, err = acc.Multiply(v); err != nil {
			return Numeric{}, err
		}
	}
	return acc, nil
}

func nonNull(values []Numeric) []Numeric {
	out := make([]Numeric, 0, len(values))
	for _, v := range values {
		if !v.null {
			out = append(out, v)
		}
	}
	return out
}

func ParseMust(s string) Numeric {
	n, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return n
}

func fmtInt(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
