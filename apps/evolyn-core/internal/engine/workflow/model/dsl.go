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
	NodeTypeService   NodeType = "service" // 数据模型先定义，执行能力 Phase 7 开放
	NodeTypeEnd       NodeType = "end"
)

// V1NodeTypes V1 支持的全部节点类型（校验器以此为准）。
var V1NodeTypes = map[NodeType]bool{
	NodeTypeStart:     true,
	NodeTypeApproval:  true,
	NodeTypeCondition: true,
	NodeTypeCC:        true,
	NodeTypeService:   true,
	NodeTypeEnd:       true,
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

// ServiceConfig service 节点配置占位（Phase 7 落地执行能力，
// Phase 0 仅冻结数据模型，校验器对 service 节点不做深度校验）。
type ServiceConfig struct {
	Action string `json:"action,omitempty"`
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
	// Recipients 抄送对象（cc 节点必填）
	Recipients *AssigneeSpec `json:"recipients,omitempty"`
	// Service 服务节点配置（Phase 7 执行）
	Service *ServiceConfig `json:"service,omitempty"`
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
