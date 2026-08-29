package task

import (
	"context"
	"errors"
	"fmt"

	"evolyn/internal/engine/workflow/event"
	"evolyn/internal/engine/workflow/model"
	"evolyn/internal/engine/workflow/provider"
	"evolyn/internal/engine/workflow/repository"
)

// 任务域 sentinel 错误（平台层 errors.Is → WORKFLOW_* 稳定码）。
var (
	// ErrTaskNotFound 任务不存在或无权访问
	ErrTaskNotFound = errors.New("workflow task not found")
	// ErrTaskNotPending 任务不在 PENDING（已被处理/取消/转办；双击防护命中点）
	ErrTaskNotPending = errors.New("workflow task not pending")
	// ErrTaskForbidden 当前操作者不是该任务的参与人（实例级校验，第 21/27 章）
	ErrTaskForbidden = errors.New("operator is not an actor of the task")
	// ErrInstanceNotFound 实例不存在或无权访问
	ErrInstanceNotFound = errors.New("workflow instance not found")
	// ErrInstanceNotRunning 实例不在 RUNNING（终态实例不可再操作）
	ErrInstanceNotRunning = errors.New("workflow instance not running")
)

// Engine 人工任务引擎：任务级动作的状态裁决与落库（第 12.2 章「Task Engine」）。
// 与 Runtime 分离——本包只负责任务与节点完成判定，不做寻路与节点推进；
// 事务由平台层包裹（行锁 FOR UPDATE 在仓储适配层实现）。
type Engine struct {
	tasks      repository.TaskRepository
	instances  repository.InstanceRepository
	operations repository.OperationRepository
	publisher  provider.EventPublisher
}

// NewEngine 构造任务引擎（publisher 可为 nil：跳过事件发布，便于单测）。
func NewEngine(
	tasks repository.TaskRepository,
	instances repository.InstanceRepository,
	operations repository.OperationRepository,
	publisher provider.EventPublisher,
) *Engine {
	return &Engine{tasks: tasks, instances: instances, operations: operations, publisher: publisher}
}

// ApproveInput 同意动作输入。
type ApproveInput struct {
	TenantID uint
	TaskID   uint
	// OperatorMemberID 操作人成员 ID；必须命中任务参与人快照（实例级校验，
	// RBAC 只决定操作能力，第 21 章）
	OperatorMemberID uint
	// Comment 审批意见（入 operation 载荷）
	Comment string
}

// ApproveOutcome 同意动作产出：任务与实例的最新状态（节点是否达成完成条件
// 由 Runtime 结合审批模式判定后推进）。
type ApproveOutcome struct {
	Task           *model.Task
	Instance       *model.Instance
	NodeInstanceID uint
	NodeKey        string
}

// Approve 同意任务（第 13.2 章事务模板 1–8 步）：
// 行锁取任务 → PENDING 状态机校验（双击防护命中点：第二次 Approve 时任务
// 已 APPROVED，0 推进）→ 实例 RUNNING 校验 → 参与人校验 → 更新任务 →
// 追加操作流水 → 发布事件（经端口加入调用方事务，通知失败不回滚审批）。
func (e *Engine) Approve(ctx context.Context, in ApproveInput) (*ApproveOutcome, error) {
	// 1. 行锁读取任务（并发双击在同一行上串行化）
	task, err := e.tasks.FindTaskByIDForUpdate(ctx, in.TenantID, in.TaskID)
	if err != nil {
		return nil, fmt.Errorf("task %d: %w", in.TaskID, ErrTaskNotFound)
	}
	// 2. 状态机校验：迁移表是唯一事实源，未登记迁移一律非法
	if !CanTransitionTask(task.Status, model.TaskStatusAPPROVED) {
		return nil, fmt.Errorf("task %d status %s: %w", task.ID, task.Status, ErrTaskNotPending)
	}
	// 3. 行锁读取实例并校验 RUNNING
	instance, err := e.instances.FindInstanceByIDForUpdate(ctx, in.TenantID, task.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("instance of task %d: %w", task.ID, ErrInstanceNotFound)
	}
	if instance.Status != model.InstanceStatusRUNNING {
		return nil, fmt.Errorf("instance %d status %s: %w", instance.ID, instance.Status, ErrInstanceNotRunning)
	}
	// 4. 参与人快照实例级校验（第 27 章安全要求）
	actors, err := e.tasks.ListActorsOfTask(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	if !containsActor(actors, in.OperatorMemberID) {
		return nil, ErrTaskForbidden
	}
	// 5-6. 更新任务 + 追加操作流水（同事务）
	task.Status = model.TaskStatusAPPROVED
	if err := e.tasks.SaveTask(ctx, task); err != nil {
		return nil, err
	}
	if err := e.operations.AppendOperation(ctx, &model.Operation{
		TenantID:         in.TenantID,
		InstanceID:       task.InstanceID,
		TaskID:           task.ID,
		OperatorMemberID: in.OperatorMemberID,
		Type:             model.OperationTypeApprove,
		Payload:          map[string]any{"nodeKey": task.NodeKey, "comment": in.Comment},
	}); err != nil {
		return nil, err
	}
	// 7. 发布事件（Outbox 模式：写入随事务提交，消费失败不回滚审批）
	e.publish(ctx, event.TaskApproved, instance.ID, task.ID, in.OperatorMemberID)
	return &ApproveOutcome{Task: task, Instance: instance, NodeInstanceID: task.NodeInstanceID, NodeKey: task.NodeKey}, nil
}

// NodeCompleted 节点完成判定（第 11 章冻结规则）：
//   - single / or-sign：任意一名参与人 APPROVED 即节点完成（剩余 PENDING
//     由 Runtime 联动取消）；
//   - countersign：APPROVED 数达到 ceil(totalActors * passRatio)。
//
// tasks 为本节点实例的全部任务（含已取消），totalActors 以创建时快照为准。
func NodeCompleted(mode model.ApprovalMode, passRatio float64, tasks []model.Task) bool {
	approved := 0
	for i := range tasks {
		if tasks[i].Status == model.TaskStatusAPPROVED {
			approved++
		}
	}
	switch mode {
	case model.ApprovalModeCountersign:
		return approved >= RequiredApprovals(len(tasks), passRatio)
	default:
		// single 与 or-sign 在 V1 共用「首个通过即完成」判定
		return approved >= 1
	}
}

func containsActor(actors []model.Actor, memberID uint) bool {
	for _, actor := range actors {
		if actor.MemberID == memberID {
			return true
		}
	}
	return false
}

func (e *Engine) publish(ctx context.Context, eventName string, instanceID, taskID, actorMemberID uint) {
	if e.publisher == nil {
		return
	}
	_ = e.publisher.PublishInTx(ctx, provider.Event{
		EventName:     eventName,
		InstanceID:    instanceID,
		TaskID:        taskID,
		ActorMemberID: actorMemberID,
	})
}
