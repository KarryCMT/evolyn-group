// Package runtime 流程运行时契约（Phase 0 冻结 WorkflowContext 与
// Navigator SPI；Runtime 推进主逻辑 Phase 2 落地）。
package runtime

import "evolyn/internal/engine/workflow/model"

// UserContext 发起人/操作人上下文（表达式白名单 starter.* 数据源，
// 第 15.3 章）。字段即表达式可引用面，新增字段必须同步表达式文档。
type UserContext struct {
	// MemberID 租户成员 ID
	MemberID uint
	// UserID 成员关联用户标识（组织维度键）
	UserID string
	// DepartmentID 所属部门 ID
	DepartmentID uint
	// DisplayName 显示名快照
	DisplayName string
}

// WorkflowContext 流程运行上下文：一次节点执行/任务操作期间传递的
// 只读视图（第 15.3 章）。TenantID 来自平台租户链，不接受客户端直传；
// Form 为发起时绑定的 Form Snapshot 业务数据；Variables 为流程变量。
type WorkflowContext struct {
	TenantID  uint
	Instance  InstanceContext
	Starter   UserContext
	Form      map[string]any
	Variables map[string]any
}

// InstanceContext 实例绑定上下文（冻结绑定关系，运行期不可变更）。
type InstanceContext struct {
	InstanceID          uint
	DefinitionCode      string
	DefinitionVersionNo int
	BusinessType        string
	BusinessID          string
	// FormVersionID 发起时冻结的表单版本（渲染/校验/表达式均以此为准）
	FormVersionID uint
}

// ExpressionEnv 由 WorkflowContext 构造表达式白名单环境
// （form / starter / variables 三组变量，键名冻结）。
func (c *WorkflowContext) ExpressionEnv() map[string]any {
	starter := map[string]any{
		"user_id":       c.Starter.UserID,
		"member_id":     c.Starter.MemberID,
		"department_id": c.Starter.DepartmentID,
	}
	if c.Form == nil {
		c.Form = map[string]any{}
	}
	if c.Variables == nil {
		c.Variables = map[string]any{}
	}
	return map[string]any{
		"form":      c.Form,
		"starter":   starter,
		"variables": c.Variables,
	}
}

// Navigator 寻路 SPI（Phase 2 实现顺序流寻路，Phase 8 扩展并行）：
// 按发布快照与当前节点实例判定下一个（组）节点。实现必须从
// dsl_snapshot 读取节点配置，不读草稿。
type Navigator interface {
	// FindNext 返回当前节点完成后应进入的节点 key 集合（条件节点按
	// 边表达式求值取首个命中分支；并行场景返回多节点，Phase 8 启用）。
	FindNext(ctx *WorkflowContext, instance *model.Instance, nodeKey string) ([]string, error)
}
