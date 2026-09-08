// 存储元数据模型（物理表存储方案 §5.2）：存储绑定、物理模型版本与 DDL Job。
// 三表均由 Service 在发布事务内写入；状态枚举与迁移 CHECK 一致，由本文件
// 常量维护（唯一事实源），数据库 CHECK 只保护已冻结状态集。
//
// 三表均为机器运维元数据（无人工操作者语义、无软删），持久化列与迁移
// 000070 严格一致：tenant_id + created_at/updated_at，不嵌入带 DeletedAt
// 的通用基类（GORM 会追加不存在的 deleted_at 过滤）。
package model

import (
	"time"

	kernel "evolyn/internal/model"
)

// StorageBackend 存储后端枚举。
type StorageBackend string

const (
	// StorageBackendPhysical 物理值表（新建表单唯一选项）。
	StorageBackendPhysical StorageBackend = "PHYSICAL"
	// StorageBackendLegacyJSONB 存量 JSONB 兼容（不创建新绑定，仅供读取判定）。
	StorageBackendLegacyJSONB StorageBackend = "LEGACY_JSONB"
)

// StorageState 存储绑定状态。
type StorageState string

const (
	// StorageStateReady 可用（无未完成 DDL）。
	StorageStateReady StorageState = "READY"
	// StorageStatePublishing 有未完成 DDL Job。
	StorageStatePublishing StorageState = "PUBLISHING"
	// StorageStateFailed 最近一次发布 DDL 终态失败（重试成功后恢复 READY）。
	StorageStateFailed StorageState = "FAILED"
)

// SchemaVersionState 物理模型版本状态。
type SchemaVersionState string

const (
	// SchemaVersionPending 待执行（DDL Job 排队/执行中）。
	SchemaVersionPending SchemaVersionState = "PENDING"
	// SchemaVersionApplied 已应用（物理表结构已就位）。
	SchemaVersionApplied SchemaVersionState = "APPLIED"
	// SchemaVersionFailed 终态失败（重试超限）。
	SchemaVersionFailed SchemaVersionState = "FAILED"
)

// DDLJobStatus DDL Job 执行状态（与 wf_job 状态机同形：PENDING→PROCESSING→
// SUCCEEDED/FAILED，失败退避回 PENDING）。
type DDLJobStatus string

const (
	DDLJobPending    DDLJobStatus = "PENDING"
	DDLJobProcessing DDLJobStatus = "PROCESSING"
	DDLJobSucceeded  DDLJobStatus = "SUCCEEDED"
	DDLJobFailed     DDLJobStatus = "FAILED"
)

// storageMetaColumns 三表公共持久化列（与迁移 000070 一致；无软删/无操作者）。
type StorageMetaColumns struct {
	TenantID  uint            `json:"tenantId" gorm:"not null"`
	CreatedAt kernel.JSONTime `json:"createdAt"`
	UpdatedAt kernel.JSONTime `json:"updatedAt"`
}

// FormStorage 表单存储绑定：每表单一行，创建表单事务内落库；table_name 由
// 服务端按 tn_fd_<app4>_<rand6> 分配，唯一约束防重（冲突重试至多五次），
// 分配后永久不变（应用/表单改名、复制、归档均不改表名）。
type FormStorage struct {
	ID                     uint           `json:"id" gorm:"autoIncrement;primaryKey"`
	FormID                 uint           `json:"formId" gorm:"not null"`
	Backend                StorageBackend `json:"backend" gorm:"size:16;not null"`
	PhysicalTable          string         `json:"tableName" gorm:"column:table_name;size:63;not null"`
	State                  StorageState   `json:"state" gorm:"size:16;not null"`
	AppliedSchemaVersionID *uint          `json:"appliedSchemaVersionId"`

	StorageMetaColumns
}

func (*FormStorage) TableName() string { return "tn_form_storages" }

// FormStorageSchemaVersion 物理模型版本：发布事务内创建（不可变，仅状态
// 推进），model/plan 为 engine/data/storage 纯模型的 JSONB 持久化形态。
type FormStorageSchemaVersion struct {
	ID            uint               `json:"id" gorm:"autoIncrement;primaryKey"`
	StorageID     uint               `json:"storageId" gorm:"not null"`
	FormVersionID uint               `json:"formVersionId" gorm:"not null"`
	Model         JSONContent        `json:"model" gorm:"type:jsonb;not null"`
	Plan          JSONContent        `json:"plan" gorm:"type:jsonb;not null"`
	Checksum      string             `json:"checksum" gorm:"size:64;not null"`
	State         SchemaVersionState `json:"state" gorm:"size:16;not null"`
	AppliedAt     *kernel.JSONTime   `json:"appliedAt"`
	ErrorCode     string             `json:"errorCode" gorm:"size:64"`
	ErrorDetail   string             `json:"errorDetail"`

	StorageMetaColumns
}

func (*FormStorageSchemaVersion) TableName() string { return "tn_form_storage_schema_versions" }

// FormDDLJob DDL 异步执行 Job：同一 storage_schema_version 至多一行；
// Worker 以 FOR UPDATE SKIP LOCKED 领取，claim+执行+回写同事务。
type FormDDLJob struct {
	ID                     uint             `json:"id" gorm:"autoIncrement;primaryKey"`
	StorageSchemaVersionID uint             `json:"storageSchemaVersionId" gorm:"not null"`
	Status                 DDLJobStatus     `json:"status" gorm:"size:16;not null"`
	RetryCount             int              `json:"retryCount" gorm:"not null;default:0"`
	NextAttemptAt          kernel.JSONTime  `json:"nextAttemptAt"`
	LastErrorCode          string           `json:"lastErrorCode" gorm:"size:64"`
	LastErrorDetail        string           `json:"lastErrorDetail"`
	StartedAt              *kernel.JSONTime `json:"startedAt"`
	FinishedAt             *kernel.JSONTime `json:"finishedAt"`

	StorageMetaColumns
}

func (*FormDDLJob) TableName() string { return "tn_form_ddl_jobs" }

// FormStorageChild 子表单物理子表永久映射（方案 §5.4）：fieldId→表名一次性
// 分配后永不变更；表名全局唯一约束是子表名防重边界。
type FormStorageChild struct {
	ID            uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	StorageID     uint   `json:"storageId" gorm:"not null"`
	ParentFieldID string `json:"parentFieldId" gorm:"column:parent_field_id;size:16;not null"`
	PhysicalTable string `json:"tableName" gorm:"column:table_name;size:63;not null"`

	StorageMetaColumns
}

func (*FormStorageChild) TableName() string { return "tn_form_storage_children" }

// JSONTimeOf 构造存储元数据时间列值（Worker/Service 共用）。
func JSONTimeOf(t time.Time) kernel.JSONTime { return kernel.JSONTime(t) }
