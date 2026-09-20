package service

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"evolyn/internal/platform/form/model"
)

// normalizeFrontendEventDependencies 以模板重新构建依赖索引。客户端提交的
// request_rely/action_rely 只是缓存，不能成为字段生命周期与执行安全的事实源。
func normalizeFrontendEventDependencies(raw model.JSONContent) (model.JSONContent, error) {
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	content, _ := root["content"].(map[string]any)
	events, _ := content["formEvents"].([]any)
	for _, rawEvent := range events {
		event, _ := rawEvent.(map[string]any)
		if event == nil {
			continue
		}
		requestTemplates := make([]string, 0)
		request, _ := event["request"].(map[string]any)
		if value, ok := request["url"].(string); ok {
			requestTemplates = append(requestTemplates, value)
		}
		for _, key := range []string{"header", "body"} {
			entries, _ := request[key].([]any)
			for _, rawEntry := range entries {
				entry, _ := rawEntry.(map[string]any)
				if value, ok := entry["key"].(string); ok {
					requestTemplates = append(requestTemplates, value)
				}
				if value, ok := entry["value"].(string); ok {
					requestTemplates = append(requestTemplates, value)
				}
			}
		}
		event["request_rely"] = frontendEventDependencies(requestTemplates...)
		actionTemplates := make([]string, 0)
		actions, _ := event["action"].([]any)
		for _, rawAction := range actions {
			action, _ := rawAction.(map[string]any)
			if value, ok := action["value"].(string); ok {
				actionTemplates = append(actionTemplates, value)
			}
		}
		event["action_rely"] = frontendEventDependencies(actionTemplates...)
	}
	encoded, err := json.Marshal(root)
	return model.JSONContent(encoded), err
}

const (
	frontendEventMaxEvents  = 50
	frontendEventMaxEntries = 20
	frontendEventMaxActions = 50
	frontendEventURLMax     = 4000
)

var (
	frontendEventIDPattern       = regexp.MustCompile(`^evt_[A-Za-z0-9_-]{4,60}$`)
	frontendEventTokenPattern    = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)
	frontendEventHeaderPattern   = regexp.MustCompile("^[!#$%&'*+.^_`|~0-9A-Za-z-]+$")
	frontendEventJSONPathPattern = regexp.MustCompile(`^\$response(?:\.[A-Za-z_][A-Za-z0-9_]*|\[[0-9]+\])*$`)
	frontendEventXPathPattern    = regexp.MustCompile(`^/(?:[A-Za-z_][A-Za-z0-9_.-]*)(?:/[A-Za-z_][A-Za-z0-9_.-]*)*$`)
)

// frontendEventRestrictedHeaders 保留会干扰代理会话或响应语义的请求头。
// Authorization 是业务 API 的常规鉴权方式，允许表单事件按需透传。
var frontendEventRestrictedHeaders = map[string]bool{
	"cookie":              true,
	"proxy-authorization": true,
	"set-cookie":          true,
}

// validateFrontendEvents 校验 v10 content.formEvents。依赖数组不是安全事实源；
// 此处只校验其形状，实际执行时会从模板重新提取字段引用。
func validateFrontendEvents(content map[string]any, items []any, issues *[]SchemaIssue) {
	rawEvents, ok := content["formEvents"].([]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: "content.formEvents", Message: "formEvents 必须是数组（v10 起必填）"})
		return
	}
	if len(rawEvents) > frontendEventMaxEvents {
		*issues = append(*issues, SchemaIssue{Path: "content.formEvents", Message: fmt.Sprintf("前端事件数量不能超过 %d", frontendEventMaxEvents)})
	}
	fields := buildSnapshotFieldsFromItems(items)
	ids := map[string]bool{}
	names := map[string]bool{}
	for index, rawEvent := range rawEvents {
		path := fmt.Sprintf("content.formEvents[%d]", index)
		event, ok := rawEvent.(map[string]any)
		if !ok {
			*issues = append(*issues, SchemaIssue{Path: path, Message: "前端事件必须是 JSON 对象"})
			continue
		}
		rejectUnknownKeys(event, []string{"id", "enabled", "name", "description", "trigger", "trigger_type", "request_type", "request", "request_rely", "action", "action_rely", "subform_fill_rule"}, path, issues)
		validateFrontendEventIdentity(event, path, ids, names, issues)
		trigger := frontendEventString(event["trigger"])
		if field, exists := fields[trigger]; !exists || !frontendEventValueField(field.widgetType) {
			*issues = append(*issues, SchemaIssue{Path: path + ".trigger", Message: "触发字段必须是表单中存在的可填写字段"})
		}
		if event["trigger_type"] != "widget" {
			*issues = append(*issues, SchemaIssue{Path: path + ".trigger_type", Message: `trigger_type 必须固定为 "widget"`})
		}
		if requestType, ok := asInteger(event["request_type"]); !ok || requestType != 0 {
			*issues = append(*issues, SchemaIssue{Path: path + ".request_type", Message: "request_type 必须固定为 0"})
		}
		request, _ := event["request"].(map[string]any)
		validateFrontendEventRequest(request, path+".request", fields, issues)
		validateFrontendEventStringList(event["request_rely"], path+".request_rely", fields, issues)
		validateFrontendEventActions(event["action"], path+".action", trigger, request, fields, issues)
		validateFrontendEventStringList(event["action_rely"], path+".action_rely", fields, issues)
		if rule := frontendEventString(event["subform_fill_rule"]); rule != "merge" && rule != "replace" {
			*issues = append(*issues, SchemaIssue{Path: path + ".subform_fill_rule", Message: "subform_fill_rule 必须是 merge / replace"})
		}
	}
}

func validateFrontendEventIdentity(event map[string]any, path string, ids, names map[string]bool, issues *[]SchemaIssue) {
	id := frontendEventString(event["id"])
	if !frontendEventIDPattern.MatchString(id) || len(id) > 64 {
		*issues = append(*issues, SchemaIssue{Path: path + ".id", Message: "事件 id 必须以 evt_ 开头且长度为 8–64"})
	} else if ids[id] {
		*issues = append(*issues, SchemaIssue{Path: path + ".id", Message: "事件 id 不能重复"})
	} else {
		ids[id] = true
	}
	name := strings.TrimSpace(frontendEventString(event["name"]))
	if name == "" || len([]rune(name)) > 64 {
		*issues = append(*issues, SchemaIssue{Path: path + ".name", Message: "事件名称必须为 1–64 个字符"})
	} else if names[name] {
		*issues = append(*issues, SchemaIssue{Path: path + ".name", Message: "事件名称不能重复"})
	} else {
		names[name] = true
	}
	description, ok := event["description"].(string)
	if !ok || len([]rune(description)) > 500 {
		*issues = append(*issues, SchemaIssue{Path: path + ".description", Message: "事件说明必须是不超过 500 个字符的字符串"})
	}
	if _, ok := event["enabled"].(bool); !ok {
		*issues = append(*issues, SchemaIssue{Path: path + ".enabled", Message: "enabled 必须是布尔值"})
	}
}

func validateFrontendEventRequest(request map[string]any, path string, fields map[string]snapshotField, issues *[]SchemaIssue) {
	if request == nil {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "request 必须是 JSON 对象"})
		return
	}
	rejectUnknownKeys(request, []string{"method", "url", "header", "body", "format"}, path, issues)
	method := frontendEventString(request["method"])
	if method != "get" && method != "post" {
		*issues = append(*issues, SchemaIssue{Path: path + ".method", Message: "method 必须是 get / post"})
	}
	rawURL := frontendEventString(request["url"])
	parsed, err := url.Parse(rawURL)
	if rawURL == "" || len(rawURL) > frontendEventURLMax || err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		*issues = append(*issues, SchemaIssue{Path: path + ".url", Message: "url 必须是不超过 4000 字符且不含用户信息的 HTTPS 地址"})
	}
	validateFrontendEventTemplate(rawURL, path+".url", fields, issues)
	validateFrontendEventEntries(request["header"], path+".header", true, fields, issues)
	validateFrontendEventEntries(request["body"], path+".body", false, fields, issues)
	if format := frontendEventString(request["format"]); format != "json" && format != "xml" {
		*issues = append(*issues, SchemaIssue{Path: path + ".format", Message: "format 必须是 json / xml"})
	}
}

func validateFrontendEventEntries(raw any, path string, header bool, fields map[string]snapshotField, issues *[]SchemaIssue) {
	entries, ok := raw.([]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "请求参数必须是数组"})
		return
	}
	if len(entries) > frontendEventMaxEntries {
		*issues = append(*issues, SchemaIssue{Path: path, Message: fmt.Sprintf("请求参数不能超过 %d 条", frontendEventMaxEntries)})
	}
	for index, rawEntry := range entries {
		entryPath := fmt.Sprintf("%s[%d]", path, index)
		entry, ok := rawEntry.(map[string]any)
		if !ok {
			*issues = append(*issues, SchemaIssue{Path: entryPath, Message: "请求参数必须是 JSON 对象"})
			continue
		}
		rejectUnknownKeys(entry, []string{"key", "value"}, entryPath, issues)
		key, keyOK := entry["key"].(string)
		value, valueOK := entry["value"].(string)
		if !keyOK || strings.TrimSpace(key) == "" || len(key) > 256 || !valueOK || len(value) > frontendEventURLMax {
			*issues = append(*issues, SchemaIssue{Path: entryPath, Message: "请求参数 key/value 不能为空且不能超过长度限制"})
			continue
		}
		if header {
			lower := strings.ToLower(strings.TrimSpace(key))
			shape := frontendEventTokenPattern.ReplaceAllString(key, "X")
			if !frontendEventHeaderPattern.MatchString(shape) || frontendEventRestrictedHeaders[lower] {
				*issues = append(*issues, SchemaIssue{Path: entryPath + ".key", Message: "Header 名称无效或属于禁止的敏感 Header"})
			}
		}
		validateFrontendEventTemplate(key, entryPath+".key", fields, issues)
		validateFrontendEventTemplate(value, entryPath+".value", fields, issues)
	}
}

func validateFrontendEventActions(raw any, path, trigger string, request map[string]any, fields map[string]snapshotField, issues *[]SchemaIssue) {
	actions, ok := raw.([]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "action 必须是数组"})
		return
	}
	if len(actions) > frontendEventMaxActions {
		*issues = append(*issues, SchemaIssue{Path: path, Message: fmt.Sprintf("返回值映射不能超过 %d 条", frontendEventMaxActions)})
	}
	format := frontendEventString(request["format"])
	targets := map[string]bool{}
	for index, rawAction := range actions {
		actionPath := fmt.Sprintf("%s[%d]", path, index)
		action, ok := rawAction.(map[string]any)
		if !ok {
			*issues = append(*issues, SchemaIssue{Path: actionPath, Message: "返回值映射必须是 JSON 对象"})
			continue
		}
		rejectUnknownKeys(action, []string{"field", "value"}, actionPath, issues)
		fieldName := frontendEventString(action["field"])
		field, exists := fields[fieldName]
		if !exists || !frontendEventValueField(field.widgetType) || fieldName == trigger || targets[fieldName] {
			*issues = append(*issues, SchemaIssue{Path: actionPath + ".field", Message: "目标字段不存在、重复、不可写或与触发字段相同"})
		} else {
			targets[fieldName] = true
		}
		value := frontendEventString(action["value"])
		validPath := format == "xml" && frontendEventXPathPattern.MatchString(value) || format != "xml" && frontendEventJSONPathPattern.MatchString(value)
		if value == "" || (!validPath && !strings.Contains(value, "${")) {
			*issues = append(*issues, SchemaIssue{Path: actionPath + ".value", Message: "返回值路径或字段模板格式无效"})
		}
		validateFrontendEventTemplate(value, actionPath+".value", fields, issues)
	}
}

func validateFrontendEventTemplate(template, path string, fields map[string]snapshotField, issues *[]SchemaIssue) {
	for _, match := range frontendEventTokenPattern.FindAllStringSubmatch(template, -1) {
		if len(match) < 2 {
			continue
		}
		field, exists := fields[match[1]]
		if !exists || !frontendEventValueField(field.widgetType) {
			*issues = append(*issues, SchemaIssue{Path: path, Message: fmt.Sprintf("模板引用的字段「%s」不存在或不可读取", match[1])})
		}
	}
	stripped := frontendEventTokenPattern.ReplaceAllString(template, "")
	if strings.Contains(stripped, "${") {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "字段模板包含不完整或非法令牌"})
	}
}

func validateFrontendEventStringList(raw any, path string, fields map[string]snapshotField, issues *[]SchemaIssue) {
	values, ok := raw.([]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "依赖索引必须是字符串数组"})
		return
	}
	for index, value := range values {
		name, ok := value.(string)
		if !ok || fields[name].widgetName == "" {
			*issues = append(*issues, SchemaIssue{Path: fmt.Sprintf("%s[%d]", path, index), Message: "依赖字段不存在"})
		}
	}
}

func frontendEventValueField(widgetType string) bool {
	return widgetType != "" && widgetType != "separator" && widgetType != "button" && widgetType != "richtext"
}

func frontendEventString(raw any) string {
	value, _ := raw.(string)
	return value
}

// frontendEventDependencies 从模板重新构建稳定有序依赖索引。
func frontendEventDependencies(templates ...string) []string {
	seen := map[string]bool{}
	for _, template := range templates {
		for _, match := range frontendEventTokenPattern.FindAllStringSubmatch(template, -1) {
			if len(match) > 1 {
				seen[match[1]] = true
			}
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
