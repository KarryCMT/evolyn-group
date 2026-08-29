package model

import "time"

// Job 持久化延时任务（超时/提醒/延迟执行；V1 不引入 Asynq，
// 采用 wf_job + Worker 轮询 FOR UPDATE SKIP LOCKED，第 19 章）。
type Job struct {
	ID uint
	// TenantID 归属租户
	TenantID uint
	// Type 任务类型
	Type JobType
	// InstanceID / NodeInstanceID / TaskID 关联运行态对象（按需为 0）
	InstanceID     uint
	NodeInstanceID uint
	TaskID         uint
	// ExecuteAt 计划执行时间（Worker 轮询窗口；UTC 标准库时间，
	// 出网格式化由平台层 JSONTime 统一承载）
	ExecuteAt time.Time
	// Status 任务状态机（迁移表见 task/state_machine.go）
	Status JobStatus
	// RetryCount / MaxRetryCount 已重试次数与上限（超限 FAILED 终态）
	RetryCount    int
	MaxRetryCount int
	// Payload 任务载荷（提醒文案、超时动作参数等）
	Payload map[string]any
	// LastError 最近一次失败摘要（只入日志/诊断，不出网）
	LastError string
}

// JobType 任务类型目录（Phase 7 定版：service.invoke 承载 service 节点
// 的异步 HTTP 调用，失败重试经 wf_job 重试记账退避回队——即第 19.1 章
// 预留的 service.retry 语义；PostgreSQL 仍是流程状态唯一事实源）。
type JobType string

const (
	JobTypeTaskReminder  JobType = "task.reminder"
	JobTypeTaskTimeout   JobType = "task.timeout"
	JobTypeServiceInvoke JobType = "service.invoke"
)

// JobStatus Job 状态机（第 19.1 章）。
type JobStatus string

const (
	// JobStatusPENDING 待执行（Worker 领取窗口）
	JobStatusPENDING JobStatus = "PENDING"
	// JobStatusPROCESSING 已领取执行中（crash 后由 Worker 回收恢复）
	JobStatusPROCESSING JobStatus = "PROCESSING"
	// JobStatusSUCCEEDED 执行成功（终态）
	JobStatusSUCCEEDED JobStatus = "SUCCEEDED"
	// JobStatusFAILED 执行失败（终态：超过 max_retry_count）
	JobStatusFAILED JobStatus = "FAILED"
	// JobStatusCANCELLED 已取消（任务完成/实例终止时联动取消，终态）
	JobStatusCANCELLED JobStatus = "CANCELLED"
)
