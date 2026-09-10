// 数值字段族语义（Phase 4，设计 §17/§36/§58-4）：decimal/money/percent 三种
// 控件的属性约束、有效默认与值校验语义在本文件收口，schema.go（保存/发布
// 校验）、value.go（提交值终审）与 storage_publish.go（物理列 NUMERIC(p,s)
// 解析）共用；前端 packages/form/src/schema/numeric.ts 按同一语义镜像，
// 修改本文件必须同步 TS 镜像与两端测试。
//
// 值协议：业务十进制一律 canonical decimal string（设计 §15/§27），空值 null；
// 现有 number 控件保持 JS number 语义，不在本族范围。
package service

import (
	"fmt"
	"regexp"
	"strings"

	"evolyn/internal/platform/numeric"
)

// decimalWidgetTypes 数值字段族的三种控件类型。
var decimalWidgetTypes = map[string]bool{
	"decimal": true,
	"money":   true,
	"percent": true,
}

// numericFieldDefault 数值字段族未显式配置时的有效精度默认
// （与 TS schema/numeric.ts NUMERIC_FIELD_DEFAULTS 逐字一致，设计 §17）。
type numericFieldDefault struct {
	Precision int
	Scale     int
}

var numericFieldDefaults = map[string]numericFieldDefault{
	"decimal": {Precision: 20, Scale: 6},
	"money":   {Precision: 20, Scale: 2},
	"percent": {Precision: 10, Scale: 6},
}

// 精度护栏（与 internal/platform/numeric DefaultPrecision 40 / DefaultMaxScale
// 18 及前端 NUMERIC_FIELD_LIMITS 对齐）。
const (
	numericPrecisionMin = 1
	numericPrecisionMax = 40
	numericScaleMin     = 0
	numericScaleMax     = 18
)

// roundingModeValues 计算链舍入模式枚举（设计 §11；与 numeric.RoundingMode
// 常量值逐字一致，schema 属性校验按字符串枚举消费）。
var roundingModeValues = []string{
	"UP", "DOWN", "CEIL", "FLOOR", "HALF_UP", "HALF_DOWN", "HALF_EVEN",
}

// decimalTextPattern 十进制文本输入形状：可选负号 + 整数位 + 可选小数位；
// 拒绝指数记法、正号与空白（canonical 协议 §49 无指数）。
var decimalTextPattern = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)

func isDecimalWidgetType(widgetType string) bool {
	return decimalWidgetTypes[widgetType]
}

// validDecimalText 报告文本是否符合协议形状。
func validDecimalText(text string) bool {
	return decimalTextPattern.MatchString(text)
}

// effectiveNumericPrecisionScale 解析控件的有效 (precision, scale)：显式配置
// 优先（防御式读取，非法值按未配置），否则按类型取默认。发布期物理列与
// 提交终审共用同一解析结果。
func effectiveNumericPrecisionScale(widgetType string, widget map[string]any) (int, int) {
	fallback := numericFieldDefaults[widgetType]
	precision := fallback.Precision
	scale := fallback.Scale
	if declared, ok := jsonInt(widget["precision"]); ok &&
		declared >= numericPrecisionMin && declared <= numericPrecisionMax {
		precision = declared
	}
	if declared, ok := jsonInt(widget["scale"]); ok &&
		declared >= numericScaleMin && declared <= numericScaleMax {
		scale = declared
	}
	return precision, scale
}

// decimalFractionDigits 小数位计数（按值语义）：尾随零不计位（"1.500" 计 1 位）。
func decimalFractionDigits(text string) int {
	dot := strings.IndexByte(text, '.')
	if dot < 0 {
		return 0
	}
	end := len(text)
	for end > dot+1 && text[end-1] == '0' {
		end--
	}
	return end - dot - 1
}

// decimalIntegerDigits 整数位计数（按值语义）：前导零不计位；零值恒为 0 位。
func decimalIntegerDigits(text string) int {
	start := 0
	if strings.HasPrefix(text, "-") {
		start = 1
	}
	dot := strings.IndexByte(text, '.')
	intText := text[start:]
	if dot >= 0 {
		intText = text[start:dot]
	}
	trimmed := strings.TrimLeft(intText, "0")
	if trimmed == "" {
		return 0
	}
	if trimmed == "0" {
		return 0
	}
	return len([]rune(trimmed))
}

// decimalDigitIssue 位数约束违规码：空串通过 / "scale" 小数位超限 /
// "precision" 整数位超限。
func decimalDigitIssue(text string, precision, scale int) string {
	if decimalFractionDigits(text) > scale {
		return "scale"
	}
	if decimalIntegerDigits(text) > precision-scale {
		return "precision"
	}
	return ""
}

// compareDecimalText 十进制文本精确比较（-1/0/1）：走 internal/platform/numeric
// （shopspring 唯一触达点），入参已过形状校验；解析失败按不可比较返回
// (0, false)，由调用方决定保守结论。
func compareDecimalText(a, b string) (int, bool) {
	left, err := numeric.Parse(a)
	if err != nil {
		return 0, false
	}
	right, err := numeric.Parse(b)
	if err != nil {
		return 0, false
	}
	order, err := left.Compare(right)
	if err != nil {
		return 0, false
	}
	return order, true
}

// resolveNumericColumnSpec 发布期为数值字段族解析物理列精度修饰：显式配置
// 优先，否则按类型默认；返回值直接写入 storage.ColumnSpec（NUMERIC(p,s)）。
func resolveNumericColumnSpec(widgetType string, widget map[string]any) (int16, int16) {
	precision, scale := effectiveNumericPrecisionScale(widgetType, widget)
	return int16(precision), int16(scale)
}

// validateNumericFamilyCrossRules 数值字段族交叉约束（与 TS
// validateNumericFamilyCrossRules 逐字一致）：有效 scale ≤ 有效 precision；
// min ≤ max；defaultValue 落在范围内；min/max/defaultValue 自身须满足
// scale/precision 位数约束（防不可满足范围）。属性级形状已在
// validateWidgetProp(kind=decimal) 拒绝，此处防御式读取。
func validateNumericFamilyCrossRules(widgetType string, widget map[string]any, path string, issues *[]SchemaIssue) {
	precision, scale := effectiveNumericPrecisionScale(widgetType, widget)
	if scale > precision {
		*issues = append(*issues, SchemaIssue{Path: path + ".scale", Message: "scale 不能大于 precision"})
	}
	decimalOf := func(key string) (string, bool) {
		text, ok := widget[key].(string)
		if !ok || !validDecimalText(text) {
			return "", false
		}
		return text, true
	}
	min, hasMin := decimalOf("min")
	max, hasMax := decimalOf("max")
	defaultValue, hasDefault := decimalOf("defaultValue")
	for _, entry := range []struct {
		key  string
		text string
		ok   bool
	}{{"min", min, hasMin}, {"max", max, hasMax}, {"defaultValue", defaultValue, hasDefault}} {
		if !entry.ok {
			continue
		}
		switch decimalDigitIssue(entry.text, precision, scale) {
		case "scale":
			*issues = append(*issues, SchemaIssue{
				Path:    fmt.Sprintf("%s.%s", path, entry.key),
				Message: fmt.Sprintf("%s 最多支持 %d 位小数", entry.key, scale),
			})
		case "precision":
			*issues = append(*issues, SchemaIssue{
				Path:    fmt.Sprintf("%s.%s", path, entry.key),
				Message: fmt.Sprintf("%s 整数位最多 %d 位", entry.key, precision-scale),
			})
		}
	}
	if hasMin && hasMax {
		if order, ok := compareDecimalText(min, max); ok && order > 0 {
			*issues = append(*issues, SchemaIssue{Path: path + ".max", Message: "max 不能小于 min"})
		}
	}
	if hasDefault {
		if hasMin {
			if order, ok := compareDecimalText(defaultValue, min); ok && order < 0 {
				*issues = append(*issues, SchemaIssue{Path: path + ".defaultValue", Message: "defaultValue 不能小于 min"})
			}
		}
		if hasMax {
			if order, ok := compareDecimalText(defaultValue, max); ok && order > 0 {
				*issues = append(*issues, SchemaIssue{Path: path + ".defaultValue", Message: "defaultValue 不能大于 max"})
			}
		}
	}
}

// orderedDecimalMatch 数值字段族的 gt/gte/lt/lte/between 求值（与 TS
// rules.ts compareOrdered 的 decimal 分支同语义）：值与常量均为 decimal
// string，经 compareDecimalText 精确比较；任何不可比较输入令条件不成立。
func orderedDecimalMatch(method string, rawValue any, expected []any) bool {
	left, ok := rawValue.(string)
	if !ok || !validDecimalText(left) {
		return false
	}
	bounds := make([]string, 0, 2)
	for _, entry := range expected {
		text, isString := entry.(string)
		if !isString || !validDecimalText(text) {
			return false
		}
		bounds = append(bounds, text)
	}
	orderOf := func(index int) (int, bool) {
		return compareDecimalText(left, bounds[index])
	}
	switch method {
	case "gt":
		order, ok := orderOf(0)
		return len(bounds) == 1 && ok && order > 0
	case "gte":
		order, ok := orderOf(0)
		return len(bounds) == 1 && ok && order >= 0
	case "lt":
		order, ok := orderOf(0)
		return len(bounds) == 1 && ok && order < 0
	case "lte":
		order, ok := orderOf(0)
		return len(bounds) == 1 && ok && order <= 0
	case "between":
		if len(bounds) != 2 {
			return false
		}
		lower, lowerOK := orderOf(0)
		upper, upperOK := orderOf(1)
		return lowerOK && upperOK && lower >= 0 && upper <= 0
	}
	return false
}
