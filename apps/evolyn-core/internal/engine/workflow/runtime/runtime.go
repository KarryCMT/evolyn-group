package runtime

import (
	"context"

	"evolyn/internal/engine/workflow/event"
	"evolyn/internal/engine/workflow/executor"
	"evolyn/internal/engine/workflow/model"
	"evolyn/internal/engine/workflow/provider"
	"evolyn/internal/engine/workflow/repository"
	"evolyn/internal/engine/workflow/task"
)

// Runtime 流程运行时（第 12 章）：StartWorkflow 与人工任务后的推进编排。
// 事务由平台层经 TxManager 包裹后调用本包方法（仓储适配层经 ResolveDB
// 加入同一事务，任意一步失败整体回滚，第 13 章）；本包不做事务开启。
type Runtime struct {
	definitions DefinitionReader
	instances   repository.InstanceRepository
	executions  repository.ExecutionRepository
	nodes       repository.NodeInstanceRepository
	tasks       repository.TaskRepository
	operations  repository.OperationRepository

	executors executor.Registry
	navigator *Navigator
	publisher provider.EventPublisher
	// taskEngine 人工任务引擎（Runtime 与 Task Engine 分离，第 12.2 章）
	taskEngine *task.Engine
}

// DefinitionReader 运行时对定义快照的最小读取面（内核 SPI 细化：Runtime
// 只需要按 code/版本号/版本 ID 读快照，不感知定义仓储全貌）。
type DefinitionReader interface {
	FindDefinitionByCode(ctx context.Context, tenantID uint, code string) (*model.Definition, error)
	FindVersion(ctx context.Context, tenantID, definitionID uint, versionNo int) (*model.DefinitionVersion, error)
	FindVersionByID(ctx context.Context, tenantID, versionID uint) (*model.DefinitionVersion, error)
}

// NewRuntime 构造运行时（publisher 可为 nil：跳过事件发布，便于单测）。
func NewRuntime(
	definitions DefinitionReader,
	instances repository.InstanceRepository,
	executions repository.ExecutionRepository,
	nodes repository.NodeInstanceRepository,
	tasks repository.TaskRepository,
	operations repository.OperationRepository,
	executors executor.Registry,
	publisher provider.EventPublisher,
) *Runtime {
	return &Runtime{
		definitions: definitions,
		instances:   instances,
		executions:  executions,
		nodes:       nodes,
		tasks:       tasks,
		operations:  operations,
		executors:   executors,
		navigator:   NewNavigator(nil),
		publisher:   publisher,
		taskEngine:  task.NewEngine(tasks, instances, operations, publisher),
	}
}

// StartInput 发起流程输入（第 14 章）：业务绑定与请求幂等双口令。
type StartInput struct {
	TenantID uint
	// Code 流程定义稳定公开标识（内部 ID 不出网）
	Code string
	// BusinessType / BusinessID 业务绑定（业务幂等键，第 14.1 章）
	BusinessType string
	BusinessID   string
	// StarterMemberID 发起人成员 ID
	StarterMemberID uint
	// AppID / FormID / FormVersionID 表单绑定（0=未绑定；Phase 3 接入
	// Form Domain 后由 FormDirectory 窄端口解析填充）
	AppID         uint
	FormID        uint
	FormVersionID uint
	// IdempotencyKey 请求幂等键（空=未携带；命中重放返回同一实例，第 14.2 章）
	IdempotencyKey string
}

// StartResult 发起结果。
type StartResult struct {
	InstanceID uint
	Status     model.InstanceStatus
	// IdempotentReplay 命中请求幂等重放（未重复创建实例）
	IdempotentReplay bool
}

// Start 发起流程实例（第 12.2 章主链路）：
// 定义校验 → 双层幂等校验 → 创建实例/根执行路径 → START 操作流水 →
// 推进环执行至人工审批挂起（WAIT）或实例完成（无审批节点的直通流）。
func (r *Runtime) Start(ctx context.Context, in StartInput) (*StartResult, error) {
	def, err := r.definitions.FindDefinitionByCode(ctx, in.TenantID, in.Code)
	if err != nil {
		return nil, ErrDefinitionNotFound
	}
	// 请求幂等（第 14.2 章）：命中即重放，禁止生成第二个实例
	if in.IdempotencyKey != "" {
		existing, err := r.instances.FindInstanceByIdempotencyKey(ctx, in.TenantID, in.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return &StartResult{InstanceID: existing.ID, Status: existing.Status, IdempotentReplay: true}, nil
		}
	}
	// 业务幂等（第 14.1 章）：同业务键 RUNNING 实例唯一（部分唯一索引兜底）
	existing, err := r.instances.FindRunningInstanceByBusiness(ctx, in.TenantID, in.BusinessType, in.BusinessID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrInstanceAlreadyRunning
	}
	// 版本绑定：只允许已发布定义的最新版本（发布后冻结，第 8.1 章）
	if def.PublishedVersion == 0 || def.Status == model.DefinitionStatusDeleted {
		return nil, ErrDefinitionNotPublished
	}
	version, err := r.definitions.FindVersion(ctx, in.TenantID, def.ID, def.PublishedVersion)
	if err != nil {
		return nil, ErrDefinitionNotPublished
	}

	instance := &model.Instance{
		TenantID:            in.TenantID,
		DefinitionID:        def.ID,
		DefinitionVersionID: version.ID,
		BusinessType:        in.BusinessType,
		BusinessID:          in.BusinessID,
		AppID:               in.AppID,
		FormID:              in.FormID,
		FormVersionID:       in.FormVersionID,
		Status:              model.InstanceStatusRUNNING,
		StarterMemberID:     in.StarterMemberID,
		IdempotencyKey:      in.IdempotencyKey,
	}
	if err := r.instances.CreateInstance(ctx, instance); err != nil {
		return nil, err
	}
	execution := &model.Execution{
		TenantID:   in.TenantID,
		InstanceID: instance.ID,
		Status:     model.ExecutionStatusRUNNING,
	}
	if err := r.executions.CreateExecution(ctx, execution); err != nil {
		return nil, err
	}
	if err := r.operations.AppendOperation(ctx, &model.Operation{
		TenantID:         in.TenantID,
		InstanceID:       instance.ID,
		OperatorMemberID: in.StarterMemberID,
		Type:             model.OperationTypeStart,
		Payload:          map[string]any{"definitionCode": def.Code, "versionNo": version.VersionNo},
	}); err != nil {
		return nil, err
	}
	r.publish(ctx, event.InstanceStarted, instance, 0, in.StarterMemberID)

	advance, err := r.newAdvanceContext(ctx, instance, version, def.Code)
	if err != nil {
		return nil, err
	}
	if err := r.advance(ctx, advance, advanceStartKey(&version.Snapshot)); err != nil {
		return nil, err
	}
	return &StartResult{InstanceID: instance.ID, Status: instance.Status}, nil
}

// advanceStartKey 定位快照中的 start 节点（校验器保证恰好一个）。
func advanceStartKey(doc *model.Document) string {
	for i := range doc.Nodes {
		if doc.Nodes[i].Type == model.NodeTypeStart {
			return doc.Nodes[i].Key
		}
	}
	return ""
}

// ApproveInput 审批同意输入。
type ApproveInput struct {
	TenantID         uint
	TaskID           uint
	OperatorMemberID uint
	Comment          string
}

// ApproveResult 审批同意结果。
type ApproveResult struct {
	InstanceID uint
	// InstanceStatus 实例推进后状态（完成最后一个审批时变为 COMPLETED）
	InstanceStatus model.InstanceStatus
	// NodeCompleted 本次审批是否使节点达成完成条件
	NodeCompleted bool
}

// Approve 审批同意（第 13.2 章事务模板）：Task Engine 完成任务级裁决后，
// Runtime 判定节点完成并推进后续节点（同一事务内同步推进，禁止先更新
// 任务再异步推进——第 13.3 章禁止事项）。
func (r *Runtime) Approve(ctx context.Context, in ApproveInput) (*ApproveResult, error) {
	outcome, err := r.taskEngine.Approve(ctx, task.ApproveInput{
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
	node, ok := version.Snapshot.NodeOf(outcome.NodeKey)
	if !ok || node.Type != model.NodeTypeApproval {
		return nil, ErrRouteStuck
	}
	// 节点完成判定（第 11 章冻结规则，含会签 ceil 阈值）
	nodeTasks, err := r.tasks.ListTasksByNodeInstance(ctx, outcome.NodeInstanceID)
	if err != nil {
		return nil, err
	}
	if !task.NodeCompleted(node.Config.ApprovalMode, node.Config.PassRatio, nodeTasks) {
		return &ApproveResult{InstanceID: instance.ID, InstanceStatus: instance.Status, NodeCompleted: false}, nil
	}
	// 节点完成：状态机 WAITING → COMPLETED，联动取消剩余 PENDING 任务
	//（或签淘汰/会签达标后剩余任务，第 10/11 章；Runtime 仅允许推进一次）
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
	r.publish(ctx, event.NodeCompleted, instance, outcome.NodeInstanceID, in.OperatorMemberID)

	advance, err := r.newAdvanceContext(ctx, instance, version, "")
	if err != nil {
		return nil, err
	}
	// 从已完成节点寻路并推进后续节点
	next, err := r.navigator.FindNext(advance.env, &version.Snapshot, outcome.NodeKey)
	if err != nil {
		return nil, err
	}
	if err := r.advance(ctx, advance, next); err != nil {
		return nil, err
	}
	return &ApproveResult{InstanceID: instance.ID, InstanceStatus: instance.Status, NodeCompleted: true}, nil
}

// advanceContext 推进环上下文：一次推进内共享实例/执行路径/快照/表达式环境。
type advanceContext struct {
	instance  *model.Instance
	execution *model.Execution
	env       *model.WorkflowContext
	doc       *model.Document
}

func (r *Runtime) newAdvanceContext(ctx context.Context, instance *model.Instance, version *model.DefinitionVersion, definitionCode string) (*advanceContext, error) {
	executions, err := r.executions.ListExecutionsByInstance(ctx, instance.ID)
	if err != nil {
		return nil, err
	}
	if len(executions) == 0 {
		return nil, ErrRouteStuck
	}
	env := &model.WorkflowContext{
		TenantID: instance.TenantID,
		Instance: model.InstanceContext{
			InstanceID:          instance.ID,
			DefinitionCode:      definitionCode,
			DefinitionVersionNo: version.VersionNo,
			BusinessType:        instance.BusinessType,
			BusinessID:          instance.BusinessID,
			FormVersionID:       instance.FormVersionID,
		},
		Starter:   model.UserContext{MemberID: instance.StarterMemberID},
		Form:      map[string]any{},
		Variables: map[string]any{},
	}
	return &advanceContext{
		instance:  instance,
		execution: &executions[0],
		env:       env,
		doc:       &version.Snapshot,
	}, nil
}

// advance 推进环：执行 current 节点并沿导航链推进，直到人工审批挂起
// （WAIT）或实例到达 End（COMPLETED）。步数上限防御快照异常死循环。
func (r *Runtime) advance(ctx context.Context, a *advanceContext, current string) error {
	doc := a.doc
	if doc == nil {
		return ErrRouteStuck
	}
	for step := 0; step <= len(doc.Nodes); step++ {
		node, ok := doc.NodeOf(current)
		if !ok {
			return ErrRouteStuck
		}
		exec, ok := r.executors.ExecutorOf(node.Type)
		if !ok {
			return ErrNodeUnsupported
		}
		// 节点实例创建即 RUNNING（V1 无「创建后不激活」场景）
		nodeInstance := &model.NodeInstance{
			TenantID:    a.instance.TenantID,
			InstanceID:  a.instance.ID,
			ExecutionID: a.execution.ID,
			NodeKey:     node.Key,
			Status:      model.NodeInstanceStatusRUNNING,
		}
		if err := r.nodes.CreateNodeInstance(ctx, nodeInstance); err != nil {
			return err
		}
		result, err := exec.Execute(ctx, executor.ExecuteInput{
			Ctx:          a.env,
			Instance:     a.instance,
			NodeInstance: nodeInstance,
			Node:         *node,
		})
		if err != nil {
			return err
		}
		// 落库执行器产出的任务与参与人快照（第 13.2 章事务模板第 11 步）
		if err := r.persistTasks(ctx, a, nodeInstance, result); err != nil {
			return err
		}
		if result.Wait {
			// 人工审批：节点挂起等待，Runtime 停止推进（第 12.2 章）
			nodeInstance.Status = model.NodeInstanceStatusWAITING
			if err := r.nodes.SaveNodeInstance(ctx, nodeInstance); err != nil {
				return err
			}
			r.publish(ctx, event.NodeEntered, a.instance, 0, 0)
			return nil
		}
		// 瞬时节点完成
		nodeInstance.Status = model.NodeInstanceStatusCOMPLETED
		if err := r.nodes.SaveNodeInstance(ctx, nodeInstance); err != nil {
			return err
		}
		r.publish(ctx, event.NodeCompleted, a.instance, 0, 0)
		if node.Type == model.NodeTypeEnd {
			// 实例终态：执行路径与实例同事务收口（状态机 RUNNING → COMPLETED）
			if !task.CanTransitionExecution(a.execution.Status, model.ExecutionStatusCOMPLETED) {
				return ErrRouteStuck
			}
			a.execution.Status = model.ExecutionStatusCOMPLETED
			if err := r.executions.SaveExecution(ctx, a.execution); err != nil {
				return err
			}
			if !task.CanTransitionInstance(a.instance.Status, model.InstanceStatusCOMPLETED) {
				return ErrRouteStuck
			}
			a.instance.Status = model.InstanceStatusCOMPLETED
			if err := r.instances.SaveInstance(ctx, a.instance); err != nil {
				return err
			}
			r.publish(ctx, event.InstanceCompleted, a.instance, 0, 0)
			return nil
		}
		next, err := r.navigator.FindNext(a.env, doc, node.Key)
		if err != nil {
			return err
		}
		current = next
	}
	return ErrAdvanceOverflow
}

// persistTasks 落库执行器创建的任务蓝图与参与人快照，并发布 task.created。
func (r *Runtime) persistTasks(ctx context.Context, a *advanceContext, nodeInstance *model.NodeInstance, result executor.ExecuteResult) error {
	for i := range result.CreatedTasks {
		blueprint := result.CreatedTasks[i]
		blueprint.InstanceID = a.instance.ID
		blueprint.NodeInstanceID = nodeInstance.ID
		// 仓储适配层负责回填 blueprint.ID（GORM 主键回填语义）
		if err := r.tasks.CreateTask(ctx, &blueprint); err != nil {
			return err
		}
		if i < len(result.CreatedActors) {
			if err := r.tasks.ReplaceActors(ctx, blueprint.ID, result.CreatedActors[i]); err != nil {
				return err
			}
		}
		r.publish(ctx, event.TaskCreated, a.instance, blueprint.ID, a.env.Starter.MemberID)
	}
	return nil
}

func (r *Runtime) publish(ctx context.Context, eventName string, instance *model.Instance, taskID, actorMemberID uint) {
	if r.publisher == nil {
		return
	}
	_ = r.publisher.PublishInTx(ctx, provider.Event{
		EventName:     eventName,
		TenantID:      instance.TenantID,
		InstanceID:    instance.ID,
		TaskID:        taskID,
		ActorMemberID: actorMemberID,
	})
}
