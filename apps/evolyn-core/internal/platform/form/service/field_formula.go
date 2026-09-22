package service

// v12 字段公式的发布期编译与服务端权威执行。公式源文只存在于不可变表单
// 快照；运行时仅消费同事务冻结的受控程序，禁止把客户端计算结果直接落库。

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"regexp"
	"sort"
	"strings"

	"evolyn/internal/platform/form/model"
)

const (
	fieldFormulaProtocolVersion = 12
	maxFieldFormulas            = 200
	maxFieldFormulaLength       = 4000
	maxFieldFormulaRemarkLength = 500
)

var fieldFormulaIDPattern = regexp.MustCompile(`^formula_[A-Za-z0-9_-]{4,56}$`)

type compiledFieldFormulas struct {
	Digest   string                 `json:"digest"`
	Formulas []compiledFieldFormula `json:"formulas"`
}

type compiledFieldFormula struct {
	ID           string   `json:"id"`
	Target       string   `json:"target"`
	Program      string   `json:"program"`
	Dependencies []string `json:"dependencies"`
}

// FieldFormulaEvaluationError 表示请求值导致的可回填错误；产物缺失、摘要不一致
// 等部署异常不会包装为此类型，调用方应按服务端错误处理。
type FieldFormulaEvaluationError struct {
	Target string
	Cause  error
}

func (e *FieldFormulaEvaluationError) Error() string {
	return fmt.Sprintf("field formula target %s: %v", e.Target, e.Cause)
}

func (e *FieldFormulaEvaluationError) Unwrap() error { return e.Cause }

type fieldFormulaDraft struct {
	id           string
	target       string
	program      string
	dependencies []string
	enabled      bool
}

// validateFieldFormulaProtocol 提供保存期 JSON Path 级反馈；表达式、依赖图与
// 数据联动互斥规则由 CompileFieldFormulas 复用同一终审器，避免两套语义漂移。
func validateFieldFormulaProtocol(content map[string]any, items []any, issues *[]SchemaIssue) {
	raw, ok := content["fieldFormulas"].([]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: "content.fieldFormulas", Message: "fieldFormulas 必须是数组（v12 起必填）"})
		return
	}
	if len(raw) > maxFieldFormulas {
		*issues = append(*issues, SchemaIssue{Path: "content.fieldFormulas", Message: fmt.Sprintf("字段公式数量不能超过 %d", maxFieldFormulas)})
	}
	for i, entry := range raw {
		path := fmt.Sprintf("content.fieldFormulas[%d]", i)
		rule, ok := entry.(map[string]any)
		if !ok {
			*issues = append(*issues, SchemaIssue{Path: path, Message: "字段公式必须是 JSON 对象"})
			continue
		}
		rejectUnknownKeys(rule, []string{"id", "version", "enabled", "targetFieldId", "formula", "remark"}, path, issues)
		id, _ := rule["id"].(string)
		if !fieldFormulaIDPattern.MatchString(id) {
			*issues = append(*issues, SchemaIssue{Path: path + ".id", Message: "字段公式 id 必须使用 formula_ 前缀且格式合法"})
		}
		if version, ok := asInteger(rule["version"]); !ok || version != 1 {
			*issues = append(*issues, SchemaIssue{Path: path + ".version", Message: "version 必须固定为 1"})
		}
		if _, ok := rule["enabled"].(bool); !ok {
			*issues = append(*issues, SchemaIssue{Path: path + ".enabled", Message: "enabled 必须是布尔值"})
		}
		if target, ok := rule["targetFieldId"].(string); !ok || target == "" {
			*issues = append(*issues, SchemaIssue{Path: path + ".targetFieldId", Message: "目标字段不能为空"})
		}
		formula, formulaOK := rule["formula"].(string)
		if !formulaOK || strings.TrimSpace(formula) == "" || len(formula) > maxFieldFormulaLength {
			*issues = append(*issues, SchemaIssue{Path: path + ".formula", Message: "formula 必须是 1–4000 字符的非空字符串"})
		}
		if remark, ok := rule["remark"].(string); !ok || len(remark) > maxFieldFormulaRemarkLength {
			*issues = append(*issues, SchemaIssue{Path: path + ".remark", Message: "remark 必须是不超过 500 字符的字符串"})
		}
	}
	if _, err := compileFieldFormulaDrafts(content, items); err != nil {
		*issues = append(*issues, SchemaIssue{Path: "content.fieldFormulas", Message: err.Error()})
	}
}

// CompileFieldFormulas 将字段引用改写为 FIELD("key") 并按依赖拓扑排序。
func CompileFieldFormulas(content map[string]any, protocol int) (model.JSONContent, error) {
	if protocol < fieldFormulaProtocolVersion {
		return model.JSONContent(`{}`), nil
	}
	root := content
	inner := content
	if value, ok := content["content"].(map[string]any); ok {
		inner = value
	}
	items, ok := documentItems(map[string]any{"content": inner})
	if !ok {
		return nil, fmt.Errorf("content.items missing")
	}
	ordered, err := compileFieldFormulaDrafts(inner, items)
	if err != nil {
		return nil, err
	}
	compiled := compiledFieldFormulas{Digest: submitRulesDigest(root), Formulas: make([]compiledFieldFormula, 0, len(ordered))}
	for _, formula := range ordered {
		if !formula.enabled {
			continue
		}
		compiled.Formulas = append(compiled.Formulas, compiledFieldFormula{
			ID: formula.id, Target: formula.target, Program: formula.program, Dependencies: formula.dependencies,
		})
	}
	encoded, err := json.Marshal(compiled)
	return model.JSONContent(encoded), err
}

func compileFieldFormulaDrafts(content map[string]any, items []any) ([]fieldFormulaDraft, error) { //nolint:gocyclo // 编译器逐条执行协议终审
	raw, ok := content["fieldFormulas"].([]any)
	if !ok || len(raw) > maxFieldFormulas {
		return nil, fmt.Errorf("fieldFormulas 必须是最多 %d 项的数组", maxFieldFormulas)
	}
	fields := buildSnapshotFieldsFromItems(items)
	linkageTargets := fieldFormulaLinkageTargets(content)
	seenIDs := map[string]bool{}
	seenTargets := map[string]bool{}
	drafts := make([]fieldFormulaDraft, 0, len(raw))
	for index, entry := range raw {
		rule, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("字段公式 %d 必须是对象", index)
		}
		id, _ := rule["id"].(string)
		if !fieldFormulaIDPattern.MatchString(id) || seenIDs[id] {
			return nil, fmt.Errorf("字段公式 %d id 无效或重复", index)
		}
		seenIDs[id] = true
		if version, ok := asInteger(rule["version"]); !ok || version != 1 {
			return nil, fmt.Errorf("字段公式 %s version 必须为 1", id)
		}
		enabled, ok := rule["enabled"].(bool)
		if !ok {
			return nil, fmt.Errorf("字段公式 %s enabled 必须是布尔值", id)
		}
		target, _ := rule["targetFieldId"].(string)
		meta, exists := fields[target]
		if !exists || (meta.widgetType != "text" && meta.widgetType != "textarea") {
			return nil, fmt.Errorf("字段公式 %s 目标字段不存在或不是文本字段", id)
		}
		if seenTargets[target] {
			return nil, fmt.Errorf("目标字段 %s 只能配置一条字段公式", target)
		}
		seenTargets[target] = true
		if linkageTargets[target] {
			return nil, fmt.Errorf("字段 %s 不能同时配置数据联动和字段公式", target)
		}
		formula, _ := rule["formula"].(string)
		remark, remarkOK := rule["remark"].(string)
		if strings.TrimSpace(formula) == "" || len(formula) > maxFieldFormulaLength || !remarkOK || len(remark) > maxFieldFormulaRemarkLength {
			return nil, fmt.Errorf("字段公式 %s 内容或备注长度无效", id)
		}
		dependencies := orderedRefs(formula, submitFieldRef)
		for _, dependency := range dependencies {
			depMeta, exists := fields[dependency]
			if !exists || !submitFormulaFieldAllowed(depMeta.widgetType) {
				return nil, fmt.Errorf("字段公式 %s 引用了不存在或不支持的字段 %s", id, dependency)
			}
			if dependency == target {
				return nil, fmt.Errorf("字段公式 %s 不能引用自身", id)
			}
		}
		program := submitFieldRef.ReplaceAllString(formula, `FIELD("$1")`)
		expr, err := parser.ParseExpr(program)
		if err != nil {
			return nil, fmt.Errorf("字段公式 %s 解析失败: %w", id, err)
		}
		if err := validateFormulaAST(expr); err != nil {
			return nil, fmt.Errorf("字段公式 %s: %w", id, err)
		}
		if err := validateFieldFormulaArity(expr); err != nil {
			return nil, fmt.Errorf("字段公式 %s: %w", id, err)
		}
		drafts = append(drafts, fieldFormulaDraft{id: id, target: target, program: program, dependencies: dependencies, enabled: enabled})
	}
	return topologicalFieldFormulas(drafts)
}

func topologicalFieldFormulas(formulas []fieldFormulaDraft) ([]fieldFormulaDraft, error) {
	byTarget := map[string]fieldFormulaDraft{}
	indegree := map[string]int{}
	outgoing := map[string][]string{}
	for _, formula := range formulas {
		if !formula.enabled {
			continue
		}
		byTarget[formula.target] = formula
		indegree[formula.target] = 0
	}
	for _, formula := range formulas {
		if !formula.enabled {
			continue
		}
		for _, dependency := range formula.dependencies {
			if _, ok := byTarget[dependency]; !ok {
				continue
			}
			indegree[formula.target]++
			outgoing[dependency] = append(outgoing[dependency], formula.target)
		}
	}
	queue := make([]fieldFormulaDraft, 0, len(byTarget))
	for _, formula := range formulas {
		if formula.enabled && indegree[formula.target] == 0 {
			queue = append(queue, formula)
		}
	}
	ordered := make([]fieldFormulaDraft, 0, len(formulas))
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		ordered = append(ordered, current)
		for _, target := range outgoing[current.target] {
			indegree[target]--
			if indegree[target] == 0 {
				queue = append(queue, byTarget[target])
			}
		}
	}
	if len(ordered) != len(byTarget) {
		cycle := make([]string, 0)
		for target, degree := range indegree {
			if degree > 0 {
				cycle = append(cycle, target)
			}
		}
		sort.Strings(cycle)
		return nil, fmt.Errorf("字段公式存在循环依赖：%s", strings.Join(cycle, "、"))
	}
	// 禁用规则不执行，但保留稳定源顺序完成结构校验。
	for _, formula := range formulas {
		if !formula.enabled {
			ordered = append(ordered, formula)
		}
	}
	return ordered, nil
}

func fieldFormulaLinkageTargets(content map[string]any) map[string]bool {
	targets := map[string]bool{}
	rules, _ := content["linkages"].([]any)
	for _, rawRule := range rules {
		rule, _ := rawRule.(map[string]any)
		mappings, _ := rule["mappings"].([]any)
		for _, rawMapping := range mappings {
			mapping, _ := rawMapping.(map[string]any)
			if target, ok := mapping["targetFieldId"].(string); ok {
				targets[target] = true
			}
		}
	}
	return targets
}

func validateFieldFormulaArity(expr ast.Expr) error {
	var validationErr error
	ast.Inspect(expr, func(node ast.Node) bool {
		if validationErr != nil {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}
		min, max := 0, -1
		switch name.Name {
		case "FIELD", "NOT", "ISBLANK", "ISEMPTY", "LEN", "LOWER", "UPPER", "TRIM", "ABS":
			min, max = 1, 1
		case "IF":
			min, max = 3, 3
		case "ROUND":
			min, max = 1, 2
		case "TRUE", "FALSE":
			min, max = 0, 0
		case "CONCATENATE", "SUM", "AVERAGE", "MIN", "MAX":
			min = 1
		case "AND", "OR":
			min = 1
		}
		if len(call.Args) < min || (max >= 0 && len(call.Args) > max) {
			validationErr = fmt.Errorf("函数 %s 参数数量不匹配", name.Name)
			return false
		}
		return true
	})
	return validationErr
}

// ApplyCompiledFieldFormulas 按发布期拓扑序覆盖派生字段。调用方必须在业务
// 值终审后、校验器/幂等比较/持久化前执行，确保所有入口得到同一权威结果。
func ApplyCompiledFieldFormulas(raw model.JSONContent, content map[string]any, protocol int, values map[string]any) error {
	return evaluateCompiledFieldFormulas(raw, content, protocol, values, true)
}

// PreviewCompiledFieldFormulas 为显隐规则预计算派生值。它与权威执行消费同一
// 发布产物，但暂不校验目标字段约束；基础字段终审完成后仍必须再调用 Apply。
func PreviewCompiledFieldFormulas(raw model.JSONContent, content map[string]any, protocol int, values map[string]any) error {
	return evaluateCompiledFieldFormulas(raw, content, protocol, values, false)
}

func evaluateCompiledFieldFormulas(raw model.JSONContent, content map[string]any, protocol int, values map[string]any, validateResult bool) error {
	if protocol < fieldFormulaProtocolVersion {
		return nil
	}
	var compiled compiledFieldFormulas
	if err := json.Unmarshal(raw, &compiled); err != nil || compiled.Digest == "" {
		// 空公式集没有派生值可伪造；允许测试夹具及滚动升级期间的空产物。
		inner := content
		if value, ok := content["content"].(map[string]any); ok {
			inner = value
		}
		if formulas, ok := inner["fieldFormulas"].([]any); ok && len(formulas) == 0 {
			return nil
		}
		return fmt.Errorf("published field-formula artifact missing or malformed")
	}
	if compiled.Digest != submitRulesDigest(content) {
		return fmt.Errorf("published field-formula artifact does not match version content")
	}
	fields, err := buildSnapshotFields(content)
	if err != nil {
		return err
	}
	visible := make(map[string]bool, len(fields))
	for key := range fields {
		visible[key] = true
	}
	for _, formula := range compiled.Formulas {
		expr, err := parser.ParseExpr(formula.Program)
		if err != nil {
			return fmt.Errorf("stored field formula %s invalid", formula.ID)
		}
		result, err := evalSubmitExpr(expr, values, visible)
		if err != nil {
			return &FieldFormulaEvaluationError{Target: formula.Target, Cause: fmt.Errorf("公式计算失败: %w", err)}
		}
		value := submitDisplay(result)
		field, ok := fields[formula.Target]
		if !ok {
			return fmt.Errorf("field formula %s target missing", formula.ID)
		}
		if validateResult {
			if validationErrors := validateFieldValue(field, value); len(validationErrors) > 0 {
				return &FieldFormulaEvaluationError{Target: formula.Target, Cause: fmt.Errorf("公式结果不合法: %s", strings.Join(validationErrors, "；"))}
			}
		}
		values[formula.Target] = value
	}
	return nil
}
