// Package assignment 审批人解析 SPI（第 17 章）。
//
// 原则：IAM 不存在的组织语义，Workflow 不得自行猜测或私建第二套组织
// 模型——部门负责人/直属主管 Resolver 在 IAM 补齐 leader/reporting 前置
// 能力前不得注册启用。解析结果在任务创建时一次性快照（v1.1 定版）。
package assignment

import (
	"context"
	"fmt"

	"evolyn/internal/engine/workflow/model"
	wfruntime "evolyn/internal/engine/workflow/runtime"
)

// ResolveInput 解析输入。
type ResolveInput struct {
	// Ctx 运行上下文（form/starter/variables 均可参与解析条件）
	Ctx *wfruntime.WorkflowContext
	// Spec 审批人规格（来自发布快照节点配置）
	Spec model.AssigneeSpec
}

// AssigneeResolver 审批人解析 SPI。实现必须幂等且只读组织数据；
// 解析不到任何审批人时返回稳定错误（WORKFLOW_ASSIGNEE_NOT_FOUND），
// 禁止静默返回空集跳过节点（v1.1 补充语义）。
type AssigneeResolver interface {
	Type() model.AssigneeType
	Resolve(ctx context.Context, input ResolveInput) ([]model.Actor, error)
}

// ErrAssigneeNotFound 解析不到审批人（出网映射 WORKFLOW_ASSIGNEE_NOT_FOUND；
// 兜底策略默认转交租户管理员，由平台层 Task Engine 落实）。
type ErrAssigneeNotFound struct {
	Type    model.AssigneeType
	Message string
}

func (e *ErrAssigneeNotFound) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("审批人解析为空（type=%s）", e.Type)
}

// Registry Resolver 注册表：按 AssigneeType 查找解析器。
type Registry interface {
	ResolverOf(assigneeType model.AssigneeType) (AssigneeResolver, bool)
}

// V1 Resolver 能力矩阵（第 17.2 章）：enabled 表示该类型当前允许注册
// 启用；条件开放类型在 IAM 前置能力落地前保持关闭（Phase 3 开工前排期）。
var v1ResolverCapabilities = map[model.AssigneeType]bool{
	model.AssigneeTypeUser:              true,  // 指定用户（IAM User 已具备）
	model.AssigneeTypeRole:              true,  // 指定角色（IAM Role 已具备）
	model.AssigneeTypeFormField:         true,  // 表单用户字段（Phase 3 Form 接入后生效）
	model.AssigneeTypeDepartment:        false, // 部门成员（可选，视 IAM membership 端口排期）
	model.AssigneeTypeDepartmentManager: false, // 部门负责人（需 IAM leader 字段前置）
	model.AssigneeTypeStarterManager:    false, // 发起人直属主管（需 IAM reporting 前置）
}

// ResolverEnabled 类型在 V1 当前里程碑是否允许启用。
func ResolverEnabled(t model.AssigneeType) bool { return v1ResolverCapabilities[t] }
