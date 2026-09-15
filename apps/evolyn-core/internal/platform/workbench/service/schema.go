// Package service 自定义工作台域服务：文档校验（前端镜像）与成员工作台存取
package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// ---- 工作台文档服务端校验器 ----
// 与前端 @evolyn.do/dashboard schema/lifecycle.ts 的 validateDashboardSchema/
// migrateDashboardSchema 逐字镜像：根版本号、widgets 数组结构、卡片 id 唯一、
// 卡片类型白名单、网格坐标与 min/max 交叉约束、config 纯 JSON。两侧语义
// 漂移以测试对齐，服务端是保存终审边界（浏览器侧校验不转移后端权威职责）。
// 另追加服务端护栏：卡片数量与文档体积上限、文本字段长度上限。

// WorkbenchSchemaVersion 当前协议版本（镜像 DASHBOARD_SCHEMA_VERSION）
const WorkbenchSchemaVersion = 1

// workbenchWidgetTypes 可持久化卡片类型白名单（镜像前端 types/dashboard.ts
// 的 DashboardWidgetType 字典；新增卡片类型两侧同步维护）
var workbenchWidgetTypes = map[string]struct{}{
	"onboarding": {},
	"greeting":   {},
	"shortcut":   {},
	"todo":       {},
	"favorites":  {},
	"apps":       {},
	"charts":     {},
}

// 服务端护栏：卡片数量与文档体积上限（单文档只存布局与轻配置，重数据由
// 各卡片运行时按引用拉取，正常远达不到该量级）
const (
	maxWorkbenchWidgets  = 100
	maxWorkbenchDocBytes = 256 * 1024
	maxWorkbenchTextLen  = 200
)

// SchemaIssue 结构化校验问题（镜像前端 DashboardSchemaValidationIssue）
type SchemaIssue struct {
	Code    string
	Path    string
	Message string
}

func (i *SchemaIssue) Error() string {
	return fmt.Sprintf("%s: %s (%s)", i.Code, i.Message, i.Path)
}

// ValidateWorkbenchDocument 迁移并校验工作台文档（任意历史 JSON 输入）。
// 校验通过时返回仅含协议字段的规范化 JSON（剔除未知键，防未知结构入库）；
// 无效时返回首个 issue。迁移语义镜像前端：缺 version 的早期文档补 v1。
func ValidateWorkbenchDocument(raw []byte) ([]byte, *SchemaIssue) {
	if len(raw) > maxWorkbenchDocBytes {
		return nil, &SchemaIssue{"document-too-large", "$", "工作台配置体积超出上限。"}
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	var input any
	if err := dec.Decode(&input); err != nil {
		return nil, &SchemaIssue{"invalid-root", "$", "工作台配置必须是对象。"}
	}

	root, ok := input.(map[string]any)
	if !ok {
		return nil, &SchemaIssue{"invalid-root", "$", "工作台配置必须是对象。"}
	}
	// 迁移：无 version 的早期文档与 v1 布局字段一致，补齐版本号（前端同语义）
	if value, present := root["version"]; present {
		version, ok := value.(float64)
		if !ok || version != WorkbenchSchemaVersion {
			return nil, &SchemaIssue{"unsupported-version", "$.version",
				fmt.Sprintf("暂不支持工作台配置版本 %v。", value)}
		}
	}

	widgetsRaw, ok := root["widgets"].([]any)
	if !ok {
		return nil, &SchemaIssue{"invalid-widgets", "$.widgets", "widgets 必须是数组。"}
	}
	if len(widgetsRaw) > maxWorkbenchWidgets {
		return nil, &SchemaIssue{"too-many-widgets", "$.widgets", "工作台卡片数量超出上限。"}
	}

	ids := make(map[string]struct{}, len(widgetsRaw))
	widgets := make([]map[string]any, 0, len(widgetsRaw))
	for index, value := range widgetsRaw {
		widget, issue := validateWorkbenchWidget(value, index, ids)
		if issue != nil {
			return nil, issue
		}
		widgets = append(widgets, widget)
	}

	normalized, err := json.Marshal(map[string]any{
		"version": WorkbenchSchemaVersion,
		"widgets": widgets,
	})
	if err != nil {
		return nil, &SchemaIssue{"invalid-root", "$", "工作台配置序列化失败。"}
	}
	return normalized, nil
}

// validateWorkbenchWidget 校验单张卡片并产出仅含协议字段的规范化输出；
// ids 跨卡片累积做唯一性判定。布局读取/交叉约束/可选标志分段校验
func validateWorkbenchWidget(value any, index int, ids map[string]struct{}) (map[string]any, *SchemaIssue) {
	path := fmt.Sprintf("$.widgets[%d]", index)
	widget, ok := value.(map[string]any)
	if !ok {
		return nil, &SchemaIssue{"invalid-widget", path, "卡片必须是对象。"}
	}

	id, issue := readText(widget["id"], path+".id")
	if issue != nil {
		return nil, issue
	}
	widgetType, issue := readText(widget["type"], path+".type")
	if issue != nil {
		return nil, issue
	}
	title, issue := readText(widget["title"], path+".title")
	if issue != nil {
		return nil, issue
	}

	layout, issue := readWidgetLayout(widget, path)
	if issue != nil {
		return nil, issue
	}
	if issue := checkWidgetLayoutConstraints(path, layout); issue != nil {
		return nil, issue
	}

	flags, issue := readWidgetOptionalFields(widget, path)
	if issue != nil {
		return nil, issue
	}

	if _, duplicated := ids[id]; duplicated {
		return nil, &SchemaIssue{"duplicate-widget-id", path + ".id", "卡片 id 不能重复。"}
	}
	ids[id] = struct{}{}
	if _, known := workbenchWidgetTypes[widgetType]; !known {
		return nil, &SchemaIssue{"unknown-widget-type", path + ".type",
			fmt.Sprintf("不支持卡片类型 %s。", widgetType)}
	}

	// 仅保留协议字段输出（镜像前端 validateWidget 的白名单重建语义）
	normalized := map[string]any{
		"id":    id,
		"type":  widgetType,
		"title": title,
		"x":     layout.x,
		"y":     layout.y,
		"w":     layout.w,
		"h":     layout.h,
	}
	if layout.minW != nil {
		normalized["minW"] = *layout.minW
	}
	if layout.minH != nil {
		normalized["minH"] = *layout.minH
	}
	if layout.maxW != nil {
		normalized["maxW"] = *layout.maxW
	}
	if layout.maxH != nil {
		normalized["maxH"] = *layout.maxH
	}
	if flags.noMove != nil {
		normalized["noMove"] = flags.noMove
	}
	if flags.noResize != nil {
		normalized["noResize"] = flags.noResize
	}
	if flags.presetKey != nil {
		normalized["presetKey"] = flags.presetKey
	}
	if flags.config != nil {
		normalized["config"] = flags.config
	}
	return normalized, nil
}

// widgetLayout 卡片网格坐标与可选尺寸约束（可选字段缺省为 nil）
type widgetLayout struct {
	x, y, w, h             float64
	minW, minH, maxW, maxH *float64
}

// readWidgetLayout 读取 8 个布局数值字段（必填坐标/尺寸 + 可选约束）
func readWidgetLayout(widget map[string]any, path string) (widgetLayout, *SchemaIssue) {
	var layout widgetLayout
	for _, field := range []struct {
		key      string
		positive bool
		target   *float64
	}{
		{"x", false, &layout.x},
		{"y", false, &layout.y},
		{"w", true, &layout.w},
		{"h", true, &layout.h},
	} {
		value, issue := readInteger(widget[field.key], path+"."+field.key, field.positive)
		if issue != nil {
			return layout, issue
		}
		*field.target = value
	}
	for _, field := range []struct {
		key    string
		target **float64
	}{
		{"minW", &layout.minW},
		{"minH", &layout.minH},
		{"maxW", &layout.maxW},
		{"maxH", &layout.maxH},
	} {
		value, issue := readOptionalInteger(widget[field.key], path+"."+field.key)
		if issue != nil {
			return layout, issue
		}
		*field.target = value
	}
	return layout, nil
}

// checkWidgetLayoutConstraints min/max 与实际宽高的交叉约束（镜像前端
// validateWidget 的六项判定，返回首个命中的 issue）
func checkWidgetLayoutConstraints(path string, layout widgetLayout) *SchemaIssue {
	for _, check := range []struct {
		broken  bool
		code    string
		at      string
		message string
	}{
		{layout.minW != nil && layout.maxW != nil && *layout.minW > *layout.maxW,
			"invalid-width-range", path, "minW 不能大于 maxW。"},
		{layout.minH != nil && layout.maxH != nil && *layout.minH > *layout.maxH,
			"invalid-height-range", path, "minH 不能大于 maxH。"},
		{layout.minW != nil && layout.w < *layout.minW,
			"width-below-minimum", path + ".w", "w 不能小于 minW。"},
		{layout.maxW != nil && layout.w > *layout.maxW,
			"width-above-maximum", path + ".w", "w 不能大于 maxW。"},
		{layout.minH != nil && layout.h < *layout.minH,
			"height-below-minimum", path + ".h", "h 不能小于 minH。"},
		{layout.maxH != nil && layout.h > *layout.maxH,
			"height-above-maximum", path + ".h", "h 不能大于 maxH。"},
	} {
		if check.broken {
			return &SchemaIssue{check.code, check.at, check.message}
		}
	}
	return nil
}

// widgetOptionalFields 卡片可选字段（布尔锁定/预设键/业务配置）
type widgetOptionalFields struct {
	noMove, noResize, presetKey any
	config                      map[string]any
}

// readWidgetOptionalFields 读取可选标志位、presetKey 与 config
func readWidgetOptionalFields(widget map[string]any, path string) (*widgetOptionalFields, *SchemaIssue) {
	fields := new(widgetOptionalFields)
	switch v := widget["noMove"].(type) {
	case nil:
	case bool:
		fields.noMove = v
	default:
		return nil, &SchemaIssue{"invalid-no-move", path + ".noMove", "noMove 必须是布尔值。"}
	}
	switch v := widget["noResize"].(type) {
	case nil:
	case bool:
		fields.noResize = v
	default:
		return nil, &SchemaIssue{"invalid-no-resize", path + ".noResize", "noResize 必须是布尔值。"}
	}
	switch v := widget["presetKey"].(type) {
	case nil:
	case string:
		if len(v) > maxWorkbenchTextLen {
			return nil, &SchemaIssue{"invalid-preset-key", path + ".presetKey", "presetKey 过长。"}
		}
		fields.presetKey = v
	default:
		return nil, &SchemaIssue{"invalid-preset-key", path + ".presetKey", "presetKey 必须是字符串。"}
	}
	switch v := widget["config"].(type) {
	case nil:
	case map[string]any:
		fields.config = v
	default:
		return nil, &SchemaIssue{"invalid-config", path + ".config", "config 必须是可序列化的对象。"}
	}
	return fields, nil
}

// readText 非空字符串（去首尾空白后判空，镜像前端 readText；追加服务端长度护栏）
func readText(value any, path string) (string, *SchemaIssue) {
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return "", &SchemaIssue{"invalid-text", path, "必须是非空字符串。"}
	}
	if len(text) > maxWorkbenchTextLen {
		return "", &SchemaIssue{"invalid-text", path, "字符串长度超出上限。"}
	}
	return text, nil
}

// readInteger 整数读取：positive=true 要求正整数，否则非负整数（镜像 readInteger）
func readInteger(value any, path string, positive bool) (float64, *SchemaIssue) {
	number, ok := value.(float64)
	if !ok || number != float64(int64(number)) || (positive && number <= 0) || (!positive && number < 0) {
		if positive {
			return 0, &SchemaIssue{"invalid-layout-value", path, "必须是正整数。"}
		}
		return 0, &SchemaIssue{"invalid-layout-value", path, "必须是非负整数。"}
	}
	return number, nil
}

// readOptionalInteger 可选正整数（缺省合法，出现则必须为正整数）
func readOptionalInteger(value any, path string) (*float64, *SchemaIssue) {
	if value == nil {
		return nil, nil
	}
	number, issue := readInteger(value, path, true)
	if issue != nil {
		return nil, issue
	}
	return &number, nil
}
