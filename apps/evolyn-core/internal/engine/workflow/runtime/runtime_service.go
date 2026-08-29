package runtime

// 服务节点执行编排（Phase 7，第 12.1/19 章）：service 节点在推进环中以
// Async 挂起并排期 wf_job service.invoke，本文件承接 Worker 侧的调用执行
// 与续跑推进——HTTP 调用经 ServiceInvoker 窄端口在 Job 独立事务内完成，
// 业务事务（发起/审批推进）不承载外部请求；失败由 Worker 重试记账退避
// 重试（第 19.1 章 service.retry 语义），PostgreSQL 仍是流程状态唯一事实源。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"evolyn/internal/engine/workflow/event"

	"evolyn/internal/engine/workflow/expression"
	"evolyn/internal/engine/workflow/model"
	"evolyn/internal/engine/workflow/provider"
	"evolyn/internal/engine/workflow/task"
)

// ServiceInvokeInput 服务节点调用输入（Job Worker 触发；操作人 0=系统）。
type ServiceInvokeInput struct {
	TenantID       uint
	InstanceID     uint
	NodeInstanceID uint
}

// ServiceInvokeResult 服务节点调用结果。
type ServiceInvokeResult struct {
	InstanceID uint
	// InstanceStatus 调用后续跑推进的实例状态（可能已推进至 COMPLETED）
	InstanceStatus model.InstanceStatus
	// Completed 本次调用是否使服务节点完成（false=幂等空跑：节点已终态）
	Completed bool
}

// InvokeServiceNode 执行 service 节点的出站调用并续跑推进：
// 行锁实例校验 RUNNING → 节点实例幂等校验（已终态空跑）→ 模板插值构造
// 请求 → ServiceInvoker 出站调用 → 响应映射写流程变量 → 节点 COMPLETED →
// SERVICE 操作流水 → 从下一节点续跑推进环。任意一步失败整体回滚（含调用
// 本身无副作用），由 Worker 重试记账退避回队。
func (r *Runtime) InvokeServiceNode(ctx context.Context, in ServiceInvokeInput) (*ServiceInvokeResult, error) {
	// 1-2. 行锁实例 + 状态机校验（与审批动作同口径）
	instance, err := r.instances.FindInstanceByIDForUpdate(ctx, in.TenantID, in.InstanceID)
	if err != nil {
		return nil, task.ErrInstanceNotFound
	}
	if instance.Status != model.InstanceStatusRUNNING {
		return nil, task.ErrInstanceNotRunning
	}
	nodeInstance, err := r.nodes.FindNodeInstanceByID(ctx, in.TenantID, in.NodeInstanceID)
	if err != nil {
		return nil, fmt.Errorf("node instance %d: %w", in.NodeInstanceID, err)
	}
	// 幂等：节点已终态（前次调用已完成但回写失败等）直接空跑成功
	if nodeInstance.Status != model.NodeInstanceStatusRUNNING {
		return &ServiceInvokeResult{InstanceID: instance.ID, InstanceStatus: instance.Status, Completed: false}, nil
	}
	// 3. 快照节点配置（发布后冻结，禁止读草稿）
	version, err := r.definitions.FindVersionByID(ctx, in.TenantID, instance.DefinitionVersionID)
	if err != nil {
		return nil, err
	}
	node, ok := version.Snapshot.NodeOf(nodeInstance.NodeKey)
	if !ok || node.Type != model.NodeTypeService || node.Config.Service == nil {
		return nil, ErrRouteStuck
	}
	// 4. 运行上下文（含 variables.* 白名单数据源加载）
	definitionCode, err := r.definitionCode(ctx, instance.DefinitionID)
	if err != nil {
		return nil, err
	}
	a, err := r.newAdvanceContext(ctx, instance, version, definitionCode)
	if err != nil {
		return nil, err
	}
	// 5. 模板插值 + 出站调用（SSRF 防护/白名单/超时由平台适配层强制）
	request, err := r.buildServiceRequest(a, node, nodeInstance)
	if err != nil {
		return nil, err
	}
	if r.serviceInvoker == nil {
		return nil, ErrNodeUnsupported
	}
	response, err := r.serviceInvoker.Invoke(ctx, *request)
	if err != nil {
		// 非调用失败不落任何状态：事务回滚由 Worker 重试记账退避回队
		return nil, err
	}
	// 6. 响应映射写流程变量（后续节点条件经 variables.* 读取）
	if err := r.applyServiceVariables(ctx, node.Config.Service, response, a); err != nil {
		return nil, err
	}
	// 7. 状态机 RUNNING → COMPLETED（迁移表裁决）+ SERVICE 操作流水
	if !task.CanTransitionNodeInstance(nodeInstance.Status, model.NodeInstanceStatusCOMPLETED) {
		return nil, ErrRouteStuck
	}
	nodeInstance.Status = model.NodeInstanceStatusCOMPLETED
	if err := r.nodes.SaveNodeInstance(ctx, nodeInstance); err != nil {
		return nil, err
	}
	if err := r.operations.AppendOperation(ctx, &model.Operation{
		TenantID:   instance.TenantID,
		InstanceID: instance.ID,
		TaskID:     0,
		Type:       model.OperationTypeService,
		Payload: map[string]any{
			"nodeKey": nodeInstance.NodeKey, "status": "SUCCEEDED",
			"httpStatus": response.StatusCode, "durationMs": response.Duration.Milliseconds(),
			"variables": mappedVariableKeys(node.Config.Service),
		},
	}); err != nil {
		return nil, err
	}
	// 8. 续跑推进：服务节点完成语义与瞬时节点一致（NodeCompleted 事件 +
	// Navigator 寻路后续节点，实例终态由 advance 内 End 分支收口）
	r.publishNodeCompleted(ctx, instance, nodeInstance.ID)
	next, err := r.navigator.FindNextCompiled(a.compiled, a.env, node.Key)
	if err != nil {
		return nil, err
	}
	if err := r.advance(ctx, a, next); err != nil {
		return nil, err
	}
	return &ServiceInvokeResult{InstanceID: instance.ID, InstanceStatus: instance.Status, Completed: true}, nil
}

// buildServiceRequest 由发布预编译产物插值构造出站请求（运行期禁止重编译，
// 第 16 章）；幂等键按实例+节点实例稳定，重试/重放对对端可去重。
func (r *Runtime) buildServiceRequest(a *advanceContext, node *model.Node, nodeInstance *model.NodeInstance) (*provider.ServiceRequest, error) {
	cfg := node.Config.Service
	templates, ok := a.compiled.ServiceTemplates[node.Key]
	if !ok {
		return nil, fmt.Errorf("service node %s templates not compiled", node.Key)
	}
	env := a.env.ExpressionEnv()
	url, err := expression.RenderTemplate(templates.URL, env)
	if err != nil {
		return nil, fmt.Errorf("service node %s url render: %w", node.Key, err)
	}
	headers := make(map[string]string, len(templates.Headers))
	for name, segments := range templates.Headers {
		value, err := expression.RenderTemplate(segments, env)
		if err != nil {
			return nil, fmt.Errorf("service node %s header %s render: %w", node.Key, name, err)
		}
		headers[name] = value
	}
	var body string
	if cfg.Body != "" {
		if body, err = expression.RenderTemplate(templates.Body, env); err != nil {
			return nil, fmt.Errorf("service node %s body render: %w", node.Key, err)
		}
	}
	method := cfg.Method
	if method == "" {
		method = model.ServiceDefaultMethod
	}
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = model.ServiceDefaultTimeoutSeconds
	}
	return &provider.ServiceRequest{
		TenantID:       a.instance.TenantID,
		InstanceID:     a.instance.ID,
		NodeInstanceID: nodeInstance.ID,
		NodeKey:        node.Key,
		Method:         method,
		URL:            url,
		Headers:        headers,
		Body:           body,
		TimeoutSeconds: timeout,
	}, nil
}

// applyServiceVariables 响应映射提取并落流程变量：JSON 点分路径提取 →
// 标量类型收敛（V1 冻结 string/number/boolean）→ upsert wf_variable →
// 同步刷新本次推进的表达式环境（续跑节点立即可读）。required 提取失败
// 即整体失败回滚；非 required 失败跳过该变量。
func (r *Runtime) applyServiceVariables(ctx context.Context, cfg *model.ServiceConfig, response *provider.ServiceResponse, a *advanceContext) error {
	if len(cfg.ResponseMapping) == 0 {
		return nil
	}
	var doc any
	if len(response.Body) > 0 {
		if err := json.Unmarshal(response.Body, &doc); err != nil {
			// 响应体非 JSON：视为整体提取失败，按各映射 required 裁决
			return r.mappingExtractionError(cfg, fmt.Errorf("响应体不是合法 JSON: %w", err))
		}
	}
	vars := make([]model.Variable, 0, len(cfg.ResponseMapping))
	for i := range cfg.ResponseMapping {
		m := &cfg.ResponseMapping[i]
		value, found := extractJSONPath(doc, m.Path)
		if !found {
			if m.Required {
				return fmt.Errorf("响应映射变量 %q 提取失败（path=%q）: %w", m.Variable, m.Path, errMappingNotFound)
			}
			continue
		}
		variable, ok, err := coerceScalarVariable(m, value)
		if err != nil {
			return err
		}
		if !ok {
			// 非标量且非 required：跳过（复杂结构不进入表达式环境）
			continue
		}
		variable.InstanceID = a.instance.ID
		vars = append(vars, *variable)
	}
	for i := range vars {
		if err := r.variables.SaveVariable(ctx, &vars[i]); err != nil {
			return err
		}
		// 表达式环境同步刷新（续跑节点/条件分支立即可读）
		if a.env.Variables == nil {
			a.env.Variables = map[string]any{}
		}
		a.env.Variables[vars[i].Key] = vars[i].Value
	}
	return nil
}

// mappingExtractionError 非 required 场景的解析失败不阻断推进（变量缺失
// 容忍），required 场景整体失败；统一包一层定位信息。
func (r *Runtime) mappingExtractionError(cfg *model.ServiceConfig, cause error) error {
	for i := range cfg.ResponseMapping {
		if cfg.ResponseMapping[i].Required {
			return fmt.Errorf("响应映射变量 %q: %w", cfg.ResponseMapping[i].Variable, cause)
		}
	}
	return nil
}

// errMappingNotFound 点分路径未命中（required 提取失败的稳定语义）。
var errMappingNotFound = fmt.Errorf("响应路径未命中")

// extractJSONPath 点分路径提取（"data.orderId"；空路径=整个文档）。
// 仅支持对象键与数组下标（数字段）两级语义，够用即止不造完整 JSONPath。
func extractJSONPath(doc any, path string) (any, bool) {
	if strings.TrimSpace(path) == "" {
		return doc, true
	}
	current := doc
	for _, seg := range strings.Split(path, ".") {
		switch node := current.(type) {
		case map[string]any:
			value, ok := node[seg]
			if !ok {
				return nil, false
			}
			current = value
		case []any:
			index := 0
			if _, err := fmt.Sscanf(seg, "%d", &index); err != nil || index < 0 || index >= len(node) {
				return nil, false
			}
			current = node[index]
		default:
			return nil, false
		}
	}
	return current, true
}

// coerceScalarVariable 提取值收敛为 V1 冻结的标量变量（string/number/
// boolean）；非标量：required 即失败，否则跳过（ok=false）。
func coerceScalarVariable(m *model.ServiceResponseMapping, value any) (*model.Variable, bool, error) {
	variable := &model.Variable{Key: m.Variable}
	switch v := value.(type) {
	case string:
		variable.ValueType = model.VariableTypeString
		variable.Value = v
	case float64:
		variable.ValueType = model.VariableTypeNumber
		variable.Value = v
	case bool:
		variable.ValueType = model.VariableTypeBoolean
		variable.Value = v
	case nil:
		if m.Required {
			return nil, false, fmt.Errorf("响应映射变量 %q 提取到 null: %w", m.Variable, errMappingNotFound)
		}
		return nil, false, nil
	default:
		if m.Required {
			return nil, false, fmt.Errorf("响应映射变量 %q 为复杂结构（V1 仅支持标量）", m.Variable)
		}
		return nil, false, nil
	}
	return variable, true, nil
}

// mappedVariableKeys 映射变量名清单（操作流水载荷，便于时间线追溯）。
func mappedVariableKeys(cfg *model.ServiceConfig) []string {
	keys := make([]string, 0, len(cfg.ResponseMapping))
	for i := range cfg.ResponseMapping {
		keys = append(keys, cfg.ResponseMapping[i].Variable)
	}
	return keys
}

// scheduleServiceInvoke 排期 service.invoke Job（execute_at=now，Worker
// 下一轮领取；重试上限取节点配置，退避由 Worker 重试记账承担）。
func (r *Runtime) scheduleServiceInvoke(ctx context.Context, a *advanceContext, nodeInstance *model.NodeInstance, node *model.Node) error {
	if r.jobs == nil {
		return nil
	}
	cfg := node.Config.Service
	maxRetries := model.ServiceDefaultMaxRetries
	if cfg != nil && cfg.MaxRetries != nil {
		maxRetries = *cfg.MaxRetries
	}
	return r.jobs.CreateJob(ctx, &model.Job{
		TenantID:       a.instance.TenantID,
		Type:           model.JobTypeServiceInvoke,
		InstanceID:     a.instance.ID,
		NodeInstanceID: nodeInstance.ID,
		ExecuteAt:      time.Now(),
		Status:         model.JobStatusPENDING,
		MaxRetryCount:  maxRetries,
		Payload:        map[string]any{"nodeKey": nodeInstance.NodeKey},
	})
}

// publishNodeCompleted 节点完成事件（best-effort，第 18.3 章随事务写入）。
func (r *Runtime) publishNodeCompleted(ctx context.Context, instance *model.Instance, nodeInstanceID uint) {
	if r.publisher == nil {
		return
	}
	_ = r.publisher.PublishInTx(ctx, provider.Event{
		EventName:      event.NodeCompleted,
		TenantID:       instance.TenantID,
		InstanceID:     instance.ID,
		NodeInstanceID: nodeInstanceID,
	})
}
