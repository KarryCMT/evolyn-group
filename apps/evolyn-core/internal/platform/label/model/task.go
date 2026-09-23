package model

import kernel "evolyn/internal/model"

const (
	RenderTaskPending        = "pending"
	RenderTaskRunning        = "running"
	RenderTaskSuccess        = "success"
	RenderTaskPartialSuccess = "partial_success"
	RenderTaskFailed         = "failed"
	RenderTaskCancelled      = "cancelled"

	RenderItemPending = "pending"
	RenderItemRunning = "running"
	RenderItemSuccess = "success"
	RenderItemFailed  = "failed"
)

// RenderTask 固定指向创建时的模板发布快照，避免排队期间模板再次发布导致
// 同一批标签出现不同版式。
type RenderTask struct {
	ID                  uint             `json:"-" gorm:"autoIncrement;primaryKey"`
	TenantID            uint             `json:"-" gorm:"index;not null"`
	Code                string           `json:"taskId" gorm:"size:64;not null"`
	TemplateID          uint             `json:"-" gorm:"not null"`
	AppID               uint             `json:"-" gorm:"not null"`
	FormID              uint             `json:"-" gorm:"not null"`
	TemplateVersionID   uint             `json:"-" gorm:"not null"`
	TemplateVersionNo   int              `json:"templateVersion" gorm:"not null"`
	OutputFormat        string           `json:"format" gorm:"size:12;not null"`
	Status              string           `json:"status" gorm:"size:24;not null"`
	TotalCount          int              `json:"totalCount" gorm:"not null"`
	SuccessCount        int              `json:"successCount" gorm:"not null"`
	FailedCount         int              `json:"failedCount" gorm:"not null"`
	Progress            int              `json:"progress" gorm:"not null"`
	FileCode            string           `json:"fileId,omitempty" gorm:"size:64"`
	ErrorCode           string           `json:"errorCode,omitempty" gorm:"size:64;not null"`
	ErrorMessage        string           `json:"errorMessage,omitempty" gorm:"size:500;not null"`
	RequestedByMemberID uint             `json:"-" gorm:"not null"`
	StartedAt           *kernel.JSONTime `json:"startedAt,omitempty"`
	FinishedAt          *kernel.JSONTime `json:"finishedAt,omitempty"`
	CreatedAt           kernel.JSONTime  `json:"createdAt"`
	UpdatedAt           kernel.JSONTime  `json:"updatedAt"`
}

func (*RenderTask) TableName() string { return "tn_label_render_tasks" }

type RenderTaskItem struct {
	ID           uint             `json:"-" gorm:"autoIncrement;primaryKey"`
	TenantID     uint             `json:"-" gorm:"index;not null"`
	TaskID       uint             `json:"-" gorm:"not null"`
	RecordID     uint             `json:"-" gorm:"not null"`
	SequenceNo   int              `json:"sequence" gorm:"not null"`
	Status       string           `json:"status" gorm:"size:16;not null"`
	ErrorCode    string           `json:"errorCode,omitempty" gorm:"size:64;not null"`
	ErrorMessage string           `json:"errorMessage,omitempty" gorm:"size:500;not null"`
	StartedAt    *kernel.JSONTime `json:"startedAt,omitempty"`
	FinishedAt   *kernel.JSONTime `json:"finishedAt,omitempty"`
	CreatedAt    kernel.JSONTime  `json:"createdAt"`
	UpdatedAt    kernel.JSONTime  `json:"updatedAt"`
}

func (*RenderTaskItem) TableName() string { return "tn_label_render_task_items" }
