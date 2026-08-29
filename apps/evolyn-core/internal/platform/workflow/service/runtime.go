package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"evolyn/internal/contextx"
	"evolyn/internal/engine/workflow/assignment"
	"evolyn/internal/engine/workflow/expression"
	"evolyn/internal/engine/workflow/provider"
	engineruntime "evolyn/internal/engine/workflow/runtime"
	enginetask "evolyn/internal/engine/workflow/task"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	wfapp "evolyn/internal/platform/workflow"
	"evolyn/internal/platform/workflow/adapter"
	"evolyn/internal/platform/workflow/model"
	"evolyn/internal/platform/workflow/repository"

	"gorm.io/gorm"
)

// runtimeService 最小 Runtime 应用服务（Phase 2）：事务边界在本层建立
// （TxManager 唯一事实源），内核 Runtime 在事务内推进；引擎 sentinel
// 错误在此统一映射为 WORKFLOW_* 稳定业务错误码。
type runtimeService struct {
	tx          TxManager
	engine      *engineruntime.Runtime
	reader      repository.RuntimeReader
	definitions *repository.EngineDefinitionReader
	formDir     adapter.FormDirectory
	access      AccessEvaluator
	audit       auditservice.Recorder
	// identity 身份窄端口（转办目标成员有效性校验，Phase 4）
	identity provider.IdentityProvider
	// formData 业务数据窄端口（任务详情表单数据投影，Phase 4）
	formData provider.BusinessDataProvider
}

// NewRuntimeService 构造最小 Runtime 服务（identity/formData 可为 nil：
// 转办校验与详情表单数据投影降级，便于单测）。
func NewRuntimeService(
	tx TxManager,
	engine *engineruntime.Runtime,
	reader repository.RuntimeReader,
	definitions *repository.EngineDefinitionReader,
	formDir adapter.FormDirectory,
	access AccessEvaluator,
	audit auditservice.Recorder,
	identity provider.IdentityProvider,
	formData provider.BusinessDataProvider,
) RuntimeService {
	return &runtimeService{
		tx: tx, engine: engine, reader: reader,
		definitions: definitions, formDir: formDir, access: access, audit: audit,
		identity: identity, formData: formData,
	}
}

// permissions 取当前成员权限集（nil 成员视为空集）。
func (s *runtimeService) permissions(ctx context.Context, member *iammodel.User) map[string]bool {
	if member == nil {
		return map[string]bool{}
	}
	return s.access.Permissions(ctx, member)
}

// mapEngineError 引擎 sentinel → 稳定业务错误码（细节只入日志）。
func mapEngineError(err error) error {
	switch {
	case errors.Is(err, engineruntime.ErrDefinitionNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		return httpx.Wrap(wfapp.ErrWorkflowNotFound, err)
	case errors.Is(err, engineruntime.ErrDefinitionNotPublished):
		return httpx.Wrap(wfapp.ErrNotPublished, err)
	case errors.Is(err, engineruntime.ErrInstanceAlreadyRunning):
		return httpx.Wrap(wfapp.ErrInstanceAlreadyRunning, err)
	case errors.Is(err, enginetask.ErrInstanceNotFound), errors.Is(err, enginetask.ErrTaskNotFound):
		return httpx.Wrap(wfapp.ErrTaskNotFound, err)
	case errors.Is(err, enginetask.ErrTaskNotPending):
		return httpx.Wrap(wfapp.ErrTaskNotPending, err)
	case errors.Is(err, enginetask.ErrTaskForbidden):
		return httpx.Wrap(wfapp.ErrTaskForbidden, err)
	case errors.Is(err, enginetask.ErrInstanceNotRunning):
		return httpx.Wrap(wfapp.ErrInstanceNotRunning, err)
	case errors.Is(err, engineruntime.ErrFormFieldForbidden):
		return httpx.Wrap(wfapp.ErrFormFieldForbidden, err)
	case errors.Is(err, engineruntime.ErrActionNotAllowed),
		errors.Is(err, engineruntime.ErrResubmitNodeMissing):
		return httpx.Wrap(wfapp.ErrActionNotAllowed, err)
	case errors.Is(err, engineruntime.ErrNotStarter):
		return httpx.Wrap(wfapp.ErrForbidden, err)
	case errors.Is(err, engineruntime.ErrRouteStuck), errors.Is(err, engineruntime.ErrNodeUnsupported),
		errors.Is(err, engineruntime.ErrAdvanceOverflow):
		// 校验器已保证的路由不变量运行期被打破：细节只入日志，出网统一口径
		return httpx.Wrap(wfapp.ErrDefinitionInvalid, err)
	}
	// 类型化错误：表达式求值失败 / 审批人解析为空（struct 错误走 errors.As）
	var exprErr *expression.ErrExpressionInvalid
	if errors.As(err, &exprErr) {
		return httpx.Wrap(wfapp.ErrExpressionInvalid, err)
	}
	var assigneeErr *assignment.ErrAssigneeNotFound
	if errors.As(err, &assigneeErr) {
		return httpx.Wrap(wfapp.ErrAssigneeNotFound, err)
	}
	return err
}

func (s *runtimeService) Start(ctx context.Context, member *iammodel.User, req *model.StartInstanceRequest) (*model.InstanceDetail, error) {
	if !s.permissions(ctx, member)["workflow-instances:create"] {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot start workflow instance"))
	}
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member not in tenant %d", tenantID))
	}
	if strings.TrimSpace(req.DefinitionCode) == "" || strings.TrimSpace(req.BusinessType) == "" ||
		strings.TrimSpace(req.BusinessID) == "" {
		return nil, httpx.Wrap(wfapp.ErrWorkflowCodeInvalid, fmt.Errorf("definitionCode/businessType/businessId required"))
	}

	// 表单绑定解析（可选）：内部 ID 出库前经窄端口校验归属与版本存在性
	var formID, formVersionID uint
	if req.FormCode != "" {
		if req.FormVersionNo <= 0 {
			return nil, wfapp.ErrFormVersionInvalid
		}
		resolvedFormID, resolvedVersionID, err := s.formDir.ResolveFormVersion(ctx, req.FormCode, req.FormVersionNo)
		if err != nil {
			return nil, err
		}
		formID, formVersionID = resolvedFormID, resolvedVersionID
	}

	input := engineruntime.StartInput{
		TenantID:        tenantID,
		Code:            req.DefinitionCode,
		BusinessType:    req.BusinessType,
		BusinessID:      req.BusinessID,
		StarterMemberID: member.ID,
		AppID:           req.AppID,
		FormID:          formID,
		FormVersionID:   formVersionID,
		IdempotencyKey:  req.IdempotencyKey,
	}
	var instanceID uint
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		result, err := s.engine.Start(tctx, input)
		if err != nil {
			return mapEngineError(err)
		}
		instanceID = result.InstanceID
		return nil
	}); err != nil {
		return nil, err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "workflow", Action: "start", ResourceType: "workflow_instance",
			ResourceID: fmt.Sprintf("%d", instanceID),
			After:      map[string]any{"definitionCode": req.DefinitionCode, "businessType": req.BusinessType, "businessId": req.BusinessID},
		})
	}
	return s.GetInstance(ctx, member, instanceID)
}

func (s *runtimeService) GetInstance(ctx context.Context, member *iammodel.User, instanceID uint) (*model.InstanceDetail, error) {
	if !s.permissions(ctx, member)["workflow-instances:get"] {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot read workflow instance %d", instanceID))
	}
	return s.buildDetail(ctx, instanceID)
}

func (s *runtimeService) Approve(ctx context.Context, member *iammodel.User, req *model.ApproveTaskRequest) (*model.ApproveTaskResult, error) {
	if !s.permissions(ctx, member)["workflow-tasks:create"] {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot approve tasks"))
	}
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member not in tenant %d", tenantID))
	}
	if req.TaskID == 0 {
		return nil, httpx.Wrap(wfapp.ErrTaskNotFound, fmt.Errorf("taskId required"))
	}
	// 审批编辑值：RawMessage → any（JSON 数字为 float64，与表单域校验口径一致）
	formValues := make(map[string]any, len(req.Values))
	for key, raw := range req.Values {
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, httpx.Wrap(wfapp.ErrFormFieldForbidden, fmt.Errorf("field %s: %w", key, err))
		}
		formValues[key] = value
	}

	var result *model.ApproveTaskResult
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		outcome, err := s.engine.Approve(tctx, engineruntime.ApproveInput{
			TenantID:         tenantID,
			TaskID:           req.TaskID,
			OperatorMemberID: member.ID,
			Comment:          req.Comment,
			FormValues:       formValues,
		})
		if err != nil {
			return mapEngineError(err)
		}
		result = &model.ApproveTaskResult{
			InstanceID:     outcome.InstanceID,
			InstanceStatus: string(outcome.InstanceStatus),
			NodeCompleted:  outcome.NodeCompleted,
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "workflow", Action: "approve", ResourceType: "workflow_task",
			ResourceID: fmt.Sprintf("%d", req.TaskID),
			After:      map[string]any{"instanceId": result.InstanceID, "nodeCompleted": result.NodeCompleted},
		})
	}
	return result, nil
}

// buildDetail 组装实例详情（绑定关系 + 节点/任务/参与人/操作时间线）。
func (s *runtimeService) buildDetail(ctx context.Context, instanceID uint) (*model.InstanceDetail, error) {
	instance, err := s.reader.FindInstanceRow(ctx, instanceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(wfapp.ErrInstanceNotFound, err)
		}
		return nil, err
	}
	definitionCode, err := s.definitions.FindCodeByID(ctx, instance.DefinitionID)
	if err != nil {
		return nil, err
	}
	versionNo, err := s.definitions.FindVersionNoByID(ctx, instance.DefinitionVersionID)
	if err != nil {
		return nil, err
	}

	nodeRows, err := s.reader.ListNodeRowsByInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	taskRows, err := s.reader.ListTaskRowsByInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	actorRows, err := s.reader.ListActorRowsByInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	operationRows, err := s.reader.ListOperationRowsByInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	detail := &model.InstanceDetail{
		ID:                  instance.ID,
		DefinitionCode:      definitionCode,
		DefinitionVersionNo: versionNo,
		BusinessType:        instance.BusinessType,
		BusinessID:          instance.BusinessID,
		AppID:               instance.AppID,
		FormID:              instance.FormID,
		FormVersionID:       instance.FormVersionID,
		Status:              instance.Status,
		StarterMemberID:     instance.StarterMemberID,
		CreatedAt:           instance.CreatedAt,
		Nodes:               make([]model.InstanceNodeView, 0, len(nodeRows)),
		Tasks:               make([]model.InstanceTaskView, 0, len(taskRows)),
		Operations:          make([]model.InstanceOperationView, 0, len(operationRows)),
	}
	if instance.IdempotencyKey != nil {
		detail.IdempotencyKey = *instance.IdempotencyKey
	}
	for i := range nodeRows {
		detail.Nodes = append(detail.Nodes, model.InstanceNodeView{
			ID: nodeRows[i].ID, NodeKey: nodeRows[i].NodeKey, Status: nodeRows[i].Status,
		})
	}
	actorsByTask := make(map[uint][]model.TaskActorView)
	for i := range actorRows {
		actorsByTask[actorRows[i].TaskID] = append(actorsByTask[actorRows[i].TaskID], model.TaskActorView{
			MemberID: actorRows[i].MemberID, DisplayName: actorRows[i].DisplayName,
		})
	}
	for i := range taskRows {
		actors := actorsByTask[taskRows[i].ID]
		if actors == nil {
			actors = []model.TaskActorView{}
		}
		detail.Tasks = append(detail.Tasks, model.InstanceTaskView{
			ID: taskRows[i].ID, NodeInstanceID: taskRows[i].NodeInstanceID,
			NodeKey: taskRows[i].NodeKey, Status: taskRows[i].Status,
			Actors: actors, CreatedAt: taskRows[i].CreatedAt,
		})
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
