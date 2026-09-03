package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 与前端 schema/rules.spec.ts 对拍的规则求值用例（条件矩阵、拓扑传播与
// 条件源可达性语义两侧结论必须一致）。

func showCond(field, typ, method string, value []any) fieldShowCondition {
	return fieldShowCondition{field: field, typ: typ, method: method, value: value}
}

func TestMatchFieldShowConditionMatrix(t *testing.T) {
	cases := []struct {
		name     string
		cond     fieldShowCondition
		value    any
		expected bool
	}{
		// text：eq/ne/contains/notContains 与空值语义
		{"text eq 命中", showCond("_s", "text", "eq", []any{"甲"}), "甲", true},
		{"text eq 未命中", showCond("_s", "text", "eq", []any{"甲"}), "乙", false},
		{"text ne 命中", showCond("_s", "text", "ne", []any{"甲"}), "乙", true},
		{"text contains", showCond("_s", "text", "contains", []any{"工时"}), "总工时统计", true},
		{"text notContains", showCond("_s", "text", "notContains", []any{"工时"}), "备注", true},
		{"text 空值 ne 不成立", showCond("_s", "text", "ne", []any{"甲"}), nil, false},
		{"text isEmpty 空串", showCond("_s", "text", "isEmpty", nil), "", true},
		{"text isEmpty 空数组", showCond("_s", "text", "isEmpty", nil), []any{}, true},
		{"text notEmpty", showCond("_s", "text", "notEmpty", nil), "x", true},
		// number：有序比较与 between 含边界
		{"number gt", showCond("_s", "number", "gt", []any{10.0}), 10.5, true},
		{"number gte 边界", showCond("_s", "number", "gte", []any{10.0}), 10.0, true},
		{"number lt", showCond("_s", "number", "lt", []any{10.0}), 9.9, true},
		{"number lte 边界", showCond("_s", "number", "lte", []any{10.0}), 10.0, true},
		{"number between 含边界", showCond("_s", "number", "between", []any{10.0, 20.0}), 20.0, true},
		{"number between 越界", showCond("_s", "number", "between", []any{10.0, 20.0}), 20.1, false},
		{"number 类型不符", showCond("_s", "number", "eq", []any{10.0}), "10", false},
		// datetime：规范字符串字典序
		{"datetime gte", showCond("_s", "datetime", "gte", []any{"2026-01-01"}), "2026-01-02", true},
		{"datetime between", showCond("_s", "datetime", "between", []any{"2026-01-01", "2026-12-31"}), "2026-06-15", true},
		{"datetime lt", showCond("_s", "datetime", "lt", []any{"2026-01-01"}), "2025-12-31", true},
		// 单选语义：eq/ne 与 in/notIn 同构
		{"radio eq", showCond("_s", "radiogroup", "eq", []any{"A"}), "A", true},
		{"radio ne", showCond("_s", "radiogroup", "ne", []any{"A"}), "B", true},
		{"combo in", showCond("_s", "combo", "in", []any{"A", "B"}), "B", true},
		{"combo notIn", showCond("_s", "combo", "notIn", []any{"A", "B"}), "C", true},
		{"combo notIn 空值不成立", showCond("_s", "combo", "notIn", []any{"A", "B"}), nil, false},
		// 多选语义
		{"multi containsAny", showCond("_s", "checkboxgroup", "containsAny", []any{"A", "B"}), []any{"B", "C"}, true},
		{"multi containsAny 未命中", showCond("_s", "checkboxgroup", "containsAny", []any{"A", "B"}), []any{"C"}, false},
		{"multi containsAll", showCond("_s", "checkboxgroup", "containsAll", []any{"A", "B"}), []any{"B", "A", "C"}, true},
		{"multi containsAll 缺项", showCond("_s", "checkboxgroup", "containsAll", []any{"A", "B"}), []any{"A"}, false},
		{"multi containsNone", showCond("_s", "checkboxgroup", "containsNone", []any{"A"}), []any{"C"}, true},
		{"multi isEmpty", showCond("_s", "checkboxgroup", "isEmpty", nil), []any{}, true},
		// includeCurrentMember：当前成员注入比较集合
		{"user eq 含当前成员", fieldShowCondition{field: "_s", typ: "user", method: "eq", value: []any{"member_a"}, includeSelf: true}, "member_current", true},
	}
	for _, testCase := range cases {
		if testCase.name == "user eq 含当前成员" {
			assert.Equal(t, testCase.expected,
				matchFieldShowCondition(testCase.cond, testCase.value, "member_current"), testCase.name)
			continue
		}
		assert.Equal(t, testCase.expected, matchFieldShowCondition(testCase.cond, testCase.value, ""), testCase.name)
	}
	// 匿名（未注入当前成员）时 includeCurrentMember 不参与
	assert.False(t, matchFieldShowCondition(
		fieldShowCondition{field: "_s", typ: "user", method: "eq", value: []any{"member_a"}, includeSelf: true},
		"member_current", ""))
}

func TestRuleFieldVisibilityTopologicalPropagation(t *testing.T) {
	// A eq 是 → B；B notEmpty → C：上游隐藏时下游不读取其隐藏值。
	content := map[string]any{"content": map[string]any{
		"fieldShowRules": []any{
			map[string]any{
				"id":     "r1",
				"filter": map[string]any{"rel": "and", "cond": []any{map[string]any{"field": "_a", "type": "text", "method": "eq", "value": []any{"是"}}}},
				"fields": []any{"_b"},
			},
			map[string]any{
				"id":     "r2",
				"filter": map[string]any{"rel": "and", "cond": []any{map[string]any{"field": "_b", "type": "text", "method": "notEmpty"}}},
				"fields": []any{"_c"},
			},
		},
	}}
	values := map[string]any{"_a": "是", "_b": "内容"}
	visible := compileFieldShowRules(content).ruleFieldVisibility(
		func(name string) bool { return true },
		func(name string) any { return values[name] },
		"")
	assert.True(t, visible["_b"])
	assert.True(t, visible["_c"])

	values = map[string]any{"_a": "否", "_b": "内容"}
	visible = compileFieldShowRules(content).ruleFieldVisibility(
		func(name string) bool { return true },
		func(name string) any { return values[name] },
		"")
	assert.False(t, visible["_b"])
	assert.False(t, visible["_c"])
}

func TestRuleFieldVisibilityConditionSourceNotReadable(t *testing.T) {
	// 条件源基础不可见（静态/权限隐藏）→ 条件视为不成立，不读取其值。
	content := map[string]any{"content": map[string]any{
		"fieldShowRules": []any{
			map[string]any{
				"id":     "r1",
				"filter": map[string]any{"rel": "and", "cond": []any{map[string]any{"field": "_src", "type": "text", "method": "eq", "value": []any{"甲"}}}},
				"fields": []any{"_target"},
			},
		},
	}}
	visible := compileFieldShowRules(content).ruleFieldVisibility(
		func(name string) bool { return name != "_src" },
		func(name string) any { return "甲" },
		"")
	assert.False(t, visible["_target"])
}

func TestRuleFieldVisibilityAndOr(t *testing.T) {
	content := map[string]any{"content": map[string]any{
		"fieldShowRules": []any{
			map[string]any{
				"id": "r1",
				"filter": map[string]any{"rel": "and", "cond": []any{
					map[string]any{"field": "_a", "type": "text", "method": "eq", "value": []any{"甲"}},
					map[string]any{"field": "_n", "type": "number", "method": "gt", "value": []any{10.0}},
				}},
				"fields": []any{"_t"},
			},
		},
	}}
	eval := compileFieldShowRules(content)
	values := map[string]any{"_a": "甲", "_n": 20.0}
	visible := eval.ruleFieldVisibility(func(string) bool { return true }, func(name string) any { return values[name] }, "")
	assert.True(t, visible["_t"])
	values["_n"] = 5.0
	visible = eval.ruleFieldVisibility(func(string) bool { return true }, func(name string) any { return values[name] }, "")
	assert.False(t, visible["_t"])
}

func TestCompileFieldShowRulesSkipsMalformed(t *testing.T) {
	content := map[string]any{"content": map[string]any{
		"fieldShowRules": []any{
			"not-a-map",
			map[string]any{"id": "", "filter": map[string]any{"rel": "and", "cond": []any{map[string]any{"field": "_a"}}}, "fields": []any{"_t"}},
			map[string]any{
				"id":     "ok",
				"filter": map[string]any{"rel": "and", "cond": []any{map[string]any{"field": "_a", "type": "text", "method": "eq", "value": []any{"甲"}}}},
				"fields": []any{"_t"},
			},
		},
	}}
	eval := compileFieldShowRules(content)
	assert.Len(t, eval.rules, 1)
	assert.Equal(t, "ok", eval.ownerOf["_t"])
}
