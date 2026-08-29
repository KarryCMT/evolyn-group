// Package definition 流程定义引擎：DSL 严格校验器与发布预编译器。
// 发布 Definition Version 前必须整体通过本包校验（第 7.5 章清单），
// 校验规则是前后端共同协议的一部分，逐字冻结、不得运行时放宽。
package definition

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"evolyn/internal/engine/workflow/assignment"
	"evolyn/internal/engine/workflow/expression"
	"evolyn/internal/engine/workflow/model"
)

// KeyPattern Node/Edge key 命名规则（冻结）：字母开头，字母/数字/下划线，
// 1~64 位。前后端一致采用，避免各端自定义命名校验漂移。
var KeyPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,63}$`)

// ValidationError 单条校验错误：Path 为协议内的定位路径（如
// nodes[2].config.assignee），Code 为稳定错误标识（出网映射
// WORKFLOW_DEFINITION_INVALID，前端按码分支），Message 为中文说明。
type ValidationError struct {
	Path    string
	Code    string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// ValidationErrors 校验错误集合（一次校验尽量报全，便于设计器一次性展示）。
type ValidationErrors []*ValidationError

func (errs ValidationErrors) Error() string {
	msgs := make([]string, 0, len(errs))
	for _, e := range errs {
		msgs = append(msgs, e.Error())
	}
	sort.Strings(msgs)
	return strings.Join(msgs, "; ")
}

// 错误码（稳定协议，errorCodes.ts 对齐维护）。
const (
	ErrCodeSchemaVersion   = "SCHEMA_VERSION_INVALID"
	ErrCodeKeyInvalid      = "KEY_INVALID"
	ErrCodeKeyDuplicate    = "KEY_DUPLICATE"
	ErrCodeNodeUnknown     = "NODE_TYPE_INVALID"
	ErrCodeRefMissing      = "EDGE_REF_MISSING"
	ErrCodeStartCardinal   = "START_CARDINALITY_INVALID"
	ErrCodeEndCardinal     = "END_CARDINALITY_INVALID"
	ErrCodeEdgeDirection   = "EDGE_DIRECTION_INVALID"
	ErrCodeConfigInvalid   = "NODE_CONFIG_INVALID"
	ErrCodeConditionEdge   = "EDGE_CONDITION_INVALID"
	ErrCodeExprInvalid     = "EXPRESSION_INVALID"
	ErrCodeFieldPermission = "FIELD_PERMISSION_INVALID"
	ErrCodeUnreachable     = "NODE_UNREACHABLE"
	ErrCodeNoExit          = "NODE_NO_PATH_TO_END"
	ErrCodeCycle           = "CYCLE_UNSUPPORTED"
)

// Validator DSL 严格校验器。表达式校验经 ExpressionEngine 发布前预编译；
// 零值即可用（内置 expr 引擎），测试或嵌入方可注入替身。
type Validator struct {
	expressions expression.Engine
}

// NewValidator 构造校验器（expressions 为 nil 时使用内置 expr 引擎）。
func NewValidator(expressions expression.Engine) *Validator {
	if expressions == nil {
		expressions = expression.NewExprEngine()
	}
	return &Validator{expressions: expressions}
}

// Validate 对 DSL 文档执行全量严格校验（第 7.5 章清单逐条实现）：
// 通过返回 nil；存在问题时返回 ValidationErrors（非 error 包装，逐条定位）。
func (v *Validator) Validate(doc *model.Document) ValidationErrors {
	var errs ValidationErrors
	if doc == nil {
		return append(errs, &ValidationError{Path: "$", Code: ErrCodeSchemaVersion, Message: "DSL 文档为空"})
	}
	errs = v.validateSchemaVersion(errs, doc)
	errs = v.validateNodes(errs, doc)
	errs = v.validateEdges(errs, doc)
	errs = v.validateCardinality(errs, doc)
	errs = v.validateGraph(errs, doc)
	return errs
}

// validateSchemaVersion 规则 1：schemaVersion 必须精确等于 DSLSchemaVersion。
func (v *Validator) validateSchemaVersion(errs ValidationErrors, doc *model.Document) ValidationErrors {
	if doc.SchemaVersion != model.DSLSchemaVersion {
		errs = append(errs, &ValidationError{
			Path: "$.schemaVersion", Code: ErrCodeSchemaVersion,
			Message: fmt.Sprintf("schemaVersion 必须为 %q，当前为 %q", model.DSLSchemaVersion, doc.SchemaVersion),
		})
	}
	return errs
}

// validateNodes 规则 2/10/11：key 唯一且命名合法、类型在目录内、
// 按 NodeType 校验 config 必填项与字段权限值域。
func (v *Validator) validateNodes(errs ValidationErrors, doc *model.Document) ValidationErrors {
	keys := make(map[string]bool, len(doc.Nodes))
	for i := range doc.Nodes {
		n := &doc.Nodes[i]
		path := fmt.Sprintf("$.nodes[%d]", i)
		if !KeyPattern.MatchString(n.Key) {
			errs = append(errs, &ValidationError{Path: path + ".key", Code: ErrCodeKeyInvalid,
				Message: "节点 key 必须以字母开头，仅含字母/数字/下划线，长度 1~64"})
		}
		if keys[n.Key] {
			errs = append(errs, &ValidationError{Path: path + ".key", Code: ErrCodeKeyDuplicate,
				Message: fmt.Sprintf("节点 key %q 重复", n.Key)})
		}
		keys[n.Key] = true
		if !model.V1NodeTypes[n.Type] {
			errs = append(errs, &ValidationError{Path: path + ".type", Code: ErrCodeNodeUnknown,
				Message: fmt.Sprintf("不支持的节点类型 %q", n.Type)})
			continue
		}
		errs = v.validateNodeConfig(errs, n, path+".config")
	}
	return errs
}

// validateNodeConfig 按 NodeType 校验配置（规则 9：approval config 必填项）。
func (v *Validator) validateNodeConfig(errs ValidationErrors, n *model.Node, path string) ValidationErrors {
	switch n.Type {
	case model.NodeTypeApproval:
		if !model.V1ApprovalModes[n.Config.ApprovalMode] {
			errs = append(errs, &ValidationError{Path: path + ".approvalMode", Code: ErrCodeConfigInvalid,
				Message: fmt.Sprintf("审批节点 approvalMode 必须为 single/or-sign/countersign，当前为 %q", n.Config.ApprovalMode)})
		}
		errs = v.validateAssigneeSpec(errs, n.Config.Assignee, path+".assignee")
		// V1 仅支持终止型驳回（第 10.2 章）
		if n.Config.RejectStrategy != "" && n.Config.RejectStrategy != model.RejectStrategyTerminate {
			errs = append(errs, &ValidationError{Path: path + ".rejectStrategy", Code: ErrCodeConfigInvalid,
				Message: "V1 仅支持 rejectStrategy=terminate"})
		}
		if n.Config.ApprovalMode == model.ApprovalModeCountersign {
			if n.Config.PassRatio <= 0 || n.Config.PassRatio > 1 {
				errs = append(errs, &ValidationError{Path: path + ".passRatio", Code: ErrCodeConfigInvalid,
					Message: "会签 passRatio 必须在 (0,1] 区间"})
			}
		}
		for field, perm := range n.Config.FormPermissions {
			if !model.V1FieldPermissions[perm] {
				errs = append(errs, &ValidationError{Path: path + ".formPermissions." + field, Code: ErrCodeFieldPermission,
					Message: fmt.Sprintf("字段权限值必须为 hidden/readonly/editable/required，当前为 %q", perm)})
			}
		}
		errs = v.validateTimeoutConfig(errs, n.Config.Timeout, path+".timeout")
		errs = v.validateReminderConfig(errs, n.Config.Reminder, path+".reminder")
	case model.NodeTypeCC:
		if n.Config.Recipients == nil {
			errs = append(errs, &ValidationError{Path: path + ".recipients", Code: ErrCodeConfigInvalid,
				Message: "抄送节点必须配置 recipients"})
			break
		}
		errs = v.validateAssigneeSpec(errs, n.Config.Recipients, path+".recipients")
	case model.NodeTypeService:
		errs = v.validateServiceConfig(errs, n.Config.Service, path+".service")
	}
	return errs
}

// ServiceHeaderPattern 请求头命名规则（冻结）：RFC 7230 token 子集，
// 1~64 位，防畸形头注入。
var ServiceHeaderPattern = regexp.MustCompile(`^[a-zA-Z0-9-]{1,64}$`)

// ServiceBlockedHeaders V1 禁止明文写入 DSL 的敏感请求头（第 27 章「密钥
// 不得直接存 DSL」）：鉴权/凭据类头必须由平台侧安全注入，禁止设计期硬编码。
// 键为小写规范化形态。
var ServiceBlockedHeaders = map[string]bool{
	"authorization": true, "cookie": true, "set-cookie": true,
	"proxy-authorization": true, "www-authenticate": true,
}

// validateServiceConfig service 节点配置深度校验（Phase 7 定版）：
// 动作/方法/URL/模板表达式/敏感头/上限逐项冻结，与 Runtime 执行语义同口径。
func (v *Validator) validateServiceConfig(errs ValidationErrors, cfg *model.ServiceConfig, path string) ValidationErrors {
	if cfg == nil {
		return append(errs, &ValidationError{Path: path, Code: ErrCodeConfigInvalid,
			Message: "service 节点必须配置 service 配置块"})
	}
	if !model.V1ServiceActions[cfg.Action] {
		errs = append(errs, &ValidationError{Path: path + ".action", Code: ErrCodeConfigInvalid,
			Message: fmt.Sprintf("service 节点 action 必须为 http，当前为 %q", cfg.Action)})
	}
	method := cfg.Method
	if method == "" {
		method = model.ServiceDefaultMethod
	}
	if !model.V1ServiceMethods[method] {
		errs = append(errs, &ValidationError{Path: path + ".method", Code: ErrCodeConfigInvalid,
			Message: fmt.Sprintf("service 节点 method 必须为 GET/POST/PUT/PATCH/DELETE，当前为 %q", method)})
	}
	if !model.ServiceBodyMethods[method] && cfg.Body != "" {
		errs = append(errs, &ValidationError{Path: path + ".body", Code: ErrCodeConfigInvalid,
			Message: fmt.Sprintf("method=%s 不允许携带 body（仅 POST/PUT/PATCH）", method)})
	}
	// URL：模板形态校验（scheme 判定基于字面前缀；插值段表达式发布期编译）
	if strings.TrimSpace(cfg.URL) == "" {
		errs = append(errs, &ValidationError{Path: path + ".url", Code: ErrCodeConfigInvalid,
			Message: "service 节点必须配置 url"})
	} else if !strings.HasPrefix(cfg.URL, "http://") && !strings.HasPrefix(cfg.URL, "https://") {
		errs = append(errs, &ValidationError{Path: path + ".url", Code: ErrCodeConfigInvalid,
			Message: "service 节点 url 必须以 http:// 或 https:// 开头"})
	} else if err := v.compileTemplate(cfg.URL); err != nil {
		errs = append(errs, &ValidationError{Path: path + ".url", Code: ErrCodeExprInvalid,
			Message: "url 模板表达式非法: " + err.Error()})
	}
	// 请求头：数量/命名/敏感头/模板表达式
	if len(cfg.Headers) > model.ServiceMaxHeaders {
		errs = append(errs, &ValidationError{Path: path + ".headers", Code: ErrCodeConfigInvalid,
			Message: fmt.Sprintf("请求头数量不得超过 %d", model.ServiceMaxHeaders)})
	}
	for name, value := range cfg.Headers {
		headerPath := path + ".headers." + name
		if !ServiceHeaderPattern.MatchString(name) {
			errs = append(errs, &ValidationError{Path: headerPath, Code: ErrCodeConfigInvalid,
				Message: "请求头命名非法（仅允许字母/数字/连字符，1~64 位）"})
		}
		if ServiceBlockedHeaders[strings.ToLower(name)] {
			errs = append(errs, &ValidationError{Path: headerPath, Code: ErrCodeConfigInvalid,
				Message: "敏感请求头禁止明文写入 DSL（鉴权凭据由平台侧注入，第 27 章）"})
		}
		if err := v.compileTemplate(value); err != nil {
			errs = append(errs, &ValidationError{Path: headerPath, Code: ErrCodeExprInvalid,
				Message: "请求头模板表达式非法: " + err.Error()})
		}
	}
	if len(cfg.Body) > model.ServiceMaxBodyBytes {
		errs = append(errs, &ValidationError{Path: path + ".body", Code: ErrCodeConfigInvalid,
			Message: fmt.Sprintf("请求体不得超过 %d 字节", model.ServiceMaxBodyBytes)})
	}
	if cfg.Body != "" {
		if err := v.compileTemplate(cfg.Body); err != nil {
			errs = append(errs, &ValidationError{Path: path + ".body", Code: ErrCodeExprInvalid,
				Message: "请求体模板表达式非法: " + err.Error()})
		}
	}
	// 超时：1~120s（缺省 10s，封顶防执行事务被外部服务长挂拖垮）
	if cfg.TimeoutSeconds < 0 || cfg.TimeoutSeconds > model.ServiceMaxTimeoutSeconds {
		errs = append(errs, &ValidationError{Path: path + ".timeoutSeconds", Code: ErrCodeConfigInvalid,
			Message: fmt.Sprintf("timeoutSeconds 必须在 1~%d 之间（0=缺省 %ds）",
				model.ServiceMaxTimeoutSeconds, model.ServiceDefaultTimeoutSeconds)})
	}
	// 重试上限：0~8（缺省 3，wf_job 重试记账承担退避回队）
	if cfg.MaxRetries != nil && (*cfg.MaxRetries < 0 || *cfg.MaxRetries > model.ServiceMaxRetries) {
		errs = append(errs, &ValidationError{Path: path + ".maxRetries", Code: ErrCodeConfigInvalid,
			Message: fmt.Sprintf("maxRetries 必须在 0~%d 之间（缺省 %d）",
				model.ServiceMaxRetries, model.ServiceDefaultMaxRetries)})
	}
	// 响应映射：变量命名/唯一性/路径合法性
	seen := make(map[string]bool, len(cfg.ResponseMapping))
	for i := range cfg.ResponseMapping {
		m := &cfg.ResponseMapping[i]
		mapPath := fmt.Sprintf("%s.responseMapping[%d]", path, i)
		if !KeyPattern.MatchString(m.Variable) {
			errs = append(errs, &ValidationError{Path: mapPath + ".variable", Code: ErrCodeConfigInvalid,
				Message: "映射变量名必须为字母开头的标识符（1~64 位，字母/数字/下划线）"})
		}
		if seen[m.Variable] {
			errs = append(errs, &ValidationError{Path: mapPath + ".variable", Code: ErrCodeConfigInvalid,
				Message: fmt.Sprintf("映射变量名 %q 重复", m.Variable)})
		}
		seen[m.Variable] = true
		if strings.TrimSpace(m.Path) != "" && strings.ContainsAny(m.Path, " 	") {
			errs = append(errs, &ValidationError{Path: mapPath + ".path", Code: ErrCodeConfigInvalid,
				Message: "映射路径为点分路径形态，不允许空白字符"})
		}
	}
	return errs
}

// compileTemplate 模板表达式编译校验（URL/Headers/Body 共用；Compile 阶段
// 再次编译并冻结产物，此处仅产出可读的校验错误）。
func (v *Validator) compileTemplate(source string) error {
	_, err := expression.ParseTemplate(v.expressions, source)
	return err
}

// validateAssigneeSpec 审批人规格校验（规则 12：Resolver 类型校验；
// 参数必填项按类型逐一冻结，与 assignment 包注册表一致）。
func (v *Validator) validateAssigneeSpec(errs ValidationErrors, spec *model.AssigneeSpec, path string) ValidationErrors {
	if spec == nil {
		return append(errs, &ValidationError{Path: path, Code: ErrCodeConfigInvalid, Message: "必须配置审批人 assignee"})
	}
	switch spec.Type {
	case model.AssigneeTypeUser:
		if len(spec.UserIDs) == 0 {
			errs = append(errs, &ValidationError{Path: path + ".userIds", Code: ErrCodeConfigInvalid,
				Message: "指定用户审批必须提供非空 userIds"})
		}
	case model.AssigneeTypeRole:
		if strings.TrimSpace(spec.RoleCode) == "" {
			errs = append(errs, &ValidationError{Path: path + ".roleCode", Code: ErrCodeConfigInvalid,
				Message: "指定角色审批必须提供 roleCode"})
		}
	case model.AssigneeTypeFormField:
		if strings.TrimSpace(spec.FormField) == "" {
			errs = append(errs, &ValidationError{Path: path + ".formField", Code: ErrCodeConfigInvalid,
				Message: "表单用户字段审批必须提供 formField"})
		}
	case model.AssigneeTypeDepartment, model.AssigneeTypeDepartmentManager:
		if spec.DeptID == 0 {
			errs = append(errs, &ValidationError{Path: path + ".deptId", Code: ErrCodeConfigInvalid,
				Message: "部门类审批人必须提供 deptId"})
		}
	case model.AssigneeTypeStarterManager:
		// 无参数；实际可用性受 IAM reporting 前置能力约束（运行期解析不到
		// 返回稳定错误 WORKFLOW_ASSIGNEE_NOT_FOUND，第 17 章补充语义）
	default:
		errs = append(errs, &ValidationError{Path: path + ".type", Code: ErrCodeConfigInvalid,
			Message: fmt.Sprintf("不支持的审批人类型 %q", spec.Type)})
		return errs
	}
	// Resolver 能力门（第 17.2 章能力矩阵）：IAM 前置能力未落地的类型
	// 不允许发布，避免「发布成功、运行必失败」的定义进入快照
	if !assignment.ResolverEnabled(spec.Type) {
		errs = append(errs, &ValidationError{Path: path + ".type", Code: ErrCodeConfigInvalid,
			Message: fmt.Sprintf("审批人类型 %q 暂未启用（IAM 前置能力未落地）", spec.Type)})
	}
	return errs
}

// validateTimeoutConfig 超时配置校验（Phase 5，第 19 章）：秒数在
// [1, MaxJobSeconds] 区间，动作仅 approve/reject。
func (v *Validator) validateTimeoutConfig(errs ValidationErrors, config *model.TimeoutConfig, path string) ValidationErrors {
	if config == nil {
		return errs
	}
	if config.Seconds < 1 || config.Seconds > model.MaxJobSeconds {
		errs = append(errs, &ValidationError{Path: path + ".seconds", Code: ErrCodeConfigInvalid,
			Message: fmt.Sprintf("超时秒数必须在 1~%d 区间", model.MaxJobSeconds)})
	}
	if !model.V1TimeoutActions[config.Action] {
		errs = append(errs, &ValidationError{Path: path + ".action", Code: ErrCodeConfigInvalid,
			Message: fmt.Sprintf("超时自动动作必须为 approve/reject，当前为 %q", config.Action)})
	}
	return errs
}

// validateReminderConfig 提醒配置校验（Phase 5）：秒数在
// [1, MaxJobSeconds] 区间（V1 单次提醒，不循环）。
func (v *Validator) validateReminderConfig(errs ValidationErrors, config *model.ReminderConfig, path string) ValidationErrors {
	if config == nil {
		return errs
	}
	if config.Seconds < 1 || config.Seconds > model.MaxJobSeconds {
		errs = append(errs, &ValidationError{Path: path + ".seconds", Code: ErrCodeConfigInvalid,
			Message: fmt.Sprintf("提醒秒数必须在 1~%d 区间", model.MaxJobSeconds)})
	}
	return errs
}

// validateEdges 规则 3/4/8/9：edge key 唯一、source/target 存在、
// 仅 condition 节点出边允许携带条件、condition 出边必须有 default 分支
// 且非 default 出边必须携带表达式、表达式预编译校验。
func (v *Validator) validateEdges(errs ValidationErrors, doc *model.Document) ValidationErrors {
	edgeKeys := make(map[string]bool, len(doc.Edges))
	outBySource := make(map[string][]*model.Edge)
	for i := range doc.Edges {
		e := &doc.Edges[i]
		path := fmt.Sprintf("$.edges[%d]", i)
		if !KeyPattern.MatchString(e.Key) {
			errs = append(errs, &ValidationError{Path: path + ".key", Code: ErrCodeKeyInvalid,
				Message: "连线 key 必须以字母开头，仅含字母/数字/下划线，长度 1~64"})
		}
		if edgeKeys[e.Key] {
			errs = append(errs, &ValidationError{Path: path + ".key", Code: ErrCodeKeyDuplicate,
				Message: fmt.Sprintf("连线 key %q 重复", e.Key)})
		}
		edgeKeys[e.Key] = true
		if _, ok := doc.NodeOf(e.Source); !ok {
			errs = append(errs, &ValidationError{Path: path + ".source", Code: ErrCodeRefMissing,
				Message: fmt.Sprintf("source 节点 %q 不存在", e.Source)})
		}
		if _, ok := doc.NodeOf(e.Target); !ok {
			errs = append(errs, &ValidationError{Path: path + ".target", Code: ErrCodeRefMissing,
				Message: fmt.Sprintf("target 节点 %q 不存在", e.Target)})
		}
		outBySource[e.Source] = append(outBySource[e.Source], e)

		// 条件表达式归属与语法校验
		sourceNode, _ := doc.NodeOf(e.Source)
		if e.Condition != nil {
			if sourceNode == nil || sourceNode.Type != model.NodeTypeCondition {
				errs = append(errs, &ValidationError{Path: path + ".condition", Code: ErrCodeConditionEdge,
					Message: "仅 condition 节点的出边允许携带条件"})
			} else if strings.TrimSpace(e.Condition.Expression) == "" {
				errs = append(errs, &ValidationError{Path: path + ".condition.expression", Code: ErrCodeExprInvalid,
					Message: "条件表达式不能为空"})
			} else if _, err := v.expressions.Compile(e.Condition.Expression); err != nil {
				errs = append(errs, &ValidationError{Path: path + ".condition.expression", Code: ErrCodeExprInvalid,
					Message: fmt.Sprintf("条件表达式编译失败：%v", err)})
			}
		}
	}

	// condition 节点出边分支规则（第 7.3 章）：必须存在 default（无条件）
	// 出边保证路由闭合，其余出边必须携带明确条件
	for i := range doc.Nodes {
		n := &doc.Nodes[i]
		if n.Type != model.NodeTypeCondition {
			continue
		}
		outs := outBySource[n.Key]
		var hasDefault, hasExpr bool
		for _, e := range outs {
			if e.Condition == nil {
				hasDefault = true
			} else {
				hasExpr = true
			}
		}
		path := fmt.Sprintf("$.nodes[%d]", i)
		if !hasDefault {
			errs = append(errs, &ValidationError{Path: path, Code: ErrCodeConditionEdge,
				Message: fmt.Sprintf("条件节点 %q 必须存在一条 default（无条件）出边", n.Key)})
		}
		if len(outs) > 1 && !hasExpr {
			errs = append(errs, &ValidationError{Path: path, Code: ErrCodeConditionEdge,
				Message: fmt.Sprintf("条件节点 %q 存在多条出边时，非 default 出边必须携带条件表达式", n.Key)})
		}
	}
	return errs
}

// validateCardinality 规则 5/6/7：恰好一个 start、至少一个 end、
// start 不允许入边、end 不允许出边。
func (v *Validator) validateCardinality(errs ValidationErrors, doc *model.Document) ValidationErrors {
	var starts, ends int
	for i := range doc.Nodes {
		switch doc.Nodes[i].Type {
		case model.NodeTypeStart:
			starts++
		case model.NodeTypeEnd:
			ends++
		}
	}
	if starts != 1 {
		errs = append(errs, &ValidationError{Path: "$.nodes", Code: ErrCodeStartCardinal,
			Message: fmt.Sprintf("必须且只能存在一个 start 节点，当前 %d 个", starts)})
	}
	if ends < 1 {
		errs = append(errs, &ValidationError{Path: "$.nodes", Code: ErrCodeEndCardinal,
			Message: "至少存在一个 end 节点"})
	}
	for i := range doc.Edges {
		e := &doc.Edges[i]
		path := fmt.Sprintf("$.edges[%d]", i)
		if src, ok := doc.NodeOf(e.Source); ok && src.Type == model.NodeTypeEnd {
			errs = append(errs, &ValidationError{Path: path + ".source", Code: ErrCodeEdgeDirection,
				Message: fmt.Sprintf("end 节点 %q 不允许存在出边", e.Source)})
		}
		if tgt, ok := doc.NodeOf(e.Target); ok && tgt.Type == model.NodeTypeStart {
			errs = append(errs, &ValidationError{Path: path + ".target", Code: ErrCodeEdgeDirection,
				Message: fmt.Sprintf("start 节点 %q 不允许存在入边", e.Target)})
		}
	}
	return errs
}

// validateGraph 规则 13/14/15：可达性（start 可达所有节点、所有节点可达
// end）、死节点检查与环检测。V1 为顺序流，不支持任何语义的环。
func (v *Validator) validateGraph(errs ValidationErrors, doc *model.Document) ValidationErrors {
	adj := make(map[string][]string, len(doc.Nodes))
	for i := range doc.Edges {
		e := &doc.Edges[i]
		adj[e.Source] = append(adj[e.Source], e.Target)
	}

	// start 正向可达性：不可达节点即死节点
	var startKey string
	for i := range doc.Nodes {
		if doc.Nodes[i].Type == model.NodeTypeStart {
			startKey = doc.Nodes[i].Key
			break
		}
	}
	forward := reachable(adj, startKey)
	// 反向可达性：所有节点必须存在到任一 end 的路径
	reverse := make(map[string][]string, len(doc.Edges))
	for i := range doc.Edges {
		e := &doc.Edges[i]
		reverse[e.Target] = append(reverse[e.Target], e.Source)
	}
	var endKeys []string
	for i := range doc.Nodes {
		if doc.Nodes[i].Type == model.NodeTypeEnd {
			endKeys = append(endKeys, doc.Nodes[i].Key)
		}
	}
	toEnd := reachable(reverse, endKeys...)

	for i := range doc.Nodes {
		n := &doc.Nodes[i]
		path := fmt.Sprintf("$.nodes[%d]", i)
		if _, ok := forward[n.Key]; !ok {
			errs = append(errs, &ValidationError{Path: path, Code: ErrCodeUnreachable,
				Message: fmt.Sprintf("死节点 %q：从 start 不可达", n.Key)})
		}
		if _, ok := toEnd[n.Key]; !ok {
			errs = append(errs, &ValidationError{Path: path, Code: ErrCodeNoExit,
				Message: fmt.Sprintf("节点 %q 不存在到 end 的路径", n.Key)})
		}
	}

	// 环检测：V1 顺序流不支持环（含自环），DFS 三色标记
	if hasCycle(adj) {
		errs = append(errs, &ValidationError{Path: "$.edges", Code: ErrCodeCycle,
			Message: "流程图存在环，V1 不支持任何环语义"})
	}
	return errs
}

// reachable 从 roots 出发沿邻接表可达的节点集合。
func reachable(adj map[string][]string, roots ...string) map[string]bool {
	seen := make(map[string]bool)
	stack := append([]string{}, roots...)
	for len(stack) > 0 {
		k := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[k] {
			continue
		}
		seen[k] = true
		stack = append(stack, adj[k]...)
	}
	return seen
}

// hasCycle 有向图环检测（白/灰/黑三色标记）。
func hasCycle(adj map[string][]string) bool {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int)
	var dfs func(k string) bool
	dfs = func(k string) bool {
		color[k] = gray
		for _, next := range adj[k] {
			switch color[next] {
			case gray:
				return true
			case white:
				if dfs(next) {
					return true
				}
			}
		}
		color[k] = black
		return false
	}
	for k := range adj {
		if color[k] == white && dfs(k) {
			return true
		}
	}
	return false
}
