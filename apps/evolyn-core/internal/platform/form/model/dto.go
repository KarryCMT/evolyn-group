package model

import (
	kernel "evolyn/internal/model"
)

// CreateFormRequest 创建表单（POST /forms）：草稿初始化为空协议文档。
// ParentEntryCode 可选：应用菜单中目标分组的节点编码（侧栏资产 code），
// 传入时表单菜单节点挂到该分组下，否则挂应用根级。
type CreateFormRequest struct {
	ApplicationID   uint   `json:"applicationId" binding:"required" example:"1"`
	Name            string `json:"name" binding:"required" example:"报名表"`
	ParentEntryCode string `json:"parentEntryCode" example:"menu_ab12cd34ef56ab12"`
}

// UpdateFormRequest 白名单改名（PATCH /forms/:id）。
type UpdateFormRequest struct {
	Name *string `json:"name"`
}

// SaveDraftRequest 保存草稿（PUT /forms/:id/draft）：全量替换 + 乐观锁口令。
// content 为目标保存协议根结构；校验失败返回 FORM_SCHEMA_INVALID + issues。
type SaveDraftRequest struct {
	DraftRevision int64       `json:"draftRevision" binding:"required"`
	Content       JSONContent `json:"content" binding:"required"`
}

// SaveDraftResult 草稿保存结果：新口令供下次保存回传。
type SaveDraftResult struct {
	DraftRevision int64 `json:"draftRevision"`
}

// PublishRequest 发布（POST /forms/:id/publish）：携带草稿口令防止发布并发旧草稿。
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
	ID               uint            `json:"id"`
	ApplicationID    uint            `json:"applicationId"`
	Name             string          `json:"name"`
	DraftRevision    int64           `json:"draftRevision"`
	PublishedVersion int             `json:"publishedVersion"`
	Draft            JSONContent     `json:"draft"`
	CreatedAt        kernel.JSONTime `json:"createdAt"`
	UpdatedAt        kernel.JSONTime `json:"updatedAt"`
}

// FormSummary 列表条目（不含草稿全文）。
type FormSummary struct {
	ID               uint            `json:"id"`
	ApplicationID    uint            `json:"applicationId"`
	Name             string          `json:"name"`
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
	FormID           uint        `json:"formId"`
	Name             string      `json:"name"`
	PublishedVersion int         `json:"publishedVersion"`
	SchemaRevision   string      `json:"schemaRevision"`
	Content          JSONContent `json:"content"`
}

// SubmitRecordRequest 提交记录（POST /form-records）：双口令定位发布快照。
type SubmitRecordRequest struct {
	FormID           uint                   `json:"formId" binding:"required"`
	PublishedVersion int                    `json:"publishedVersion" binding:"required"`
	SchemaRevision   string                 `json:"schemaRevision" binding:"required"`
	Values           map[string]JSONContent `json:"values" binding:"required"`
}

// SubmitRecordResult 提交受理结果。
type SubmitRecordResult struct {
	RecordID uint `json:"recordId"`
}
