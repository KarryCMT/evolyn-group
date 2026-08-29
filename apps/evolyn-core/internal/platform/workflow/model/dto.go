package model

import (
	"encoding/json"

	kernel "evolyn/internal/model"
)

// ---- 出网 DTO（不暴露引擎内部模型与内部自增 ID；流程一律以 code 定位，
// 版本以 version_no 标识；时间统一 kernel.JSONTime 出网格式） ----

// CreateWorkflowRequest 创建流程定义请求。
type CreateWorkflowRequest struct {
	Name        string `json:"name"`        // 必填，trim 后 1–128 字符
	Description string `json:"description"` // 可选，≤512 字符
}

// UpdateWorkflowRequest 白名单更新（PATCH，指针区分未提交字段）。
type UpdateWorkflowRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// SaveDraftRequest 保存草稿：draft_revision 乐观锁 + DSL 全量替换
// （PUT 语义；draft 为 Workflow DSL v1 全文档）。
type SaveDraftRequest struct {
	DraftRevision int64           `json:"draftRevision"`
	Draft         json.RawMessage `json:"draft"`
}

// SaveDraftResult 草稿保存结果：新口令供下次保存回传。
type SaveDraftResult struct {
	DraftRevision int64 `json:"draftRevision"`
}

// PublishRequest 发布：按草稿当前口令发布（口令不符即并发冲突）。
type PublishRequest struct {
	DraftRevision int64 `json:"draftRevision"`
}

// PublishResult 发布结果：version_no 即运行实例（Phase 2）定位版本的公开标识。
type PublishResult struct {
	VersionNo int `json:"versionNo"`
}

// WorkflowSummary 列表条目（草稿全文不出网，详情接口携带）。
type WorkflowSummary struct {
	Code             string          `json:"code"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	PublishedVersion int             `json:"publishedVersion"`
	DraftRevision    int64           `json:"draftRevision"`
	CreatorMemberID  uint            `json:"creatorMemberId"`
	CreatedAt        kernel.JSONTime `json:"createdAt"`
	UpdatedAt        kernel.JSONTime `json:"updatedAt"`
}

// WorkflowDetail 定义详情：含 DSL 草稿全文与修订口令。
type WorkflowDetail struct {
	WorkflowSummary
	Draft json.RawMessage `json:"draft"`
}

// WorkflowPage 游标分页（id 倒序，新定义靠前；cursor 不透明值原样回传）。
type WorkflowPage struct {
	Items      []WorkflowSummary `json:"items"`
	NextCursor string            `json:"nextCursor"`
}

// ListWorkflowsQuery 列表查询参数。
type ListWorkflowsQuery struct {
	Limit  int
	Cursor string
}

// VersionSummary 发布版本条目（快照全文不出网，详情接口携带）。
type VersionSummary struct {
	VersionNo           int             `json:"versionNo"`
	PublishedByMemberID uint            `json:"publishedByMemberId"`
	PublishedAt         kernel.JSONTime `json:"publishedAt"`
}

// VersionDetail 指定版本详情：含冻结的 DSL 快照全文。
type VersionDetail struct {
	VersionSummary
	DSL json.RawMessage `json:"dsl"`
}
