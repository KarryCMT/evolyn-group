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
	// FormCode 可选：绑定的表单公开编码（form_ 前缀）。流程型表单的流程
	// 设计页懒建定义时传入，租户内一条表单至多绑定一条未删除定义
	//（uk_wf_definition_form_code 兜底）；空串=独立定义。
	FormCode string `json:"formCode"`
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
	FormCode         string          `json:"formCode"`
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
	// FormCode 精确过滤：流程设计页按绑定表单定位定义（空=不过滤）
	FormCode string
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
	// Values 审批编辑的表单字段值（键=widgetName，可选）：仅允许节点
	// 字段权限授权（editable/required）的字段，越权字段整体拒绝
	//（WORKFLOW_FORM_FIELD_FORBIDDEN），授权字段经 Form Domain 按冻结
	// 快照校验后同事务写回（FORM_RECORD_INVALID + fieldErrors 回填）
	Values map[string]json.RawMessage `json:"values,omitempty"`
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
	InstanceNo string `json:"instanceNo"`

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

// ---- 审批中心与完整人工任务（Phase 4，第 20.3/20.4 章） ----

// RejectTaskRequest 驳回请求（V1 仅 terminate 语义）。
type RejectTaskRequest struct {
	TaskID  uint   `json:"taskId"`
	Comment string `json:"comment"`
}

// ReturnTaskRequest 退回发起人请求。
type ReturnTaskRequest struct {
	TaskID  uint   `json:"taskId"`
	Comment string `json:"comment"`
}

// TransferTaskRequest 转办请求。
type TransferTaskRequest struct {
	TaskID uint `json:"taskId"`
	// TargetMemberID 转办目标成员（同租户有效成员，服务层校验）
	TargetMemberID uint   `json:"targetMemberId"`
	Comment        string `json:"comment"`
}

// InstanceActionRequest 实例级动作请求（撤回/终止/重提交共用 comment 位）。
type InstanceActionRequest struct {
	Comment string `json:"comment"`
}

// ResubmitInstanceRequest 发起人重新提交请求：修改后的表单字段值（可选，
// 经 Form Domain 按冻结快照整体校验）。
type ResubmitInstanceRequest struct {
	Values map[string]json.RawMessage `json:"values,omitempty"`
}

// ActionTaskResult 任务级动作结果（驳回/退回/转办共用）。
type ActionTaskResult struct {
	InstanceID     uint   `json:"instanceId"`
	InstanceStatus string `json:"instanceStatus"`
	// NewTaskID 转办产生的新任务 ID（非转办动作为 0）
	NewTaskID uint `json:"newTaskId,omitempty"`
}

// TaskActorView 任务参与人（快照）—— 位置见上，Phase 2 已定义。

// TaskSummary 审批中心任务条目。
// TaskSummaryField 仅包含节点可见的标量摘要，最多展示三个字段。
type TaskSummaryField struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type TaskSummary struct {
	InstanceNo string `json:"instanceNo"`

	Title         string             `json:"title"`
	NodeName      string             `json:"nodeName"`
	StarterName   string             `json:"starterName"`
	SummaryFields []TaskSummaryField `json:"summaryFields"`

	ID              uint            `json:"id"`
	InstanceID      uint            `json:"instanceId"`
	NodeKey         string          `json:"nodeKey"`
	Status          string          `json:"status"`
	Actors          []TaskActorView `json:"actors"`
	TransferredFrom uint            `json:"transferredFrom,omitempty"`
	CreatedAt       kernel.JSONTime `json:"createdAt"`
}

// TaskPage 任务游标分页（id 倒序；cursor 不透明值原样回传）。
type TaskPage struct {
	Items      []TaskSummary `json:"items"`
	NextCursor string        `json:"nextCursor"`
}

// PendingTaskFormCount 是待办按流程表单归属的聚合项。未绑定表单的独立流程
// 仍计入总数，但不会出现在表单菜单的二级筛选项中。
type PendingTaskFormCount struct {
	FormCode string `json:"formCode"`
	Count    int64  `json:"count"`
}

// PendingTaskSummary 是流程侧栏的真实待办徽标数据，避免前端以当前分页条数
// 伪装总量。FormCounts 只包含实际存在待办的已绑定流程型表单。
type PendingTaskSummary struct {
	Total      int64                  `json:"total"`
	FormCounts []PendingTaskFormCount `json:"formCounts"`
}

// CCRecordView 抄送我的条目。
type CCRecordView struct {
	ID          uint            `json:"id"`
	InstanceID  uint            `json:"instanceId"`
	NodeKey     string          `json:"nodeKey"`
	MemberID    uint            `json:"memberId"`
	DisplayName string          `json:"displayName"`
	CreatedAt   kernel.JSONTime `json:"createdAt"`
}

// CCPage 抄送游标分页。
type CCPage struct {
	Items      []CCRecordView `json:"items"`
	NextCursor string         `json:"nextCursor"`
}

// InstanceSummary 审批中心实例条目（我发起的）。
type InstanceSummary struct {
	InstanceNo string `json:"instanceNo"`

	ID                  uint            `json:"id"`
	DefinitionCode      string          `json:"definitionCode"`
	DefinitionVersionNo int             `json:"definitionVersionNo"`
	BusinessType        string          `json:"businessType"`
	BusinessID          string          `json:"businessId"`
	Status              string          `json:"status"`
	StarterMemberID     uint            `json:"starterMemberId"`
	CreatedAt           kernel.JSONTime `json:"createdAt"`
}

// InstancePage 实例游标分页。
type InstancePage struct {
	Items      []InstanceSummary `json:"items"`
	NextCursor string            `json:"nextCursor"`
}

// TaskDetail 任务详情上下文（第 4 章审批详情返回协议）：任务 + 实例绑定
// + 表单快照/数据 + 节点字段权限 + 允许动作 + 操作时间线。
type TaskDetail struct {
	Task       TaskSummary     `json:"task"`
	Instance   InstanceSummary `json:"instance"`
	NodeKey    string          `json:"nodeKey"`
	NodeStatus string          `json:"nodeStatus"`
	// FormPermissions 当前节点字段权限（widgetName → 权限；无配置为空对象）
	FormPermissions map[string]string `json:"formPermissions"`
	// AllowedActions 当前任务允许的动作（PENDING：approve/reject/
	// return-to-starter/transfer；终态为空数组）
	AllowedActions []string `json:"allowedActions"`
	// FormCode / FormVersionNo / FormContent 表单冻结绑定投影（未绑定为空）
	FormCode      string          `json:"formCode,omitempty"`
	FormVersionNo int             `json:"formVersionNo,omitempty"`
	FormContent   json.RawMessage `json:"formContent,omitempty"`
	// FormValues 业务数据当前值（未绑定为空对象）
	FormValues map[string]any          `json:"formValues"`
	Operations []InstanceOperationView `json:"operations"`
}

// ListTasksQuery 审批中心任务查询参数。
type ListTasksQuery struct {
	// Scope pending=我的待办 / completed=我的已办 / cc-to-me=抄送我的
	Scope  string
	Limit  int
	Cursor string
	// FormCode 仅允许在 pending 中使用：按绑定流程表单筛选待办。
	FormCode string
}

// ListInstancesQuery 实例查询参数（scope=started-by-me 我发起的）。
type ListInstancesQuery struct {
	Scope  string
	Limit  int
	Cursor string
}
