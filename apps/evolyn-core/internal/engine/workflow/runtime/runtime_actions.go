// Phase 4 实例级与任务级动作编排（第 10 章审批动作语义）：任务级前置裁决
// 在 task.Engine（行锁 + 状态机 + 参与人校验），节点/实例联动终态落账在本包
// （Runtime 感知节点/执行路径/实例仓储）。全部动作仍由平台层 TxManager
// 包裹，任一步失败整体回滚（第 13 章）。
package runtime

import (
	"context"

	"evolyn/internal/engine/workflow/event"
	"evolyn/internal/engine/workflow/model"
	"evolyn/internal/engine/workflow/provider"
	"evolyn/internal/engine/workflow/task"
)

// RejectInput 驳回输入（terminate 语义，第 10.2 章）。
type RejectInput struct {
	TenantID         uint
	TaskID           uint
	OperatorMemberID uint
	Comment          string
	// AutoTimeout 超时自动驳回触发（第 19.4 章，语义同 ApproveInput）
	AutoTimeout bool
}

// RejectResult 驳回结果。
type RejectResult struct {
	InstanceID     uint
	InstanceStatus model.InstanceStatus
}

// Reject 驳回（第 10.2 章）：Task → REJECTED（Task Engine）→ 节点实例
// WAITING → REJECTED → 其余 PENDING 任务 CANCELLED → 执行路径与实例
// REJECTED/CANCELLED → 事件。任一审批人驳回即整个实例终止，V1 无
// 「驳回到历史节点」。
func (r *Runtime) Reject(ctx context.Context, in RejectInput) (*RejectResult, error) {
	outcome, err := r.taskEngine.Reject(ctx, task.RejectInput{
		TenantID:         in.TenantID,
		TaskID:           in.TaskID,
		OperatorMemberID: in.OperatorMemberID,
		Comment:          in.Comment,
		AutoTimeout:      in.AutoTimeout,
	})
	if err != nil {
		return nil, err
	}
	instance := outcome.Instance
	version, err := r.definitions.FindVersionByID(ctx, in.TenantID, instance.DefinitionVersionID)
	if err != nil {
		return nil, err
	}
	if _, ok := version.Snapshot.NodeOf(outcome.NodeKey); !ok {
		return nil, ErrRouteStuck
	}
	nodeInstance, err := r.nodes.FindNodeInstanceByID(ctx, in.TenantID, outcome.NodeInstanceID)
	if err != nil {
		return nil, err
	}
	if !task.CanTransitionNodeInstance(nodeInstance.Status, model.NodeInstanceStatusREJECTED) {
		return nil, ErrRouteStuck
	}
	nodeInstance.Status = model.NodeInstanceStatusREJECTED
	if err := r.nodes.SaveNodeInstance(ctx, nodeInstance); err != nil {
		return nil, err
	}
	if _, err := r.tasks.CancelPendingTasksByNode(ctx, outcome.NodeInstanceID); err != nil {
		return nil, err
	}
	// 任务取消联动排期 Job（第 19 章）
	r.cancelJobsByNode(ctx, outcome.NodeInstanceID)
	res, err := r.cancelInstance(ctx, instance, model.InstanceStatusREJECTED, in.OperatorMemberID)
	if err != nil {
		return nil, err
	}
	return asRejectResult(res), nil
}

// ReturnInput 退回发起人输入（第 10.3 章）。
type ReturnInput struct {
	TenantID         uint
	TaskID           uint
	OperatorMemberID uint
	Comment          string
}

// ReturnResult 退回结果：实例保持 RUNNING，进入发起人修改等待态。
type ReturnResult struct {
	InstanceID uint
	Status     model.InstanceStatus
	// ResubmitNodeInstanceID 新建发起人修改节点实例（WAITING_RESUBMIT）
	ResubmitNodeInstanceID uint
}

// ReturnToStarter 退回发起人（第 10.3 章）：任务 CANCELLED（Task Engine）→
// 当前节点实例 COMPLETED（RETURNED 语义由 RETURN_TO_STARTER operation 表达）
// → 其余 PENDING 任务 CANCELLED → 新建发起人修改节点实例 WAITING_RESUBMIT。
// 发起人重新提交后从退回节点继续（Resubmit）。
func (r *Runtime) ReturnToStarter(ctx context.Context, in ReturnInput) (*ReturnResult, error) {
	outcome, err := r.taskEngine.ReturnToStarter(ctx, task.ReturnInput{
		TenantID:         in.TenantID,
		TaskID:           in.TaskID,
		OperatorMemberID: in.OperatorMemberID,
		Comment:          in.Comment,
	})
	if err != nil {
		return nil, err
	}
	instance := outcome.Instance
	version, err := r.definitions.FindVersionByID(ctx, in.TenantID, instance.DefinitionVersionID)
	if err != nil {
		return nil, err
	}
	if _, ok := version.Snapshot.NodeOf(outcome.NodeKey); !ok {
		return nil, ErrRouteStuck
	}
	nodeInstance, err := r.nodes.FindNodeInstanceByID(ctx, in.TenantID, outcome.NodeInstanceID)
	if err != nil {
		return nil, err
	}
	if !task.CanTransitionNodeInstance(nodeInstance.Status, model.NodeInstanceStatusCOMPLETED) {
		return nil, ErrRouteStuck
	}
	nodeInstance.Status = model.NodeInstanceStatusCOMPLETED
	if err := r.nodes.SaveNodeInstance(ctx, nodeInstance); err != nil {
		return nil, err
	}
	if _, err := r.tasks.CancelPendingTasksByNode(ctx, outcome.NodeInstanceID); err != nil {
		return nil, err
	}
	// 任务取消联动排期 Job（第 19 章）
	r.cancelJobsByNode(ctx, outcome.NodeInstanceID)
	// 发起人修改节点实例：NodeKey 记录退回来源节点，重提交后从该节点继续
	resubmit := &model.NodeInstance{
		TenantID:    instance.TenantID,
		InstanceID:  instance.ID,
		ExecutionID: nodeInstance.ExecutionID,
		NodeKey:     outcome.NodeKey,
		Status:      model.NodeInstanceStatusWAITING_RESUBMIT,
	}
	if err := r.nodes.CreateNodeInstance(ctx, resubmit); err != nil {
		return nil, err
	}
	return &ReturnResult{InstanceID: instance.ID, Status: instance.Status, ResubmitNodeInstanceID: resubmit.ID}, nil
}

// ResubmitInput 发起人重新提交输入（第 10.3 章）：可携带修改后的表单值。
type ResubmitInput struct {
	TenantID         uint
	InstanceID       uint
	OperatorMemberID uint
	// FormValues 修改后的表单字段值（发起人为数据属主，不按审批节点字段
	// 权限过滤，但仍经 Form Domain 按冻结快照整体校验）
	FormValues map[string]any
}

// ResubmitResult 重提交结果（流程从退回节点继续）。
type ResubmitResult struct {
	InstanceID     uint
	InstanceStatus model.InstanceStatus
}

// Resubmit 发起人重新提交：实例 RUNNING + 发起人校验 → 表单值经业务窄端口
// 写回 → 重提交节点实例 WAITING_RESUBMIT → RUNNING → COMPLETED →
// RESUBMIT 操作流水 → 从退回节点重新执行（审批人重新审批）。
func (r *Runtime) Resubmit(ctx context.Context, in ResubmitInput) (*ResubmitResult, error) {
	instance, err := r.instances.FindInstanceByIDForUpdate(ctx, in.TenantID, in.InstanceID)
	if err != nil {
		return nil, task.ErrInstanceNotFound
	}
	if instance.Status != model.InstanceStatusRUNNING {
		return nil, task.ErrInstanceNotRunning
	}
	if instance.StarterMemberID != in.OperatorMemberID {
		return nil, ErrNotStarter
	}
	// 定位发起人修改节点实例（同一实例至多一个 WAITING_RESUBMIT：退回是
	// 串行流程中的节点动作，重提交前不可能二次退回）
	nodeInstances, err := r.nodes.ListNodeInstancesByInstance(ctx, instance.ID)
	if err != nil {
		return nil, err
	}
	var resubmit *model.NodeInstance
	for i := range nodeInstances {
		if nodeInstances[i].Status == model.NodeInstanceStatusWAITING_RESUBMIT {
			resubmit = &nodeInstances[i]
			break
		}
	}
	if resubmit == nil {
		return nil, ErrResubmitNodeMissing
	}
	// 发起人修改写回（Form Domain 按冻结快照校验，失败整体回滚）
	if err := r.applyStarterFormValues(ctx, instance, in.FormValues); err != nil {
		return nil, err
	}
	// 状态机：WAITING_RESUBMIT → RUNNING → COMPLETED（迁移表两步落账）
	if !task.CanTransitionNodeInstance(resubmit.Status, model.NodeInstanceStatusRUNNING) {
		return nil, ErrRouteStuck
	}
	resubmit.Status = model.NodeInstanceStatusRUNNING
	if err := r.nodes.SaveNodeInstance(ctx, resubmit); err != nil {
		return nil, err
	}
	if !task.CanTransitionNodeInstance(resubmit.Status, model.NodeInstanceStatusCOMPLETED) {
		return nil, ErrRouteStuck
	}
	resubmit.Status = model.NodeInstanceStatusCOMPLETED
	if err := r.nodes.SaveNodeInstance(ctx, resubmit); err != nil {
		return nil, err
	}
	if err := r.operations.AppendOperation(ctx, &model.Operation{
		TenantID:         in.TenantID,
		InstanceID:       instance.ID,
		OperatorMemberID: in.OperatorMemberID,
		Type:             model.OperationTypeResubmit,
		Payload:          map[string]any{"nodeKey": resubmit.NodeKey},
	}); err != nil {
		return nil, err
	}
	// 从退回来源节点重新执行（新节点实例 + 新任务，审批人重新审批）
	version, err := r.definitions.FindVersionByID(ctx, in.TenantID, instance.DefinitionVersionID)
	if err != nil {
		return nil, err
	}
	definitionCode, err := r.definitionCode(ctx, instance.DefinitionID)
	if err != nil {
		return nil, err
	}
	advance, err := r.newAdvanceContext(ctx, instance, version, definitionCode)
	if err != nil {
		return nil, err
	}
	if err := r.advance(ctx, advance, resubmit.NodeKey); err != nil {
		return nil, err
	}
	return &ResubmitResult{InstanceID: instance.ID, InstanceStatus: instance.Status}, nil
}

// InstanceActionInput 实例级动作输入（撤回/管理员终止）。
type InstanceActionInput struct {
	TenantID         uint
	InstanceID       uint
	OperatorMemberID uint
	Comment          string
}

// Withdraw 发起人撤回（第 10.4 章）：发起人专属 + 撤回窗口校验（实例尚不
// 存在任何已完成的人工审批任务）→ 实例 CANCELLED，全部 PENDING 任务取消。
func (r *Runtime) Withdraw(ctx context.Context, in InstanceActionInput) (*InstanceStatusResult, error) {
	return r.cancelRunningInstance(ctx, in, true)
}

// Terminate 管理员终止（第 10.4 章）：独立 terminate 权限（平台服务层校验），
// 不受撤回窗口限制。
func (r *Runtime) Terminate(ctx context.Context, in InstanceActionInput) (*InstanceStatusResult, error) {
	return r.cancelRunningInstance(ctx, in, false)
}

// InstanceStatusResult 实例级动作结果。
type InstanceStatusResult struct {
	InstanceID     uint
	InstanceStatus model.InstanceStatus
}

// TransferInput 转办输入（第 10.5 章，任务级动作透传 Task Engine）。
type TransferInput = task.TransferInput

// Transfer 转办任务：原任务 TRANSFERRED + 新任务另建，节点实例保持挂起，
// 无节点/实例联动（完全在任务域内完成，Runtime 仅做端口透传）。
func (r *Runtime) Transfer(ctx context.Context, in TransferInput) (*task.TransferOutcome, error) {
	return r.taskEngine.Transfer(ctx, in)
}

// cancelRunningInstance 撤回/终止公共链路：行锁取实例 → RUNNING →（撤回：
// 发起人 + 窗口校验）→ PENDING 任务/挂起节点实例取消 → 执行路径与实例
// CANCELLED → 操作流水 → 事件。
func (r *Runtime) cancelRunningInstance(ctx context.Context, in InstanceActionInput, withdrawWindow bool) (*InstanceStatusResult, error) {
	instance, err := r.instances.FindInstanceByIDForUpdate(ctx, in.TenantID, in.InstanceID)
	if err != nil {
		return nil, task.ErrInstanceNotFound
	}
	if instance.Status != model.InstanceStatusRUNNING {
		return nil, task.ErrInstanceNotRunning
	}
	if withdrawWindow {
		if instance.StarterMemberID != in.OperatorMemberID {
			return nil, ErrNotStarter
		}
		// 撤回窗口（第 10.4 章冻结规则）：任何已完成的人工审批任务即关闭窗口
		tasks, err := r.tasks.ListTasksByInstance(ctx, instance.ID)
		if err != nil {
			return nil, err
		}
		if !task.WithdrawAllowed(tasks) {
			return nil, ErrActionNotAllowed
		}
	}
	if err := r.operations.AppendOperation(ctx, &model.Operation{
		TenantID:         in.TenantID,
		InstanceID:       instance.ID,
		OperatorMemberID: in.OperatorMemberID,
		Type:             map[bool]model.OperationType{true: model.OperationTypeWithdraw, false: model.OperationTypeTerminate}[withdrawWindow],
		Payload:          map[string]any{"comment": in.Comment},
	}); err != nil {
		return nil, err
	}
	return r.cancelInstance(ctx, instance, model.InstanceStatusCANCELLED, in.OperatorMemberID)
}

// cancelInstance 实例终态落账（REJECTED / CANCELLED 共用）：PENDING 任务
// 取消 → 挂起节点实例取消 → 执行路径取消 → 实例终态 → 事件。
func (r *Runtime) cancelInstance(ctx context.Context, instance *model.Instance, target model.InstanceStatus, operatorMemberID uint) (*InstanceStatusResult, error) {
	if _, err := r.tasks.CancelPendingTasksByInstance(ctx, instance.ID); err != nil {
		return nil, err
	}
	// 实例终态联动取消全部排期 Job（第 19 章）
	r.cancelJobsByInstance(ctx, instance.ID)
	// 挂起中的节点实例（WAITING/WAITING_RESUBMIT）随实例终态取消
	nodeInstances, err := r.nodes.ListNodeInstancesByInstance(ctx, instance.ID)
	if err != nil {
		return nil, err
	}
	for i := range nodeInstances {
		switch nodeInstances[i].Status {
		case model.NodeInstanceStatusWAITING, model.NodeInstanceStatusWAITING_RESUBMIT:
			if !task.CanTransitionNodeInstance(nodeInstances[i].Status, model.NodeInstanceStatusCANCELLED) {
				return nil, ErrRouteStuck
			}
			nodeInstances[i].Status = model.NodeInstanceStatusCANCELLED
			if err := r.nodes.SaveNodeInstance(ctx, &nodeInstances[i]); err != nil {
				return nil, err
			}
		}
	}
	executions, err := r.executions.ListExecutionsByInstance(ctx, instance.ID)
	if err != nil {
		return nil, err
	}
	for i := range executions {
		if executions[i].Status != model.ExecutionStatusRUNNING {
			continue
		}
		if !task.CanTransitionExecution(executions[i].Status, model.ExecutionStatusCANCELLED) {
			return nil, ErrRouteStuck
		}
		executions[i].Status = model.ExecutionStatusCANCELLED
		if err := r.executions.SaveExecution(ctx, &executions[i]); err != nil {
			return nil, err
		}
	}
	if !task.CanTransitionInstance(instance.Status, target) {
		return nil, ErrRouteStuck
	}
	instance.Status = target
	if err := r.instances.SaveInstance(ctx, instance); err != nil {
		return nil, err
	}
	eventName := event.InstanceRejected
	if target == model.InstanceStatusCANCELLED {
		eventName = event.InstanceCancelled
	}
	r.publish(ctx, eventName, instance, 0, operatorMemberID)
	return &InstanceStatusResult{InstanceID: instance.ID, InstanceStatus: instance.Status}, nil
}

// asRejectResult 适配实例终态结果为驳回结果出参。
func asRejectResult(res *InstanceStatusResult) *RejectResult {
	return &RejectResult{InstanceID: res.InstanceID, InstanceStatus: res.InstanceStatus}
}

// applyStarterFormValues 发起人修改写回（第 10.3 章）：发起人是表单数据
// 属主，不按审批节点字段权限过滤，但必须经 Form Domain 按冻结快照校验。
func (r *Runtime) applyStarterFormValues(ctx context.Context, instance *model.Instance, values map[string]any) error {
	if len(values) == 0 {
		return nil
	}
	if r.business == nil || instance.FormVersionID == 0 {
		return ErrFormFieldForbidden
	}
	return r.business.UpdateData(ctx, provider.BusinessRef{
		TenantID:      instance.TenantID,
		AppID:         instance.AppID,
		FormID:        instance.FormID,
		FormVersionID: instance.FormVersionID,
		BusinessID:    instance.BusinessID,
	}, values)
}

// definitionCode 详情投影辅助：按定义行 ID 反查公开编码（表达式上下文与
// 操作载荷使用，不参与状态裁决）。
func (r *Runtime) definitionCode(ctx context.Context, definitionID uint) (string, error) {
	return r.definitions.FindDefinitionCodeByID(ctx, 0, definitionID)
}
