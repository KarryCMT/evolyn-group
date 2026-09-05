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
	"strings"

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

// SnapshotFieldMapping 是发布版本中冻结的查询字段白名单。widgetName 是已独立
// 于 label 的稳定记录键；JSONB 与未来物理表模式仅在后端解析器层选择不同目标。
type SnapshotFieldMapping struct {
	WidgetName     string `json:"widgetName"`
	WidgetType     string `json:"widgetType"`
	JSONBKey       string `json:"jsonbKey"`
	PhysicalColumn string `json:"physicalColumn"`
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
	return buildSnapshotFieldsFromItems(itemsAny), nil
}

// buildSnapshotFieldsFromItems 从同一作用域的字段项构建快照视图。子表单行复用它，
// 但子项键仅在该子表单作用域内有效，绝不能混入顶层字段映射。
func buildSnapshotFieldsFromItems(itemsAny []any) map[string]snapshotField {
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
	return fields
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

// ExtractSnapshotFieldMappings 冻结顶层值字段的存储映射。布局与按钮没有记录值，
// 不得进入查询白名单；物理列只描述意图，DDL 仍只能由后端迁移服务执行。
func ExtractSnapshotFieldMappings(root map[string]any) []SnapshotFieldMapping {
	itemsAny, _ := documentItems(root)
	mappings := make([]SnapshotFieldMapping, 0, len(itemsAny))
	for _, rawItem := range itemsAny {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		widget, _ := item["widget"].(map[string]any)
		name, _ := widget["widgetName"].(string)
		widgetType, _ := widget["type"].(string)
		if name == "" || widgetType == "separator" || widgetType == "button" {
			continue
		}
		mappings = append(mappings, SnapshotFieldMapping{
			WidgetName:     name,
			WidgetType:     widgetType,
			JSONBKey:       name,
			PhysicalColumn: "f_" + strings.ToLower(strings.TrimPrefix(name, "_widget_")),
		})
	}
	return mappings
}

// ValidateRecordValues 校验并清洗提交值：返回可直接落库的 values 与按 widgetName
// 回填的错误集合。values 的键为 widgetName、值为 JSON 原文（逐字段原样收网）。
// v5 起按字段显隐规则动态求值：静态隐藏字段携值拒绝；规则隐藏字段按
// 「正式记录只保存有效可见字段的值」收网为 null（本入口供流程写回等系统
// 合并路径复用，客户端伪造隐藏在包装协议入口先行拒绝）。
func ValidateRecordValues(
	content map[string]any,
	rawValues map[string]json.RawMessage,
) (map[string]any, RecordFieldErrors) {
	// 写回路径无提交人身份上下文：includeCurrentMember 以匿名口径求值
	// （不注入任何成员），保证按快照+值的重算确定性。
	return validateRecordValues(content, rawValues, nil, "")
}

// validateRecordValues 终审核心；baseReadable 为 nil 时按快照静态 visible
// 作为规则条件源可达性基线，权限管线可注入「静态 ∧ 权限可见」的合成基线。
func validateRecordValues(
	content map[string]any,
	rawValues map[string]json.RawMessage,
	baseReadable func(name string) bool,
	currentMemberID string,
) (map[string]any, RecordFieldErrors) {
	fields, err := buildSnapshotFields(content)
	if err != nil {
		return nil, RecordFieldErrors{"": {"表单快照异常，请刷新后重试"}}
	}
	if baseReadable == nil {
		baseReadable = func(name string) bool {
			field, ok := fields[name]
			return ok && field.visible
		}
	}
	// 条件值取自合并后的候选值；条件源不可见时不读取其值（条件视为不成立）。
	ruleVisible := compileFieldShowRules(content).ruleFieldVisibility(
		baseReadable,
		func(name string) any { return decodeShowValue(rawValues[name]) },
		currentMemberID,
	)
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
		if visible, computed := ruleVisible[name]; computed && !visible {
			// 规则隐藏：跳过必填校验，既有值收网为 null（隐藏字段不落正式值）。
			cleaned[name] = nil
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
// 终审完成类型、范围与必填复核（v≤5 快照的存量入口；v6 起由
// value_resolution.go 的 ResolveSubmittedValues 按「有效可见性 → 不可见字段
// 赋值策略 → 值终审」管线接管）。所有数据字段必须显式携带 visible 且与
// 「静态可见 ∧ 显隐规则求值结果」一致；布局字段不得进入 values，隐藏字段
// 不得携带 data（v5 起浏览器可见性只是交互，服务端按提交值独立重算终审）。
func ValidateSubmittedRecordValues(
	content map[string]any,
	submitted map[string]model.SubmitFieldValue,
	currentMemberID string,
) (map[string]any, RecordFieldErrors) {
	fields, err := buildSnapshotFields(content)
	if err != nil {
		return nil, RecordFieldErrors{"": {"表单快照异常，请刷新后重试"}}
	}
	// 规则求值：条件源可达性 = 静态可见 ∧ 已算上游规则可见性；条件值取提交
	// data；currentMemberID 为提交人（includeCurrentMember 注入比较集合）。
	ruleVisible := compileFieldShowRules(content).ruleFieldVisibility(
		func(name string) bool {
			field, ok := fields[name]
			return ok && field.visible
		},
		func(name string) any {
			wrapped, ok := submitted[name]
			if !ok || len(wrapped.Data) == 0 {
				return nil
			}
			return decodeShowValue(json.RawMessage(wrapped.Data))
		},
		currentMemberID,
	)
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
		effective := field.visible
		if visible, computed := ruleVisible[name]; computed {
			effective = field.visible && visible
		}
		if *wrapped.Visible != effective {
			fieldErrors[name] = []string{"字段可见状态与发布快照不一致"}
			continue
		}
		if !effective {
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

	cleaned, valueErrors := validateRecordValues(content, rawValues, nil, currentMemberID)
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
	case "datetime", "radiogroup", "checkboxgroup", "combo", "combocheck", "user", "usergroup":
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
	case "user":
		return validateMemberValue(field, value)
	case "usergroup":
		return validateMemberGroupValue(field, value)
	case "subform":
		return validateSubformValue(field, value)
	default:
		// 白名单外控件不可发布，正常到不了这里；防御性拒绝。
		return []string{fmt.Sprintf("字段类型「%s」暂不支持提交", field.widgetType)}
	}
}

// validateSubformValue 终审子表单二维值：行必须为对象、每行仅可提交冻结子字段，
// 并复用基础字段校验。错误统一挂在子表单顶层键，便于既有字段错误回填协议消费。
func validateSubformValue(field snapshotField, value any) []string {
	rows, ok := value.([]any)
	if !ok {
		return []string{fmt.Sprintf("%s的值类型不正确", field.label)}
	}
	if len(rows) > 200 {
		return []string{fmt.Sprintf("%s不能超过 200 行", field.label)}
	}
	if min, ok := jsonInt(field.widget["minRowCount"]); ok && len(rows) < min {
		return []string{fmt.Sprintf("%s至少填写 %d 行", field.label, min)}
	}
	if max, ok := jsonInt(field.widget["maxRowCount"]); ok && len(rows) > max {
		return []string{fmt.Sprintf("%s不能超过 %d 行", field.label, max)}
	}
	childrenAny, ok := field.widget["items"].([]any)
	if !ok {
		// 发布快照已在发布期校验；写入路径仍防御性拒绝损坏快照。
		return []string{fmt.Sprintf("%s的字段配置异常", field.label)}
	}
	children := buildSnapshotFieldsFromItems(childrenAny)
	var errs []string
	for rowIndex, rawRow := range rows {
		row, ok := rawRow.(map[string]any)
		if !ok {
			errs = append(errs, fmt.Sprintf("%s第 %d 行的值类型不正确", field.label, rowIndex+1))
			continue
		}
		for key := range row {
			if _, known := children[key]; !known {
				errs = append(errs, fmt.Sprintf("%s第 %d 行提交了不存在的字段「%s」", field.label, rowIndex+1, key))
			}
		}
		for name, child := range children {
			if child.widgetType == "separator" || child.widgetType == "button" || !child.visible {
				continue
			}
			childValue, submitted := row[name]
			if !submitted || childValue == nil {
				if !child.allowBlank {
					errs = append(errs, fmt.Sprintf("%s第 %d 行：请%s%s", field.label, rowIndex+1, choosingVerb(child.widgetType), child.label))
				}
				continue
			}
			for _, childErr := range validateFieldValue(child, childValue) {
				errs = append(errs, fmt.Sprintf("%s第 %d 行：%s", field.label, rowIndex+1, childErr))
			}
		}
	}
	return errs
}

// validateMemberValue 只接受成员目录返回的稳定字符串 ID。成员有效性和字段范围由
// 选择器保证；提交侧仍须拒绝对象、数字等越过 UI 构造的值。
func validateMemberValue(field snapshotField, value any) []string {
	memberID, ok := value.(string)
	if !ok || strings.TrimSpace(memberID) == "" {
		return []string{fmt.Sprintf("%s的值类型不正确", field.label)}
	}
	return nil
}

// validateMemberGroupValue 保留选择顺序，但拒绝非字符串、重复成员及超出 Schema
// defaultValue 同口径的 200 人上限，避免异常的大数组进入记录 JSONB。
func validateMemberGroupValue(field snapshotField, value any) []string {
	members, ok := value.([]any)
	if !ok {
		return []string{fmt.Sprintf("%s的值类型不正确", field.label)}
	}
	if len(members) > 200 {
		return []string{fmt.Sprintf("%s最多选择 200 名成员", field.label)}
	}
	seen := make(map[string]struct{}, len(members))
	for _, rawMemberID := range members {
		memberID, ok := rawMemberID.(string)
		if !ok || strings.TrimSpace(memberID) == "" {
			return []string{fmt.Sprintf("%s的值类型不正确", field.label)}
		}
		if _, duplicated := seen[memberID]; duplicated {
			return []string{fmt.Sprintf("%s的值存在重复成员", field.label)}
		}
		seen[memberID] = struct{}{}
	}
	return nil
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
