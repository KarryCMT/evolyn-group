package runtime

import (
	"context"
	"sync"
	"time"

	"evolyn/internal/engine/workflow/definition"
	"evolyn/internal/engine/workflow/event"
	"evolyn/internal/engine/workflow/executor"
	"evolyn/internal/engine/workflow/expression"
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

	// ccRecords 抄送记录仓储（第 10.6 章，CC 节点执行时追加写）
	ccRecords repository.CCRepository
	// jobs 延时任务仓储（Phase 5：任务创建排期 + 实例终态联动取消；
	// Phase 7 扩展 service.invoke 异步执行排期）
	jobs repository.JobRepository
	// variables 流程变量仓储（Phase 7：表达式 variables.* 数据源加载与
	// service 节点响应映射写入；可为 nil——单测或无变量场景）
	variables repository.VariableRepository
	// serviceInvoker 服务节点出站调用窄端口（Phase 7，平台适配层承担
	// HTTP/SSRF 防护；可为 nil——无服务节点场景）
	serviceInvoker provider.ServiceInvoker

	// business 业务数据窄端口（第 15 章，Phase 3）：form.* 表达式数据源与
	// 审批编辑写回的唯一通道，内核不感知 form_records；可为 nil（单测）
	business provider.BusinessDataProvider
	// identity 身份窄端口：starter.* 表达式上下文填充；可为 nil（单测）
	identity provider.IdentityProvider

	// expressions 表达式引擎（发布期已预编译，运行期仅取产物求值）
	expressions expression.Engine
	// compiledCache 发布预编译产物缓存（版本 ID → CompiledDefinition；
	// 快照不可变故进程内缓存安全），互斥锁保护首次构建
	compiledMu    sync.Mutex
	compiledCache map[uint]*definition.CompiledDefinition
}

// DefinitionReader 运行时对定义快照的最小读取面（内核 SPI 细化：Runtime
// 只需要按 code/版本号/版本 ID 读快照，不感知定义仓储全貌）。
type DefinitionReader interface {
	FindDefinitionByCode(ctx context.Context, tenantID uint, code string) (*model.Definition, error)
	FindVersion(ctx context.Context, tenantID, definitionID uint, versionNo int) (*model.DefinitionVersion, error)
	FindVersionByID(ctx context.Context, tenantID, versionID uint) (*model.DefinitionVersion, error)
	// FindDefinitionCodeByID 按定义行 ID 反查公开编码（重提交等实例级动作
	// 构造表达式上下文使用；tenantID 仅为接口对齐，租户过滤由 ctx 承载）
	FindDefinitionCodeByID(ctx context.Context, tenantID, definitionID uint) (string, error)
}

// NewRuntime 构造运行时（publisher/business/identity/ccRecords 可为 nil：
// 跳过事件发布、表单数据接入与抄送落库，便于单测）。
func NewRuntime(
	definitions DefinitionReader,
	instances repository.InstanceRepository,
	executions repository.ExecutionRepository,
	nodes repository.NodeInstanceRepository,
	tasks repository.TaskRepository,
	operations repository.OperationRepository,
	executors executor.Registry,
	publisher provider.EventPublisher,
	business provider.BusinessDataProvider,
	identity provider.IdentityProvider,
	ccRecords repository.CCRepository,
	jobs repository.JobRepository,
	variables repository.VariableRepository,
	serviceInvoker provider.ServiceInvoker,
) *Runtime {
	return &Runtime{
		definitions:    definitions,
		instances:      instances,
		executions:     executions,
		nodes:          nodes,
		tasks:          tasks,
		operations:     operations,
		executors:      executors,
		navigator:      NewNavigator(nil),
		publisher:      publisher,
		taskEngine:     task.NewEngine(tasks, instances, operations, publisher, identity, jobs),
		business:       business,
		identity:       identity,
		expressions:    expression.NewExprEngine(),
		compiledCache:  make(map[uint]*definition.CompiledDefinition),
		ccRecords:      ccRecords,
		jobs:           jobs,
		variables:      variables,
		serviceInvoker: serviceInvoker,
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
	// FormValues 审批编辑的表单字段值（键=widgetName，可选）。Runtime 按节
	// 点字段权限过滤后经业务数据窄端口在同一事务内写回（第 13.2/15.4 章）：
	// 携带未授权字段即整体失败回滚，禁止直接改业务表
	FormValues map[string]any
	// AutoTimeout 超时自动动作触发（第 19.4 章）：操作人 0，跳过参与人
	// 校验（Actor 语义替代），操作流水记 TIMEOUT；仍走 Task Engine 正常
	// 执行路径，Worker 不得绕过
	AutoTimeout bool
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
	node, ok := version.Snapshot.NodeOf(outcome.NodeKey)
	if !ok || node.Type != model.NodeTypeApproval {
		return nil, ErrRouteStuck
	}
	// 审批编辑写回（第 13.2 章事务模板第 5/6 步）：按节点字段权限过滤后
	// 经业务数据窄端口由 Form Domain 校验写回，失败即整体回滚（含任务更新）
	if err := r.applyFormValues(ctx, instance, node, in.FormValues); err != nil {
		return nil, err
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
	// 剩余任务取消：排期 Job 联动取消（第 19 章）
	r.cancelJobsByNode(ctx, outcome.NodeInstanceID)
	r.publish(ctx, event.NodeCompleted, instance, outcome.NodeInstanceID, in.OperatorMemberID)

	advance, err := r.newAdvanceContext(ctx, instance, version, "")
	if err != nil {
		return nil, err
	}
	// 从已完成节点寻路并推进后续节点（取发布预编译产物求值）
	next, err := r.navigator.FindNextCompiled(advance.compiled, advance.env, outcome.NodeKey)
	if err != nil {
		return nil, err
	}
	if err := r.advance(ctx, advance, next); err != nil {
		return nil, err
	}
	return &ApproveResult{InstanceID: instance.ID, InstanceStatus: instance.Status, NodeCompleted: true}, nil
}

// applyFormValues 审批编辑写回（第 15.4 章审批提交协议）：
//   - 未携带编辑值：直通；
//   - 携带编辑值但实例未绑定表单/端口未装配：WORKFLOW_FORM_FIELD_FORBIDDEN；
//   - 请求字段未获节点 editable/required 授权：WORKFLOW_FORM_FIELD_FORBIDDEN；
//   - 授权字段经业务数据窄端口由 Form Domain 按冻结快照校验，校验失败
//     整体报错（同事务回滚），内核不做任何 form_records 直改。
func (r *Runtime) applyFormValues(ctx context.Context, instance *model.Instance, node *model.Node, values map[string]any) error {
	if len(values) == 0 {
		return nil
	}
	if r.business == nil || instance.FormVersionID == 0 {
		return ErrFormFieldForbidden
	}
	allowed := make(map[string]bool, len(node.Config.FormPermissions))
	for field, perm := range node.Config.FormPermissions {
		if perm == model.FieldPermissionEditable || perm == model.FieldPermissionRequired {
			allowed[field] = true
		}
	}
	for field := range values {
		if !allowed[field] {
			return ErrFormFieldForbidden
		}
	}
	return r.business.UpdateData(ctx, provider.BusinessRef{
		TenantID:      instance.TenantID,
		AppID:         instance.AppID,
		FormID:        instance.FormID,
		FormVersionID: instance.FormVersionID,
		BusinessID:    instance.BusinessID,
	}, values)
}

// advanceContext 推进环上下文：一次推进内共享实例/执行路径/快照/表达式环境。
type advanceContext struct {
	instance  *model.Instance
	execution *model.Execution
	env       *model.WorkflowContext
	doc       *model.Document
	compiled  *definition.CompiledDefinition
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
	// 身份窄端口：填充 starter.* 表达式上下文（登录名/归属部门，Phase 3）
	if r.identity != nil {
		userID, departmentID, err := r.identity.MemberContext(ctx, instance.TenantID, instance.StarterMemberID)
		if err != nil {
			return nil, err
		}
		env.Starter.UserID = userID
		env.Starter.DepartmentID = departmentID
	}
	// 流程变量：表达式 variables.* 数据源（Phase 7；service 节点响应映射
	// 写入后立即可读；仓储未装配时为空环境，单测场景）
	if r.variables != nil {
		vars, err := r.variables.ListVariablesByInstance(ctx, instance.ID)
		if err != nil {
			return nil, err
		}
		env.Variables = make(map[string]any, len(vars))
		for i := range vars {
			env.Variables[vars[i].Key] = vars[i].Value
		}
	}
	// 业务数据窄端口：填充 form.* 表达式数据源（发起时冻结的 Form 快照，
	// 第 8.2 章双版本冻结；读取失败即本次推进失败）
	if r.business != nil && instance.FormVersionID != 0 {
		values, err := r.business.GetData(ctx, provider.BusinessRef{
			TenantID:      instance.TenantID,
			AppID:         instance.AppID,
			FormID:        instance.FormID,
			FormVersionID: instance.FormVersionID,
			BusinessID:    instance.BusinessID,
		})
		if err != nil {
			return nil, err
		}
		env.Form = values
	}
	compiled, err := r.compiledFor(version)
	if err != nil {
		return nil, err
	}
	return &advanceContext{
		instance:  instance,
		execution: &executions[0],
		env:       env,
		doc:       &version.Snapshot,
		compiled:  compiled,
	}, nil
}

// compiledFor 取版本预编译产物（第 16 章「发布时预编译」）：快照不可变，
// 进程内按版本 ID 缓存一次构建结果，运行期禁止逐次重编译。
func (r *Runtime) compiledFor(version *model.DefinitionVersion) (*definition.CompiledDefinition, error) {
	r.compiledMu.Lock()
	defer r.compiledMu.Unlock()
	if compiled, ok := r.compiledCache[version.ID]; ok {
		return compiled, nil
	}
	compiled, err := definition.Compile(&version.Snapshot, r.expressions)
	if err != nil {
		return nil, err
	}
	r.compiledCache[version.ID] = compiled
	return compiled, nil
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
		// 抄送记录落库（第 10.6 章：非审批任务，独立记录 + CC 操作流水）
		if err := r.persistCC(ctx, a, nodeInstance, result); err != nil {
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
		if result.Async {
			// 服务节点异步挂起（Phase 7，第 19 章）：节点实例保持 RUNNING，
			// 排期 service.invoke Job 后停止推进——业务事务内不发外部请求，
			// Worker 独立事务调用完成后续跑（InvokeServiceNode）
			if err := r.scheduleServiceInvoke(ctx, a, nodeInstance, node); err != nil {
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
		next, err := r.navigator.FindNextCompiled(a.compiled, a.env, node.Key)
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
		// 排期超时/提醒 Job（Phase 5，第 19 章：任务创建时一次性排期）
		if err := r.scheduleJobs(ctx, a, nodeInstance, blueprint); err != nil {
			return err
		}
		r.publish(ctx, event.TaskCreated, a.instance, blueprint.ID, a.env.Starter.MemberID)
	}
	return nil
}

// defaultJobMaxRetry 排期 Job 默认重试上限（Worker 失败重试回队）。
const defaultJobMaxRetry = 3

// scheduleJobs 按节点配置为任务排期 task.timeout / task.reminder Job
// （jobs 未装配时跳过，单测场景；校验器已保证秒数与动作合法）。
func (r *Runtime) scheduleJobs(ctx context.Context, a *advanceContext, nodeInstance *model.NodeInstance, created model.Task) error {
	if r.jobs == nil {
		return nil
	}
	node, ok := a.doc.NodeOf(nodeInstance.NodeKey)
	if !ok {
		return nil
	}
	now := time.Now()
	jobs := make([]model.Job, 0, 2)
	if node.Config.Timeout != nil {
		jobs = append(jobs, model.Job{
			TenantID: a.instance.TenantID, Type: model.JobTypeTaskTimeout,
			InstanceID: a.instance.ID, NodeInstanceID: nodeInstance.ID, TaskID: created.ID,
			ExecuteAt: now.Add(time.Duration(node.Config.Timeout.Seconds) * time.Second),
			Status:    model.JobStatusPENDING, MaxRetryCount: defaultJobMaxRetry,
			Payload: map[string]any{"action": string(node.Config.Timeout.Action), "nodeKey": nodeInstance.NodeKey},
		})
	}
	if node.Config.Reminder != nil {
		jobs = append(jobs, model.Job{
			TenantID: a.instance.TenantID, Type: model.JobTypeTaskReminder,
			InstanceID: a.instance.ID, NodeInstanceID: nodeInstance.ID, TaskID: created.ID,
			ExecuteAt: now.Add(time.Duration(node.Config.Reminder.Seconds) * time.Second),
			Status:    model.JobStatusPENDING, MaxRetryCount: defaultJobMaxRetry,
			Payload: map[string]any{"nodeKey": nodeInstance.NodeKey},
		})
	}
	for i := range jobs {
		if err := r.jobs.CreateJob(ctx, &jobs[i]); err != nil {
			return err
		}
	}
	return nil
}

// persistCC 落库抄送记录与 CC 操作流水（第 10.6 章）：抄送对象解析快照
// 一次性写入；ccRecords 未装配时跳过（单测场景）。
func (r *Runtime) persistCC(ctx context.Context, a *advanceContext, nodeInstance *model.NodeInstance, result executor.ExecuteResult) error {
	if len(result.CCRecipients) == 0 {
		return nil
	}
	if r.ccRecords != nil {
		records := make([]model.CCRecord, 0, len(result.CCRecipients))
		for _, recipient := range result.CCRecipients {
			records = append(records, model.CCRecord{
				TenantID:       a.instance.TenantID,
				InstanceID:     a.instance.ID,
				NodeInstanceID: nodeInstance.ID,
				NodeKey:        nodeInstance.NodeKey,
				MemberID:       recipient.MemberID,
				DisplayName:    recipient.DisplayName,
			})
		}
		if err := r.ccRecords.CreateCCRecords(ctx, records); err != nil {
			return err
		}
	}
	recipients := make([]map[string]any, 0, len(result.CCRecipients))
	for _, recipient := range result.CCRecipients {
		recipients = append(recipients, map[string]any{
			"memberId": recipient.MemberID, "displayName": recipient.DisplayName,
		})
	}
	return r.operations.AppendOperation(ctx, &model.Operation{
		TenantID:   a.instance.TenantID,
		InstanceID: a.instance.ID,
		Type:       model.OperationTypeCC,
		Payload:    map[string]any{"nodeKey": nodeInstance.NodeKey, "recipients": recipients},
	})
}

// cancelJobsByNode 节点完成联动取消排期 Job（jobs 未装配时跳过）。
func (r *Runtime) cancelJobsByNode(ctx context.Context, nodeInstanceID uint) {
	if r.jobs == nil {
		return
	}
	_ = r.jobs.CancelJobsByNodeInstance(ctx, nodeInstanceID)
}

// cancelJobsByInstance 实例终态联动取消排期 Job（jobs 未装配时跳过）。
func (r *Runtime) cancelJobsByInstance(ctx context.Context, instanceID uint) {
	if r.jobs == nil {
		return
	}
	_ = r.jobs.CancelJobsByInstance(ctx, instanceID)
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

// publishWithNode 带节点实例维度的事件发布：同一实例内可重复发生的事件
// （退回发起人）以 NodeInstanceID 参与平台侧幂等键构造（Phase 6）。
func (r *Runtime) publishWithNode(ctx context.Context, eventName string, instance *model.Instance, nodeInstanceID, actorMemberID uint) {
	if r.publisher == nil {
		return
	}
	_ = r.publisher.PublishInTx(ctx, provider.Event{
		EventName:      eventName,
		TenantID:       instance.TenantID,
		InstanceID:     instance.ID,
		NodeInstanceID: nodeInstanceID,
		ActorMemberID:  actorMemberID,
	})
}
