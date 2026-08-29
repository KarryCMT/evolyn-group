// parallel.go 并行区域分析（Phase 8，第 31 章 Parallel Execution）：
// 校验器与发布预编译器共用的唯一图分析实现。并行 split 的每条分支路径
// 必须汇聚到同一 join（join token 计数的正确性前提），区域必须封闭：
// 不允许外部路径中途进入分支、不允许分支路径泄漏到区域外、不允许分支
// 内出现 End 或再嵌套 parallel（V1 冻结为扁平并行）。
package definition

import (
	"fmt"
	"sort"

	"evolyn/internal/engine/workflow/model"
)

// ParallelRegion 一个 split → join 并行区域的预编译产物（发布期冻结，
// 运行期按 join key / split key 查表，禁止重分析）。
type ParallelRegion struct {
	// SplitKey 并行分流节点 key
	SplitKey string
	// JoinKey 并行汇聚节点 key
	JoinKey string
	// BranchTargets 分支目标节点 key（按 split 出边声明顺序，即分支推进顺序）
	BranchTargets []string
}

// parallelIssue 区域分析问题（校验器转 ValidationError，编译器直接报错）。
type parallelIssue struct {
	Path    string
	Message string
}

// analyzeParallelRegions 对 DSL 文档做并行区域全量分析：
// 返回按节点声明顺序排列的区域清单与全部问题（一次报全，便于设计器展示）。
// 前置假设：节点/边引用完整（validateEdges 已保证），split 出边无条件
// （「仅 condition 出边可携带条件」规则已覆盖）。
func analyzeParallelRegions(doc *model.Document) ([]*ParallelRegion, []parallelIssue) {
	var issues []parallelIssue
	// 出边邻接表（保持声明顺序，即分支推进顺序）
	outEdges := make(map[string][]*model.Edge, len(doc.Nodes))
	for i := range doc.Edges {
		e := &doc.Edges[i]
		outEdges[e.Source] = append(outEdges[e.Source], e)
	}
	// 节点分类
	isJoin := make(map[string]bool, len(doc.Nodes))
	for i := range doc.Nodes {
		n := &doc.Nodes[i]
		if n.Type == model.NodeTypeParallel && n.Config.Parallel != nil && n.Config.Parallel.Role == model.ParallelRoleJoin {
			isJoin[n.Key] = true
		}
	}
	// reachUntilJoin 从 root 出发沿出边可达的节点集合，join 节点本身不展开
	// 也不纳入（分支区域的「出口」由出边单独判定）
	reachUntilJoin := func(root string) map[string]bool {
		seen := make(map[string]bool)
		stack := []string{root}
		for len(stack) > 0 {
			k := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if seen[k] || isJoin[k] {
				continue
			}
			seen[k] = true
			for _, e := range outEdges[k] {
				stack = append(stack, e.Target)
			}
		}
		return seen
	}

	var regions []*ParallelRegion
	for i := range doc.Nodes {
		n := &doc.Nodes[i]
		if n.Type != model.NodeTypeParallel || n.Config.Parallel == nil || n.Config.Parallel.Role != model.ParallelRoleSplit {
			continue
		}
		path := fmt.Sprintf("$.nodes[%d]", i)
		branches := outEdges[n.Key]
		if len(branches) < 2 {
			issues = append(issues, parallelIssue{Path: path,
				Message: fmt.Sprintf("并行分流节点 %q 至少需要两条出边分支", n.Key)})
			continue
		}
		region := &ParallelRegion{SplitKey: n.Key}
		// 分支区域与出口 join 逐支分析
		union := make(map[string]bool)
		var exitJoin string
		consistent := true
		for bi, e := range branches {
			branchPath := fmt.Sprintf("%s（分支 %d：%s）", path, bi+1, e.Key)
			if isJoin[e.Target] {
				issues = append(issues, parallelIssue{Path: branchPath,
					Message: fmt.Sprintf("并行分支不允许为空（%q 直连汇聚节点 %q）", n.Key, e.Target)})
				consistent = false
				continue
			}
			reach := reachUntilJoin(e.Target)
			// 分支区域内不允许出现 End / 再嵌套 parallel（V1 扁平并行）
			for key := range reach {
				node, ok := doc.NodeOf(key)
				if !ok {
					continue
				}
				switch node.Type {
				case model.NodeTypeEnd:
					issues = append(issues, parallelIssue{Path: branchPath,
						Message: fmt.Sprintf("并行分支路径不允许经过 End 节点 %q（分支必须汇聚到 join）", key)})
					consistent = false
				case model.NodeTypeParallel:
					issues = append(issues, parallelIssue{Path: branchPath,
						Message: fmt.Sprintf("并行分支路径不允许包含 parallel 节点 %q（V1 不支持嵌套并行）", key)})
					consistent = false
				}
			}
			// 本分支区域的出口 join 集合：区域节点（含分支起点）出边指向的 join
			exits := make(map[string]bool)
			checkExit := func(source string) {
				for _, oe := range outEdges[source] {
					if isJoin[oe.Target] {
						exits[oe.Target] = true
					}
				}
			}
			checkExit(e.Target)
			for key := range reach {
				checkExit(key)
			}
			if len(exits) == 0 {
				issues = append(issues, parallelIssue{Path: branchPath,
					Message: fmt.Sprintf("分支 %q 不存在到任何汇聚节点的路径", e.Target)})
				consistent = false
				continue
			}
			if len(exits) > 1 {
				issues = append(issues, parallelIssue{Path: branchPath,
					Message: fmt.Sprintf("分支 %q 汇聚到多个 join 节点（%v），一个 split 的全部分支必须汇聚到同一 join", e.Target, sortedKeys(exits))})
				consistent = false
				continue
			}
			for j := range exits {
				if exitJoin == "" {
					exitJoin = j
				} else if exitJoin != j {
					issues = append(issues, parallelIssue{Path: branchPath,
						Message: fmt.Sprintf("分支 %q 汇聚到 %q，与其他分支汇聚点 %q 不一致", e.Target, j, exitJoin)})
					consistent = false
				}
			}
			for key := range reach {
				union[key] = true
			}
		}
		if !consistent || exitJoin == "" {
			continue
		}
		region.JoinKey = exitJoin
		for _, e := range branches {
			region.BranchTargets = append(region.BranchTargets, e.Target)
		}
		// 区域封闭性：入口唯一（仅 split 出边）+ 出口唯一（仅通向 join）
		for ei := range doc.Edges {
			e := &doc.Edges[ei]
			edgePath := fmt.Sprintf("$.edges[%d]", ei)
			targetInRegion := union[e.Target] || e.Target == exitJoin
			sourceInRegion := union[e.Source]
			if targetInRegion && e.Source != n.Key && !sourceInRegion {
				issues = append(issues, parallelIssue{Path: edgePath,
					Message: fmt.Sprintf("外部路径不允许中途进入并行区域（%s → %s）", e.Source, e.Target)})
			}
			if sourceInRegion && !union[e.Target] && e.Target != exitJoin {
				issues = append(issues, parallelIssue{Path: edgePath,
					Message: fmt.Sprintf("并行分支路径不允许泄漏到区域外（%s → %s），必须汇聚到 %q", e.Source, e.Target, exitJoin)})
			}
		}
		regions = append(regions, region)
	}
	return regions, issues
}

// sortedKeys 排序 key（错误信息稳定输出）。
func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
