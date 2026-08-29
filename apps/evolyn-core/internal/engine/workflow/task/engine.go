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
	// identity 身份窄端口（转办目标显示名快照；可为 nil：单测/无快照场景）
	identity provider.IdentityProvider
}

// NewEngine 构造任务引擎（publisher/identity 可为 nil：跳过事件发布与
// 显示名快照，便于单测）。
func NewEngine(
	tasks repository.TaskRepository,
	instances repository.InstanceRepository,
	operations repository.OperationRepository,
	publisher provider.EventPublisher,
	identity provider.IdentityProvider,
) *Engine {
	return &Engine{tasks: tasks, instances: instances, operations: operations, publisher: publisher, identity: identity}
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

// RejectOutcome 驳回动作产出：任务与实例最新状态（节点 REJECTED 与实例
// REJECTED 终态落账由 Runtime 编排——Runtime 感知节点/实例仓储）。
type RejectOutcome struct {
	Task           *model.Task
	Instance       *model.Instance
	NodeInstanceID uint
	NodeKey        string
}

// RejectInput 驳回动作输入。
type RejectInput struct {
	TenantID         uint
	TaskID           uint
	OperatorMemberID uint
	Comment          string
}

// Reject 驳回任务（第 10.2 章 terminate 语义）：行锁 → PENDING 状态机校验 →
// 实例 RUNNING 校验 → 参与人校验 → 任务 REJECTED → 操作流水 → 事件。
// 节点实例 REJECTED、其余 PENDING 任务 CANCELLED、实例 REJECTED 由
// Runtime 在同一事务内联动落账（Reject 语义不可拆分：任一审批人驳回
// 即整个实例终止，V1 无「驳回到历史节点」）。
func (e *Engine) Reject(ctx context.Context, in RejectInput) (*RejectOutcome, error) {
	task, instance, err := e.lockPendingTask(ctx, in.TenantID, in.TaskID, in.OperatorMemberID)
	if err != nil {
		return nil, err
	}
	task.Status = model.TaskStatusREJECTED
	if err := e.tasks.SaveTask(ctx, task); err != nil {
		return nil, err
	}
	if err := e.operations.AppendOperation(ctx, &model.Operation{
		TenantID:         in.TenantID,
		InstanceID:       task.InstanceID,
		TaskID:           task.ID,
		OperatorMemberID: in.OperatorMemberID,
		Type:             model.OperationTypeReject,
		Payload: map[string]any{
			"nodeKey": task.NodeKey, "comment": in.Comment, "rejectStrategy": string(model.RejectStrategyTerminate),
		},
	}); err != nil {
		return nil, err
	}
	e.publish(ctx, event.TaskRejected, instance.ID, task.ID, in.OperatorMemberID)
	return &RejectOutcome{Task: task, Instance: instance, NodeInstanceID: task.NodeInstanceID, NodeKey: task.NodeKey}, nil
}

// ReturnInput 退回发起人动作输入。
type ReturnInput struct {
	TenantID         uint
	TaskID           uint
	OperatorMemberID uint
	Comment          string
}

// ReturnOutcome 退回动作产出（发起人修改节点实例由 Runtime 创建）。
type ReturnOutcome struct {
	Task           *model.Task
	Instance       *model.Instance
	NodeInstanceID uint
	NodeKey        string
}

// ReturnToStarter 退回发起人（第 10.3 章）：不等价于 REJECTED——任务以
// CANCELLED 关闭（退回重开语义见状态机注释），当前节点实例由 Runtime
// 以 COMPLETED + RETURN_TO_STARTER operation 表达，随后创建
// WAITING_RESUBMIT 节点实例等待发起人修改重提；实例保持 RUNNING。
func (e *Engine) ReturnToStarter(ctx context.Context, in ReturnInput) (*ReturnOutcome, error) {
	task, instance, err := e.lockPendingTask(ctx, in.TenantID, in.TaskID, in.OperatorMemberID)
	if err != nil {
		return nil, err
	}
	task.Status = model.TaskStatusCANCELLED
	if err := e.tasks.SaveTask(ctx, task); err != nil {
		return nil, err
	}
	if err := e.operations.AppendOperation(ctx, &model.Operation{
		TenantID:         in.TenantID,
		InstanceID:       task.InstanceID,
		TaskID:           task.ID,
		OperatorMemberID: in.OperatorMemberID,
		Type:             model.OperationTypeReturnToStarter,
		Payload:          map[string]any{"nodeKey": task.NodeKey, "comment": in.Comment},
	}); err != nil {
		return nil, err
	}
	e.publish(ctx, event.TaskCancelled, instance.ID, task.ID, in.OperatorMemberID)
	return &ReturnOutcome{Task: task, Instance: instance, NodeInstanceID: task.NodeInstanceID, NodeKey: task.NodeKey}, nil
}

// TransferInput 转办动作输入。
type TransferInput struct {
	TenantID         uint
	TaskID           uint
	OperatorMemberID uint
	// TargetMemberID 转办目标成员（同租户有效性由平台服务层校验后传入）
	TargetMemberID uint
	Comment        string
}

// TransferOutcome 转办动作产出：原任务关闭为 TRANSFERRED，新任务已创建
// （参与人快照仅目标成员；节点实例保持 WAITING，不因转办自动完成，
// 第 10.5 章）。
type TransferOutcome struct {
	OriginalTask *model.Task
	NewTask      *model.Task
	Instance     *model.Instance
}

// Transfer 转办任务（第 10.5 章）：行锁 → PENDING 校验 → 实例 RUNNING →
// 参与人校验 → 原任务 TRANSFERRED（记录去向）→ 新建 PENDING 任务
// （TransferredFromTaskID 回链，历史链可追溯）→ 操作流水 → 事件。
func (e *Engine) Transfer(ctx context.Context, in TransferInput) (*TransferOutcome, error) {
	if in.TargetMemberID == 0 {
		return nil, fmt.Errorf("transfer target member required")
	}
	task, instance, err := e.lockPendingTask(ctx, in.TenantID, in.TaskID, in.OperatorMemberID)
	if err != nil {
		return nil, err
	}
	if in.TargetMemberID == in.OperatorMemberID {
		return nil, ErrTaskForbidden
	}
	task.Status = model.TaskStatusTRANSFERRED
	task.TransferredToMemberID = in.TargetMemberID
	if err := e.tasks.SaveTask(ctx, task); err != nil {
		return nil, err
	}
	// 新任务另建（TransferredFromTaskID 回链原任务，历史链可追溯）
	newTask := &model.Task{
		TenantID:              in.TenantID,
		InstanceID:            task.InstanceID,
		NodeInstanceID:        task.NodeInstanceID,
		NodeKey:               task.NodeKey,
		Status:                model.TaskStatusPENDING,
		TransferredFromTaskID: task.ID,
		TransferredToMemberID: 0,
	}
	if err := e.tasks.CreateTask(ctx, newTask); err != nil {
		return nil, err
	}
	displayName := ""
	if e.identity != nil {
		displayName = e.identity.MemberDisplayName(ctx, in.TenantID, in.TargetMemberID)
	}
	if err := e.tasks.ReplaceActors(ctx, newTask.ID, []model.Actor{
		{MemberID: in.TargetMemberID, DisplayName: displayName},
	}); err != nil {
		return nil, err
	}
	if err := e.operations.AppendOperation(ctx, &model.Operation{
		TenantID:         in.TenantID,
		InstanceID:       task.InstanceID,
		TaskID:           newTask.ID,
		OperatorMemberID: in.OperatorMemberID,
		Type:             model.OperationTypeTransfer,
		Payload: map[string]any{
			"nodeKey": task.NodeKey, "fromTaskId": task.ID, "toTaskId": newTask.ID,
			"targetMemberId": in.TargetMemberID, "comment": in.Comment,
		},
	}); err != nil {
		return nil, err
	}
	e.publish(ctx, event.TaskTransferred, instance.ID, newTask.ID, in.OperatorMemberID)
	return &TransferOutcome{OriginalTask: task, NewTask: newTask, Instance: instance}, nil
}

// lockPendingTask 审批动作公共前置（第 13.2 章事务模板 1–4 步）：行锁取
// 任务 → PENDING 状态机校验（双击防护命中点）→ 行锁取实例校验 RUNNING →
// 参与人快照实例级校验。返回任务与实例供动作继续落账。
func (e *Engine) lockPendingTask(ctx context.Context, tenantID, taskID, operatorMemberID uint) (*model.Task, *model.Instance, error) {
	// 1. 行锁读取任务（并发双击在同一行上串行化）
	task, err := e.tasks.FindTaskByIDForUpdate(ctx, tenantID, taskID)
	if err != nil {
		return nil, nil, fmt.Errorf("task %d: %w", taskID, ErrTaskNotFound)
	}
	// 2. 状态机校验：仅 PENDING 可执行动作（迁移表中 PENDING 无自环，
	// 直接按状态判定；目标终态由各动作自行经迁移表校验）
	if task.Status != model.TaskStatusPENDING {
		return nil, nil, fmt.Errorf("task %d status %s: %w", task.ID, task.Status, ErrTaskNotPending)
	}
	// 3. 行锁读取实例并校验 RUNNING
	instance, err := e.instances.FindInstanceByIDForUpdate(ctx, tenantID, task.InstanceID)
	if err != nil {
		return nil, nil, fmt.Errorf("instance of task %d: %w", task.ID, ErrInstanceNotFound)
	}
	if instance.Status != model.InstanceStatusRUNNING {
		return nil, nil, fmt.Errorf("instance %d status %s: %w", instance.ID, instance.Status, ErrInstanceNotRunning)
	}
	// 4. 参与人快照实例级校验（第 27 章安全要求）
	actors, err := e.tasks.ListActorsOfTask(ctx, task.ID)
	if err != nil {
		return nil, nil, err
	}
	if !containsActor(actors, operatorMemberID) {
		return nil, nil, ErrTaskForbidden
	}
	return task, instance, nil
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
