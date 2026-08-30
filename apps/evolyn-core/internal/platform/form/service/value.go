// 提交值校验（P2 基础字段，字段字典 §4）。
//
// 服务端按发布快照终审（方案 §3.7）：未知键拒绝、隐藏/布局字段携值拒绝、
// 类型/范围/选项命中逐项复核；错误文案与浏览器侧 schema/codec 逐字一致，
// 按 widgetName 回填（FORM_RECORD_INVALID + fieldErrors）。
package service

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"

	"evolyn/internal/platform/form/model"
)

// RecordFieldError 按字段键回填的错误集合。
type RecordFieldErrors map[string][]string

var timeShapePatterns = map[string]*regexp.Regexp{
	"date":     regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`),
	"datetime": regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2}):(\d{2})$`),
	"month":    regexp.MustCompile(`^(\d{4})-(\d{2})$`),
	"time":     regexp.MustCompile(`^(\d{2}):(\d{2})$`),
}

// snapshotField 提交校验用的字段快照视图。
type snapshotField struct {
	widgetName string
	widgetType string
	label      string
	allowBlank bool
	visible    bool
	widget     map[string]any
}

// documentItems 从协议根文档中取 content.items（入参为 {content:{type,items}}）。
func documentItems(root map[string]any) ([]any, bool) {
	content, ok := root["content"].(map[string]any)
	if !ok {
		return nil, false
	}
	items, ok := content["items"].([]any)
	return items, ok
}

// buildSnapshotFields 从发布快照文档提取顶层字段视图（布局项同样收集以拒绝携值）。
func buildSnapshotFields(root map[string]any) (map[string]snapshotField, error) {
	itemsAny, ok := documentItems(root)
	if !ok {
		return nil, fmt.Errorf("快照缺少 items")
	}
	fields := make(map[string]snapshotField, len(itemsAny))
	for _, rawItem := range itemsAny {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		widget, _ := item["widget"].(map[string]any)
		widgetName, _ := widget["widgetName"].(string)
		widgetType, _ := widget["type"].(string)
		label, _ := item["label"].(string)
		allowBlank, _ := widget["allowBlank"].(bool)
		visible := true
		if v, ok := widget["visible"].(bool); ok {
			visible = v
		}
		if widgetName == "" {
			continue
		}
		fields[widgetName] = snapshotField{
			widgetName: widgetName,
			widgetType: widgetType,
			label:      label,
			allowBlank: allowBlank,
			visible:    visible,
			widget:     widget,
		}
	}
	return fields, nil
}

// ExtractSnapshotTopFieldKeys 提取顶层字段键有序数组（发布时写入 field_keys）。
func ExtractSnapshotTopFieldKeys(root map[string]any) []string {
	itemsAny, _ := documentItems(root)
	keys := make([]string, 0, len(itemsAny))
	for _, rawItem := range itemsAny {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		widget, _ := item["widget"].(map[string]any)
		if name, ok := widget["widgetName"].(string); ok && name != "" {
			keys = append(keys, name)
		}
	}
	return keys
}

// ValidateRecordValues 校验并清洗提交值：返回可直接落库的 values 与按 widgetName
// 回填的错误集合。values 的键为 widgetName、值为 JSON 原文（逐字段原样收网）。
func ValidateRecordValues(
	content map[string]any,
	rawValues map[string]json.RawMessage,
) (map[string]any, RecordFieldErrors) {
	fields, err := buildSnapshotFields(content)
	if err != nil {
		return nil, RecordFieldErrors{"": {"表单快照异常，请刷新后重试"}}
	}
	fieldErrors := RecordFieldErrors{}
	cleaned := make(map[string]any, len(fields))

	for name, field := range fields {
		raw, submitted := rawValues[name]
		if field.widgetType == "separator" || field.widgetType == "button" {
			// 布局项无值：不收集，携带任何值（含显式 null 之外的值）即拒绝。
			if submitted && !isNullJSON(raw) {
				fieldErrors[name] = []string{"分割线等布局字段不能携带值"}
			}
			continue
		}
		if !field.visible {
			if submitted && !isNullJSON(raw) {
				fieldErrors[name] = []string{"隐藏字段不能提交值"}
			}
			continue
		}
		if !submitted || isNullJSON(raw) {
			// 显式 null 与缺省同语义：未填写（字典 1.2 空值两态一致）。
			if !field.allowBlank {
				fieldErrors[name] = []string{
					fmt.Sprintf("请%s%s", choosingVerb(field.widgetType), field.label),
				}
				continue
			}
			cleaned[name] = nil
			continue
		}
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			fieldErrors[name] = []string{fmt.Sprintf("%s的值类型不正确", field.label)}
			continue
		}
		if errs := validateFieldValue(field, value); len(errs) > 0 {
			fieldErrors[name] = errs
			continue
		}
		cleaned[name] = value
	}

	for key := range rawValues {
		if _, known := fields[key]; !known {
			fieldErrors[key] = []string{"提交了表单中不存在的字段"}
		}
	}
	return cleaned, fieldErrors
}

// ValidateSubmittedRecordValues 校验新版字段包装协议并解包 data，再复用
// ValidateRecordValues 完成类型、范围与必填终审。所有数据字段必须显式携带
// visible；布局字段不得进入 values，隐藏字段不得携带 data。
func ValidateSubmittedRecordValues(
	content map[string]any,
	submitted map[string]model.SubmitFieldValue,
) (map[string]any, RecordFieldErrors) {
	fields, err := buildSnapshotFields(content)
	if err != nil {
		return nil, RecordFieldErrors{"": {"表单快照异常，请刷新后重试"}}
	}
	rawValues := make(map[string]json.RawMessage, len(submitted))
	fieldErrors := RecordFieldErrors{}
	for name, field := range fields {
		wrapped, exists := submitted[name]
		if field.widgetType == "separator" || field.widgetType == "button" {
			if exists {
				fieldErrors[name] = []string{"分割线等布局字段不能进入提交值"}
			}
			continue
		}
		if !exists || wrapped.Visible == nil {
			fieldErrors[name] = []string{"缺少字段可见状态"}
			continue
		}
		if *wrapped.Visible != field.visible {
			fieldErrors[name] = []string{"字段可见状态与发布快照不一致"}
			continue
		}
		if !field.visible {
			if len(wrapped.Data) > 0 && !isNullJSON(json.RawMessage(wrapped.Data)) {
				fieldErrors[name] = []string{"隐藏字段不能提交值"}
			}
			continue
		}
		if len(wrapped.Data) == 0 {
			rawValues[name] = json.RawMessage(`null`)
			continue
		}
		rawValues[name] = json.RawMessage(wrapped.Data)
	}
	for name := range submitted {
		if _, known := fields[name]; !known {
			fieldErrors[name] = []string{"提交了表单中不存在的字段"}
		}
	}

	cleaned, valueErrors := ValidateRecordValues(content, rawValues)
	for name, messages := range valueErrors {
		if _, exists := fieldErrors[name]; !exists {
			fieldErrors[name] = messages
		}
	}
	return cleaned, fieldErrors
}

func isNullJSON(raw json.RawMessage) bool {
	return string(raw) == "null"
}

func isEmptyValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return v == ""
	case []any:
		return len(v) == 0
	default:
		return false
	}
}

func choosingVerb(widgetType string) string {
	switch widgetType {
	case "datetime", "radiogroup", "checkboxgroup", "combo", "combocheck":
		return "选择"
	default:
		return "输入"
	}
}

// validateFieldValue 与前端 validateWidgetValue 逐字一致的基础字段值校验。
func validateFieldValue(field snapshotField, value any) []string {
	if isEmptyValue(value) {
		if field.allowBlank {
			return nil
		}
		return []string{fmt.Sprintf("请%s%s", choosingVerb(field.widgetType), field.label)}
	}
	switch field.widgetType {
	case "text", "textarea":
		return validateTextValue(field, value)
	case "number":
		return validateNumberValue(field, value)
	case "datetime":
		return validateDateTimeValue(field, value)
	case "radiogroup", "combo":
		return validateSingleOptionValue(field, value)
	case "checkboxgroup", "combocheck":
		return validateMultiOptionValue(field, value)
	default:
		// 白名单外控件不可发布，正常到不了这里；防御性拒绝。
		return []string{fmt.Sprintf("字段类型「%s」暂不支持提交", field.widgetType)}
	}
}

func validateTextValue(field snapshotField, value any) []string {
	text, ok := value.(string)
	if !ok {
		return []string{fmt.Sprintf("%s的值类型不正确", field.label)}
	}
	var errs []string
	minLength, minOK := jsonInt(field.widget["minLength"])
	maxLength, maxOK := jsonInt(field.widget["maxLength"])
	if minOK && len([]rune(text)) < minLength {
		errs = append(errs, fmt.Sprintf("%s最少输入 %d 个字符", field.label, minLength))
	}
	if maxOK && len([]rune(text)) > maxLength {
		errs = append(errs, fmt.Sprintf("%s不能超过 %d 个字符", field.label, maxLength))
	}
	if field.widget["format"] == "email" && !emailPattern.MatchString(text) {
		errs = append(errs, fmt.Sprintf("%s格式不正确", field.label))
	}
	return errs
}

func validateNumberValue(field snapshotField, value any) []string {
	num, ok := value.(float64)
	if !ok || math.IsNaN(num) || math.IsInf(num, 0) {
		return []string{fmt.Sprintf("%s的值类型不正确", field.label)}
	}
	var errs []string
	if min, ok := jsonFloat(field.widget["min"]); ok && num < min {
		errs = append(errs, fmt.Sprintf("%s不能小于 %s", field.label, jsNumber(min)))
	}
	if max, ok := jsonFloat(field.widget["max"]); ok && num > max {
		errs = append(errs, fmt.Sprintf("%s不能大于 %s", field.label, jsNumber(max)))
	}
	if precision, ok := jsonInt(field.widget["precision"]); ok {
		scaled := num * math.Pow(10, float64(precision))
		if math.Abs(scaled-math.Round(scaled)) > 1e-9 {
			errs = append(errs, fmt.Sprintf("%s最多支持 %d 位小数", field.label, precision))
		}
	}
	return errs
}

func validateDateTimeValue(field snapshotField, value any) []string {
	text, ok := value.(string)
	if !ok {
		return []string{fmt.Sprintf("%s的值类型不正确", field.label)}
	}
	format, _ := field.widget["format"].(string)
	if format == "" {
		format = "datetime"
	}
	if !isCanonicalDateTime(text, format) {
		return []string{fmt.Sprintf("%s的日期格式不正确", field.label)}
	}
	return nil
}

// isCanonicalDateTime 规范形状 + 真实日历校验（禁止依赖各引擎 Date 解析行为）。
func isCanonicalDateTime(value, format string) bool {
	match := timeShapePatterns[format].FindStringSubmatch(value)
	if match == nil {
		return false
	}
	switch format {
	case "date":
		year, month, day := atoi(match[1]), atoi(match[2]), atoi(match[3])
		return isRealDate(year, month, day)
	case "datetime":
		year, month, day := atoi(match[1]), atoi(match[2]), atoi(match[3])
		return isRealDate(year, month, day) &&
			atoi(match[4]) <= 23 && atoi(match[5]) <= 59 && atoi(match[6]) <= 59
	case "month":
		return atoi(match[2]) >= 1 && atoi(match[2]) <= 12
	case "time":
		return atoi(match[1]) <= 23 && atoi(match[2]) <= 59
	}
	return false
}

func isRealDate(year, month, day int) bool {
	if month < 1 || month > 12 || day < 1 {
		return false
	}
	leap := year%4 == 0 && (year%100 != 0 || year%400 == 0)
	daysInMonth := []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if leap {
		daysInMonth[1] = 29
	}
	return day <= daysInMonth[month-1]
}

func validateSingleOptionValue(field snapshotField, value any) []string {
	text, ok := value.(string)
	if !ok {
		return []string{fmt.Sprintf("%s的值类型不正确", field.label)}
	}
	if !optionValues(field.widget).Contains(text) {
		return []string{fmt.Sprintf("%s的值不在选项范围内", field.label)}
	}
	return nil
}

func validateMultiOptionValue(field snapshotField, value any) []string {
	arr, ok := value.([]any)
	if !ok {
		return []string{fmt.Sprintf("%s的值类型不正确", field.label)}
	}
	values := optionValues(field.widget)
	seen := map[string]bool{}
	for _, entry := range arr {
		text, isString := entry.(string)
		if !isString || !values.Contains(text) {
			return []string{fmt.Sprintf("%s的值不在选项范围内", field.label)}
		}
		if seen[text] {
			return []string{fmt.Sprintf("%s的值存在重复选项", field.label)}
		}
		seen[text] = true
	}
	return nil
}

type stringSet map[string]struct{}

func (s stringSet) Contains(v string) bool {
	_, ok := s[v]
	return ok
}

func optionValues(widget map[string]any) stringSet {
	set := stringSet{}
	options, _ := widget["options"].([]any)
	for _, rawOption := range options {
		if option, ok := rawOption.(map[string]any); ok {
			if value, ok := option["value"].(string); ok {
				set[value] = struct{}{}
			}
		}
	}
	return set
}

// jsonInt/jsonFloat 防御式读取（null/缺省=未启用）：JSON 解码值为 float64，
// 同时容忍 int/int64（测试桩与内存构造的快照）。
func jsonInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		if v != math.Trunc(v) || math.Abs(v) > 1<<31 {
			return 0, false
		}
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	default:
		return 0, false
	}
}

func jsonFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, !math.IsNaN(v) && !math.IsInf(v, 0)
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return math.NaN(), false
	}
}

// jsNumber 与 JS 数值转字符串口径一致（整数无小数点），保证文案与前端对拍。
func jsNumber(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func atoi(text string) int {
	value, _ := strconv.Atoi(text)
	return value
}
