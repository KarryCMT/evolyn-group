package runtime

import (
	"evolyn/internal/engine/workflow/definition"
	"evolyn/internal/engine/workflow/expression"
	"evolyn/internal/engine/workflow/model"
)

// Navigator 顺序流寻路器（V1 无并行；并行 split/join 随 Phase 8 扩展为
// 多节点返回）。实现必须基于发布快照的编译产物，不读草稿。
type Navigator struct {
	expressions expression.Engine
}

// NewNavigator 构造寻路器（expressions 为 nil 时使用内置 Expr 引擎）。
func NewNavigator(expressions expression.Engine) *Navigator {
	if expressions == nil {
		expressions = expression.NewExprEngine()
	}
	return &Navigator{expressions: expressions}
}

// FindNext 判定当前节点完成后应进入的节点 key（第 12.2 章「Navigator.FindNext」）：
//   - 普通节点：V1 仅允许单出边（多出边属未支持语义，快速失败）；
//   - 条件节点：按快照出边声明顺序求值边表达式，取首个命中的分支；
//     全部未命中时回落 default（无条件）出边——校验器已保证 default 必存在。
//
// 兼容入口：按需即时编译表达式（测试/直接调用方使用）；生产链路走
// FindNextCompiled 取发布预编译产物。
func (n *Navigator) FindNext(env *model.WorkflowContext, doc *model.Document, nodeKey string) (string, error) {
	node, ok := doc.NodeOf(nodeKey)
	if !ok {
		return "", ErrRouteStuck
	}
	edges := n.outgoingEdges(doc, nodeKey)
	if len(edges) == 0 {
		return "", ErrRouteStuck
	}
	if node.Type != model.NodeTypeCondition {
		if len(edges) > 1 {
			// 条件分支只允许出现在 condition 节点（校验器同口径，双保险）
			return "", ErrRouteStuck
		}
		return edges[0].Target, nil
	}
	return n.pickConditionBranch(env, edges)
}

// FindNextCompiled 语义同 FindNext，但条件表达式取发布预编译产物
// （CompiledDefinition.EdgeExpressions，第 16 章「发布时预编译、运行期
// 禁止重编译」）；产物缺失的出边防御性回退到即时编译。
func (n *Navigator) FindNextCompiled(compiled *definition.CompiledDefinition, env *model.WorkflowContext, nodeKey string) (string, error) {
	doc := compiled.Document
	node, ok := doc.NodeOf(nodeKey)
	if !ok {
		return "", ErrRouteStuck
	}
	edges := n.outgoingEdges(doc, nodeKey)
	if len(edges) == 0 {
		return "", ErrRouteStuck
	}
	if node.Type != model.NodeTypeCondition {
		if len(edges) > 1 {
			return "", ErrRouteStuck
		}
		return edges[0].Target, nil
	}
	var defaultTarget string
	for _, edge := range edges {
		if edge.Condition == nil {
			defaultTarget = edge.Target
			continue
		}
		program, ok := compiled.EdgeExpressions[edge.Key]
		if !ok {
			// 防御：快照含未编译表达式（正常发布链路不会发生）
			recompiled, err := n.expressions.Compile(edge.Condition.Expression)
			if err != nil {
				return "", err
			}
			program = recompiled
		}
		hit, err := program.Eval(env.ExpressionEnv())
		if err != nil {
			return "", err
		}
		if truthy(hit) {
			return edge.Target, nil
		}
	}
	if defaultTarget == "" {
		return "", ErrRouteStuck
	}
	return defaultTarget, nil
}

// outgoingEdges 返回节点出边（保持快照声明顺序，条件求值以此为准）。
func (n *Navigator) outgoingEdges(doc *model.Document, nodeKey string) []model.Edge {
	edges := make([]model.Edge, 0)
	for i := range doc.Edges {
		if doc.Edges[i].Source == nodeKey {
			edges = append(edges, doc.Edges[i])
		}
	}
	return edges
}

// pickConditionBranch 条件分支决策：首个命中表达式的出边；全未命中走 default。
func (n *Navigator) pickConditionBranch(env *model.WorkflowContext, edges []model.Edge) (string, error) {
	var defaultTarget string
	for _, edge := range edges {
		if edge.Condition == nil {
			defaultTarget = edge.Target
			continue
		}
		program, err := n.expressions.Compile(edge.Condition.Expression)
		if err != nil {
			return "", err
		}
		hit, err := program.Eval(env.ExpressionEnv())
		if err != nil {
			return "", err
		}
		if truthy(hit) {
			return edge.Target, nil
		}
	}
	if defaultTarget == "" {
		return "", ErrRouteStuck
	}
	return defaultTarget, nil
}

// truthy 求值结果布尔化（Expr 布尔直接命中；非布尔结果视为未命中）。
func truthy(v any) bool {
	hit, ok := v.(bool)
	return ok && hit
}
