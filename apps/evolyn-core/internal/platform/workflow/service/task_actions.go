// Phase 4 完整人工任务应用服务（第 10/20.3/20.4 章）：驳回/退回/转办/
// 撤回/终止/重提交 + 审批中心四类查询 + 任务详情上下文。事务边界在本层
// 建立（TxManager 唯一事实源）；引擎 sentinel 错误统一经 mapEngineError
// 映射稳定业务错误码。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"evolyn/internal/contextx"
	enginemodel "evolyn/internal/engine/workflow/model"
	"evolyn/internal/engine/workflow/provider"
	engineruntime "evolyn/internal/engine/workflow/runtime"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	wfapp "evolyn/internal/platform/workflow"
	"evolyn/internal/platform/workflow/model"
	"evolyn/internal/platform/workflow/repository"

	"gorm.io/gorm"
)

// 人工审批动作权限门（第 21 章）：approve/reject/return/transfer 是
// 同族 POST 动作，URL 门与 Service 复核统一使用 workflow-tasks:create
// （与鉴权中间件同口径）；实例级能力由 TaskActor 快照校验兜底。
const taskActionPermission = "workflow-tasks:create"

// pendingStatuses 我的待办任务状态集。
var pendingStatuses = []string{"PENDING"}

// completedStatuses 我的已办任务状态集（本人参与且已达终态的任务）。
var completedStatuses = []string{"APPROVED", "REJECTED", "TRANSFERRED", "CANCELLED"}

// rejectTask 驳回任务（第 10.2 章 terminate 语义）。
func (s *runtimeService) RejectTask(ctx context.Context, member *iammodel.User, req *model.RejectTaskRequest) (*model.ActionTaskResult, error) {
	if !s.hasPermission(ctx, member, taskActionPermission) {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot reject tasks"))
	}
	tenantID, memberID, err := s.taskActionContext(ctx, member)
	if err != nil {
		return nil, err
	}
	if req.TaskID == 0 {
		return nil, httpx.Wrap(wfapp.ErrTaskNotFound, fmt.Errorf("taskId required"))
	}
	var result *model.ActionTaskResult
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		out, err := s.engine.Reject(tctx, engineruntime.RejectInput{
			TenantID: tenantID, TaskID: req.TaskID,
			OperatorMemberID: memberID, Comment: req.Comment,
		})
		if err != nil {
			return mapEngineError(err)
		}
		result = &model.ActionTaskResult{InstanceID: out.InstanceID, InstanceStatus: string(out.InstanceStatus)}
		return nil
	}); err != nil {
		return nil, err
	}
	s.recordActionAudit(ctx, member, "reject", req.TaskID, result.InstanceID)
	return result, nil
}

// ReturnTask 退回发起人（第 10.3 章）。
func (s *runtimeService) ReturnTask(ctx context.Context, member *iammodel.User, req *model.ReturnTaskRequest) (*model.ActionTaskResult, error) {
	if !s.hasPermission(ctx, member, taskActionPermission) {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot return tasks"))
	}
	tenantID, memberID, err := s.taskActionContext(ctx, member)
	if err != nil {
		return nil, err
	}
	if req.TaskID == 0 {
		return nil, httpx.Wrap(wfapp.ErrTaskNotFound, fmt.Errorf("taskId required"))
	}
	var result *model.ActionTaskResult
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		out, err := s.engine.ReturnToStarter(tctx, engineruntime.ReturnInput{
			TenantID: tenantID, TaskID: req.TaskID,
			OperatorMemberID: memberID, Comment: req.Comment,
		})
		if err != nil {
			return mapEngineError(err)
		}
		result = &model.ActionTaskResult{InstanceID: out.InstanceID, InstanceStatus: string(out.Status)}
		return nil
	}); err != nil {
		return nil, err
	}
	s.recordActionAudit(ctx, member, "return", req.TaskID, result.InstanceID)
	return result, nil
}

// TransferTask 转办任务（第 10.5 章）：目标成员同租户有效（身份端口校验）。
func (s *runtimeService) TransferTask(ctx context.Context, member *iammodel.User, req *model.TransferTaskRequest) (*model.ActionTaskResult, error) {
	if !s.hasPermission(ctx, member, taskActionPermission) {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot transfer tasks"))
	}
	tenantID, memberID, err := s.taskActionContext(ctx, member)
	if err != nil {
		return nil, err
	}
	if req.TaskID == 0 || req.TargetMemberID == 0 {
		return nil, httpx.Wrap(wfapp.ErrTaskNotFound, fmt.Errorf("taskId/targetMemberId required"))
	}
	if s.identity != nil {
		if err := s.identity.ValidateMembers(ctx, tenantID, []uint{req.TargetMemberID}); err != nil {
			return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("transfer target invalid: %w", err))
		}
	}
	var result *model.ActionTaskResult
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		out, err := s.engine.Transfer(tctx, engineruntime.TransferInput{
			TenantID: tenantID, TaskID: req.TaskID, OperatorMemberID: memberID,
			TargetMemberID: req.TargetMemberID, Comment: req.Comment,
		})
		if err != nil {
			return mapEngineError(err)
		}
		result = &model.ActionTaskResult{InstanceID: out.Instance.ID, InstanceStatus: string(out.Instance.Status), NewTaskID: out.NewTask.ID}
		return nil
	}); err != nil {
		return nil, err
	}
	s.recordActionAudit(ctx, member, "transfer", req.TaskID, result.InstanceID)
	return result, nil
}

// WithdrawInstance 发起人撤回（第 10.4 章）：发起人专属 + 撤回窗口校验
// 在引擎内完成；权限门与 POST URL 一致（workflow-instances:create）。
func (s *runtimeService) WithdrawInstance(ctx context.Context, member *iammodel.User, instanceID uint, req *model.InstanceActionRequest) (*model.ActionTaskResult, error) {
	return s.instanceCancelAction(ctx, member, instanceID, req, true)
}

// TerminateInstance 管理员终止（第 10.4 章）：独立权限（workflow-instances:
// update，基线管理员经 '*' 拥有；普通成员仅 create/get），不经撤回窗口限制。
func (s *runtimeService) TerminateInstance(ctx context.Context, member *iammodel.User, instanceID uint, req *model.InstanceActionRequest) (*model.ActionTaskResult, error) {
	if !s.hasPermission(ctx, member, "workflow-instances:update") {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot terminate workflow instances"))
	}
	return s.instanceCancelAction(ctx, member, instanceID, req, false)
}

// instanceCancelAction 撤回/终止公共链路。
func (s *runtimeService) instanceCancelAction(ctx context.Context, member *iammodel.User, instanceID uint, req *model.InstanceActionRequest, withdraw bool) (*model.ActionTaskResult, error) {
	if !s.hasPermission(ctx, member, "workflow-instances:create") {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot operate workflow instances"))
	}
	tenantID, memberID, err := s.taskActionContext(ctx, member)
	if err != nil {
		return nil, err
	}
	if instanceID == 0 {
		return nil, httpx.Wrap(wfapp.ErrInstanceNotFound, fmt.Errorf("instanceId required"))
	}
	comment := ""
	if req != nil {
		comment = req.Comment
	}
	var result *model.ActionTaskResult
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		var out *engineruntime.InstanceStatusResult
		var err error
		if withdraw {
			out, err = s.engine.Withdraw(tctx, engineruntime.InstanceActionInput{
				TenantID: tenantID, InstanceID: instanceID, OperatorMemberID: memberID, Comment: comment,
			})
		} else {
			out, err = s.engine.Terminate(tctx, engineruntime.InstanceActionInput{
				TenantID: tenantID, InstanceID: instanceID, OperatorMemberID: memberID, Comment: comment,
			})
		}
		if err != nil {
			return mapEngineError(err)
		}
		result = &model.ActionTaskResult{InstanceID: out.InstanceID, InstanceStatus: string(out.InstanceStatus)}
		return nil
	}); err != nil {
		return nil, err
	}
	action := "terminate"
	if withdraw {
		action = "withdraw"
	}
	s.recordActionAudit(ctx, member, action, instanceID, instanceID)
	return result, nil
}

// ResubmitInstance 发起人重新提交（第 10.3 章）：修改后的表单值经 Form
// Domain 校验后同事务写回，流程从退回节点继续。
func (s *runtimeService) ResubmitInstance(ctx context.Context, member *iammodel.User, instanceID uint, req *model.ResubmitInstanceRequest) (*model.ActionTaskResult, error) {
	if !s.hasPermission(ctx, member, "workflow-instances:create") {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot resubmit workflow instances"))
	}
	tenantID, memberID, err := s.taskActionContext(ctx, member)
	if err != nil {
		return nil, err
	}
	if instanceID == 0 {
		return nil, httpx.Wrap(wfapp.ErrInstanceNotFound, fmt.Errorf("instanceId required"))
	}
	formValues := map[string]any{}
	if req != nil {
		for key, raw := range req.Values {
			var value any
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, httpx.Wrap(wfapp.ErrFormFieldForbidden, fmt.Errorf("field %s: %w", key, err))
			}
			formValues[key] = value
		}
	}
	var result *model.ActionTaskResult
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		out, err := s.engine.Resubmit(tctx, engineruntime.ResubmitInput{
			TenantID: tenantID, InstanceID: instanceID,
			OperatorMemberID: memberID, FormValues: formValues,
		})
		if err != nil {
			return mapEngineError(err)
		}
		result = &model.ActionTaskResult{InstanceID: out.InstanceID, InstanceStatus: string(out.InstanceStatus)}
		return nil
	}); err != nil {
		return nil, err
	}
	s.recordActionAudit(ctx, member, "resubmit", instanceID, instanceID)
	return result, nil
}

// ListTasks 审批中心任务查询（第 20.4 章）：我的待办/我的已办/抄送我的。
func (s *runtimeService) ListTasks(ctx context.Context, member *iammodel.User, query model.ListTasksQuery) (*model.TaskPage, error) {
	if !s.hasPermission(ctx, member, "workflow-tasks:get") {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot list workflow tasks"))
	}
	tenantID, memberID, err := s.taskActionContext(ctx, member)
	if err != nil {
		return nil, err
	}
	formCode := strings.TrimSpace(query.FormCode)
	if formCode != "" && !strings.HasPrefix(formCode, "form_") {
		return nil, httpx.Wrap(wfapp.ErrWorkflowCodeInvalid, fmt.Errorf("invalid formCode %q", formCode))
	}
	_ = tenantID
	limit, afterID, err := parsePhase4Cursor(query.Limit, query.Cursor)
	if err != nil {
		return nil, httpx.Wrap(wfapp.ErrWorkflowCodeInvalid, err)
	}

	switch strings.TrimSpace(query.Scope) {
	case "cc-to-me":
		if formCode != "" {
			return nil, httpx.Wrap(wfapp.ErrWorkflowCodeInvalid, fmt.Errorf("formCode only supports pending scope"))
		}
		return s.listCCTasks(ctx, memberID, "", limit, afterID)
	case "completed":
		if formCode != "" {
			return nil, httpx.Wrap(wfapp.ErrWorkflowCodeInvalid, fmt.Errorf("formCode only supports pending scope"))
		}
		return s.listMemberTasks(ctx, memberID, completedStatuses, "", limit, afterID)
	case "pending", "":
		return s.listMemberTasks(ctx, memberID, pendingStatuses, formCode, limit, afterID)
	default:
		return nil, httpx.Wrap(wfapp.ErrWorkflowCodeInvalid, fmt.Errorf("unknown scope %q", query.Scope))
	}
}

// PendingTaskSummary 为流程侧栏提供准确的未处理总量和按绑定表单的聚合数。
// 独立流程没有 formCode，仍累加到 Total，但不会虚构一个不可跳转的子菜单项。
func (s *runtimeService) PendingTaskSummary(ctx context.Context, member *iammodel.User) (*model.PendingTaskSummary, error) {
	if !s.hasPermission(ctx, member, "workflow-tasks:get") {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot read workflow task summary"))
	}
	_, memberID, err := s.taskActionContext(ctx, member)
	if err != nil {
		return nil, err
	}
	rows, err := s.reader.CountPendingTasksByMember(ctx, memberID, pendingStatuses)
	if err != nil {
		return nil, err
	}
	summary := &model.PendingTaskSummary{FormCounts: make([]model.PendingTaskFormCount, 0, len(rows))}
	for _, row := range rows {
		summary.Total += row.Count
		if row.FormCode != "" {
			summary.FormCounts = append(summary.FormCounts, row)
		}
	}
	return summary, nil
}

// listMemberTasks 我的待办/我的已办组装。
func (s *runtimeService) listMemberTasks(ctx context.Context, memberID uint, statuses []string, formCode string, limit int, afterID uint) (*model.TaskPage, error) {
	rows, hasMore, err := s.reader.ListTaskRowsByMemberAndStatuses(ctx, memberID, statuses, formCode, limit, afterID)
	if err != nil {
		return nil, err
	}
	actorsByTask, err := s.actorsByTask(ctx, rows)
	if err != nil {
		return nil, err
	}
	page := &model.TaskPage{Items: make([]model.TaskSummary, 0, len(rows))}
	for i := range rows {
		page.Items = append(page.Items, taskSummaryFromRow(rows[i], actorsByTask))
	}
	if hasMore && len(rows) > 0 {
		page.NextCursor = strconv.FormatUint(uint64(rows[len(rows)-1].ID), 10)
	}
	return page, nil
}

// listCCTasks 抄送我的组装。
func (s *runtimeService) listCCTasks(ctx context.Context, memberID uint, formCode string, limit int, afterID uint) (*model.TaskPage, error) {
	rows, hasMore, err := s.reader.ListCCRowsByMember(ctx, memberID, formCode, limit, afterID)
	if err != nil {
		return nil, err
	}
	page := &model.TaskPage{Items: make([]model.TaskSummary, 0, len(rows))}
	for i := range rows {
		page.Items = append(page.Items, model.TaskSummary{
			ID:         rows[i].ID,
			InstanceID: rows[i].InstanceID,
			NodeKey:    rows[i].NodeKey,
			Status:     "CC",
			Actors: []model.TaskActorView{{
				MemberID: rows[i].MemberID, DisplayName: rows[i].DisplayName,
			}},
			CreatedAt: rows[i].CreatedAt,
		})
	}
	if hasMore && len(rows) > 0 {
		page.NextCursor = strconv.FormatUint(uint64(rows[len(rows)-1].ID), 10)
	}
	return page, nil
}

// ListInstances 审批中心实例查询（第 20.4 章）：我发起的。
func (s *runtimeService) ListInstances(ctx context.Context, member *iammodel.User, query model.ListInstancesQuery) (*model.InstancePage, error) {
	if !s.hasPermission(ctx, member, "workflow-instances:get") {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot list workflow instances"))
	}
	tenantID, memberID, err := s.taskActionContext(ctx, member)
	if err != nil {
		return nil, err
	}
	_ = tenantID
	if scope := strings.TrimSpace(query.Scope); scope != "" && scope != "started-by-me" {
		return nil, httpx.Wrap(wfapp.ErrWorkflowCodeInvalid, fmt.Errorf("unknown scope %q", query.Scope))
	}
	limit, afterID, err := parsePhase4Cursor(query.Limit, query.Cursor)
	if err != nil {
		return nil, httpx.Wrap(wfapp.ErrWorkflowCodeInvalid, err)
	}
	rows, hasMore, err := s.reader.ListInstanceRowsByStarter(ctx, memberID, limit, afterID)
	if err != nil {
		return nil, err
	}
	page := &model.InstancePage{Items: make([]model.InstanceSummary, 0, len(rows))}
	for i := range rows {
		definitionCode, versionNo, err := s.definitionProjection(ctx, rows[i].DefinitionID, rows[i].DefinitionVersionID)
		if err != nil {
			return nil, err
		}
		page.Items = append(page.Items, model.InstanceSummary{
			ID:                  rows[i].ID,
			DefinitionCode:      definitionCode,
			DefinitionVersionNo: versionNo,
			BusinessType:        rows[i].BusinessType,
			BusinessID:          rows[i].BusinessID,
			Status:              rows[i].Status,
			StarterMemberID:     rows[i].StarterMemberID,
			CreatedAt:           rows[i].CreatedAt,
		})
	}
	if hasMore && len(rows) > 0 {
		page.NextCursor = strconv.FormatUint(uint64(rows[len(rows)-1].ID), 10)
	}
	return page, nil
}

// GetTask 任务详情上下文（审批详情返回协议，第 4 章）：任务 + 实例绑定 +
// 表单快照/数据 + 节点字段权限 + 允许动作 + 操作时间线。
// 实例级访问控制：任务参与人或发起人可读（第 27 章安全要求）。
func (s *runtimeService) GetTask(ctx context.Context, member *iammodel.User, taskID uint) (*model.TaskDetail, error) {
	if !s.hasPermission(ctx, member, "workflow-tasks:get") {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot read workflow task %d", taskID))
	}
	tenantID, memberID, err := s.taskActionContext(ctx, member)
	if err != nil {
		return nil, err
	}
	_ = tenantID
	taskRow, err := s.reader.FindTaskRow(ctx, taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(wfapp.ErrTaskNotFound, err)
		}
		return nil, err
	}
	instance, err := s.reader.FindInstanceRow(ctx, taskRow.InstanceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(wfapp.ErrInstanceNotFound, err)
		}
		return nil, err
	}
	// 实例级访问控制：参与人快照或发起人
	actors, err := s.reader.ListActorRowsByTaskIDs(ctx, []uint{taskID})
	if err != nil {
		return nil, err
	}
	allowed := instance.StarterMemberID == memberID
	for i := range actors {
		if actors[i].MemberID == memberID {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, httpx.Wrap(wfapp.ErrTaskForbidden, fmt.Errorf("member %d is neither actor nor starter of task %d", memberID, taskID))
	}

	definitionCode, versionNo, err := s.definitionProjection(ctx, instance.DefinitionID, instance.DefinitionVersionID)
	if err != nil {
		return nil, err
	}
	version, err := s.definitions.FindVersionByID(ctx, instance.TenantID, instance.DefinitionVersionID)
	if err != nil {
		return nil, err
	}
	node, nodeOK := version.Snapshot.NodeOf(taskRow.NodeKey)

	nodeInstanceStatus := ""
	nodeRows, err := s.reader.ListNodeRowsByInstance(ctx, instance.ID)
	if err != nil {
		return nil, err
	}
	for i := range nodeRows {
		if nodeRows[i].ID == taskRow.NodeInstanceID {
			nodeInstanceStatus = nodeRows[i].Status
			break
		}
	}

	actorsByTask, err := s.actorsByTask(ctx, []model.WfTask{*taskRow})
	if err != nil {
		return nil, err
	}
	detail := &model.TaskDetail{
		Task:            taskSummaryFromRow(*taskRow, actorsByTask),
		NodeKey:         taskRow.NodeKey,
		NodeStatus:      nodeInstanceStatus,
		FormPermissions: map[string]string{},
		FormValues:      map[string]any{},
		AllowedActions:  []string{},
		Operations:      []model.InstanceOperationView{},
	}
	detail.Instance = model.InstanceSummary{
		ID:                  instance.ID,
		DefinitionCode:      definitionCode,
		DefinitionVersionNo: versionNo,
		BusinessType:        instance.BusinessType,
		BusinessID:          instance.BusinessID,
		Status:              instance.Status,
		StarterMemberID:     instance.StarterMemberID,
		CreatedAt:           instance.CreatedAt,
	}
	if nodeOK && node.Type == enginemodel.NodeTypeApproval {
		for field, perm := range node.Config.FormPermissions {
			detail.FormPermissions[field] = string(perm)
		}
	}
	if taskRow.Status == "PENDING" {
		actions := []string{"approve", "reject", "return-to-starter", "transfer"}
		// 含并行网关的定义冻结不支持退回发起人（Phase 8）：重提交会从退回
		// 节点二次推进，若路径上有 split 将二次扇出分支致 join 到达计数失真，
		// 动作投影与 Runtime.ReturnToStarter 裁决同口径
		for i := range version.Snapshot.Nodes {
			if version.Snapshot.Nodes[i].Type == enginemodel.NodeTypeParallel {
				actions = []string{"approve", "reject", "transfer"}
				break
			}
		}
		detail.AllowedActions = actions
	}

	// 表单绑定投影：冻结快照全文 + 业务数据当前值（第 8.2 章双版本冻结）
	if instance.FormVersionID != 0 {
		content, formCode, formVersionNo, err := s.formDir.GetVersionContent(ctx, instance.FormVersionID)
		if err != nil {
			return nil, err
		}
		detail.FormCode = formCode
		detail.FormVersionNo = formVersionNo
		detail.FormContent = content
		if s.formData != nil {
			values, err := s.formData.GetData(ctx, provider.BusinessRef{
				TenantID:      instance.TenantID,
				AppID:         instance.AppID,
				FormID:        instance.FormID,
				FormVersionID: instance.FormVersionID,
				BusinessID:    instance.BusinessID,
			})
			if err != nil {
				return nil, err
			}
			detail.FormValues = values
		}
	}

	operationRows, err := s.reader.ListOperationRowsByInstance(ctx, instance.ID)
	if err != nil {
		return nil, err
	}
	for i := range operationRows {
		detail.Operations = append(detail.Operations, model.InstanceOperationView{
			ID: operationRows[i].ID, TaskID: operationRows[i].TaskID,
			OperatorMemberID: operationRows[i].OperatorMemberID,
			Type:             operationRows[i].OperationType,
			Payload:          []byte(operationRows[i].Payload),
			CreatedAt:        operationRows[i].CreatedAt,
		})
	}
	return detail, nil
}

// ---- 内部辅助 ----

// taskActionContext 任务级动作公共上下文：租户 + 成员校验（与发起/同意同口径）。
func (s *runtimeService) taskActionContext(ctx context.Context, member *iammodel.User) (uint, uint, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return 0, 0, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return 0, 0, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member not in tenant %d", tenantID))
	}
	return tenantID, member.ID, nil
}

// hasPermission 权限集判定（nil 成员视为空集）。
func (s *runtimeService) hasPermission(ctx context.Context, member *iammodel.User, permission string) bool {
	return s.permissions(ctx, member)[permission]
}

// recordActionAudit 提交后 best-effort 审计（与既有域同口径）。
func (s *runtimeService) recordActionAudit(ctx context.Context, member *iammodel.User, action string, taskID, instanceID uint) {
	if s.audit == nil {
		return
	}
	s.audit.Record(ctx, auditservice.Entry{
		Module: "workflow", Action: action, ResourceType: "workflow_task",
		ResourceID: strconv.FormatUint(uint64(taskID), 10),
		After:      map[string]any{"instanceId": instanceID, "operatorMemberId": memberIDPtr(member)},
	})
}

// actorsByTask 批量组装任务参与人视图。
func (s *runtimeService) actorsByTask(ctx context.Context, rows []model.WfTask) (map[uint][]model.TaskActorView, error) {
	ids := make([]uint, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].ID)
	}
	actorRows, err := s.reader.ListActorRowsByTaskIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[uint][]model.TaskActorView, len(rows))
	for i := range actorRows {
		if actorRows[i].ActorRole != "assignee" {
			continue
		}
		out[actorRows[i].TaskID] = append(out[actorRows[i].TaskID], model.TaskActorView{
			MemberID: actorRows[i].MemberID, DisplayName: actorRows[i].DisplayName,
		})
	}
	return out, nil
}

// taskSummaryFromRow 任务行 → 出网条目。
func taskSummaryFromRow(row model.WfTask, actorsByTask map[uint][]model.TaskActorView) model.TaskSummary {
	actors := actorsByTask[row.ID]
	if actors == nil {
		actors = []model.TaskActorView{}
	}
	return model.TaskSummary{
		ID: row.ID, InstanceID: row.InstanceID, NodeKey: row.NodeKey,
		Status: row.Status, Actors: actors,
		TransferredFrom: row.TransferredFromTaskID,
		CreatedAt:       row.CreatedAt,
	}
}

// definitionProjection 定义编码与版本号投影（我发起的/任务详情共用）。
func (s *runtimeService) definitionProjection(ctx context.Context, definitionID, versionID uint) (string, int, error) {
	definitionCode, err := s.definitions.FindCodeByID(ctx, definitionID)
	if err != nil {
		return "", 0, err
	}
	versionNo, err := s.definitions.FindVersionNoByID(ctx, versionID)
	if err != nil {
		return "", 0, err
	}
	return definitionCode, versionNo, nil
}

// parsePhase4Cursor 游标与分页上限（同定义域口径）。
func parsePhase4Cursor(limit int, cursor string) (int, uint, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	afterID, _, err := repository.ParseCursor(cursor)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid cursor")
	}
	return limit, afterID, nil
}

// memberIDPtr 审计载荷辅助（保持与既有 Entry 形态一致）。
func memberIDPtr(member *iammodel.User) uint {
	if member == nil {
		return 0
	}
	return member.ID
}
