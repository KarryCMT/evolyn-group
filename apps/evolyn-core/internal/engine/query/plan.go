package query

// BuildLogicalPlan 将规范化 AST 投影为存储无关计划；适配器需要在该计划上合并
// 当前租户、字段权限和行级数据范围，再选择 JSONB 或物理表编译路径。
func BuildLogicalPlan(document Document) LogicalPlan {
	complexity := measureExpression(document.Filter)
	return LogicalPlan{
		Filter:     document.Filter,
		Projection: cloneStrings(document.Projection),
		GroupBy:    cloneStrings(document.GroupBy),
		Aggregates: append([]Aggregate{}, document.Aggregates...),
		Sorts:      append([]Sort{}, document.Sorts...),
		Paging:     document.Paging,
		Aggregate:  len(document.GroupBy) > 0 || len(document.Aggregates) > 0,
		Complexity: complexity,
	}
}

func measureExpression(expression *Expression) Complexity {
	if expression == nil {
		return Complexity{}
	}
	if expression.Type == "condition" {
		return Complexity{Depth: 1, Conditions: 1}
	}
	result := Complexity{Depth: 1}
	for i := range expression.Children {
		child := measureExpression(&expression.Children[i])
		if child.Depth+1 > result.Depth {
			result.Depth = child.Depth + 1
		}
		result.Conditions += child.Conditions
	}
	return result
}

func cloneStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string{}, values...)
}
