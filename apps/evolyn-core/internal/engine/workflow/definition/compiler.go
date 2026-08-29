package definition

import (
	"evolyn/internal/engine/workflow/expression"
	"evolyn/internal/engine/workflow/model"
)

// CompiledDefinition 发布预编译产物（第 16 章「发布时预编译表达式」）：
// 随 dsl_snapshot 一同冻结，运行期直接取用编译产物求值，禁止运行期重编译。
type CompiledDefinition struct {
	// Document 来源 DSL 快照
	Document *model.Document
	// EdgeExpressions 出边 key → 预编译条件表达式
	EdgeExpressions map[string]expression.Program
}

// Compile 校验并预编译 DSL 文档：任一表达式编译失败即返回错误，
// 发布流程必须先经 Validator 校验再经 Compile 产出快照伴随产物。
// Phase 0 仅冻结契约与行为；持久化与版本号分配由 Phase 1 发布服务承担。
func Compile(doc *model.Document, engine expression.Engine) (*CompiledDefinition, error) {
	if engine == nil {
		engine = expression.NewExprEngine()
	}
	compiled := &CompiledDefinition{
		Document:        doc,
		EdgeExpressions: make(map[string]expression.Program),
	}
	for i := range doc.Edges {
		e := &doc.Edges[i]
		if e.Condition == nil {
			continue
		}
		program, err := engine.Compile(e.Condition.Expression)
		if err != nil {
			return nil, err
		}
		compiled.EdgeExpressions[e.Key] = program
	}
	return compiled, nil
}
