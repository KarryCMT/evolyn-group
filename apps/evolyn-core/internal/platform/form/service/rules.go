// 字段显隐规则编译器与求值器（v5，Go 侧镜像实现）。
//
// 与前端 packages/form/src/schema/rules.ts 对同一规则集的求值结论必须一致：
// 条件源不可见（静态隐藏、权限隐藏或被上游规则隐藏）时条件视为「不满足」；
// 除 isEmpty/notEmpty 外空值一律令条件不成立；上游隐藏时下游按拓扑序自然
// 隐藏。本文件只做防御式编译与求值，结构与依赖图合法性由 schema.go 的
// 设计期校验（validateFieldShowRules）保证。
package service

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
)

// fieldShowCondition 求值用的条件窄化视图。
type fieldShowCondition struct {
	field       string
	typ         string
	method      string
	value       []any
	includeSelf bool // includeCurrentMember：把当前登录成员加入比较集合
}

// fieldShowRule 求值用的规则窄化视图。
type fieldShowRule struct {
	id         string
	rel        string // and / or
	conditions []fieldShowCondition
	targets    []string
}

// fieldShowEvaluator 已编译的规则求值器：同一发布快照内可复用。
type fieldShowEvaluator struct {
	rules   []fieldShowRule
	ownerOf map[string]string   // 目标字段 → 所属规则 id
	edges   map[string][]string // 依赖图出边：条件源 → 目标（保持插入序）
	order   []string            // 参与规则字段的拓扑序（条件源在前、目标在后）
}

// compileFieldShowRules 从发布快照文档（{content:{...}} 根）防御式编译规则集；
// 无合法规则时返回空求值器（所有字段规则可见性恒真，不产生任何求值开销）。
func compileFieldShowRules(root map[string]any) *fieldShowEvaluator {
	eval := &fieldShowEvaluator{
		ownerOf: map[string]string{},
		edges:   map[string][]string{},
	}
	indegree := map[string]int{}
	content, _ := root["content"].(map[string]any)
	rawRules, _ := content["fieldShowRules"].([]any)
	for _, rawRule := range rawRules {
		rule, ok := parseFieldShowRule(rawRule)
		if !ok {
			continue
		}
		eval.rules = append(eval.rules, rule)
		for _, target := range rule.targets {
			if _, exists := eval.ownerOf[target]; !exists {
				eval.ownerOf[target] = rule.id
			}
			indegree[target] += 0 // 确保目标进入节点表
		}
		for _, condition := range rule.conditions {
			indegree[condition.field] += 0 // 确保纯条件源进入节点表
			for _, target := range rule.targets {
				if !containsString(eval.edges[condition.field], target) {
					eval.edges[condition.field] = append(eval.edges[condition.field], target)
					indegree[target]++
				}
			}
		}
	}
	eval.order = topologicalShowOrder(indegree, eval.edges)
	return eval
}

// parseFieldShowRule 结构不完整即跳过（合法性由设计期校验单独保证）。
func parseFieldShowRule(raw any) (fieldShowRule, bool) {
	ruleMap, ok := raw.(map[string]any)
	if !ok {
		return fieldShowRule{}, false
	}
	id, _ := ruleMap["id"].(string)
	if id == "" {
		return fieldShowRule{}, false
	}
	rel := "and"
	filter, _ := ruleMap["filter"].(map[string]any)
	if filter != nil && filter["rel"] == "or" {
		rel = "or"
	}
	rawConditions, _ := filter["cond"].([]any)
	conditions := make([]fieldShowCondition, 0, len(rawConditions))
	for _, rawCondition := range rawConditions {
		conditionMap, ok := rawCondition.(map[string]any)
		if !ok {
			continue
		}
		field, _ := conditionMap["field"].(string)
		typ, _ := conditionMap["type"].(string)
		method, _ := conditionMap["method"].(string)
		if field == "" || method == "" {
			continue
		}
		values, _ := conditionMap["value"].([]any)
		includeSelf, _ := conditionMap["includeCurrentMember"].(bool)
		conditions = append(conditions, fieldShowCondition{
			field:       field,
			typ:         typ,
			method:      method,
			value:       values,
			includeSelf: includeSelf,
		})
	}
	rawTargets, _ := ruleMap["fields"].([]any)
	targets := make([]string, 0, len(rawTargets))
	for _, rawTarget := range rawTargets {
		if target, ok := rawTarget.(string); ok && target != "" {
			targets = append(targets, target)
		}
	}
	if len(conditions) == 0 || len(targets) == 0 {
		return fieldShowRule{}, false
	}
	return fieldShowRule{id: id, rel: rel, conditions: conditions, targets: targets}, true
}

// topologicalShowOrder Kahn 排序；环内节点（设计期已拒绝）不进入序列，
// 其求值结果收敛为隐藏（fail-closed）。节点名排序仅为遍历确定性，
// 求值结果只依赖图结构，与遍历顺序无关。
func topologicalShowOrder(indegree map[string]int, edges map[string][]string) []string {
	nodes := make([]string, 0, len(indegree))
	for node := range indegree {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	queue := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if indegree[node] == 0 {
			queue = append(queue, node)
		}
	}
	order := make([]string, 0, len(nodes))
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)
		for _, target := range edges[node] {
			indegree[target]--
			if indegree[target] == 0 {
				queue = append(queue, target)
			}
		}
	}
	return order
}

// ruleFieldVisibility 按拓扑序计算每个规则目标字段的可见性（仅含参与规则
// 的目标字段）。baseReadable 为基础可达性（静态 visible，权限管线可叠加
// 权限矩阵）；valueOf 返回字段当前值；currentMemberID 为空表示匿名/未注入。
// 调用方以「静态可见 ∧（规则结果或真）」合成最终可见性。
func (e *fieldShowEvaluator) ruleFieldVisibility(
	baseReadable func(name string) bool,
	valueOf func(name string) any,
	currentMemberID string,
) map[string]bool {
	if len(e.ownerOf) == 0 {
		return nil
	}
	ruleIndex := make(map[string]fieldShowRule, len(e.rules))
	for _, rule := range e.rules {
		ruleIndex[rule.id] = rule
	}
	// 条件源可达性 = 基础可达性 ∧ 已求值的上游规则可见性（未涉及规则视为真）。
	ruleVisible := map[string]bool{}
	for _, field := range e.order {
		ruleID, isTarget := e.ownerOf[field]
		if !isTarget {
			continue
		}
		rule := ruleIndex[ruleID]
		ruleVisible[field] = e.matchRule(rule, func(name string) bool {
			if !baseReadable(name) {
				return false
			}
			if visible, computed := ruleVisible[name]; computed {
				return visible
			}
			return true
		}, valueOf, currentMemberID)
	}
	return ruleVisible
}

// matchRule 单规则匹配：不可见条件源一律不成立。
func (e *fieldShowEvaluator) matchRule(
	rule fieldShowRule,
	isFieldVisible func(name string) bool,
	valueOf func(name string) any,
	currentMemberID string,
) bool {
	results := make([]bool, 0, len(rule.conditions))
	for _, condition := range rule.conditions {
		if !isFieldVisible(condition.field) {
			results = append(results, false)
			continue
		}
		results = append(results, matchFieldShowCondition(condition, valueOf(condition.field), currentMemberID))
	}
	if rule.rel == "or" {
		for _, matched := range results {
			if matched {
				return true
			}
		}
		return false
	}
	for _, matched := range results {
		if !matched {
			return false
		}
	}
	return true
}

// matchFieldShowCondition 单条件求值（与 TS matchFieldShowCondition 对齐）。
func matchFieldShowCondition(condition fieldShowCondition, rawValue any, currentMemberID string) bool {
	empty := isEmptyValue(rawValue)
	switch condition.method {
	case "isEmpty":
		return empty
	case "notEmpty":
		return !empty
	}
	if empty {
		return false
	}
	expected := expectedShowValues(condition, currentMemberID)
	switch condition.method {
	case "eq", "in":
		return scalarIncludes(condition.typ, rawValue, expected)
	case "ne", "notIn":
		return !scalarIncludes(condition.typ, rawValue, expected)
	case "contains":
		text, ok := rawValue.(string)
		return ok && strings.Contains(text, firstShowText(expected))
	case "notContains":
		text, ok := rawValue.(string)
		return ok && !strings.Contains(text, firstShowText(expected))
	case "gt", "gte", "lt", "lte", "between":
		return orderedShowMatch(condition, rawValue, expected)
	case "containsAny", "containsAll", "containsNone":
		selected, ok := rawValue.([]any)
		if !ok {
			return false
		}
		expectedSet := stringSetOf(expected)
		hits := 0
		for _, entry := range selected {
			if text, isString := entry.(string); isString && expectedSet.Contains(text) {
				hits++
			}
		}
		switch condition.method {
		case "containsAny":
			return hits > 0
		case "containsAll":
			return hits == len(expectedSet)
		default:
			return hits == 0
		}
	default:
		return false
	}
}

// orderedShowMatch number/datetime 的有序比较：数值按大小，datetime 规范
// 形状字符串按字典序（同格式可比）。
func orderedShowMatch(condition fieldShowCondition, rawValue any, expected []any) bool {
	compare := func(a, b string) int {
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	}
	if condition.typ == "number" {
		left, ok := rawValue.(float64)
		if !ok || math.IsNaN(left) || math.IsInf(left, 0) {
			return false
		}
		bounds := make([]float64, 0, 2)
		for _, entry := range expected {
			num, ok := entry.(float64)
			if !ok || math.IsNaN(num) || math.IsInf(num, 0) {
				return false
			}
			bounds = append(bounds, num)
		}
		switch condition.method {
		case "gt":
			return len(bounds) == 1 && left > bounds[0]
		case "gte":
			return len(bounds) == 1 && left >= bounds[0]
		case "lt":
			return len(bounds) == 1 && left < bounds[0]
		case "lte":
			return len(bounds) == 1 && left <= bounds[0]
		case "between":
			return len(bounds) == 2 && left >= bounds[0] && left <= bounds[1]
		}
		return false
	}
	left, ok := rawValue.(string)
	if !ok {
		return false
	}
	bounds := make([]string, 0, 2)
	for _, entry := range expected {
		text, isString := entry.(string)
		if !isString {
			return false
		}
		bounds = append(bounds, text)
	}
	switch condition.method {
	case "gt":
		return len(bounds) == 1 && compare(left, bounds[0]) > 0
	case "gte":
		return len(bounds) == 1 && compare(left, bounds[0]) >= 0
	case "lt":
		return len(bounds) == 1 && compare(left, bounds[0]) < 0
	case "lte":
		return len(bounds) == 1 && compare(left, bounds[0]) <= 0
	case "between":
		return len(bounds) == 2 && compare(left, bounds[0]) >= 0 && compare(left, bounds[1]) <= 0
	}
	return false
}

// expectedShowValues 组装比较集合：value 常量 + includeCurrentMember 注入。
func expectedShowValues(condition fieldShowCondition, currentMemberID string) []any {
	merged := make([]any, 0, len(condition.value)+1)
	hasCurrent := false
	for _, entry := range condition.value {
		switch v := entry.(type) {
		case string:
			if v == "" {
				continue
			}
			if v == currentMemberID {
				hasCurrent = true
			}
			merged = append(merged, v)
		case float64:
			if !math.IsNaN(v) && !math.IsInf(v, 0) {
				merged = append(merged, v)
			}
		}
	}
	if condition.includeSelf && currentMemberID != "" && !hasCurrent {
		merged = append(merged, currentMemberID)
	}
	return merged
}

// scalarIncludes 单值语义控件的集合命中（eq/ne 与 in/notIn 同构）。
func scalarIncludes(typ string, rawValue any, expected []any) bool {
	if typ == "number" {
		num, ok := rawValue.(float64)
		if !ok || math.IsNaN(num) || math.IsInf(num, 0) {
			return false
		}
		for _, entry := range expected {
			if expectedNum, ok := entry.(float64); ok && expectedNum == num {
				return true
			}
		}
		return false
	}
	text, ok := rawValue.(string)
	if !ok {
		return false
	}
	return containsString(expectedStrings(expected), text)
}

func expectedStrings(expected []any) []string {
	values := make([]string, 0, len(expected))
	for _, entry := range expected {
		if text, ok := entry.(string); ok {
			values = append(values, text)
		}
	}
	return values
}

func firstShowText(expected []any) string {
	for _, entry := range expected {
		if text, ok := entry.(string); ok && text != "" {
			return text
		}
	}
	return ""
}

func stringSetOf(expected []any) stringSet {
	set := stringSet{}
	for _, entry := range expected {
		if text, ok := entry.(string); ok {
			set[text] = struct{}{}
		}
	}
	return set
}

// decodeShowValue 按需解码 JSON 原文为求值值（失败按空值处理）。
func decodeShowValue(raw json.RawMessage) any {
	if len(raw) == 0 || isNullJSON(raw) {
		return nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return value
}
