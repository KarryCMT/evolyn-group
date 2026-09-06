package model

import (
	kernel "evolyn/internal/model"
)

// CreateFormRequest 创建表单（POST /forms）：formType 在创建时固化，草稿初始化为空协议文档。
// ParentEntryCode 可选：应用菜单中目标分组的节点编码（侧栏资产 code），
// 传入时表单菜单节点挂到该分组下，否则挂应用根级。
type CreateFormRequest struct {
	ApplicationID   uint     `json:"applicationId" binding:"required" example:"1"`
	Name            string   `json:"name" binding:"required" example:"报名表"`
	FormType        FormType `json:"formType" example:"workflow"`
	ParentEntryCode string   `json:"parentEntryCode" example:"menu_ab12cd34ef56ab12"`
}

// UpdateFormRequest 白名单更新（PATCH /forms/:code）：名称为资产事实源，
// 图标/颜色经菜单维护端口同步到节点展示属性（ADR-011「修改名称和图标」
// 单一权限点 forms:patch）；指针字段区分未提交与提交零值（空串图标=清空）。
type UpdateFormRequest struct {
	Name  *string `json:"name"`
	Icon  *string `json:"icon"`
	Color *string `json:"color"`
}

// SwitchFormTypeRequest 切换表单类型（POST /forms/:code/switch-type，
// ADR-011）：standard↔workflow 互转；流程表单切标准后原流程数据保留，
// 仅不可再发起流程；与当前类型相同返回 FORM_TYPE_UNCHANGED。
type SwitchFormTypeRequest struct {
	FormType FormType `json:"formType" binding:"required" example:"workflow"`
}

// CopyFormRequest 复制表单（POST /forms/:code/copy，ADR-011）：
// targetApplicationId 为空或等于源应用 → copy-in-app 动作；非空且不同 →
// copy-cross-app 动作（目标应用须可创建表单）。parentEntryCode 为目标应用
// 菜单中的目标分组节点编码，为空挂目标应用根级。
type CopyFormRequest struct {
	TargetApplicationID *uint  `json:"targetApplicationId" example:"2"`
	ParentEntryCode     string `json:"parentEntryCode" example:"menu_ab12cd34ef56ab12"`
}

// SaveDraftRequest 保存草稿（PUT /forms/:code/draft）：全量替换 + 乐观锁口令。
// content 为目标保存协议根结构；校验失败返回 FORM_SCHEMA_INVALID + issues。
type SaveDraftRequest struct {
	DraftRevision   int64       `json:"draftRevision" binding:"required"`
	ProtocolVersion int         `json:"protocolVersion" binding:"required"`
	Content         JSONContent `json:"content" binding:"required"`
}

// SaveDraftResult 草稿保存结果：新口令供下次保存回传。
type SaveDraftResult struct {
	DraftRevision int64 `json:"draftRevision"`
}

// PublishRequest 发布（POST /forms/:code/publish）：携带草稿口令防止发布并发旧草稿。
type PublishRequest struct {
	DraftRevision int64 `json:"draftRevision" binding:"required"`
}

// PublishResult 发布结果：双口令（publishedVersion + schemaRevision）。
type PublishResult struct {
	PublishedVersion int    `json:"publishedVersion"`
	SchemaRevision   string `json:"schemaRevision"`
}

// FormDetail 表单详情出网（含草稿全文与修订口令）。
type FormDetail struct {
	ApplicationID    uint            `json:"applicationId"`
	Code             string          `json:"code"`
	Name             string          `json:"name"`
	FormType         FormType        `json:"formType"`
	DraftRevision    int64           `json:"draftRevision"`
	PublishedVersion int             `json:"publishedVersion"`
	ProtocolVersion  int             `json:"protocolVersion"`
	Draft            JSONContent     `json:"draft"`
	CreatedAt        kernel.JSONTime `json:"createdAt"`
	UpdatedAt        kernel.JSONTime `json:"updatedAt"`
}

// FormSummary 列表条目（不含草稿全文）。
type FormSummary struct {
	ApplicationID    uint            `json:"applicationId"`
	Code             string          `json:"code"`
	Name             string          `json:"name"`
	FormType         FormType        `json:"formType"`
	PublishedVersion int             `json:"publishedVersion"`
	UpdatedAt        kernel.JSONTime `json:"updatedAt"`
}

// ListFormsQuery 应用内表单列表查询（游标按 id 倒序）。
type ListFormsQuery struct {
	ApplicationID uint
	Limit         int
	Cursor        string
}

// FormPage 游标分页结果。
type FormPage struct {
	Items      []FormSummary `json:"items"`
	NextCursor string        `json:"nextCursor"`
	HasMore    bool          `json:"hasMore"`
}

// FormRuntimeFieldPermission 运行时字段权限投影（设计 §8.2 v2.3）：单字段
// 的权限可见/可编辑（合成规则由前端执行：effectiveVisible = 快照可见 ∧ 权限
// 可见，effectiveEditable = 快照可见 ∧ 权限可编辑）。
type FormRuntimeFieldPermission struct {
	Visible  bool `json:"visible"`
	Editable bool `json:"editable"`
}

// FormRuntimePermissions 运行时 permissions 投影：operations 为记录无关操作
// 可用性（add/import，记录级操作由记录接口逐行判定）；viewFields 与
// addFields 是两个独立矩阵——viewFields = FieldsForNew("view")（列表/查看
// 模式列基准），addFields = FieldsForNew("add")（填写模式可填字段），查看与
// 填写的字段集可能不同（如仅录入组），不得合并或按模式二选一返回。
type FormRuntimePermissions struct {
	Operations []string                              `json:"operations" example:"add,import"`
	ViewFields map[string]FormRuntimeFieldPermission `json:"viewFields"`
	AddFields  map[string]FormRuntimeFieldPermission `json:"addFields"`
}

// FormRuntime 运行时 bootstrap 出网（后端契约 §2.2）：已发布快照 + 双口令；
// Permissions 为权限组投影（无权限组基线 S4 下为全量放行矩阵，判定器未接入
// 时为 nil）。
type FormRuntime struct {
	FormCode         string                  `json:"formCode"`
	Name             string                  `json:"name"`
	PublishedVersion int                     `json:"publishedVersion"`
	SchemaRevision   string                  `json:"schemaRevision"`
	ProtocolVersion  int                     `json:"protocolVersion"`
	Content          JSONContent             `json:"content"`
	Permissions      *FormRuntimePermissions `json:"permissions,omitempty"`
}

// SubmitFieldValue 单字段提交快照：Data 缺省与显式 null 均表示空值；Visible
// 必须显式携带，由服务端与发布快照复核，避免客户端伪造隐藏状态绕过必填校验。
type SubmitFieldValue struct {
	Data    JSONContent `json:"data"`
	Visible *bool       `json:"visible"`
}

// SubmitRecordRequest 提交记录（POST /form-records）：应用/菜单上下文、客户端
// 幂等键、发布双口令及按 widgetName 包装的字段快照共同组成稳定提交协议。
type SubmitRecordRequest struct {
	AppCode          string                      `json:"appCode" binding:"required"`
	EntryCode        string                      `json:"entryCode"`
	FormCode         string                      `json:"formCode" binding:"required"`
	PublishedVersion int                         `json:"publishedVersion" binding:"required"`
	SchemaRevision   string                      `json:"schemaRevision" binding:"required"`
	Values           map[string]SubmitFieldValue `json:"values" binding:"required"`
	HasResult        *bool                       `json:"hasResult" binding:"required"`
	DataOpID         string                      `json:"dataOpId" binding:"required"`
}

// SubmitRecordResult 提交受理结果。
type SubmitRecordResult struct {
	WorkflowInstanceNo string `json:"workflowInstanceNo"`

	RecordID uint `json:"recordId"`
}

// RecordQueryExpression 是 POST /forms/:code/records 的受控 Query DSL AST。
// field 只接受发布快照 field_mappings 中的 widgetName；服务端绝不接收 JSONB
// 路径、数据库列名或 SQL 片段。
type RecordQueryExpression struct {
	Type        string                  `json:"type"`
	Conjunction string                  `json:"conjunction,omitempty"`
	Children    []RecordQueryExpression `json:"children,omitempty"`
	Field       string                  `json:"field,omitempty"`
	Operator    string                  `json:"operator,omitempty"`
	Value       any                     `json:"value,omitempty"`
}

type RecordQuerySort struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

// RecordQueryDocument 与 @evolyn.do/query 的可序列化文档同形；列表当前只执行
// filter 与 paging，其他能力必须在服务端具备明确结果语义后才会开放。
type RecordQueryDocument struct {
	Version    int                    `json:"version"`
	Filter     *RecordQueryExpression `json:"filter,omitempty"`
	Sorts      []RecordQuerySort      `json:"sorts,omitempty"`
	Projection []string               `json:"projection,omitempty"`
	GroupBy    []string               `json:"groupBy,omitempty"`
	Aggregates []any                  `json:"aggregates,omitempty"`
	Paging     RecordQueryPaging      `json:"paging"`
	Keyword    string                 `json:"keyword,omitempty"`
}

type RecordQueryPaging struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

// FormRecordDTO 是受 record-level view 权限及字段矩阵裁剪后的记录投影。
// submittedByName/updatedAt 是系统字段数据源（000067：展示名快照 + 最后写回时间），
// 不属于字段矩阵，凡行可见即出网。
type FormRecordDTO struct {
	WorkflowInstanceNo string `json:"workflowInstanceNo"`

	ID                  uint            `json:"id"`
	Values              map[string]any  `json:"values"`
	SubmittedByMemberID uint            `json:"submittedByMemberId"`
	SubmittedByName     string          `json:"submittedByName"`
	SubmittedAt         kernel.JSONTime `json:"submittedAt"`
	UpdatedAt           kernel.JSONTime `json:"updatedAt"`
}

type FormRecordPage struct {
	Items    []FormRecordDTO `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
}
