package definition

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"evolyn/internal/engine/workflow/model"
)

// validDoc 最小合法 DSL：start → approval → end（校验器基准用例）。
func validDoc() *model.Document {
	return &model.Document{
		SchemaVersion: model.DSLSchemaVersion,
		Nodes: []model.Node{
			{Key: "start", Type: model.NodeTypeStart, Name: "发起"},
			{
				Key: "approval", Type: model.NodeTypeApproval, Name: "审批",
				Config: model.NodeConfig{
					ApprovalMode: model.ApprovalModeSingle,
					Assignee:     &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{1}},
				},
			},
			{Key: "end", Type: model.NodeTypeEnd, Name: "结束"},
		},
		Edges: []model.Edge{
			{Key: "e1", Source: "start", Target: "approval"},
			{Key: "e2", Source: "approval", Target: "end"},
		},
	}
}

func codesOf(errs ValidationErrors) []string {
	out := make([]string, 0, len(errs))
	for _, e := range errs {
		out = append(out, e.Code)
	}
	return out
}

func TestValidateValidDoc(t *testing.T) {
	v := NewValidator(nil)
	assert.Empty(t, v.Validate(validDoc()))
}

func TestValidateSchemaVersion(t *testing.T) {
	doc := validDoc()
	doc.SchemaVersion = "2.0"
	errs := NewValidator(nil).Validate(doc)
	require.Len(t, errs, 1)
	assert.Equal(t, ErrCodeSchemaVersion, errs[0].Code)
}

func TestValidateNodeKeyRules(t *testing.T) {
	// 非法命名（数字开头）；同步修正边引用，隔离出命名错误本身
	doc := validDoc()
	doc.Nodes[1].Key = "1bad"
	doc.Edges[0].Target = "1bad"
	doc.Edges[1].Source = "1bad"
	codes := codesOf(NewValidator(nil).Validate(doc))
	assert.Contains(t, codes, ErrCodeKeyInvalid)
	assert.NotContains(t, codes, ErrCodeKeyDuplicate)

	// 重复 key：追加同 key 节点（同时触发死节点错误，仅断言重复项）
	doc2 := validDoc()
	doc2.Nodes = append(doc2.Nodes, model.Node{Key: "start", Type: model.NodeTypeEnd})
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc2)), ErrCodeKeyDuplicate)
}

func TestValidateUnknownNodeType(t *testing.T) {
	doc := validDoc()
	doc.Nodes[1].Type = "parallel_gateway"
	errs := NewValidator(nil).Validate(doc)
	require.Len(t, errs, 1)
	assert.Equal(t, ErrCodeNodeUnknown, errs[0].Code)
}

func TestValidateApprovalConfig(t *testing.T) {
	// 缺少 assignee
	doc := validDoc()
	doc.Nodes[1].Config.Assignee = nil
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)

	// 非法审批模式
	doc = validDoc()
	doc.Nodes[1].Config.ApprovalMode = "vote"
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)

	// V1 仅支持 terminate
	doc = validDoc()
	doc.Nodes[1].Config.RejectStrategy = "back-to-prev"
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)

	// 会签必须提供 (0,1] 的 passRatio
	doc = validDoc()
	doc.Nodes[1].Config.ApprovalMode = model.ApprovalModeCountersign
	doc.Nodes[1].Config.PassRatio = 1.5
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)

	// 非法字段权限值
	doc = validDoc()
	doc.Nodes[1].Config.FormPermissions = map[string]model.FieldPermission{"amount": "write"}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeFieldPermission)
}

func TestValidateTimeoutAndReminderConfig(t *testing.T) {
	// 合法配置通过
	doc := validDoc()
	doc.Nodes[1].Config.Timeout = &model.TimeoutConfig{Seconds: 86400, Action: model.TimeoutActionApprove}
	doc.Nodes[1].Config.Reminder = &model.ReminderConfig{Seconds: 3600}
	assert.Empty(t, NewValidator(nil).Validate(doc))

	// 非法动作
	doc = validDoc()
	doc.Nodes[1].Config.Timeout = &model.TimeoutConfig{Seconds: 60, Action: "escalate"}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)

	// 秒数越界（0 与超过 30 天）
	doc = validDoc()
	doc.Nodes[1].Config.Timeout = &model.TimeoutConfig{Seconds: 0, Action: model.TimeoutActionReject}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)
	doc = validDoc()
	doc.Nodes[1].Config.Reminder = &model.ReminderConfig{Seconds: model.MaxJobSeconds + 1}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)
}

func TestValidateAssigneeCapabilityGate(t *testing.T) {
	// IAM 前置能力未落地的类型不允许发布（能力矩阵门，第 17.2 章）
	doc := validDoc()
	doc.Nodes[1].Config.Assignee = &model.AssigneeSpec{Type: model.AssigneeTypeStarterManager}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)

	doc = validDoc()
	doc.Nodes[1].Config.Assignee = &model.AssigneeSpec{Type: model.AssigneeTypeDepartment, DeptID: 1}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)

	// 已启用类型正常通过
	doc = validDoc()
	doc.Nodes[1].Config.Assignee = &model.AssigneeSpec{Type: model.AssigneeTypeRole, RoleCode: "finance"}
	assert.Empty(t, NewValidator(nil).Validate(doc))

	doc = validDoc()
	doc.Nodes[1].Config.Assignee = &model.AssigneeSpec{Type: model.AssigneeTypeFormField, FormField: "manager_id"}
	assert.Empty(t, NewValidator(nil).Validate(doc))
}

func TestValidateConditionEdges(t *testing.T) {
	// 合法条件节点：两条出边，一条带表达式一条 default
	doc := validDoc()
	doc.Nodes = append(doc.Nodes[:1],
		model.Node{Key: "cond", Type: model.NodeTypeCondition, Name: "条件"},
		doc.Nodes[2])
	doc.Edges = []model.Edge{
		{Key: "e1", Source: "start", Target: "cond"},
		{Key: "e2", Source: "cond", Target: "end", Condition: &model.EdgeCondition{Expression: "form.amount > 10000"}},
		{Key: "e3", Source: "cond", Target: "end"},
	}
	assert.Empty(t, NewValidator(nil).Validate(doc))

	// 缺 default 出边
	docBad := validDoc()
	docBad.Nodes = append(docBad.Nodes[:1],
		model.Node{Key: "cond", Type: model.NodeTypeCondition, Name: "条件"},
		docBad.Nodes[2])
	docBad.Edges = []model.Edge{
		{Key: "e1", Source: "start", Target: "cond"},
		{Key: "e2", Source: "cond", Target: "end", Condition: &model.EdgeCondition{Expression: "form.amount > 10000"}},
	}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(docBad)), ErrCodeConditionEdge)

	// 表达式引用白名单外变量 → 编译失败
	docExpr := validDoc()
	docExpr.Nodes = append(docExpr.Nodes[:1],
		model.Node{Key: "cond", Type: model.NodeTypeCondition, Name: "条件"},
		docExpr.Nodes[2])
	docExpr.Edges = []model.Edge{
		{Key: "e1", Source: "start", Target: "cond"},
		{Key: "e2", Source: "cond", Target: "end", Condition: &model.EdgeCondition{Expression: "env.SECRET"}},
		{Key: "e3", Source: "cond", Target: "end"},
	}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(docExpr)), ErrCodeExprInvalid)

	// 非条件节点出边不允许携带条件
	docMisplaced := validDoc()
	docMisplaced.Edges[0].Condition = &model.EdgeCondition{Expression: "form.amount > 1"}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(docMisplaced)), ErrCodeConditionEdge)
}

func TestValidateCardinalityAndDirection(t *testing.T) {
	// 无 start
	doc := validDoc()
	doc.Nodes = doc.Nodes[1:]
	doc.Edges = doc.Edges[1:]
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeStartCardinal)

	// 无 end
	doc = validDoc()
	doc.Nodes = doc.Nodes[:2]
	doc.Edges = doc.Edges[:1]
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeEndCardinal)

	// start 存在入边
	doc = validDoc()
	doc.Edges = append(doc.Edges, model.Edge{Key: "e3", Source: "end", Target: "start"})
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeEdgeDirection)
}

func TestValidateRefMissing(t *testing.T) {
	doc := validDoc()
	doc.Edges[0].Target = "ghost"
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeRefMissing)
}

func TestValidateGraph(t *testing.T) {
	// 死节点：不可达的游离 end
	doc := validDoc()
	doc.Nodes = append(doc.Nodes, model.Node{Key: "orphan_end", Type: model.NodeTypeEnd, Name: "孤儿"})
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeUnreachable)

	// 无到 end 的路径：approval 出边被删
	doc = validDoc()
	doc.Edges = doc.Edges[:1]
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeNoExit)

	// 环：V1 不支持
	doc = validDoc()
	doc.Nodes = append(doc.Nodes[:2], model.Node{Key: "extra", Type: model.NodeTypeApproval, Name: "加签"},
		doc.Nodes[2])
	// 重建边：start→approval→extra→end 与 extra→approval 成环
	doc.Edges = []model.Edge{
		{Key: "e1", Source: "start", Target: "approval"},
		{Key: "e2", Source: "approval", Target: "extra"},
		{Key: "e3", Source: "extra", Target: "end"},
		{Key: "e4", Source: "extra", Target: "approval"},
	}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeCycle)
}

func TestValidateCCAndServiceConfig(t *testing.T) {
	// cc 节点必须配置 recipients
	doc := validDoc()
	doc.Nodes[1].Type = model.NodeTypeCC
	doc.Nodes[1].Config = model.NodeConfig{}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)

	doc.Nodes[1].Type = model.NodeTypeCC
	doc.Nodes[1].Config = model.NodeConfig{
		Recipients: &model.AssigneeSpec{Type: model.AssigneeTypeUser, UserIDs: []uint{2}},
	}
	assert.Empty(t, NewValidator(nil).Validate(doc))

	// service 节点深度校验（Phase 7 定版）：合法 http 配置整体通过
	doc = validDoc()
	maxRetries := 2
	doc.Nodes[1].Type = model.NodeTypeService
	doc.Nodes[1].Config = model.NodeConfig{Service: &model.ServiceConfig{
		Action: "http", Method: "POST",
		URL:  "https://api.example.com/orders/{{form.orderId}}",
		Body: `{"amount": {{form.amount}}}`,
		ResponseMapping: []model.ServiceResponseMapping{
			{Variable: "orderNo", Path: "data.id", Required: true},
		},
		MaxRetries: &maxRetries,
	}}
	assert.Empty(t, NewValidator(nil).Validate(doc))

	// 非法 action / 缺 URL / 非 http(s) 前缀
	doc.Nodes[1].Config.Service.Action = "grpc"
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)
	doc.Nodes[1].Config.Service.Action = "http"
	doc.Nodes[1].Config.Service.URL = ""
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)
	doc.Nodes[1].Config.Service.URL = "ftp://api.example.com/x"
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)

	// 敏感请求头禁止明文写入 DSL（第 27 章密钥不入 DSL）
	doc.Nodes[1].Config.Service.URL = "https://api.example.com/x"
	doc.Nodes[1].Config.Service.Headers = map[string]string{"Authorization": "Bearer hard-coded-secret"}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)

	// 模板表达式非法 → EXPRESSION_INVALID；body 方法约束
	doc.Nodes[1].Config.Service.Headers = map[string]string{"X-Trace": "{{form.amount >}}"}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeExprInvalid)
	doc.Nodes[1].Config.Service.Headers = nil
	doc.Nodes[1].Config.Service.Method = "GET"
	doc.Nodes[1].Config.Service.Body = `{"x": 1}`
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)

	// 映射变量重复 / 非法命名 / 超时越界
	doc.Nodes[1].Config.Service.Method = "POST"
	doc.Nodes[1].Config.Service.Body = ""
	doc.Nodes[1].Config.Service.ResponseMapping = []model.ServiceResponseMapping{
		{Variable: "a"}, {Variable: "a"},
	}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)
	doc.Nodes[1].Config.Service.ResponseMapping = []model.ServiceResponseMapping{{Variable: "1bad"}}
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)
	doc.Nodes[1].Config.Service.ResponseMapping = nil
	doc.Nodes[1].Config.Service.TimeoutSeconds = 121
	assert.Contains(t, codesOf(NewValidator(nil).Validate(doc)), ErrCodeConfigInvalid)
	doc.Nodes[1].Config.Service.TimeoutSeconds = 0

	// Compile：模板段预编译产物登记 + 非法模板发布失败
	doc.Nodes[1].Config.Service.ResponseMapping = []model.ServiceResponseMapping{{Variable: "orderNo"}}
	compiled, err := Compile(doc, nil)
	require.NoError(t, err)
	require.Contains(t, compiled.ServiceTemplates, "approval")
	_, err = Compile(func() *model.Document {
		d := validDoc()
		d.Nodes[1].Type = model.NodeTypeService
		d.Nodes[1].Config = model.NodeConfig{Service: &model.ServiceConfig{
			Action: "http", URL: "https://api.example.com/{{bad >}}",
		}}
		return d
	}(), nil)
	assert.Error(t, err)
}

func TestCompilePrecompilesEdges(t *testing.T) {
	doc := validDoc()
	doc.Edges[1].Condition = &model.EdgeCondition{Expression: "form.amount > 10000"}
	compiled, err := Compile(doc, nil)
	require.NoError(t, err)
	require.Contains(t, compiled.EdgeExpressions, "e2")

	// 白名单环境求值：form.amount 命中
	out, err := compiled.EdgeExpressions["e2"].Eval(map[string]any{
		"form": map[string]any{"amount": 20000},
	})
	require.NoError(t, err)
	assert.Equal(t, true, out)

	// 非法表达式预编译失败
	_, err = Compile(validDoc(), nil)
	require.NoError(t, err) // 无条件边不报错
	bad := validDoc()
	bad.Edges[1].Condition = &model.EdgeCondition{Expression: "len(form.amount) > 1"}
	_, err = Compile(bad, nil)
	require.Error(t, err)
}
