package service

// v7 提交校验的服务端编译与执行器。该文件刻意只依赖标准库：公式不会进入
// JavaScript、SQL 或模板引擎；发布期将受控程序冻结到版本，提交只消费该产物。

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"evolyn/internal/platform/form/model"
)

const maxSubmitValidators = 50

var (
	submitFieldRef = regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)#`)
	submitTokenRef = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)
)

type compiledSubmitRules struct {
	Digest     string               `json:"digest"`
	Validators []compiledSubmitRule `json:"validators"`
}

type compiledSubmitRule struct {
	Index      int      `json:"index"`
	Program    string   `json:"program"`
	Remind     string   `json:"remind"`
	FailAction int      `json:"failAction"`
	Fields     []string `json:"fields"`
	Tokens     []string `json:"tokens"`
}

type submitValidatorError struct {
	Index  int      `json:"index"`
	Remind string   `json:"remind"`
	Fields []string `json:"fields"`
}

// ValidateSubmitRulePermissionGroups is the publish-time counterpart of the
// permission-group edit check. A draft can add a rule after groups already exist;
// publishing it is forbidden if an enabled add group cannot expose and repair every
// dependency, or if any enabled group cannot view confirmation-template fields.
func ValidateSubmitRulePermissionGroups(content map[string]any, protocol int, groups []model.AssetPermissionGroup) error {
	if protocol < 7 || len(groups) == 0 {
		return nil
	}
	raw, err := CompileSubmitRules(content, protocol)
	if err != nil {
		return err
	}
	var compiled compiledSubmitRules
	if err := json.Unmarshal(raw, &compiled); err != nil {
		return err
	}
	root := content
	if inner, ok := content["content"].(map[string]any); ok {
		root = inner
	}
	confirmFields := map[string]bool{}
	if confirm, ok := root["preSubmitConfirm"].(map[string]any); ok {
		for _, key := range []string{"title", "content"} {
			value, _ := confirm[key].(string)
			for _, field := range orderedRefs(value, submitTokenRef) {
				confirmFields[field] = true
			}
		}
	}
	for _, group := range groups {
		if !group.Enabled {
			continue
		}
		grants := map[string]model.PermissionFieldRule{}
		for _, grant := range group.FieldPermissions {
			grants[grant.Field] = grant
		}
		hasAdd := false
		for _, op := range group.Operations {
			if op == model.PermissionOpAdd {
				hasAdd = true
				break
			}
		}
		for field := range confirmFields {
			if grant, ok := grants[field]; !ok || !grant.Visible {
				return fmt.Errorf("permission group %s cannot view confirmation field %s", group.Code, field)
			}
		}
		if !hasAdd {
			continue
		}
		for _, validator := range compiled.Validators {
			for _, field := range validator.Fields {
				grant, ok := grants[field]
				if !ok || !grant.Visible || !grant.Editable {
					return fmt.Errorf("permission group %s cannot add records because validator field %s is not visible and editable", group.Code, field)
				}
			}
		}
	}
	return nil
}

// validateSubmitValidationProtocol mirrors the v7 structural contract before any
// draft is saved. Detailed parser diagnostics remain JSON-path scoped.
func validateSubmitValidationProtocol(content map[string]any, issues *[]SchemaIssue) {
	raw, ok := content["validators"].([]any)
	if !ok || len(raw) > maxSubmitValidators {
		*issues = append(*issues, SchemaIssue{Path: "content.validators", Message: "validators 必须是最多 50 项的数组"})
		return
	}
	for i, value := range raw {
		item, ok := value.(map[string]any)
		path := fmt.Sprintf("content.validators[%d]", i)
		if !ok {
			*issues = append(*issues, SchemaIssue{Path: path, Message: "提交校验规则必须是 JSON 对象"})
			continue
		}
		formula, fok := item["formula"].(string)
		remind, rok := item["remind"].(string)
		if !fok || strings.TrimSpace(formula) == "" || len(formula) > 4000 {
			*issues = append(*issues, SchemaIssue{Path: path + ".formula", Message: "formula 必须是 1–4000 字符的非空字符串"})
		}
		if !rok || strings.TrimSpace(remind) == "" || len(remind) > 500 {
			*issues = append(*issues, SchemaIssue{Path: path + ".remind", Message: "remind 必须是 1–500 字符的非空字符串"})
		}
		if _, ok := item["realtime"].(bool); !ok {
			*issues = append(*issues, SchemaIssue{Path: path + ".realtime", Message: "realtime 必须是 boolean"})
		}
		if action, ok := asInteger(item["failAction"]); !ok || (action != 0 && action != 1) {
			*issues = append(*issues, SchemaIssue{Path: path + ".failAction", Message: "failAction 必须是 0 或 1"})
		}
		if remark, ok := item["remark"].(string); !ok || len(remark) > 200 {
			*issues = append(*issues, SchemaIssue{Path: path + ".remark", Message: "remark 必须是最多 200 字符的字符串"})
		}
	}
	confirm, ok := content["preSubmitConfirm"].(map[string]any)
	if !ok {
		*issues = append(*issues, SchemaIssue{Path: "content.preSubmitConfirm", Message: "preSubmitConfirm 必须是 JSON 对象"})
		return
	}
	if _, ok := confirm["enable"].(bool); !ok {
		*issues = append(*issues, SchemaIssue{Path: "content.preSubmitConfirm.enable", Message: "enable 必须是 boolean"})
	}
	for _, key := range []string{"title", "content"} {
		value, ok := confirm[key].(string)
		max := 100
		if key == "content" {
			max = 1000
		}
		if !ok || strings.TrimSpace(value) == "" || len(value) > max {
			*issues = append(*issues, SchemaIssue{Path: "content.preSubmitConfirm." + key, Message: key + " 必须是非空且长度合法的模板字符串"})
		}
	}
}

// CompileSubmitRules returns a JSONB-safe immutable artifact. Program is a normalized
// form of the formula (field references become FIELD("key")); it is not user source.
func CompileSubmitRules(content map[string]any, protocol int) (model.JSONContent, error) {
	if protocol < 7 {
		return model.JSONContent(`{}`), nil
	}
	root := content
	if inner, ok := content["content"].(map[string]any); ok {
		content = inner
	}
	items, ok := documentItems(map[string]any{"content": content})
	if !ok {
		return nil, fmt.Errorf("content.items missing")
	}
	fields := buildSnapshotFieldsFromItems(items)
	raw, ok := content["validators"].([]any)
	if !ok {
		return nil, fmt.Errorf("validators must be an array")
	}
	if len(raw) > maxSubmitValidators {
		return nil, fmt.Errorf("too many validators")
	}
	compiled := compiledSubmitRules{Digest: submitRulesDigest(root), Validators: make([]compiledSubmitRule, 0, len(raw))}
	for i, rawRule := range raw {
		rule, ok := rawRule.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("validator %d must be an object", i)
		}
		formula, ok := rule["formula"].(string)
		if !ok || strings.TrimSpace(formula) == "" {
			return nil, fmt.Errorf("validator %d formula invalid", i)
		}
		remind, ok := rule["remind"].(string)
		if !ok || strings.TrimSpace(remind) == "" {
			return nil, fmt.Errorf("validator %d remind invalid", i)
		}
		action, ok := asInteger(rule["failAction"])
		if !ok || (action != 0 && action != 1) {
			return nil, fmt.Errorf("validator %d failAction invalid", i)
		}
		deps := orderedRefs(formula, submitFieldRef)
		for _, field := range deps {
			meta, exists := fields[field]
			if !exists {
				return nil, fmt.Errorf("validator %d references unknown field %s", i, field)
			}
			if !submitFormulaFieldAllowed(meta.widgetType) {
				return nil, fmt.Errorf("validator %d references unsupported field %s", i, field)
			}
		}
		program := submitFieldRef.ReplaceAllString(formula, `FIELD("$1")`)
		expr, err := parser.ParseExpr(program)
		if err != nil {
			return nil, fmt.Errorf("validator %d formula parse: %w", i, err)
		}
		if err := validateFormulaAST(expr); err != nil {
			return nil, fmt.Errorf("validator %d formula: %w", i, err)
		}
		if _, ok := rule["realtime"].(bool); !ok {
			return nil, fmt.Errorf("validator %d realtime invalid", i)
		}
		compiled.Validators = append(compiled.Validators, compiledSubmitRule{Index: i, Program: program, Remind: remind, FailAction: action, Fields: deps, Tokens: orderedRefs(remind, submitTokenRef)})
	}
	for _, key := range []string{"title", "content"} {
		confirm, _ := content["preSubmitConfirm"].(map[string]any)
		value, _ := confirm[key].(string)
		for _, field := range orderedRefs(value, submitTokenRef) {
			if _, ok := fields[field]; !ok {
				return nil, fmt.Errorf("preSubmitConfirm.%s references unknown field %s", key, field)
			}
		}
	}
	encoded, err := json.Marshal(compiled)
	return model.JSONContent(encoded), err
}

func orderedRefs(source string, pattern *regexp.Regexp) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, match := range pattern.FindAllStringSubmatch(source, -1) {
		if !seen[match[1]] {
			seen[match[1]] = true
			out = append(out, match[1])
		}
	}
	return out
}
func submitRulesDigest(content map[string]any) string {
	raw, _ := json.Marshal(content)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func submitFormulaFieldAllowed(widget string) bool {
	switch widget {
	case "text", "textarea", "phone", "number", "datetime", "radiogroup", "combo", "checkboxgroup", "combocheck":
		return true
	}
	return false
}

func validateFormulaAST(node ast.Expr) error {
	switch n := node.(type) {
	case *ast.BasicLit:
		if n.Kind != token.STRING && n.Kind != token.CHAR && n.Kind != token.INT && n.Kind != token.FLOAT {
			return fmt.Errorf("unsupported literal")
		}
	case *ast.CallExpr:
		name, ok := n.Fun.(*ast.Ident)
		if !ok {
			return fmt.Errorf("unsupported call")
		}
		allowed := map[string]bool{"FIELD": true, "AND": true, "OR": true, "NOT": true, "IF": true, "ISBLANK": true, "ISEMPTY": true, "LEN": true, "CONCATENATE": true, "LOWER": true, "UPPER": true, "TRIM": true, "ABS": true, "ROUND": true}
		if !allowed[name.Name] {
			return fmt.Errorf("function %s is not allowed", name.Name)
		}
		for _, a := range n.Args {
			if err := validateFormulaAST(a); err != nil {
				return err
			}
		}
	case *ast.BinaryExpr:
		if err := validateFormulaAST(n.X); err != nil {
			return err
		}
		return validateFormulaAST(n.Y)
	case *ast.UnaryExpr:
		return validateFormulaAST(n.X)
	case *ast.ParenExpr:
		return validateFormulaAST(n.X)
	case *ast.Ident:
		if n.Name != "true" && n.Name != "false" {
			return fmt.Errorf("unknown identifier %s", n.Name)
		}
	default:
		return fmt.Errorf("unsupported expression")
	}
	return nil
}

// ValidateCompiledSubmitRules evaluates only the stored program. An artifact whose
// digest no longer matches its version content is a deployment/configuration error.
func ValidateCompiledSubmitRules(raw model.JSONContent, content map[string]any, protocol int, values map[string]any) ([]submitValidatorError, error) {
	if protocol < 7 {
		return nil, nil
	}
	var compiled compiledSubmitRules
	if err := json.Unmarshal(raw, &compiled); err != nil || compiled.Digest == "" {
		return nil, fmt.Errorf("published submit-rule artifact missing or malformed")
	}
	if compiled.Digest != submitRulesDigest(content) {
		return nil, fmt.Errorf("published submit-rule artifact does not match version content")
	}
	fields, err := buildSnapshotFields(content)
	if err != nil {
		return nil, err
	}
	visibility := effectiveFieldVisibility(fields, content, nil, func(name string) any { return values[name] }, "")
	failures := []submitValidatorError{}
	for _, rule := range compiled.Validators {
		expr, err := parser.ParseExpr(rule.Program)
		if err != nil {
			return nil, fmt.Errorf("stored submit-rule program invalid")
		}
		result, err := evalSubmitExpr(expr, values, visibility)
		passed, ok := result.(bool)
		if err != nil || !ok || !passed {
			if rule.FailAction == 0 {
				failures = append(failures, submitValidatorError{Index: rule.Index, Remind: renderSubmitRemind(rule.Remind, values, visibility), Fields: rule.Fields})
			}
		}
	}
	sort.Slice(failures, func(i, j int) bool { return failures[i].Index < failures[j].Index })
	return failures, nil
}

func renderSubmitRemind(source string, values map[string]any, visible map[string]bool) string {
	return submitTokenRef.ReplaceAllStringFunc(source, func(raw string) string {
		key := submitTokenRef.FindStringSubmatch(raw)[1]
		if !visible[key] {
			return ""
		}
		return submitDisplay(values[key])
	})
}
func submitDisplay(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case []any:
		parts := make([]string, len(x))
		for i, v := range x {
			parts[i] = submitDisplay(v)
		}
		return strings.Join(parts, "、")
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	default:
		b, _ := json.Marshal(x)
		return string(b)
	}
}

func evalSubmitExpr(node ast.Expr, values map[string]any, visible map[string]bool) (any, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		if n.Kind == token.STRING || n.Kind == token.CHAR {
			text, err := strconv.Unquote(n.Value)
			return text, err
		}
		return strconv.ParseFloat(n.Value, 64)
	case *ast.Ident:
		if n.Name == "true" {
			return true, nil
		}
		if n.Name == "false" {
			return false, nil
		}
		return nil, fmt.Errorf("unknown identifier")
	case *ast.ParenExpr:
		return evalSubmitExpr(n.X, values, visible)
	case *ast.UnaryExpr:
		v, e := evalSubmitExpr(n.X, values, visible)
		if e != nil {
			return nil, e
		}
		number, e := submitNumber(v)
		if e != nil {
			return nil, e
		}
		if n.Op == token.SUB {
			return -number, nil
		}
		return number, nil
	case *ast.BinaryExpr:
		l, e := evalSubmitExpr(n.X, values, visible)
		if e != nil {
			return nil, e
		}
		r, e := evalSubmitExpr(n.Y, values, visible)
		if e != nil {
			return nil, e
		}
		return evalSubmitBinary(n.Op, l, r)
	case *ast.CallExpr:
		name := n.Fun.(*ast.Ident).Name
		args := make([]any, len(n.Args))
		for i, a := range n.Args {
			v, e := evalSubmitExpr(a, values, visible)
			if e != nil {
				return nil, e
			}
			args[i] = v
		}
		if name == "FIELD" {
			if len(args) != 1 {
				return nil, fmt.Errorf("FIELD arity")
			}
			key, _ := args[0].(string)
			if !visible[key] {
				return nil, nil
			}
			return values[key], nil
		}
		return evalSubmitFunction(name, args)
	}
	return nil, fmt.Errorf("unsupported program")
}
func submitNumber(v any) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case int:
		return float64(x), nil
	case json.Number:
		return x.Float64()
	case string:
		return strconv.ParseFloat(x, 64)
	}
	return 0, fmt.Errorf("not a number")
}
func evalSubmitBinary(op token.Token, l, r any) (any, error) {
	switch op {
	case token.ADD, token.SUB, token.MUL, token.QUO:
		a, e := submitNumber(l)
		if e != nil {
			return nil, e
		}
		b, e := submitNumber(r)
		if e != nil {
			return nil, e
		}
		if op == token.ADD {
			return a + b, nil
		}
		if op == token.SUB {
			return a - b, nil
		}
		if op == token.MUL {
			return a * b, nil
		}
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return a / b, nil
	case token.EQL:
		return submitCompare(l, r) == 0, nil
	case token.NEQ:
		return submitCompare(l, r) != 0, nil
	case token.GTR:
		return submitCompare(l, r) > 0, nil
	case token.GEQ:
		return submitCompare(l, r) >= 0, nil
	case token.LSS:
		return submitCompare(l, r) < 0, nil
	case token.LEQ:
		return submitCompare(l, r) <= 0, nil
	}
	return nil, fmt.Errorf("operator not allowed")
}
func submitCompare(a, b any) int {
	if af, e := submitNumber(a); e == nil {
		if bf, e := submitNumber(b); e == nil {
			if af < bf {
				return -1
			}
			if af > bf {
				return 1
			}
			return 0
		}
	}
	return strings.Compare(submitDisplay(a), submitDisplay(b))
}
func evalSubmitFunction(name string, args []any) (any, error) {
	switch name {
	case "AND":
		for _, a := range args {
			if b, ok := a.(bool); !ok || !b {
				return false, nil
			}
		}
		return true, nil
	case "OR":
		for _, a := range args {
			if b, _ := a.(bool); b {
				return true, nil
			}
		}
		return false, nil
	case "NOT":
		b, ok := args[0].(bool)
		return !b, func() error {
			if !ok {
				return fmt.Errorf("NOT expects boolean")
			}
			return nil
		}()
	case "IF":
		b, _ := args[0].(bool)
		if b {
			return args[1], nil
		}
		return args[2], nil
	case "ISBLANK", "ISEMPTY":
		return args[0] == nil || submitDisplay(args[0]) == "", nil
	case "LEN":
		return float64(len([]rune(submitDisplay(args[0])))), nil
	case "CONCATENATE":
		var b strings.Builder
		for _, a := range args {
			b.WriteString(submitDisplay(a))
		}
		return b.String(), nil
	case "LOWER":
		return strings.ToLower(submitDisplay(args[0])), nil
	case "UPPER":
		return strings.ToUpper(submitDisplay(args[0])), nil
	case "TRIM":
		return strings.Join(strings.Fields(submitDisplay(args[0])), " "), nil
	case "ABS":
		n, e := submitNumber(args[0])
		return math.Abs(n), e
	case "ROUND":
		n, e := submitNumber(args[0])
		if e != nil {
			return nil, e
		}
		p := 0
		if len(args) > 1 {
			x, e := submitNumber(args[1])
			if e != nil {
				return nil, e
			}
			p = int(x)
		}
		f := math.Pow10(p)
		return math.Round(n*f) / f, nil
	}
	return nil, fmt.Errorf("function not allowed")
}

var _ = bytes.Compare // keeps bytes imported for compiler versions that elide SHA helper internals differently
