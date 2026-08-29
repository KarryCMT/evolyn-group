package expression

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/vm"
)

// whitelistEnv 表达式白名单环境（第 15.3 / 16.2 章）：以强类型环境
// 编译，引用未声明变量或调用未定义函数在编译期即失败——这是沙箱的
// 第一道防线，不依赖运行时拦截。
type whitelistEnv struct {
	// Form 表单业务数据（form.amount 等字段访问按 map 键求值）
	Form map[string]any `expr:"form"`
	// Starter 发起人上下文（user_id / department_id / member_id）
	Starter map[string]any `expr:"starter"`
	// Variables 流程上下文变量（variables.xxx）
	Variables map[string]any `expr:"variables"`
}

// exprEngine Engine 的 Expr 实现。
type exprEngine struct{}

// NewExprEngine 构造内置 Expr 表达式引擎。
func NewExprEngine() Engine { return &exprEngine{} }

func (e *exprEngine) Compile(source string) (Program, error) {
	src := strings.TrimSpace(source)
	if src == "" {
		return nil, &ErrExpressionInvalid{Cause: fmt.Errorf("表达式为空")}
	}
	if len(src) > MaxExpressionLength {
		return nil, &ErrExpressionInvalid{Cause: fmt.Errorf("表达式长度 %d 超过上限 %d", len(src), MaxExpressionLength)}
	}
	// expr.Env 强类型白名单环境：未知标识符/函数调用编译期报错
	program, err := expr.Compile(src, expr.Env(whitelistEnv{}), expr.AsAny())
	if err != nil {
		return nil, &ErrExpressionInvalid{Cause: err}
	}
	// 深度防御：即使未来白名单环境放宽，也拒绝包含函数调用节点的程序
	//（第 16.2 章「禁止任意 Go 函数调用」）
	if hasCallNode(program.Node()) {
		return nil, &ErrExpressionInvalid{Cause: fmt.Errorf("表达式不允许函数调用")}
	}
	return &exprProgram{program: program}, nil
}

// hasCallNode AST 遍历是否存在调用类节点（函数/方法调用）。
func hasCallNode(node ast.Node) bool {
	found := false
	ast.Walk(&node, &callVisitor{found: &found})
	return found
}

// callVisitor ast.Visitor 实现：命中 CallNode 后停止下钻。
type callVisitor struct{ found *bool }

func (v *callVisitor) Visit(node *ast.Node) {
	switch (*node).(type) {
	case *ast.CallNode:
		*v.found = true
	case *ast.BuiltinNode:
		// len/filter/all 等内置函数与自定义函数调用同属禁用面（第 16.2 章）
		*v.found = true
	}
}

// exprProgram Program 的 Expr 实现（预编译产物不可变，可并发复用）。
type exprProgram struct {
	program *vm.Program
}

func (p *exprProgram) Eval(env map[string]any) (any, error) {
	scope := whitelistEnv{
		Form:      map[string]any{},
		Starter:   map[string]any{},
		Variables: map[string]any{},
	}
	for k, v := range env {
		switch k {
		case "form":
			if m, ok := v.(map[string]any); ok {
				scope.Form = m
			}
		case "starter":
			if m, ok := v.(map[string]any); ok {
				scope.Starter = m
			}
		case "variables":
			if m, ok := v.(map[string]any); ok {
				scope.Variables = m
			}
		}
	}
	out, err := expr.Run(p.program, scope)
	if err != nil {
		return nil, &ErrExpressionInvalid{Cause: err}
	}
	return out, nil
}
