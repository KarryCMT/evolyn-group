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
	DraftRevision int64       `json:"draftRevision" binding:"required"`
	Content       JSONContent `json:"content" binding:"required"`
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

// FormRuntime 运行时 bootstrap 出网（后端契约 §2.2）：已发布快照 + 双口令。
type FormRuntime struct {
	FormCode         string      `json:"formCode"`
	Name             string      `json:"name"`
	PublishedVersion int         `json:"publishedVersion"`
	SchemaRevision   string      `json:"schemaRevision"`
	Content          JSONContent `json:"content"`
}

// SubmitRecordRequest 提交记录（POST /form-records）：双口令定位发布快照。
type SubmitRecordRequest struct {
	FormCode         string                 `json:"formCode" binding:"required"`
	PublishedVersion int                    `json:"publishedVersion" binding:"required"`
	SchemaRevision   string                 `json:"schemaRevision" binding:"required"`
	Values           map[string]JSONContent `json:"values" binding:"required"`
}

// SubmitRecordResult 提交受理结果。
type SubmitRecordResult struct {
	RecordID uint `json:"recordId"`
}
