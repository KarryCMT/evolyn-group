// parallel_test.go 并行网关校验与预编译产物测试（Phase 8，第 31 章）。
package definition

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"evolyn/internal/engine/workflow/model"
)

// approvalNode 构造单人审批节点。
func approvalNode(key string, userID uint) model.Node {
	return model.Node{
		Key: key, Type: model.NodeTypeApproval, Name: key,
		Config: model.NodeConfig{
			ApprovalMode: model.ApprovalModeSingle,
			Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{userID}},
		},
	}
}

// parallelNode 构造并行网关节点。
func parallelNode(key string, role model.ParallelRole) model.Node {
	return model.Node{
		Key: key, Type: model.NodeTypeParallel, Name: key,
		Config: model.NodeConfig{Parallel: &model.ParallelConfig{Role: role}},
	}
}

// parallelDoc 合法双分支并行 DSL：start → split → {a, b} → join → end。
func parallelDoc() *model.Document {
	return &model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			parallelNode("split", model.ParallelRoleSplit),
			approvalNode("a", 2),
			approvalNode("b", 3),
			parallelNode("join", model.ParallelRoleJoin),
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "split"},
			{Key: "p1", Source: "split", Target: "a"},
			{Key: "p2", Source: "split", Target: "b"},
			{Key: "j1", Source: "a", Target: "join"},
			{Key: "j2", Source: "b", Target: "join"},
			{Key: "e2", Source: "join", Target: "end"},
		},
	}
}

func TestValidateParallelValid(t *testing.T) {
	v := NewValidator(nil)
	assert.Empty(t, v.Validate(parallelDoc()))
}

func TestValidateParallelConfigRequired(t *testing.T) {
	// 缺少 parallel 配置块
	doc := parallelDoc()
	doc.Nodes[1].Config.Parallel = nil
	codes := codesOf(NewValidator(nil).Validate(doc))
	assert.Contains(t, codes, ErrCodeParallelConfig)

	// role 非法
	doc2 := parallelDoc()
	doc2.Nodes[1].Config.Parallel.Role = "fork"
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc2)), ErrCodeParallelConfig)
}

func TestValidateParallelCardinality(t *testing.T) {
	v := NewValidator(nil)

	// split 仅一条出边（退化为顺序流，禁止）
	singleBranch := &model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			parallelNode("split", model.ParallelRoleSplit),
			approvalNode("a", 2),
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "split"},
			{Key: "p1", Source: "split", Target: "a"},
			{Key: "j1", Source: "a", Target: "end"},
		},
	}
	assert.Contains(t, codesOf(v.Validate(singleBranch)), ErrCodeParallelConfig)

	// join 仅一条入边（无并行汇入，禁止）
	singleJoin := &model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			parallelNode("split", model.ParallelRoleSplit),
			approvalNode("a", 2),
			approvalNode("b", 3),
			parallelNode("join", model.ParallelRoleJoin),
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "split"},
			{Key: "p1", Source: "split", Target: "a"},
			{Key: "p2", Source: "split", Target: "b"},
			{Key: "j1", Source: "a", Target: "join"},
			{Key: "e2", Source: "join", Target: "end"},
		},
	}
	assert.Contains(t, codesOf(v.Validate(singleJoin)), ErrCodeParallelConfig)
}

func TestValidateParallelNestedForbidden(t *testing.T) {
	// 分支内再嵌套 parallel（V1 冻结为扁平并行）
	doc := &model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			parallelNode("split", model.ParallelRoleSplit),
			parallelNode("inner_split", model.ParallelRoleSplit),
			approvalNode("a", 2),
			approvalNode("b", 3),
			parallelNode("inner_join", model.ParallelRoleJoin),
			parallelNode("join", model.ParallelRoleJoin),
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "split"},
			{Key: "p1", Source: "split", Target: "inner_split"},
			{Key: "p2", Source: "split", Target: "a"},
			{Key: "i1", Source: "inner_split", Target: "b"},
			{Key: "i2", Source: "inner_split", Target: "a"},
			{Key: "ij1", Source: "b", Target: "inner_join"},
			{Key: "ij2", Source: "a", Target: "inner_join"},
			{Key: "oj1", Source: "inner_join", Target: "join"},
			{Key: "oj2", Source: "a", Target: "join"},
			{Key: "e2", Source: "join", Target: "end"},
		},
	}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeParallelRegion)
}

func TestValidateParallelEndInBranchForbidden(t *testing.T) {
	// 分支路径直达 End（绕过 join），join 永不满足到达数
	doc := &model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			parallelNode("split", model.ParallelRoleSplit),
			approvalNode("a", 2),
			approvalNode("b", 3),
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "split"},
			{Key: "p1", Source: "split", Target: "a"},
			{Key: "p2", Source: "split", Target: "b"},
			{Key: "j1", Source: "a", Target: "end"},
			{Key: "j2", Source: "b", Target: "end"},
		},
	}
	// end 被两条分支汇入：退出集合为空（无 join），区域分析报错；
	// end 在分支区域内的错误同样触发
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeParallelRegion)
}

func TestValidateParallelExternalEntryForbidden(t *testing.T) {
	// 外部路径中途进入分支区域（绕过 split 的额外 token 入口）：
	// start → pre → split 为主链，pre → a 为非法旁路入口
	doc := &model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			approvalNode("pre", 9),
			parallelNode("split", model.ParallelRoleSplit),
			approvalNode("a", 2),
			approvalNode("b", 3),
			parallelNode("join", model.ParallelRoleJoin),
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "pre"},
			{Key: "e2", Source: "pre", Target: "split"},
			{Key: "ein", Source: "pre", Target: "a"}, // 外部路径进入分支区域
			{Key: "p1", Source: "split", Target: "a"},
			{Key: "p2", Source: "split", Target: "b"},
			{Key: "j1", Source: "a", Target: "join"},
			{Key: "j2", Source: "b", Target: "join"},
			{Key: "e3", Source: "join", Target: "end"},
		},
	}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeParallelRegion)
}

func TestValidateParallelEmptyBranchForbidden(t *testing.T) {
	// split 直连 join（空分支）
	doc := parallelDoc()
	doc.Edges[2] = model.Edge{Key: "p2", Source: "split", Target: "join"}
	// b 悬空会触发死节点，改为 b → end 之外不可达；直接删除 b 与其入边
	doc.Nodes = []model.Node{doc.Nodes[0], doc.Nodes[1], doc.Nodes[2], doc.Nodes[4], doc.Nodes[5]}
	doc.Edges = []model.Edge{
		{Key: "e1", Source: "start", Target: "split"},
		{Key: "p1", Source: "split", Target: "a"},
		{Key: "p2", Source: "split", Target: "join"},
		{Key: "j1", Source: "a", Target: "join"},
		{Key: "e2", Source: "join", Target: "end"},
	}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeParallelRegion)
}

func TestCompileParallelRegions(t *testing.T) {
	compiled, err := Compile(parallelDoc(), nil)
	require.NoError(t, err)
	require.Len(t, compiled.SplitRegions, 1)
	require.Len(t, compiled.JoinRegions, 1)
	region := compiled.SplitRegions["split"]
	require.NotNil(t, region)
	assert.Equal(t, "join", region.JoinKey)
	assert.Equal(t, []string{"a", "b"}, region.BranchTargets)
	assert.Same(t, region, compiled.JoinRegions["join"])
}
