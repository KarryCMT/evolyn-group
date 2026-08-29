package repository

import (
	"context"

	"evolyn/internal/engine/workflow/model"
)

// TaskRepository 人工任务仓储（含参与人快照）。
// 并发纪律：任务状态变更必须先经 FindTaskByIDForUpdate 行锁读取再写，
// 双击 Approve 等并发场景由行锁 + PENDING 校验保证只推进一次（第 13.2 章）。
type TaskRepository interface {
	// CreateTask 创建任务（会签/或签一次创建多任务）
	CreateTask(ctx context.Context, task *model.Task) error
	// ReplaceActors 全量替换任务参与人快照（任务创建时一次性写入）
	ReplaceActors(ctx context.Context, taskID uint, actors []model.Actor) error
	// FindTaskByIDForUpdate 行锁读取任务（租户过滤）
	FindTaskByIDForUpdate(ctx context.Context, tenantID, taskID uint) (*model.Task, error)
	// ListTasksByInstance 实例任务列表（Withdraw 资格与节点完成判定）
	ListTasksByInstance(ctx context.Context, instanceID uint) ([]model.Task, error)
	// ListTasksByNodeInstance 节点实例任务列表（会签阈值判定）
	ListTasksByNodeInstance(ctx context.Context, nodeInstanceID uint) ([]model.Task, error)
	// ListActorsOfTask 任务参与人快照
	ListActorsOfTask(ctx context.Context, taskID uint) ([]model.Actor, error)
	// SaveTask 保存任务状态（迁移先经 task 包状态机校验）
	SaveTask(ctx context.Context, task *model.Task) error
	// CancelPendingTasksByNode 批量取消节点下 PENDING 任务
	//（或签淘汰/节点完成/驳回终止联动；返回受影响行数）
	CancelPendingTasksByNode(ctx context.Context, nodeInstanceID uint) (int64, error)
	// CancelPendingTasksByInstance 批量取消实例下全部 PENDING 任务
	//（撤回/管理员终止联动；返回受影响行数）
	CancelPendingTasksByInstance(ctx context.Context, instanceID uint) (int64, error)
}

// CCRepository 抄送记录仓储（追加写，禁止更新；第 10.6 章）。
type CCRepository interface {
	// CreateCCRecords 批量落抄送记录（抄送节点执行时一次性写入）
	CreateCCRecords(ctx context.Context, records []model.CCRecord) error
}

// NodeInstanceRepository 节点实例仓储。
type NodeInstanceRepository interface {
	CreateNodeInstance(ctx context.Context, nodeInstance *model.NodeInstance) error
	FindNodeInstanceByID(ctx context.Context, tenantID, nodeInstanceID uint) (*model.NodeInstance, error)
	ListNodeInstancesByInstance(ctx context.Context, instanceID uint) ([]model.NodeInstance, error)
	SaveNodeInstance(ctx context.Context, nodeInstance *model.NodeInstance) error
}

// OperationRepository 操作流水仓储：追加写，禁止更新。
type OperationRepository interface {
	// AppendOperation 与状态变更同事务追加操作记录（第 13.2 章第 8 步）
	AppendOperation(ctx context.Context, operation *model.Operation) error
	// ListOperationsByInstance 实例操作时间线（按 id 升序）
	ListOperationsByInstance(ctx context.Context, instanceID uint) ([]model.Operation, error)
}

// VariableRepository 流程变量仓储。
type VariableRepository interface {
	SaveVariable(ctx context.Context, variable *model.Variable) error
	ListVariablesByInstance(ctx context.Context, instanceID uint) ([]model.Variable, error)
}
