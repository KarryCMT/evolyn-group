// Package executor 节点执行器 SPI（第 12.1 章）。Executor 是引擎内核的
// 节点行为插件：不依赖 Gin/GORM，平台能力经 provider/ 窄端口获取。
package executor

import (
	"context"

	"evolyn/internal/engine/workflow/model"
)

// ExecuteInput 节点执行输入。
type ExecuteInput struct {
	// Ctx 运行上下文（含表达式环境）
	Ctx *model.WorkflowContext
	// Instance / NodeInstance 当前实例与节点实例
	Instance     *model.Instance
	NodeInstance *model.NodeInstance
	// Node 设计态节点（来自发布快照，禁止读草稿）
	Node model.Node
	// OperatorMemberID 触发执行的操作人（0=系统触发）
	OperatorMemberID uint
}

// ExecuteResult 节点执行结果：Wait 与 Complete 互斥，均为 false 表示
// 执行失败（错误经返回值传递，事务整体回滚）。
type ExecuteResult struct {
	// Wait 节点进入等待（人工审批创建任务后挂起，Runtime 停止推进）
	Wait bool
	// Complete 节点已同步完成，Runtime 继续 Navigator 寻路
	Complete bool
	// NextNodeKeys 覆盖默认寻路（condition 节点分支决策结果；
	// 空表示交由 Navigator 按出边判定）
	NextNodeKeys []string
	// CreatedTasks 本节点创建的人工任务（审批节点产出，供事务模板落库）
	CreatedTasks []model.Task
	// CreatedActors 任务参与人快照（与 CreatedTasks 一一对应）
	CreatedActors [][]model.Actor
	// CCRecipients 抄送对象快照（cc 节点产出，非审批任务不参与完成判定，
	// 第 10.6 章；落库由 Runtime 统一承担）
	CCRecipients []model.Actor
}

// NodeExecutor 节点执行器 SPI：按 NodeType 注册（第 12.1 章）。
// V1 执行器集合：Start / Approval / Condition / CC / End；
// ServiceExecutor Phase 7 开放。执行器只做节点语义，状态落库经
// Repository 端口由 Runtime 编排（第 13.2 章事务模板）。
type NodeExecutor interface {
	Type() model.NodeType
	Execute(ctx context.Context, input ExecuteInput) (ExecuteResult, error)
}

// Registry 执行器注册表：按 NodeType 查找执行器；未注册类型即配置
// 校验漏网之鱼，运行期直接报错（快速失败优于静默跳过）。
type Registry interface {
	ExecutorOf(nodeType model.NodeType) (NodeExecutor, bool)
}
