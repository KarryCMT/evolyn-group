// 目标保存协议校验器（Go 侧镜像实现）。
//
// 与前端 packages/form/src/schema/{dictionary,validate}.ts 对同一 JSON 的校验结论
// 必须一致（P1 验收条件）：27 种 widget.type 的属性表、取值范围、未知键拒绝与
// 错误文案逐条镜像；差异只允许出现在问题排序（Go 按键名排序保证确定性）。
// 修改本表必须同步：字段字典文档、TS 字典、发布白名单与两侧测试。
package service

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"evolyn/internal/platform/form/model"
)

// SchemaIssue 协议校验问题（与前端 FormSchemaIssue 结构对齐，随 BizError data 出网）。
type SchemaIssue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// propKind 属性取值类别（与 TS WidgetPropKind 一致）。
type propKind string

const (
	kindBoolean     propKind = "boolean"
	kindString      propKind = "string"
	kindInteger     propKind = "integer"
	kindNumber      propKind = "number"
	kindEnum        propKind = "enum"
	kindStringArray propKind = "stringArray"
	kindOptions     propKind = "options"
	kindWidgetItems propKind = "widgetItems"
	kindLinkFilters propKind = "linkFilters"
	kindLinkSorts   propKind = "linkSorts"
	kindLinkMaps    propKind = "linkMappings"
	kindExpression  propKind = "expression"
	kindSnRule      propKind = "snRule"
	kindButtonAct   propKind = "buttonAction"
)

// propSpec 单个控件属性的声明式约束（字段字典逐条对应）。
type propSpec struct {
	kind     propKind
	required bool
	enum     []string
	maxLen   int
	min, max *float64
	maxItems int
}

func f64(v float64) *float64 { return &v }

type widgetSpec struct {
	label         string
	labelOptional bool
	props         map[string]propSpec
}

// 公共上限（与 TS FORM_PROTOCOL_LIMITS / WIDGET_OPTION_LIMITS 一致）。
const (
	protoMaxItems        = 500
	protoSubformMaxItems = 200
	protoMaxLayouts      = 50
	protoMaxTabs         = 20
	protoLabelMax        = 64
	protoDescMax         = 500
	protoNameMax         = 64
	protoPlaceholderMax  = 100
	protoOptionMin       = 1
	protoOptionMax       = 200
	protoOptionTextMax   = 100
)

var (
	widgetNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	emailPattern      = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
)

// publishableWidgetTypes P2 发布白名单（字段字典 §6；与 TS PUBLISHABLE_WIDGET_TYPES 一致）。
var publishableWidgetTypes = map[string]bool{
	"text": true, "textarea": true, "number": true, "datetime": true,
	"radiogroup": true, "checkboxgroup": true, "combo": true, "combocheck": true,
	"separator": true,
}

// subformAllowedTypes 子表单子项白名单（基础 9 类 + 组织四类，禁止嵌套 subform）。
var subformAllowedTypes = map[string]bool{
	"text": true, "textarea": true, "number": true, "datetime": true,
	"radiogroup": true, "checkboxgroup": true, "combo": true, "combocheck": true,
	"separator": true, "user": true, "usergroup": true, "dept": true, "deptgroup": true,
}

var textProps = map[string]propSpec{
	"placeholder":  {kind: kindString, maxLen: protoPlaceholderMax},
	"minLength":    {kind: kindInteger, min: f64(0), max: f64(1000)},
	"maxLength":    {kind: kindInteger, min: f64(1), max: f64(1000)},
	"format":       {kind: kindEnum, enum: []string{"", "email"}},
	"defaultValue": {kind: kindString, maxLen: 1000},
}

// widgetSpecs 控件字典（27 种；标签仅用于错误文案/内部参考）。
var widgetSpecs = map[string]widgetSpec{
	"text": {label: "单行文本", props: textProps},
	"textarea": {label: "多行文本", props: map[string]propSpec{
		"placeholder":  {kind: kindString, maxLen: protoPlaceholderMax},
		"minLength":    {kind: kindInteger, min: f64(0), max: f64(2000)},
		"maxLength":    {kind: kindInteger, min: f64(1), max: f64(2000)},
		"autoHeight":   {kind: kindBoolean},
		"defaultValue": {kind: kindString, maxLen: 2000},
	}},
	"number": {label: "数字", props: map[string]propSpec{
		"placeholder":  {kind: kindString, maxLen: protoPlaceholderMax},
		"min":          {kind: kindNumber},
		"max":          {kind: kindNumber},
		"precision":    {kind: kindInteger, min: f64(0), max: f64(8)},
		"defaultValue": {kind: kindNumber},
	}},
	"datetime": {label: "日期时间", props: map[string]propSpec{
		"format":       {kind: kindEnum, enum: []string{"date", "datetime", "month", "time"}},
		"placeholder":  {kind: kindString, maxLen: protoPlaceholderMax},
		"defaultValue": {kind: kindString, maxLen: 32},
	}},
	"radiogroup": {label: "单选组", props: map[string]propSpec{
		"options":      {kind: kindOptions, required: true},
		"layout":       {kind: kindEnum, enum: []string{"vertical", "horizontal"}},
		"defaultValue": {kind: kindString, maxLen: protoOptionTextMax},
	}},
	"checkboxgroup": {label: "复选组", props: map[string]propSpec{
		"options":      {kind: kindOptions, required: true},
		"layout":       {kind: kindEnum, enum: []string{"vertical", "horizontal"}},
		"defaultValue": {kind: kindStringArray, maxItems: protoOptionMax},
	}},
	"combo": {label: "下拉框", props: map[string]propSpec{
		"options":      {kind: kindOptions, required: true},
		"placeholder":  {kind: kindString, maxLen: protoPlaceholderMax},
		"filterable":   {kind: kindBoolean},
		"defaultValue": {kind: kindString, maxLen: protoOptionTextMax},
	}},
	"combocheck": {label: "下拉多选框", props: map[string]propSpec{
		"options":      {kind: kindOptions, required: true},
		"placeholder":  {kind: kindString, maxLen: protoPlaceholderMax},
		"defaultValue": {kind: kindStringArray, maxItems: protoOptionMax},
	}},
	"separator": {label: "分割线", labelOptional: true, props: map[string]propSpec{
		"style": {kind: kindEnum, enum: []string{"solid", "dashed"}},
	}},
	"user": {label: "成员选择", props: map[string]propSpec{
		"scope":        {kind: kindEnum, enum: []string{"tenant", "department"}},
		"departments":  {kind: kindStringArray, maxItems: 100},
		"defaultValue": {kind: kindString, maxLen: protoNameMax},
	}},
	"usergroup": {label: "成员多选", props: map[string]propSpec{
		"scope":        {kind: kindEnum, enum: []string{"tenant", "department"}},
		"departments":  {kind: kindStringArray, maxItems: 100},
		"defaultValue": {kind: kindStringArray, maxItems: 200},
	}},
	"dept": {label: "部门选择", props: map[string]propSpec{
		"includeChildren": {kind: kindBoolean},
		"defaultValue":    {kind: kindString, maxLen: protoNameMax},
	}},
	"deptgroup": {label: "部门多选", props: map[string]propSpec{
		"includeChildren": {kind: kindBoolean},
		"defaultValue":    {kind: kindStringArray, maxItems: 200},
	}},
	"image": {label: "图片", props: map[string]propSpec{
		"maxCount":  {kind: kindInteger, min: f64(1), max: f64(20)},
		"maxSizeMB": {kind: kindInteger, min: f64(1), max: f64(50)},
		"accept":    {kind: kindStringArray, maxItems: 20},
	}},
	"upload": {label: "附件", props: map[string]propSpec{
		"maxCount":  {kind: kindInteger, min: f64(1), max: f64(20)},
		"maxSizeMB": {kind: kindInteger, min: f64(1), max: f64(100)},
		"accept":    {kind: kindStringArray, maxItems: 20},
	}},
	"address": {label: "地址", props: map[string]propSpec{
		"level":             {kind: kindEnum, enum: []string{"province", "city", "district", "detail"}},
		"detailPlaceholder": {kind: kindString, maxLen: protoPlaceholderMax},
	}},
	"location": {label: "定位", props: map[string]propSpec{
		"radius": {kind: kindNumber, min: f64(10), max: f64(5000)},
	}},
	"signature": {label: "签名", props: map[string]propSpec{}},
	"phone": {label: "手机号", props: map[string]propSpec{
		"areaCode":     {kind: kindString, maxLen: 5},
		"verification": {kind: kindBoolean},
	}},
	"subform": {label: "子表单", props: map[string]propSpec{
		"items":       {kind: kindWidgetItems, required: true},
		"minRowCount": {kind: kindInteger, min: f64(0), max: f64(200)},
		"maxRowCount": {kind: kindInteger, min: f64(1), max: f64(200)},
	}},
	"linkquery": {label: "关联查询", props: map[string]propSpec{
		"targetForm": {kind: kindString, maxLen: protoNameMax},
		"multiple":   {kind: kindBoolean},
		"filters":    {kind: kindLinkFilters, maxItems: 20},
		"sorts":      {kind: kindLinkSorts, maxItems: 3},
	}},
	"linkfield": {label: "关联字段", props: map[string]propSpec{
		"targetForm":    {kind: kindString, maxLen: protoNameMax},
		"displayFields": {kind: kindStringArray, maxItems: 20},
	}},
	"lookup": {label: "数据联动", props: map[string]propSpec{
		"targetForm": {kind: kindString, maxLen: protoNameMax},
		"mappings":   {kind: kindLinkMaps, maxItems: 20},
	}},
	"aggregation": {label: "聚合计算", props: map[string]propSpec{
		"expression":  {kind: kindExpression},
		"precision":   {kind: kindInteger, min: f64(0), max: f64(8)},
		"displayMode": {kind: kindEnum, enum: []string{"plain", "percent"}},
	}},
	"sn": {label: "流水号", props: map[string]propSpec{
		"rule": {kind: kindSnRule},
	}},
	"richtext": {label: "富文本", props: map[string]propSpec{
		"toolbar": {kind: kindStringArray, maxItems: 30},
	}},
	"button": {label: "按钮", labelOptional: true, props: map[string]propSpec{
		"text":   {kind: kindString, maxLen: 32},
		"action": {kind: kindButtonAct},
	}},
}

// optionWidgetTypes 选项类控件（defaultValue 命中校验）。
var optionWidgetTypes = map[string]bool{
	"radiogroup": true, "checkboxgroup": true, "combo": true, "combocheck": true,
}

// ValidateFormSchema 校验目标协议文档原文；合法返回空 issues。
func ValidateFormSchema(raw []byte, versions ...int) []SchemaIssue {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return []SchemaIssue{{Path: "content", Message: "表单文档必须是 JSON 对象"}}
	}
	issues := make([]SchemaIssue, 0)
	protocolVersion := model.CurrentProtocolVersion
	if len(versions) > 0 {
		protocolVersion = versions[0]
	}
	validateRoot(root, protocolVersion, &issues)
	return issues
}

// ValidatePublishable 在结构校验之上叠加发布白名单，返回需要提示的 issues。
func ValidatePublishable(raw []byte, versions ...int) []SchemaIssue {
	base := ValidateFormSchema(raw, versions...)
	if len(base) > 0 {
		return base
	}
	var root map[string]any
	_ = json.Unmarshal(raw, &root)
	content := root["content"].(map[string]any)
	items := content["items"].([]any)
	issues := make([]SchemaIssue, 0)
	collectUnsupported(items, "content.items", &issues)
	return issues
}

func collectUnsupported(items []any, itemsPath string, issues *[]SchemaIssue) {
	for i, rawItem := range items {
		item := rawItem.(map[string]any)
		widget := item["widget"].(map[string]any)
		widgetType, _ := widget["type"].(string)
		if !publishableWidgetTypes[widgetType] {
			*issues = append(*issues, SchemaIssue{
				Path:    fmt.Sprintf("%s[%d].widget.type", itemsPath, i),
				Message: fmt.Sprintf("控件「%s」的运行能力尚未开放，暂不能发布", widgetType),
			})
		}
		if widgetType == "subform" {
			if children, ok := widget["items"].([]any); ok {
				collectUnsupported(children, fmt.Sprintf("%s[%d].widget.items", itemsPath, i), issues)
			}
		}
	}
}

// ---- 逐层校验（文案与 TS validate.ts 逐字一致） ----

func validateRoot(root any, protocolVersion int, issues *[]SchemaIssue) {
	obj, ok := root.(map[string]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: "content", Message: "表单文档必须是 JSON 对象"})
		return
	}
	if protocolVersion < 1 || protocolVersion > model.CurrentProtocolVersion {
		*issues = append(*issues, SchemaIssue{
			Path: "content", Message: fmt.Sprintf("不支持的表单协议版本：%d", protocolVersion),
		})
		return
	}
	rejectUnknownKeys(obj, []string{"content"}, "content", issues)
	content, ok := obj["content"].(map[string]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: "content", Message: "content 必须是 JSON 对象"})
		return
	}
	contentKeys := []string{"type", "items"}
	if protocolVersion >= 2 {
		contentKeys = append(contentKeys, "layout_fields", "field_layout")
	}
	if protocolVersion >= 3 {
		contentKeys = append(contentKeys, "layout")
	}
	rejectUnknownKeys(content, contentKeys, "content", issues)
	if content["type"] != "form" {
		*issues = append(*issues, SchemaIssue{Path: "content.type", Message: `content.type 必须固定为 "form"`})
	}
	if protocolVersion >= 3 {
		layout, ok := content["layout"].(string)
		if !ok || (layout != "normal" && layout != "grid-2" && layout != "grid-3" && layout != "grid-4") {
			*issues = append(*issues, SchemaIssue{
				Path: "content.layout", Message: "layout 必须是 normal / grid-2 / grid-3 / grid-4",
			})
		}
	}
	items, ok := content["items"].([]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: "content.items", Message: "content.items 必须是数组"})
		return
	}
	if len(items) > protoMaxItems {
		*issues = append(*issues, SchemaIssue{
			Path:    "content.items",
			Message: fmt.Sprintf("字段项数量不能超过 %d", protoMaxItems),
		})
		return
	}
	scopeNames := map[string]bool{}
	for i, rawItem := range items {
		validateItem(rawItem, fmt.Sprintf("content.items[%d]", i), scopeNames, issues)
	}
	if protocolVersion >= 2 {
		validateLayouts(content, scopeNames, issues)
	}
}

// validateLayouts 校验 v2 multitab 引用图。标签页只引用顶层字段键，字段类型不限，
// 因而未来 subform 开放运行能力时无需再次修改布局协议。
func validateLayouts(content map[string]any, topLevelNames map[string]bool, issues *[]SchemaIssue) {
	rawLayouts, layoutsOK := content["layout_fields"].([]any)
	if !layoutsOK {
		*issues = append(*issues, SchemaIssue{Path: "content.layout_fields", Message: "layout_fields 必须是数组"})
		return
	}
	rawTopLayout, topOK := content["field_layout"].([]any)
	if !topOK {
		*issues = append(*issues, SchemaIssue{Path: "content.field_layout", Message: "field_layout 必须是数组"})
		return
	}
	if len(rawLayouts) > protoMaxLayouts {
		*issues = append(*issues, SchemaIssue{
			Path: "content.layout_fields", Message: fmt.Sprintf("布局数量不能超过 %d", protoMaxLayouts),
		})
	}

	layoutNames := map[string]bool{}
	nodeNames := map[string]bool{}
	for name := range topLevelNames {
		nodeNames[name] = true
	}
	placements := map[string]string{}

	for layoutIndex, rawLayout := range rawLayouts {
		path := fmt.Sprintf("content.layout_fields[%d]", layoutIndex)
		layout, ok := rawLayout.(map[string]any)
		if !ok {
			*issues = append(*issues, SchemaIssue{Path: path, Message: "布局项必须是 JSON 对象"})
			continue
		}
		rejectUnknownKeys(layout, []string{"name", "type", "tabStyle", "container"}, path, issues)
		name := validateStableLayoutName(layout["name"], "_layout_", path+".name", issues)
		if name != "" {
			if nodeNames[name] {
				*issues = append(*issues, SchemaIssue{Path: path + ".name", Message: fmt.Sprintf("布局键「%s」重复", name)})
			} else {
				nodeNames[name] = true
				layoutNames[name] = true
			}
		}
		if layout["type"] != "multitab" {
			*issues = append(*issues, SchemaIssue{Path: path + ".type", Message: "当前协议仅支持 multitab 布局"})
		}
		if layout["tabStyle"] != "style1" && layout["tabStyle"] != "style2" {
			*issues = append(*issues, SchemaIssue{Path: path + ".tabStyle", Message: "tabStyle 必须是 style1 / style2"})
		}
		container, ok := layout["container"].([]any)
		if !ok {
			*issues = append(*issues, SchemaIssue{Path: path + ".container", Message: "container 必须是标签页数组"})
			continue
		}
		if len(container) < 1 || len(container) > protoMaxTabs {
			*issues = append(*issues, SchemaIssue{
				Path: path + ".container", Message: fmt.Sprintf("标签页数量必须在 1–%d 之间", protoMaxTabs),
			})
		}
		for tabIndex, rawTab := range container {
			validateTab(rawTab, fmt.Sprintf("%s.container[%d]", path, tabIndex), topLevelNames, nodeNames, placements, issues)
		}
	}

	for index, rawReference := range rawTopLayout {
		path := fmt.Sprintf("content.field_layout[%d]", index)
		reference, ok := rawReference.(string)
		if !ok || reference == "" {
			*issues = append(*issues, SchemaIssue{Path: path, Message: "顶层布局引用必须是非空字符串"})
			continue
		}
		if !topLevelNames[reference] && !layoutNames[reference] {
			*issues = append(*issues, SchemaIssue{Path: path, Message: fmt.Sprintf("顶层引用「%s」不存在", reference)})
			continue
		}
		registerPlacement(reference, path, placements, issues)
	}

	for _, name := range sortedBoolKeys(topLevelNames) {
		if _, placed := placements[name]; !placed {
			*issues = append(*issues, SchemaIssue{Path: "content.field_layout", Message: fmt.Sprintf("顶层字段「%s」未加入布局", name)})
		}
	}
	for _, name := range sortedBoolKeys(layoutNames) {
		if _, placed := placements[name]; !placed {
			*issues = append(*issues, SchemaIssue{Path: "content.field_layout", Message: fmt.Sprintf("布局「%s」未加入顶层布局", name)})
		}
	}
}

func validateTab(
	rawTab any, path string, topLevelNames, nodeNames map[string]bool,
	placements map[string]string, issues *[]SchemaIssue,
) {
	tab, ok := rawTab.(map[string]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "标签页必须是 JSON 对象"})
		return
	}
	rejectUnknownKeys(tab, []string{"name", "title", "type", "field_layout"}, path, issues)
	name := validateStableLayoutName(tab["name"], "_tab_", path+".name", issues)
	if name != "" {
		if nodeNames[name] {
			*issues = append(*issues, SchemaIssue{Path: path + ".name", Message: fmt.Sprintf("标签页键「%s」重复", name)})
		} else {
			nodeNames[name] = true
		}
	}
	if tab["type"] != "tab" {
		*issues = append(*issues, SchemaIssue{Path: path + ".type", Message: "标签页 type 必须固定为 tab"})
	}
	title, titleOK := tab["title"].(string)
	if !titleOK || strings.TrimSpace(title) == "" {
		*issues = append(*issues, SchemaIssue{Path: path + ".title", Message: "标签页标题不能为空"})
	} else if len([]rune(title)) > protoLabelMax {
		*issues = append(*issues, SchemaIssue{Path: path + ".title", Message: fmt.Sprintf("标签页标题不能超过 %d 个字符", protoLabelMax)})
	}
	references, ok := tab["field_layout"].([]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path + ".field_layout", Message: "标签页 field_layout 必须是数组"})
		return
	}
	for index, rawReference := range references {
		refPath := fmt.Sprintf("%s.field_layout[%d]", path, index)
		reference, ok := rawReference.(string)
		if !ok || reference == "" {
			*issues = append(*issues, SchemaIssue{Path: refPath, Message: "标签页字段引用必须是非空字符串"})
			continue
		}
		if !topLevelNames[reference] {
			*issues = append(*issues, SchemaIssue{Path: refPath, Message: fmt.Sprintf("字段引用「%s」不是顶层字段", reference)})
			continue
		}
		registerPlacement(reference, refPath, placements, issues)
	}
}

func validateStableLayoutName(value any, prefix, path string, issues *[]SchemaIssue) string {
	name, ok := value.(string)
	if !ok || !strings.HasPrefix(name, prefix) || !widgetNamePattern.MatchString(name) {
		label := "布局"
		if prefix == "_tab_" {
			label = "标签页"
		}
		*issues = append(*issues, SchemaIssue{Path: path, Message: label + "键格式不正确"})
		return ""
	}
	if len(name) > protoNameMax {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "稳定键不能超过 64 个字符"})
		return ""
	}
	return name
}

func registerPlacement(reference, path string, placements map[string]string, issues *[]SchemaIssue) {
	if previous, exists := placements[reference]; exists {
		*issues = append(*issues, SchemaIssue{
			Path: path, Message: fmt.Sprintf("引用「%s」重复，已在 %s 使用", reference, previous),
		})
		return
	}
	placements[reference] = path
}

func sortedBoolKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func validateItem(raw any, path string, scopeNames map[string]bool, issues *[]SchemaIssue) {
	item, ok := raw.(map[string]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "字段项必须是 JSON 对象"})
		return
	}
	rejectUnknownKeys(item, []string{"widget", "label", "description", "labelHidden", "lineWidth"}, path, issues)

	widget, _ := item["widget"].(map[string]any)
	widgetType, _ := widget["type"].(string)
	spec, specOK := widgetSpecs[widgetType]

	validateLabel(item["label"], widgetType, path+".label", issues)
	if desc, ok := item["description"].(string); ok {
		if len([]rune(desc)) > protoDescMax {
			*issues = append(*issues, SchemaIssue{
				Path:    path + ".description",
				Message: fmt.Sprintf("说明不能超过 %d 个字符", protoDescMax),
			})
		}
	} else {
		*issues = append(*issues, SchemaIssue{Path: path + ".description", Message: "description 必须是字符串（空串即「无」）"})
	}
	if _, ok := item["labelHidden"].(bool); !ok {
		*issues = append(*issues, SchemaIssue{Path: path + ".labelHidden", Message: "labelHidden 必须是布尔值"})
	}
	if lineWidth, ok := asInteger(item["lineWidth"]); ok {
		if lineWidth < 1 || lineWidth > 12 {
			*issues = append(*issues, SchemaIssue{
				Path:    path + ".lineWidth",
				Message: "lineWidth 必须在 1–12 之间",
			})
		}
	} else {
		*issues = append(*issues, SchemaIssue{Path: path + ".lineWidth", Message: "lineWidth 必须是整数"})
	}
	if specOK {
		validateWidget(item["widget"], path+".widget", spec, scopeNames, issues)
	} else {
		validateWidget(item["widget"], path+".widget", widgetSpec{}, scopeNames, issues)
	}
}

func validateLabel(label any, widgetType, path string, issues *[]SchemaIssue) {
	text, ok := label.(string)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "label 必须是字符串"})
		return
	}
	spec := widgetSpecs[widgetType]
	if !spec.labelOptional && strings.TrimSpace(text) == "" {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "label 不能为空"})
	}
	if len([]rune(text)) > protoLabelMax {
		*issues = append(*issues, SchemaIssue{
			Path:    path,
			Message: fmt.Sprintf("label 不能超过 %d 个字符", protoLabelMax),
		})
	}
}

func validateWidget(raw any, path string, spec widgetSpec, scopeNames map[string]bool, issues *[]SchemaIssue) {
	widget, ok := raw.(map[string]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "widget 必须是 JSON 对象"})
		return
	}
	widgetType, _ := widget["type"].(string)
	if _, known := widgetSpecs[widgetType]; !known {
		typeText := widgetType
		if _, isString := widget["type"].(string); !isString {
			typeText = "非字符串"
		}
		*issues = append(*issues, SchemaIssue{
			Path:    path + ".type",
			Message: "未知的控件类型：" + typeText,
		})
		return
	}

	allowed := []string{"type", "widgetName", "enable", "visible", "allowBlank"}
	for key := range spec.props {
		allowed = append(allowed, key)
	}
	rejectUnknownKeys(widget, allowed, path, issues)

	name, _ := widget["widgetName"].(string)
	if name == "" {
		*issues = append(*issues, SchemaIssue{Path: path + ".widgetName", Message: "widgetName 必须是非空字符串"})
	} else if len(name) > protoNameMax || !widgetNamePattern.MatchString(name) {
		*issues = append(*issues, SchemaIssue{
			Path:    path + ".widgetName",
			Message: "widgetName 必须是 1–64 位的字母/数字/下划线标识符（且以字母或下划线开头）",
		})
	} else if scopeNames[name] {
		*issues = append(*issues, SchemaIssue{
			Path:    path + ".widgetName",
			Message: fmt.Sprintf("字段键「%s」在当前作用域内重复", name),
		})
	} else {
		scopeNames[name] = true
	}

	for _, key := range []string{"enable", "visible", "allowBlank"} {
		if _, ok := widget[key].(bool); !ok {
			*issues = append(*issues, SchemaIssue{
				Path:    path + "." + key,
				Message: key + " 必须是布尔值（不允许 null/缺省）",
			})
		}
	}

	for _, key := range sortedKeys(spec.props) {
		propSpec := spec.props[key]
		value, present := widget[key]
		if !present {
			if propSpec.required {
				*issues = append(*issues, SchemaIssue{
					Path:    path + "." + key,
					Message: "缺少必填属性 " + key,
				})
			}
			continue
		}
		validateWidgetProp(value, propSpec, path+"."+key, key, issues)
	}

	validateWidgetCrossRules(widget, widgetType, path, issues)
}

func validateWidgetProp(value any, spec propSpec, path, key string, issues *[]SchemaIssue) {
	switch spec.kind {
	case kindBoolean:
		if _, ok := value.(bool); !ok {
			*issues = append(*issues, SchemaIssue{Path: path, Message: key + " 必须是布尔值"})
		}
	case kindString:
		text, ok := value.(string)
		if !ok {
			*issues = append(*issues, SchemaIssue{Path: path, Message: key + " 必须是字符串"})
		} else if spec.maxLen > 0 && len([]rune(text)) > spec.maxLen {
			*issues = append(*issues, SchemaIssue{
				Path:    path,
				Message: fmt.Sprintf("%s 不能超过 %d 个字符", key, spec.maxLen),
			})
		}
	case kindInteger:
		if value == nil {
			return // 未启用语义（与 TS 一致：integer 允许 null）
		}
		if num, ok := asInteger(value); ok {
			if !inRange(float64(num), spec) {
				*issues = append(*issues, SchemaIssue{
					Path:    path,
					Message: fmt.Sprintf("%s 不在允许范围 %s 内", key, rangeText(spec)),
				})
			}
		} else {
			*issues = append(*issues, SchemaIssue{Path: path, Message: key + " 必须是整数（null 表示未启用）"})
		}
	case kindNumber:
		if value == nil {
			return
		}
		if num, ok := value.(float64); ok && !math.IsNaN(num) && !math.IsInf(num, 0) {
			if !inRange(num, spec) {
				*issues = append(*issues, SchemaIssue{
					Path:    path,
					Message: fmt.Sprintf("%s 不在允许范围 %s 内", key, rangeText(spec)),
				})
			}
		} else {
			*issues = append(*issues, SchemaIssue{Path: path, Message: key + " 必须是有限数值（null 表示未启用）"})
		}
	case kindEnum:
		text, ok := value.(string)
		if !ok || !containsString(spec.enum, text) {
			*issues = append(*issues, SchemaIssue{
				Path:    path,
				Message: fmt.Sprintf("%s 必须是以下枚举值之一：%s", key, strings.Join(spec.enum, " / ")),
			})
		}
	case kindStringArray:
		arr, ok := value.([]any)
		if !ok {
			*issues = append(*issues, SchemaIssue{Path: path, Message: key + " 必须是字符串数组"})
			return
		}
		if spec.maxItems > 0 && len(arr) > spec.maxItems {
			*issues = append(*issues, SchemaIssue{
				Path:    path,
				Message: fmt.Sprintf("%s 条目数不能超过 %d", key, spec.maxItems),
			})
		}
		for i, entry := range arr {
			if text, ok := entry.(string); !ok || text == "" {
				*issues = append(*issues, SchemaIssue{
					Path:    fmt.Sprintf("%s[%d]", path, i),
					Message: "数组条目必须是非空字符串",
				})
			}
		}
	case kindOptions:
		validateOptions(value, path, issues)
	case kindWidgetItems:
		validateSubformItems(value, path, issues)
	case kindLinkFilters:
		validateLinkFilters(value, path, issues)
	case kindLinkSorts:
		validateLinkSorts(value, path, issues)
	case kindLinkMaps:
		validateLinkMappings(value, path, issues)
	case kindExpression:
		validateAggregationExpression(value, path, issues)
	case kindSnRule:
		validateSnRule(value, path, issues)
	case kindButtonAct:
		validateButtonAction(value, path, issues)
	}
}

func validateOptions(value any, path string, issues *[]SchemaIssue) {
	arr, ok := value.([]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "options 必须是数组"})
		return
	}
	if len(arr) < protoOptionMin || len(arr) > protoOptionMax {
		*issues = append(*issues, SchemaIssue{
			Path:    path,
			Message: fmt.Sprintf("选项数量必须在 %d–%d 之间", protoOptionMin, protoOptionMax),
		})
	}
	seenValues := map[string]bool{}
	for i, rawOption := range arr {
		optionPath := fmt.Sprintf("%s[%d]", path, i)
		option, ok := rawOption.(map[string]any)
		if !ok {
			*issues = append(*issues, SchemaIssue{Path: optionPath, Message: "选项必须是 {label, value} 对象"})
			continue
		}
		rejectUnknownKeys(option, []string{"label", "value"}, optionPath, issues)
		for _, key := range []string{"label", "value"} {
			text, isString := option[key].(string)
			if !isString || text == "" {
				*issues = append(*issues, SchemaIssue{
					Path:    optionPath + "." + key,
					Message: "选项 " + key + " 必须是非空字符串",
				})
			} else if len([]rune(text)) > protoOptionTextMax {
				*issues = append(*issues, SchemaIssue{
					Path:    optionPath + "." + key,
					Message: fmt.Sprintf("选项 %s 不能超过 %d 个字符", key, protoOptionTextMax),
				})
			}
		}
		if value, isString := option["value"].(string); isString {
			if seenValues[value] {
				*issues = append(*issues, SchemaIssue{
					Path:    optionPath + ".value",
					Message: fmt.Sprintf("选项值「%s」重复", value),
				})
			}
			seenValues[value] = true
		}
	}
}

func validateSubformItems(value any, path string, issues *[]SchemaIssue) {
	arr, ok := value.([]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "子表单 items 必须是数组"})
		return
	}
	if len(arr) > protoSubformMaxItems {
		*issues = append(*issues, SchemaIssue{
			Path:    path,
			Message: fmt.Sprintf("子表单字段数不能超过 %d", protoSubformMaxItems),
		})
	}
	scopeNames := map[string]bool{}
	for i, child := range arr {
		childPath := fmt.Sprintf("%s[%d]", path, i)
		widget, _ := child.(map[string]any)["widget"].(map[string]any)
		childType, _ := widget["type"].(string)
		if childType != "" && !subformAllowedTypes[childType] {
			*issues = append(*issues, SchemaIssue{
				Path:    childPath + ".widget.type",
				Message: fmt.Sprintf("子表单内不允许使用控件「%s」", childType),
			})
			continue
		}
		validateItem(child, childPath, scopeNames, issues)
	}
}

func validateLinkFilters(value any, path string, issues *[]SchemaIssue) {
	arr, ok := value.([]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "filters 必须是数组"})
		return
	}
	ops := []string{"eq", "ne", "gt", "lt", "ge", "le", "contains"}
	for i, rawFilter := range arr {
		filterPath := fmt.Sprintf("%s[%d]", path, i)
		filter, ok := rawFilter.(map[string]any)
		if !ok {
			*issues = append(*issues, SchemaIssue{Path: filterPath, Message: "过滤条件必须是 {field, op, value} 对象"})
			continue
		}
		rejectUnknownKeys(filter, []string{"field", "op", "value"}, filterPath, issues)
		if field, ok := filter["field"].(string); !ok || field == "" {
			*issues = append(*issues, SchemaIssue{Path: filterPath + ".field", Message: "过滤条件 field 必须是非空字符串"})
		}
		if op, ok := filter["op"].(string); !ok || !containsString(ops, op) {
			*issues = append(*issues, SchemaIssue{
				Path:    filterPath + ".op",
				Message: "过滤条件 op 必须是以下枚举值之一：" + strings.Join(ops, " / "),
			})
		}
	}
}

func validateLinkSorts(value any, path string, issues *[]SchemaIssue) {
	arr, ok := value.([]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "sorts 必须是数组"})
		return
	}
	for i, rawSort := range arr {
		sortPath := fmt.Sprintf("%s[%d]", path, i)
		sortItem, ok := rawSort.(map[string]any)
		if !ok {
			*issues = append(*issues, SchemaIssue{Path: sortPath, Message: "排序项必须是 {field, order} 对象"})
			continue
		}
		rejectUnknownKeys(sortItem, []string{"field", "order"}, sortPath, issues)
		if field, ok := sortItem["field"].(string); !ok || field == "" {
			*issues = append(*issues, SchemaIssue{Path: sortPath + ".field", Message: "排序项 field 必须是非空字符串"})
		}
		if order := sortItem["order"]; order != "asc" && order != "desc" {
			*issues = append(*issues, SchemaIssue{Path: sortPath + ".order", Message: "排序项 order 必须是 asc / desc"})
		}
	}
}

func validateLinkMappings(value any, path string, issues *[]SchemaIssue) {
	arr, ok := value.([]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "mappings 必须是数组"})
		return
	}
	for i, rawMapping := range arr {
		mappingPath := fmt.Sprintf("%s[%d]", path, i)
		mapping, ok := rawMapping.(map[string]any)
		if !ok {
			*issues = append(*issues, SchemaIssue{Path: mappingPath, Message: "映射项必须是 {source, target} 对象"})
			continue
		}
		rejectUnknownKeys(mapping, []string{"source", "target"}, mappingPath, issues)
		for _, key := range []string{"source", "target"} {
			if text, ok := mapping[key].(string); !ok || text == "" {
				*issues = append(*issues, SchemaIssue{
					Path:    mappingPath + "." + key,
					Message: "映射项 " + key + " 必须是非空字符串",
				})
			}
		}
	}
}

func validateAggregationExpression(value any, path string, issues *[]SchemaIssue) {
	if value == nil {
		return
	}
	expression, ok := value.(map[string]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "expression 必须是 {op, source, field?} 对象或 null"})
		return
	}
	rejectUnknownKeys(expression, []string{"op", "source", "field"}, path, issues)
	ops := []string{"sum", "avg", "count", "min", "max"}
	if op, ok := expression["op"].(string); !ok || !containsString(ops, op) {
		*issues = append(*issues, SchemaIssue{
			Path:    path + ".op",
			Message: "聚合 op 必须是以下枚举值之一：" + strings.Join(ops, " / "),
		})
	}
	if source, ok := expression["source"].(string); !ok || source == "" {
		*issues = append(*issues, SchemaIssue{Path: path + ".source", Message: "聚合 source 必须是非空字符串（源字段键）"})
	}
	if field, present := expression["field"]; present {
		if text, ok := field.(string); !ok || text == "" {
			*issues = append(*issues, SchemaIssue{Path: path + ".field", Message: "聚合 field 必须是非空字符串或省略"})
		}
	}
	if expression["op"] == "count" {
		if _, present := expression["field"]; present {
			*issues = append(*issues, SchemaIssue{Path: path + ".field", Message: "op=count 时不允许携带 field"})
		}
	}
}

func validateSnRule(value any, path string, issues *[]SchemaIssue) {
	rule, ok := value.(map[string]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "rule 必须是对象"})
		return
	}
	rejectUnknownKeys(rule, []string{"prefix", "dateFmt", "seqLength", "resetCycle"}, path, issues)
	if prefix, present := rule["prefix"]; present {
		text, isString := prefix.(string)
		if !isString || len([]rune(text)) > 32 {
			*issues = append(*issues, SchemaIssue{Path: path + ".prefix", Message: "流水号前缀必须是 ≤32 字符的字符串"})
		}
	}
	if dateFmt, present := rule["dateFmt"]; present {
		text, isString := dateFmt.(string)
		if !isString || !containsString([]string{"none", "yyyyMM", "yyyyMMdd"}, text) {
			*issues = append(*issues, SchemaIssue{
				Path:    path + ".dateFmt",
				Message: "dateFmt 必须是以下枚举值之一：none / yyyyMM / yyyyMMdd",
			})
		}
	}
	if rawLength, present := rule["seqLength"]; present {
		if length, ok := asInteger(rawLength); ok {
			if length < 3 || length > 8 {
				*issues = append(*issues, SchemaIssue{Path: path + ".seqLength", Message: "seqLength 必须是 3–8 的整数"})
			}
		} else {
			*issues = append(*issues, SchemaIssue{Path: path + ".seqLength", Message: "seqLength 必须是 3–8 的整数"})
		}
	}
	if cycle, present := rule["resetCycle"]; present {
		text, isString := cycle.(string)
		if !isString || !containsString([]string{"none", "daily", "monthly", "yearly"}, text) {
			*issues = append(*issues, SchemaIssue{
				Path:    path + ".resetCycle",
				Message: "resetCycle 必须是以下枚举值之一：none / daily / monthly / yearly",
			})
		}
	}
}

func validateButtonAction(value any, path string, issues *[]SchemaIssue) {
	action, ok := value.(map[string]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "action 必须是 {type} 对象"})
		return
	}
	rejectUnknownKeys(action, []string{"type"}, path, issues)
	actionType, ok := action["type"].(string)
	if !ok || !containsString([]string{"none", "submit"}, actionType) {
		*issues = append(*issues, SchemaIssue{
			Path:    path + ".type",
			Message: "action.type 必须是以下枚举值之一：none / submit",
		})
	}
}

// validateWidgetCrossRules 类型间交叉约束（min ≤ max 系列与 defaultValue 命中）。
func validateWidgetCrossRules(widget map[string]any, widgetType, path string, issues *[]SchemaIssue) {
	minMaxOf := func(minKey, maxKey string) {
		min, minOK := widget[minKey].(float64)
		max, maxOK := widget[maxKey].(float64)
		if minOK && maxOK && min > max {
			*issues = append(*issues, SchemaIssue{
				Path:    path + "." + maxKey,
				Message: maxKey + " 不能小于 " + minKey,
			})
		}
	}
	switch widgetType {
	case "text", "textarea":
		minMaxOf("minLength", "maxLength")
	case "number":
		minMaxOf("min", "max")
		if def, ok := widget["defaultValue"].(float64); ok {
			if min, ok := widget["min"].(float64); ok && def < min {
				*issues = append(*issues, SchemaIssue{Path: path + ".defaultValue", Message: "defaultValue 不能小于 min"})
			}
			if max, ok := widget["max"].(float64); ok && def > max {
				*issues = append(*issues, SchemaIssue{Path: path + ".defaultValue", Message: "defaultValue 不能大于 max"})
			}
		}
	case "user", "usergroup":
		if widget["scope"] == "department" {
			if deps, ok := widget["departments"].([]any); !ok || len(deps) == 0 {
				*issues = append(*issues, SchemaIssue{
					Path:    path + ".departments",
					Message: "scope=department 时 departments 必须是非空数组",
				})
			}
		}
	case "subform":
		minMaxOf("minRowCount", "maxRowCount")
	}

	if optionWidgetTypes[widgetType] {
		def, present := widget["defaultValue"]
		if !present || def == nil {
			return
		}
		optionValues := map[string]bool{}
		if options, ok := widget["options"].([]any); ok {
			for _, rawOption := range options {
				if option, ok := rawOption.(map[string]any); ok {
					if value, ok := option["value"].(string); ok {
						optionValues[value] = true
					}
				}
			}
		}
		entries := []any{def}
		if arr, ok := def.([]any); ok {
			entries = arr
		}
		for _, entry := range entries {
			if text, ok := entry.(string); !ok || !optionValues[text] {
				*issues = append(*issues, SchemaIssue{
					Path:    path + ".defaultValue",
					Message: "defaultValue 必须是选项 value 之一",
				})
				break
			}
		}
	}
}

// ---- 助手 ----

func rejectUnknownKeys(target map[string]any, allowed []string, path string, issues *[]SchemaIssue) {
	for _, key := range sortedMapKeys(target) {
		if !containsString(allowed, key) {
			prefix := ""
			if path != "" {
				prefix = path + "."
			}
			*issues = append(*issues, SchemaIssue{
				Path:    prefix + key,
				Message: "未知属性「" + key + "」",
			})
		}
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedMapKeys(m map[string]any) []string {
	return sortedKeys(m)
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

// asInteger JSON 数字（float64）收敛为整型；非数字或非整数返回 false
func asInteger(value any) (int, bool) {
	num, ok := value.(float64)
	if !ok || num != math.Trunc(num) || math.Abs(num) > 1<<31 {
		return 0, false
	}
	return int(num), true
}

func inRange(value float64, spec propSpec) bool {
	if spec.min != nil && value < *spec.min {
		return false
	}
	if spec.max != nil && value > *spec.max {
		return false
	}
	return true
}

func rangeText(spec propSpec) string {
	min, max := "-∞", "+∞"
	if spec.min != nil {
		min = formatNumber(*spec.min)
	}
	if spec.max != nil {
		max = formatNumber(*spec.max)
	}
	return min + "–" + max
}

// formatNumber 与 JS 数值转字符串口径一致（整数无小数点）
func formatNumber(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return fmt.Sprintf("%d", int64(v))
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", v), "0"), ".")
}
