package service

import (
	"fmt"
	"regexp"
)

const (
	maxLinkageRules      = 50
	maxLinkageConditions = 20
	maxLinkageMappings   = 20
)

var linkageIDPattern = regexp.MustCompile(`^linkage_[A-Za-z0-9_-]{4,56}$`)

var linkageOperators = map[string]bool{
	"eq": true, "neq": true, "gt": true, "gte": true, "lt": true, "lte": true,
	"contains": true, "notContains": true, "in": true, "not_in": true,
	"empty": true, "not_empty": true,
}

// validateDataLinkages 镜像前端 v11 结构校验，并在保存/发布前阻断当前表单内
// 字段依赖环。跨表字段存在性与类型在发布服务中读取源表不可变快照终审。
func validateDataLinkages(content map[string]any, items []any, issues *[]SchemaIssue) { //nolint:gocyclo // 协议结构逐层给出精确 JSON Path
	rawRules, ok := content["linkages"].([]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: "content.linkages", Message: "linkages 必须是数组（v11 起必填）"})
		return
	}
	if len(rawRules) > maxLinkageRules {
		*issues = append(*issues, SchemaIssue{Path: "content.linkages", Message: fmt.Sprintf("数据联动规则数量不能超过 %d", maxLinkageRules)})
	}
	fields := make(map[string]bool)
	for _, rawItem := range items {
		item, _ := rawItem.(map[string]any)
		widget, _ := item["widget"].(map[string]any)
		name, _ := widget["widgetName"].(string)
		if name != "" {
			fields[name] = true
		}
	}
	ids := make(map[string]bool)
	graph := make(map[string]map[string]bool)
	for ruleIndex, rawRule := range rawRules {
		path := fmt.Sprintf("content.linkages[%d]", ruleIndex)
		rule, ok := rawRule.(map[string]any)
		if !ok {
			*issues = append(*issues, SchemaIssue{Path: path, Message: "数据联动规则必须是 JSON 对象"})
			continue
		}
		rejectUnknownKeys(rule, []string{"id", "version", "enabled", "source", "filter", "mappings", "result", "runtime"}, path, issues)
		id, _ := rule["id"].(string)
		if !linkageIDPattern.MatchString(id) || ids[id] {
			*issues = append(*issues, SchemaIssue{Path: path + ".id", Message: "数据联动 id 无效或重复"})
		} else {
			ids[id] = true
		}
		if version, ok := rule["version"].(float64); !ok || version != 1 {
			*issues = append(*issues, SchemaIssue{Path: path + ".version", Message: "version 必须固定为 1"})
		}
		if _, ok := rule["enabled"].(bool); !ok {
			*issues = append(*issues, SchemaIssue{Path: path + ".enabled", Message: "enabled 必须是布尔值"})
		}
		validateLinkageSource(rule["source"], path+".source", issues)

		dependencies := make(map[string]bool)
		filter, filterOK := rule["filter"].(map[string]any)
		if !filterOK {
			*issues = append(*issues, SchemaIssue{Path: path + ".filter", Message: "filter 必须是 JSON 对象"})
		} else {
			rejectUnknownKeys(filter, []string{"logic", "conditions"}, path+".filter", issues)
			logic, _ := filter["logic"].(string)
			conditions, conditionsOK := filter["conditions"].([]any)
			if (logic != "and" && logic != "or") || !conditionsOK || len(conditions) < 1 || len(conditions) > maxLinkageConditions {
				*issues = append(*issues, SchemaIssue{Path: path + ".filter", Message: "过滤条件必须包含 1–20 条且 logic 为 and / or"})
			} else {
				for index, rawCondition := range conditions {
					validateLinkageCondition(rawCondition, fmt.Sprintf("%s.filter.conditions[%d]", path, index), fields, dependencies, issues)
				}
			}
		}

		targets := make(map[string]bool)
		mappings, mappingsOK := rule["mappings"].([]any)
		if !mappingsOK || len(mappings) < 1 || len(mappings) > maxLinkageMappings {
			*issues = append(*issues, SchemaIssue{Path: path + ".mappings", Message: "联动映射必须包含 1–20 条"})
		} else {
			for index, rawMapping := range mappings {
				mappingPath := fmt.Sprintf("%s.mappings[%d]", path, index)
				mapping, ok := rawMapping.(map[string]any)
				if !ok {
					*issues = append(*issues, SchemaIssue{Path: mappingPath, Message: "联动映射必须是 JSON 对象"})
					continue
				}
				rejectUnknownKeys(mapping, []string{"sourceFieldId", "targetFieldId"}, mappingPath, issues)
				sourceFieldID, _ := mapping["sourceFieldId"].(string)
				targetFieldID, _ := mapping["targetFieldId"].(string)
				if sourceFieldID == "" {
					*issues = append(*issues, SchemaIssue{Path: mappingPath + ".sourceFieldId", Message: "请选择联动表单字段"})
				}
				if !fields[targetFieldID] || targets[targetFieldID] {
					*issues = append(*issues, SchemaIssue{Path: mappingPath + ".targetFieldId", Message: "目标字段不存在或重复"})
				} else {
					targets[targetFieldID] = true
				}
			}
		}
		for dependency := range dependencies {
			if graph[dependency] == nil {
				graph[dependency] = make(map[string]bool)
			}
			for target := range targets {
				graph[dependency][target] = true
			}
		}

		result, resultOK := rule["result"].(map[string]any)
		if !resultOK || result["mode"] != "first" {
			*issues = append(*issues, SchemaIssue{Path: path + ".result.mode", Message: "结果策略 mode 必须固定为 first"})
		}
		validateLinkageRuntime(rule["runtime"], path+".runtime", issues)
	}
	if linkageGraphHasCycle(graph) {
		*issues = append(*issues, SchemaIssue{Path: "content.linkages", Message: "数据联动字段形成循环依赖，请调整联动关系"})
	}
}

func validateLinkageSource(raw any, path string, issues *[]SchemaIssue) {
	source, ok := raw.(map[string]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "source 必须是 JSON 对象"})
		return
	}
	rejectUnknownKeys(source, []string{"type", "appId", "sourceId"}, path, issues)
	appID, appOK := source["appId"].(float64)
	sourceID, sourceOK := source["sourceId"].(string)
	if source["type"] != "form" || !appOK || appID <= 0 || appID != float64(uint(appID)) || !sourceOK || len(sourceID) <= len("form_") || sourceID[:len("form_")] != "form_" {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "联动数据源必须是有效的普通表单"})
	}
}

func validateLinkageCondition(raw any, path string, fields, dependencies map[string]bool, issues *[]SchemaIssue) {
	condition, ok := raw.(map[string]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "过滤条件必须是 JSON 对象"})
		return
	}
	rejectUnknownKeys(condition, []string{"id", "sourceFieldId", "operator", "value"}, path, issues)
	id, _ := condition["id"].(string)
	if len(id) < 4 || len(id) > 64 {
		*issues = append(*issues, SchemaIssue{Path: path + ".id", Message: "条件 id 必须为 4–64 个字符"})
	}
	if sourceFieldID, _ := condition["sourceFieldId"].(string); sourceFieldID == "" {
		*issues = append(*issues, SchemaIssue{Path: path + ".sourceFieldId", Message: "请选择联动表单字段"})
	}
	operator, _ := condition["operator"].(string)
	if !linkageOperators[operator] {
		*issues = append(*issues, SchemaIssue{Path: path + ".operator", Message: "操作符不受支持"})
		return
	}
	if operator == "empty" || operator == "not_empty" {
		return
	}
	value, ok := condition["value"].(map[string]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path + ".value", Message: "请选择当前表单字段或填写自定义值"})
		return
	}
	switch value["type"] {
	case "field":
		rejectUnknownKeys(value, []string{"type", "fieldId"}, path+".value", issues)
		fieldID, _ := value["fieldId"].(string)
		if !fields[fieldID] {
			*issues = append(*issues, SchemaIssue{Path: path + ".value.fieldId", Message: "当前表单依赖字段不存在"})
		} else {
			dependencies[fieldID] = true
		}
	case "constant":
		rejectUnknownKeys(value, []string{"type", "value", "valueType"}, path+".value", issues)
		if _, exists := value["value"]; !exists {
			*issues = append(*issues, SchemaIssue{Path: path + ".value.value", Message: "请填写自定义值"})
		}
	default:
		*issues = append(*issues, SchemaIssue{Path: path + ".value", Message: "请选择当前表单字段或填写自定义值"})
	}
}

func validateLinkageRuntime(raw any, path string, issues *[]SchemaIssue) {
	runtime, ok := raw.(map[string]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "数据联动运行参数不符合要求"})
		return
	}
	rejectUnknownKeys(runtime, []string{"trigger", "runOnInit", "debounceMs", "emptyStrategy", "errorStrategy"}, path, issues)
	debounce, debounceOK := runtime["debounceMs"].(float64)
	_, runOnInitOK := runtime["runOnInit"].(bool)
	if runtime["trigger"] != "dependency_change" || !runOnInitOK || !debounceOK || debounce < 0 || debounce > 2000 || debounce != float64(int(debounce)) || (runtime["emptyStrategy"] != "clear" && runtime["emptyStrategy"] != "keep") || (runtime["errorStrategy"] != "clear" && runtime["errorStrategy"] != "keep") {
		*issues = append(*issues, SchemaIssue{Path: path, Message: "数据联动运行参数不符合要求"})
	}
}

func linkageGraphHasCycle(graph map[string]map[string]bool) bool {
	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	var visit func(string) bool
	visit = func(field string) bool {
		if visiting[field] {
			return true
		}
		if visited[field] {
			return false
		}
		visiting[field] = true
		for target := range graph[field] {
			if visit(target) {
				return true
			}
		}
		delete(visiting, field)
		visited[field] = true
		return false
	}
	for field := range graph {
		if visit(field) {
			return true
		}
	}
	return false
}
