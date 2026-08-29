package definition

import (
	"evolyn/internal/engine/workflow/expression"
	"evolyn/internal/engine/workflow/model"
)

// ServiceTemplates service 节点模板预编译产物（Phase 7）：URL/请求头/请求体
// 的 {{expr}} 段在发布期编译冻结，运行期仅求值（第 16 章「发布时预编译」）。
type ServiceTemplates struct {
	URL     []expression.TemplateSegment
	Headers map[string][]expression.TemplateSegment
	Body    []expression.TemplateSegment
}

// CompiledDefinition 发布预编译产物（第 16 章「发布时预编译表达式」）：
// 随 dsl_snapshot 一同冻结，运行期直接取用编译产物求值，禁止运行期重编译。
type CompiledDefinition struct {
	// Document 来源 DSL 快照
	Document *model.Document
	// EdgeExpressions 出边 key → 预编译条件表达式
	EdgeExpressions map[string]expression.Program
	// ServiceTemplates service 节点 key → 模板编译产物（Phase 7；
	// 无模板段亦登记，区分「节点存在」与「未编译」）
	ServiceTemplates map[string]*ServiceTemplates
}

// Compile 校验并预编译 DSL 文档：任一表达式编译失败即返回错误，
// 发布流程必须先经 Validator 校验再经 Compile 产出快照伴随产物。
// Phase 0 仅冻结契约与行为；持久化与版本号分配由 Phase 1 发布服务承担。
func Compile(doc *model.Document, engine expression.Engine) (*CompiledDefinition, error) {
	if engine == nil {
		engine = expression.NewExprEngine()
	}
	compiled := &CompiledDefinition{
		Document:         doc,
		EdgeExpressions:  make(map[string]expression.Program),
		ServiceTemplates: make(map[string]*ServiceTemplates),
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
	// service 节点模板预编译（Phase 7）：URL/Headers/Body 任一段编译失败
	// 即发布失败，杜绝带病快照进入运行时
	for i := range doc.Nodes {
		n := &doc.Nodes[i]
		if n.Type != model.NodeTypeService || n.Config.Service == nil {
			continue
		}
		cfg := n.Config.Service
		templates := &ServiceTemplates{Headers: make(map[string][]expression.TemplateSegment, len(cfg.Headers))}
		var err error
		if templates.URL, err = expression.ParseTemplate(engine, cfg.URL); err != nil {
			return nil, err
		}
		for name, value := range cfg.Headers {
			if templates.Headers[name], err = expression.ParseTemplate(engine, value); err != nil {
				return nil, err
			}
		}
		if cfg.Body != "" {
			if templates.Body, err = expression.ParseTemplate(engine, cfg.Body); err != nil {
				return nil, err
			}
		}
		compiled.ServiceTemplates[n.Key] = templates
	}
	return compiled, nil
}
