package model

import (
	"time"

	kernel "evolyn/internal/model"
)

// ---- 运行态持久化模型（000049）：字段与引擎内核 model 一一对应，
// 由仓储适配层做双向转换；引擎内核不感知 GORM（ADR-012）。
// 运行态历史完整保留：全部无软删（GORM 无 DeletedAt）。

// RuntimeTenantBaseModel 是运行态表与迁移 000049 的公共持久化列。
//
// 通用 TenantBaseModel 带有创建人、更新人和 DeletedAt，适用于常规租户资源；
// 流程运行态则是追加历史事实，迁移刻意没有这些列且禁止软删。若直接嵌入
// TenantBaseModel，GORM 会在查询中自动追加 deleted_at IS NULL，导致列表接口
// 访问不存在的列。运行态模型必须使用这一与实际 Schema 严格一致的基类。
type RuntimeTenantBaseModel struct {
	TenantID  uint            `json:"tenantId" gorm:"index;not null;default:1"`
	CreatedAt kernel.JSONTime `json:"createdAt"`
	UpdatedAt kernel.JSONTime `json:"updatedAt"`
}

// WfInstance 流程实例。
type WfInstance struct {
	ID                  uint    `json:"id" gorm:"autoIncrement;primaryKey"`
	DefinitionID        uint    `json:"definitionId" gorm:"not null"`
	DefinitionVersionID uint    `json:"definitionVersionId" gorm:"not null"`
	BusinessType        string  `json:"businessType" gorm:"size:64;not null"`
	BusinessID          string  `json:"businessId" gorm:"size:64;not null"`
	AppID               uint    `json:"appId" gorm:"not null;default:0"`
	FormID              uint    `json:"formId" gorm:"not null;default:0"`
	FormVersionID       uint    `json:"formVersionId" gorm:"not null;default:0"`
	Status              string  `json:"status" gorm:"size:16;not null;default:RUNNING"`
	StarterMemberID     uint    `json:"starterMemberId" gorm:"not null"`
	IdempotencyKey      *string `json:"idempotencyKey" gorm:"size:64"`

	RuntimeTenantBaseModel
}

func (*WfInstance) TableName() string { return "wf_instance" }

// WfExecution 执行路径（V1 仅根路径）。
type WfExecution struct {
	ID                uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	InstanceID        uint   `json:"instanceId" gorm:"not null"`
	ParentExecutionID uint   `json:"parentExecutionId" gorm:"not null;default:0"`
	Status            string `json:"status" gorm:"size:16;not null;default:RUNNING"`

	RuntimeTenantBaseModel
}

func (*WfExecution) TableName() string { return "wf_execution" }

// WfNodeInstance 节点实例。
type WfNodeInstance struct {
	ID          uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	InstanceID  uint   `json:"instanceId" gorm:"not null"`
	ExecutionID uint   `json:"executionId" gorm:"not null"`
	NodeKey     string `json:"nodeKey" gorm:"size:64;not null"`
	Status      string `json:"status" gorm:"size:24;not null;default:PENDING"`

	RuntimeTenantBaseModel
}

func (*WfNodeInstance) TableName() string { return "wf_node_instance" }

// WfTask 人工任务。
type WfTask struct {
	ID                    uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	InstanceID            uint   `json:"instanceId" gorm:"not null"`
	NodeInstanceID        uint   `json:"nodeInstanceId" gorm:"not null"`
	NodeKey               string `json:"nodeKey" gorm:"size:64;not null"`
	Status                string `json:"status" gorm:"size:16;not null;default:PENDING"`
	TransferredFromTaskID uint   `json:"transferredFromTaskId" gorm:"not null;default:0"`
	TransferredToMemberID uint   `json:"transferredToMemberId" gorm:"not null;default:0"`

	RuntimeTenantBaseModel
}

func (*WfTask) TableName() string { return "wf_task" }

// WfTaskActor 任务参与人快照。
type WfTaskActor struct {
	ID          uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	TaskID      uint   `json:"taskId" gorm:"not null"`
	MemberID    uint   `json:"memberId" gorm:"not null"`
	DisplayName string `json:"displayName" gorm:"size:100;not null;default:''"`
	ActorRole   string `json:"actorRole" gorm:"size:16;not null;default:assignee"`

	// 任务参与人是创建时的快照，000049 没有 updated_at，故不能复用
	// RuntimeTenantBaseModel。
	TenantID  uint            `json:"tenantId" gorm:"index;not null;default:1"`
	CreatedAt kernel.JSONTime `json:"createdAt"`
}

func (*WfTaskActor) TableName() string { return "wf_task_actor" }

// WfOperation 操作流水（追加写）。
type WfOperation struct {
	ID               uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	InstanceID       uint            `json:"instanceId" gorm:"not null"`
	TaskID           uint            `json:"taskId" gorm:"not null;default:0"`
	OperatorMemberID uint            `json:"operatorMemberId" gorm:"not null;default:0"`
	OperationType    string          `json:"operationType" gorm:"size:32;not null"`
	Payload          DSLContent      `json:"payload" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt        kernel.JSONTime `json:"createdAt"`
	// TenantID 租户隔离（追加写，无更新语义故不嵌 TenantBaseModel）
	TenantID uint `json:"tenantId" gorm:"index;not null;default:1"`
}

func (*WfOperation) TableName() string { return "wf_operation" }

// WfCCRecord 抄送记录（000051，追加写；第 10.6 章）。
type WfCCRecord struct {
	ID             uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	InstanceID     uint            `json:"instanceId" gorm:"not null"`
	NodeInstanceID uint            `json:"nodeInstanceId" gorm:"not null;default:0"`
	NodeKey        string          `json:"nodeKey" gorm:"size:64;not null;default:''"`
	MemberID       uint            `json:"memberId" gorm:"not null"`
	DisplayName    string          `json:"displayName" gorm:"size:100;not null;default:''"`
	CreatedAt      kernel.JSONTime `json:"createdAt"`
	// TenantID 租户隔离（追加写，无更新语义故不嵌 TenantBaseModel）
	TenantID uint `json:"tenantId" gorm:"index;not null;default:1"`
}

func (*WfCCRecord) TableName() string { return "wf_cc_record" }

// WfJob 延时任务（000052，Phase 5）。
type WfJob struct {
	ID             uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	JobType        string          `json:"jobType" gorm:"size:32;not null"`
	InstanceID     uint            `json:"instanceId" gorm:"not null;default:0"`
	NodeInstanceID uint            `json:"nodeInstanceId" gorm:"not null;default:0"`
	TaskID         uint            `json:"taskId" gorm:"not null;default:0"`
	ExecuteAt      time.Time       `json:"executeAt" gorm:"not null"`
	Status         string          `json:"status" gorm:"size:16;not null;default:PENDING"`
	RetryCount     int             `json:"retryCount" gorm:"not null;default:0"`
	MaxRetryCount  int             `json:"maxRetryCount" gorm:"not null;default:3"`
	Payload        DSLContent      `json:"payload" gorm:"type:jsonb;not null;default:'{}'"`
	LastError      string          `json:"lastError" gorm:"type:text;not null;default:''"`
	CreatedAt      kernel.JSONTime `json:"createdAt"`
	UpdatedAt      kernel.JSONTime `json:"updatedAt"`
	// TenantID 租户隔离（Worker 全租户轮询路径 ctx 无租户上下文，本表
	// 由租户 Callback 兜底；追加/状态机回写，无业务更新语义）
	TenantID uint `json:"tenantId" gorm:"index;not null;default:1"`
}

func (*WfJob) TableName() string { return "wf_job" }

// WfVariable 流程变量（000053，Phase 7）：实例作用域 (instance_id, var_key)
// 唯一，JSONB 单值承载 V1 冻结的标量值域；service 节点响应映射写入。
type WfVariable struct {
	ID         uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	InstanceID uint            `json:"instanceId" gorm:"not null"`
	VarKey     string          `json:"varKey" gorm:"size:64;not null"`
	VarType    string          `json:"varType" gorm:"size:16;not null"`
	VarValue   DSLContent      `json:"varValue" gorm:"type:jsonb;not null;default:'null'"`
	CreatedAt  kernel.JSONTime `json:"createdAt"`
	UpdatedAt  kernel.JSONTime `json:"updatedAt"`

	TenantID uint `json:"tenantId" gorm:"index;not null;default:1"`
}

func (*WfVariable) TableName() string { return "wf_variable" }
