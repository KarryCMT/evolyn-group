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

// ---- 运行态 DTO（Phase 2 最小 Runtime） ----

// StartInstanceRequest 发起流程实例（第 14 章双层幂等）。
type StartInstanceRequest struct {
	// DefinitionCode 流程定义稳定公开编码
	DefinitionCode string `json:"definitionCode"`
	// BusinessType / BusinessID 业务绑定（业务幂等键）
	BusinessType string `json:"businessType"`
	BusinessID   string `json:"businessId"`
	// IdempotencyKey 请求幂等键（可选；双击/重试重放返回同一实例）
	IdempotencyKey string `json:"idempotencyKey"`
	// AppID 业务归属应用（可选，0=未绑定）
	AppID uint `json:"appId"`
	// FormCode + FormVersionNo 表单绑定（可选；经 FormDirectory 窄端口
	// 解析为内部 form_id/form_version_id 落库）
	FormCode      string `json:"formCode"`
	FormVersionNo int    `json:"formVersionNo"`
}

// ApproveTaskRequest 审批同意请求。
type ApproveTaskRequest struct {
	TaskID  uint   `json:"taskId"`
	Comment string `json:"comment"`
}

// TaskActorView 任务参与人（快照）。
type TaskActorView struct {
	MemberID    uint   `json:"memberId"`
	DisplayName string `json:"displayName"`
}

// InstanceTaskView 实例任务视图。
type InstanceTaskView struct {
	ID             uint            `json:"id"`
	NodeInstanceID uint            `json:"nodeInstanceId"`
	NodeKey        string          `json:"nodeKey"`
	Status         string          `json:"status"`
	Actors         []TaskActorView `json:"actors"`
	CreatedAt      kernel.JSONTime `json:"createdAt"`
}

// InstanceNodeView 节点实例视图。
type InstanceNodeView struct {
	ID      uint   `json:"id"`
	NodeKey string `json:"nodeKey"`
	Status  string `json:"status"`
}

// InstanceOperationView 操作流水视图（时间线）。
type InstanceOperationView struct {
	ID               uint            `json:"id"`
	TaskID           uint            `json:"taskId"`
	OperatorMemberID uint            `json:"operatorMemberId"`
	Type             string          `json:"type"`
	Payload          json.RawMessage `json:"payload"`
	CreatedAt        kernel.JSONTime `json:"createdAt"`
}

// InstanceDetail 实例详情：绑定关系 + 节点/任务/操作时间线。
type InstanceDetail struct {
	ID                  uint                    `json:"id"`
	DefinitionCode      string                  `json:"definitionCode"`
	DefinitionVersionNo int                     `json:"definitionVersionNo"`
	BusinessType        string                  `json:"businessType"`
	BusinessID          string                  `json:"businessId"`
	AppID               uint                    `json:"appId"`
	FormID              uint                    `json:"formId"`
	FormVersionID       uint                    `json:"formVersionId"`
	Status              string                  `json:"status"`
	StarterMemberID     uint                    `json:"starterMemberId"`
	IdempotencyKey      string                  `json:"idempotencyKey"`
	CreatedAt           kernel.JSONTime         `json:"createdAt"`
	Nodes               []InstanceNodeView      `json:"nodes"`
	Tasks               []InstanceTaskView      `json:"tasks"`
	Operations          []InstanceOperationView `json:"operations"`
}

// ApproveTaskResult 审批同意结果。
type ApproveTaskResult struct {
	InstanceID     uint   `json:"instanceId"`
	InstanceStatus string `json:"instanceStatus"`
	NodeCompleted  bool   `json:"nodeCompleted"`
}
