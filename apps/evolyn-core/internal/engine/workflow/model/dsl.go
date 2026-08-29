// Package model Workflow Engine 领域模型与 DSL v1 协议类型。
//
// 本包是前后端共同协议的唯一事实源（逐字对齐
// docs/低代码平台/流程引擎/流程引擎阶段开发功能设计v1.1.md 第 7 章），
// 仅含纯 Go 结构与枚举，不携带 GORM 标签、不落表——持久化形态由
// platform/workflow 仓储适配层负责（DSL 单文档 JSONB 存储，
// 不建 wf_node / wf_edge 独立表）。
package model

// DSLSchemaVersion Workflow DSL v1 协议版本号；校验器只接受本值。
// 协议演进时递增次版本并同步前后端与迁移器，禁止静默放宽。
const DSLSchemaVersion = "1.0"

// NodeType V1 节点类型目录（冻结，禁止 V1 未登记类型入库）。
type NodeType string

const (
	NodeTypeStart     NodeType = "start"
	NodeTypeApproval  NodeType = "approval"
	NodeTypeCondition NodeType = "condition"
	NodeTypeCC        NodeType = "cc"
	NodeTypeService   NodeType = "service"  // 数据模型先定义，执行能力 Phase 7 开放
	NodeTypeParallel  NodeType = "parallel" // 并行网关（Phase 8）：role=split/join
	NodeTypeEnd       NodeType = "end"
)

// V1NodeTypes V1 支持的全部节点类型（校验器以此为准）。
var V1NodeTypes = map[NodeType]bool{
	NodeTypeStart:     true,
	NodeTypeApproval:  true,
	NodeTypeCondition: true,
	NodeTypeCC:        true,
	NodeTypeService:   true,
	NodeTypeParallel:  true,
	NodeTypeEnd:       true,
}

// ParallelRole 并行网关角色（Phase 8，第 31 章并行执行定版）：同一节点
// 通过 role 表达 split（分流）/ join（汇聚）两种语义，避免引入两个节点类型。
type ParallelRole string

const (
	// ParallelRoleSplit 并行分流：多条无条件出边各自 fork 一条子执行路径
	ParallelRoleSplit ParallelRole = "split"
	// ParallelRoleJoin 并行汇聚：等待全部分支路径到达后放行单条出边
	//（join token 计数 = 分支数，Runtime 判定，第 12.2 章）
	ParallelRoleJoin ParallelRole = "join"
)

// V1ParallelRoles 并行网关角色合法值域。
var V1ParallelRoles = map[ParallelRole]bool{
	ParallelRoleSplit: true,
	ParallelRoleJoin:  true,
}

// MaxParallelBranches 并行分支数上限（Phase 8 复杂度冻结：校验器与
// Runtime 同口径，防 DSL 配置出超大规模扇出拖垮推进事务）。
const MaxParallelBranches = 10

// MaxParallelDepth 运行期并行嵌套/链式深度上限（Phase 8 复杂度冻结：
// 校验器禁止并行区域内再嵌套 parallel，该上限仅作运行期纵深防御）。
const MaxParallelDepth = 16

// ParallelConfig 并行网关配置（Phase 8）。
type ParallelConfig struct {
	// Role 网关角色：split（分流）/ join（汇聚）
	Role ParallelRole `json:"role"`
}

// ApprovalMode 审批模式（第 11 章）。
type ApprovalMode string

const (
	// ApprovalModeSingle 单人审批：单任务完成即节点完成
	ApprovalModeSingle ApprovalMode = "single"
	// ApprovalModeOrSign 或签：任意一人 APPROVED 即节点完成，其余任务自动取消
	ApprovalModeOrSign ApprovalMode = "or-sign"
	// ApprovalModeCountersign 会签：达到 ceil(totalActors * passRatio) 通过阈值节点完成
	ApprovalModeCountersign ApprovalMode = "countersign"
)

// V1ApprovalModes V1 支持的审批模式集合。
var V1ApprovalModes = map[ApprovalMode]bool{
	ApprovalModeSingle:      true,
	ApprovalModeOrSign:      true,
	ApprovalModeCountersign: true,
}

// RejectStrategy 驳回策略。V1 仅支持 terminate（终止型驳回，第 10.2 章），
// 「驳回到任意历史节点」等策略 V1 明确不支持。
type RejectStrategy string

const RejectStrategyTerminate RejectStrategy = "terminate"

// FieldPermission 审批节点字段权限（V1 值域冻结，第 7.4 章）。
type FieldPermission string

const (
	FieldPermissionHidden   FieldPermission = "hidden"
	FieldPermissionReadonly FieldPermission = "readonly"
	FieldPermissionEditable FieldPermission = "editable"
	FieldPermissionRequired FieldPermission = "required"
)

// V1FieldPermissions 字段权限合法值域。
var V1FieldPermissions = map[FieldPermission]bool{
	FieldPermissionHidden:   true,
	FieldPermissionReadonly: true,
	FieldPermissionEditable: true,
	FieldPermissionRequired: true,
}

// AssigneeType 审批人解析类型（第 17 章能力矩阵；实际可用性由
// assignment 包的 V1 注册表与 IAM 前置能力共同决定）。
type AssigneeType string

const (
	AssigneeTypeUser              AssigneeType = "user"               // 指定用户（V1）
	AssigneeTypeRole              AssigneeType = "role"               // 指定角色（V1）
	AssigneeTypeFormField         AssigneeType = "form_field"         // 表单用户字段（V1）
	AssigneeTypeDepartment        AssigneeType = "department"         // 部门成员（可选）
	AssigneeTypeDepartmentManager AssigneeType = "department_manager" // 部门负责人（条件开放，需 IAM leader 前置）
	AssigneeTypeStarterManager    AssigneeType = "starter_manager"    // 发起人直属主管（条件开放，需 IAM reporting 前置）
)

// AssigneeSpec 审批人规格：type 决定哪些参数字段生效，校验器按类型
// 逐一校验必填参数；解析结果在任务创建时一次性快照（v1.1 定版）。
type AssigneeSpec struct {
	Type AssigneeType `json:"type"`
	// UserIDs 指定用户成员 ID 列表（type=user 必填，非空）
	UserIDs []uint `json:"userIds,omitempty"`
	// RoleCode 指定角色编码（type=role 必填）
	RoleCode string `json:"roleCode,omitempty"`
	// FormField 表单用户字段 widgetName（type=form_field 必填）
	FormField string `json:"formField,omitempty"`
	// DeptID 部门 ID（type=department / department_manager 必填）
	DeptID uint `json:"deptId,omitempty"`
}

// MaxJobSeconds 超时/提醒排期秒数上限（30 天，冻结复杂度上限；校验器与
// 前端设计器同口径）。
const MaxJobSeconds = 30 * 24 * 3600

// TimeoutAction 超时自动动作（第 19.4 章：必须经 Task Engine 正常执行路径
// 触发，V1 仅支持同意/驳回；reject 即 terminate 联动）。
type TimeoutAction string

const (
	TimeoutActionApprove TimeoutAction = "approve"
	TimeoutActionReject  TimeoutAction = "reject"
)

// V1TimeoutActions 超时自动动作合法值域。
var V1TimeoutActions = map[TimeoutAction]bool{
	TimeoutActionApprove: true,
	TimeoutActionReject:  true,
}

// TimeoutConfig 审批节点超时配置（Phase 5）：任务创建时排 task.timeout Job，
// 到期由 Worker 经 Task Engine 执行自动动作；MaxSeconds 冻结复杂度上限。
type TimeoutConfig struct {
	// Seconds 任务创建后多少秒超时（1~2592000，即最长 30 天）
	Seconds int `json:"seconds"`
	// Action 超时自动动作（approve/reject）
	Action TimeoutAction `json:"action"`
}

// ReminderConfig 审批节点提醒配置（Phase 5）：任务创建时排 task.reminder
// Job，到期由 Worker 落 REMINDER 操作流水（V1 单次提醒，不循环）。
type ReminderConfig struct {
	// Seconds 任务创建后多少秒提醒（1~2592000，即最长 30 天）
	Seconds int `json:"seconds"`
}

// V1ServiceActions service 节点动作类型目录（Phase 7 定版：V1 仅出站
// HTTP/Webhook；grpc/插件等随后续里程碑扩展，校验器以此为准）。
var V1ServiceActions = map[string]bool{"http": true}

// V1ServiceMethods service 节点允许的 HTTP 方法（出站写语义为主）。
var V1ServiceMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
}

// ServiceBodyMethods 允许携带请求体的方法。
var ServiceBodyMethods = map[string]bool{"POST": true, "PUT": true, "PATCH": true}

// Service 节点常量（第 7 章 DSL v1，Phase 7 定版）：超时/重试/变量名等
// 复杂度上限在此冻结，校验器与 Runtime 同口径。
const (
	// ServiceActionHTTP V1 唯一动作类型
	ServiceActionHTTP = "http"
	// ServiceDefaultMethod URL/方法缺省值
	ServiceDefaultMethod = "POST"
	// ServiceDefaultTimeoutSeconds / ServiceMaxTimeoutSeconds 单次请求超时
	//（缺省 10s，封顶 120s：推进事务不因外部服务长挂而过久占用）
	ServiceDefaultTimeoutSeconds = 10
	ServiceMaxTimeoutSeconds     = 120
	// ServiceDefaultMaxRetries / ServiceMaxRetries 失败重试上限
	//（wf_job 重试记账承担退避回队，缺省 3 次）
	ServiceDefaultMaxRetries = 3
	ServiceMaxRetries        = 8
	// ServiceMaxHeaders / ServiceMaxBodyBytes / ServiceMaxResponseBytes
	// 请求头数量、请求体与响应体大小上限（防大报文拖垮执行事务）
	ServiceMaxHeaders       = 16
	ServiceMaxBodyBytes     = 64 * 1024
	ServiceMaxResponseBytes = 1024 * 1024
)

// ServiceResponseMapping 响应映射：从 JSON 响应体提取值写入流程变量，
// 后续节点条件/审批人/模板经 variables.<name> 读取（第 16 章白名单上下文）。
type ServiceResponseMapping struct {
	// Variable 变量名（合法 Go 标识符形态，实例内唯一）
	Variable string `json:"variable"`
	// Path 响应 JSON 内点分路径（如 "data.orderId"；空=整个响应体）
	Path string `json:"path,omitempty"`
	// Required 提取失败是否整体失败（缺省 false：变量落 JSON null 继续推进）
	Required bool `json:"required,omitempty"`
}

// ServiceConfig service 节点配置（Phase 7 定版）：V1 承载出站 HTTP/Webhook
// 动作。安全边界（第 27 章）：密钥不得直接存 DSL——校验器拒绝
// authorization/cookie 等敏感头明文入 DSL，鉴权凭据由平台侧扩展注入；
// SSRF 防护与主机白名单由平台适配层在调用时强制（第 7 章/第 19 章）。
// 执行模型：节点在推进环中异步挂起，HTTP 调用经 wf_job service.invoke 由
// Job Worker 独立事务执行（业务事务内不发外部请求），失败退避重试。
type ServiceConfig struct {
	// Action 动作类型（V1 仅 "http"）
	Action string `json:"action"`
	// Method HTTP 方法（缺省 POST）
	Method string `json:"method,omitempty"`
	// URL 请求地址模板：支持 {{expr}} 插值（form.*/starter.*/variables.*，
	// 发布期预编译）；必须为 http(s) 形态
	URL string `json:"url"`
	// Headers 请求头：值支持 {{expr}} 插值；V1 禁止携带明文敏感头
	Headers map[string]string `json:"headers,omitempty"`
	// Body 请求体模板（JSON 形态，支持 {{expr}} 插值；GET/DELETE 忽略）
	Body string `json:"body,omitempty"`
	// TimeoutSeconds 单次请求超时秒数（缺省 10，1~120）
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`
	// MaxRetries 失败重试上限（缺省 3，0~8；承载于 wf_job 重试记账）
	MaxRetries *int `json:"maxRetries,omitempty"`
	// ResponseMapping 响应 → 流程变量映射（成功 2xx 时执行）
	ResponseMapping []ServiceResponseMapping `json:"responseMapping,omitempty"`
}

// NodeConfig 节点配置：字段扁平承载各类型节点的配置项，校验器按
// NodeType 裁剪必填/非法字段，未声明字段的缺省语义由 Runtime 解释。
type NodeConfig struct {
	// ApprovalMode 审批模式（approval 节点必填）
	ApprovalMode ApprovalMode `json:"approvalMode,omitempty"`
	// Assignee 审批人规格（approval 节点必填）
	Assignee *AssigneeSpec `json:"assignee,omitempty"`
	// RejectStrategy 驳回策略（approval 节点可选，缺省 terminate）
	RejectStrategy RejectStrategy `json:"rejectStrategy,omitempty"`
	// PassRatio 会签通过比例 (0,1]（approvalMode=countersign 必填），
	// requiredApprovals = ceil(totalActors * passRatio)
	PassRatio float64 `json:"passRatio,omitempty"`
	// FormPermissions 审批节点字段权限：widgetName → 权限（approval 节点可选）
	FormPermissions map[string]FieldPermission `json:"formPermissions,omitempty"`
	// Timeout 审批节点超时配置（Phase 5，可选；到期自动 approve/reject）
	Timeout *TimeoutConfig `json:"timeout,omitempty"`
	// Reminder 审批节点提醒配置（Phase 5，可选；单次提醒流水）
	Reminder *ReminderConfig `json:"reminder,omitempty"`
	// Recipients 抄送对象（cc 节点必填）
	Recipients *AssigneeSpec `json:"recipients,omitempty"`
	// Service 服务节点配置（Phase 7 执行）
	Service *ServiceConfig `json:"service,omitempty"`
	// Parallel 并行网关配置（Phase 8：role=split/join）
	Parallel *ParallelConfig `json:"parallel,omitempty"`
}

// EdgeCondition 出边条件；仅 condition 节点的出边允许携带，
// 表达式语法经 Expression Engine 发布前预编译校验。
type EdgeCondition struct {
	Expression string `json:"expression"`
}

// Edge 设计态连线。普通节点的 Condition 允许为空；condition 节点的
// 出边必须携带表达式或作为 default 分支（见校验器规则）。
type Edge struct {
	Key       string         `json:"key"`
	Source    string         `json:"source"`
	Target    string         `json:"target"`
	Condition *EdgeCondition `json:"condition,omitempty"`
}

// Settings 顶层流程设置（V1 预留空结构，字段随里程碑显式追加，
// 不开放自由键值避免协议漂移）。
type Settings struct{}

// Document Workflow DSL v1 顶层结构：草稿与发布快照的唯一事实形态，
// 整体内嵌于 wf_definition.draft_content / wf_definition_version.dsl_snapshot。
type Document struct {
	SchemaVersion string   `json:"schemaVersion"`
	Nodes         []Node   `json:"nodes"`
	Edges         []Edge   `json:"edges"`
	Settings      Settings `json:"settings"`
}

// Node 设计态节点；key 为文档内稳定标识（唯一、发布后不可变），
// 运行时 Navigator 按 key 从发布快照读取配置。
type Node struct {
	Key    string     `json:"key"`
	Type   NodeType   `json:"type"`
	Name   string     `json:"name"`
	Config NodeConfig `json:"config"`
}

// NodeOf 按 key 查找节点（文档内线性扫描；发布快照编译产物提供索引）。
func (d *Document) NodeOf(key string) (*Node, bool) {
	for i := range d.Nodes {
		if d.Nodes[i].Key == key {
			return &d.Nodes[i], true
		}
	}
	return nil, false
}
