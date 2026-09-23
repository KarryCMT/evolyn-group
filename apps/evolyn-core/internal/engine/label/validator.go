package label

import (
	"fmt"
	"math"
	"regexp"
)

const maxElements = 200

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}([0-9a-fA-F]{2})?$`)

var systemKeys = map[string]bool{
	"recordId": true, "createdAt": true, "updatedAt": true,
	"createdBy": true, "currentUser": true, "currentDate": true,
}

var supportedFonts = map[string]bool{
	"Arial": true, "Noto Sans": true, "Noto Sans CJK SC": true,
	// 兼容既有模板；服务端统一回退到嵌入的 Noto Sans SC 字形。
	"sans-serif": true,
}

// Validate 对 LabelSchema V1 做与渲染器同源的封闭校验。
func Validate(schema *Schema) []Issue {
	issues := make([]Issue, 0)
	add := func(path, code, message string) {
		issues = append(issues, Issue{Path: path, Code: code, Message: message})
	}
	if schema == nil {
		add("$", "SCHEMA_REQUIRED", "标签 Schema 不能为空")
		return issues
	}
	if schema.SchemaVersion != "1.0" {
		add("$.schemaVersion", "VERSION_UNSUPPORTED", "仅支持 LabelSchema 1.0")
	}
	if schema.Name == "" {
		add("$.name", "NAME_REQUIRED", "标签名称不能为空")
	}
	if !positiveFinite(schema.Page.Width) || !positiveFinite(schema.Page.Height) {
		add("$.page", "PAGE_SIZE_INVALID", "标签宽高必须为正数")
	}
	if schema.Page.Unit != "mm" && schema.Page.Unit != "px" {
		add("$.page.unit", "PAGE_UNIT_INVALID", "标签单位仅支持 mm 或 px")
	}
	if schema.Page.DPI != 96 && schema.Page.DPI != 203 && schema.Page.DPI != 300 && schema.Page.DPI != 600 {
		add("$.page.dpi", "PAGE_DPI_INVALID", "DPI 仅支持 96、203、300、600")
	}
	if !validColor(schema.Page.Background) {
		add("$.page.background", "COLOR_INVALID", "页面背景色必须为十六进制颜色")
	}
	if schema.Settings.GridSize <= 0 || math.IsNaN(schema.Settings.GridSize) {
		add("$.settings.gridSize", "GRID_SIZE_INVALID", "网格尺寸必须为正数")
	}
	if len(schema.Elements) > maxElements {
		add("$.elements", "ELEMENT_LIMIT_EXCEEDED", fmt.Sprintf("元素数量不能超过 %d", maxElements))
	}
	ids := make(map[string]bool, len(schema.Elements))
	for index := range schema.Elements {
		element := &schema.Elements[index]
		path := fmt.Sprintf("$.elements[%d]", index)
		if element.ID == "" {
			add(path+".id", "ELEMENT_ID_REQUIRED", "元素 ID 不能为空")
		} else if ids[element.ID] {
			add(path+".id", "ELEMENT_ID_DUPLICATED", "元素 ID 不能重复")
		}
		ids[element.ID] = true
		if !finite(element.X) || !finite(element.Y) || !positiveFinite(element.Width) || !positiveFinite(element.Height) {
			add(path, "ELEMENT_BOUNDS_INVALID", "元素坐标必须有限且宽高必须为正数")
		}
		if !finite(element.Rotation) {
			add(path+".rotation", "ELEMENT_ROTATION_INVALID", "旋转角度必须是有限数值")
		}
		if element.Opacity != nil && (*element.Opacity < 0 || *element.Opacity > 1 || !finite(*element.Opacity)) {
			add(path+".opacity", "ELEMENT_OPACITY_INVALID", "透明度必须在 0 到 1 之间")
		}
		validateElement(element, path, add)
	}
	return issues
}

func validateElement(element *Element, path string, add func(string, string, string)) {
	switch element.Type {
	case "text", "field":
		validateValue(element.Value, path+".value", add)
		validateTextStyle(element.Style, path+".style", add)
	case "qrcode":
		validateValue(element.Value, path+".value", add)
		if element.Options == nil {
			add(path+".options", "QR_OPTIONS_REQUIRED", "二维码配置不能为空")
			return
		}
		if element.Options.ErrorCorrection != "L" && element.Options.ErrorCorrection != "M" && element.Options.ErrorCorrection != "Q" && element.Options.ErrorCorrection != "H" {
			add(path+".options.errorCorrection", "QR_LEVEL_INVALID", "二维码纠错级别无效")
		}
		if element.Options.QuietZone < 0 || element.Options.QuietZone > 16 {
			add(path+".options.quietZone", "QR_QUIET_ZONE_INVALID", "二维码静区必须在 0 到 16 之间")
		}
		if !validColor(element.Options.Foreground) || !validColor(element.Options.Background) {
			add(path+".options", "COLOR_INVALID", "二维码颜色必须为十六进制颜色")
		}
	case "image":
		validateValue(element.Value, path+".value", add)
		if element.ObjectFit != "contain" && element.ObjectFit != "cover" {
			add(path+".objectFit", "IMAGE_FIT_INVALID", "图片缩放方式无效")
		}
	case "rect":
		if !validColor(element.Fill) || (element.Stroke != "" && !validColor(element.Stroke)) {
			add(path, "COLOR_INVALID", "矩形颜色必须为十六进制颜色")
		}
	case "line":
		if !validColor(element.Color) || element.StrokeWidth <= 0 {
			add(path, "LINE_STYLE_INVALID", "线条颜色或宽度无效")
		}
	default:
		add(path+".type", "ELEMENT_TYPE_UNSUPPORTED", "不支持的标签元素类型")
	}
}

func validateValue(value *ValueSource, path string, add func(string, string, string)) {
	if value == nil {
		add(path, "VALUE_SOURCE_REQUIRED", "值来源不能为空")
		return
	}
	switch value.Type {
	case "static":
	case "field":
		if value.FieldID == "" {
			add(path+".fieldId", "FIELD_ID_REQUIRED", "字段绑定不能为空")
		}
	case "system":
		if !systemKeys[value.Key] {
			add(path+".key", "SYSTEM_KEY_INVALID", "系统字段无效")
		}
	case "expression":
		if value.Expression == "" {
			add(path+".expression", "EXPRESSION_REQUIRED", "表达式不能为空")
		}
	default:
		add(path+".type", "VALUE_SOURCE_TYPE_INVALID", "值来源类型无效")
	}
}

func validateTextStyle(style *TextStyle, path string, add func(string, string, string)) {
	if style == nil {
		add(path, "TEXT_STYLE_REQUIRED", "文本样式不能为空")
		return
	}
	if !supportedFonts[style.FontFamily] || !positiveFinite(style.FontSize) || style.FontWeight < 100 || style.FontWeight > 900 || !positiveFinite(style.LineHeight) {
		add(path, "TEXT_STYLE_INVALID", "字体、字号或字重无效")
	}
	if !validColor(style.Color) {
		add(path+".color", "COLOR_INVALID", "文本颜色必须为十六进制颜色")
	}
	if style.TextAlign != "left" && style.TextAlign != "center" && style.TextAlign != "right" {
		add(path+".textAlign", "TEXT_ALIGN_INVALID", "文本水平对齐方式无效")
	}
	if style.VerticalAlign != "top" && style.VerticalAlign != "middle" && style.VerticalAlign != "bottom" {
		add(path+".verticalAlign", "TEXT_ALIGN_INVALID", "文本垂直对齐方式无效")
	}
	if style.Overflow != "clip" && style.Overflow != "ellipsis" && style.Overflow != "wrap" {
		add(path+".overflow", "TEXT_OVERFLOW_INVALID", "文本溢出策略无效")
	}
}

func validColor(value string) bool      { return colorPattern.MatchString(value) }
func finite(value float64) bool         { return !math.IsNaN(value) && !math.IsInf(value, 0) }
func positiveFinite(value float64) bool { return value > 0 && finite(value) }
