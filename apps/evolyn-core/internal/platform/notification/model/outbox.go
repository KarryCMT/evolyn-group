// Package model 消息中心域数据模型：事务 Outbox 事件。
package model

import (
	"time"

	kernel "evolyn/internal/model"
)

// Outbox 处理状态（稳定枚举；processing 预留给「先领取再处理」的演进形态，
// 当前 Dispatcher 在单事务内 pending→done，崩溃回滚天然回 pending）
const (
	OutboxPending    = "pending"
	OutboxProcessing = "processing"
	OutboxDone       = "done"
	OutboxFailed     = "failed"
)

// OutboxEvent 业务事务与消息物化之间的可靠边界：业务 Service 在自身事务内
// 写入（随业务提交/回滚），Dispatcher 以 FOR UPDATE SKIP LOCKED 小批领取，
// 重试按 event_id 幂等。
type OutboxEvent struct {
	ID                uint        `json:"id" gorm:"autoIncrement;primaryKey"`
	EventID           string      `json:"eventId" gorm:"size:128;not null"` // 全局幂等键
	EventCode         string      `json:"eventCode" gorm:"size:128;not null"`
	ActorMemberID     uint        `json:"actorMemberId" gorm:"not null;default:0"`      // 0=系统发起
	AudienceMemberIDs JSONContent `json:"audienceMemberIds" gorm:"type:jsonb;not null"` // 成员 ID 数组
	Parameters        JSONContent `json:"parameters" gorm:"type:jsonb;not null"`        // 受注册表 Schema 限制
	OccurredAt        time.Time   `json:"occurredAt" gorm:"not null"`
	Status            string      `json:"status" gorm:"size:16;not null;default:pending"`
	AttemptCount      int         `json:"attemptCount" gorm:"not null;default:0"`
	NextAttemptAt     time.Time   `json:"nextAttemptAt" gorm:"not null"`
	LastErrorCode     string      `json:"lastErrorCode" gorm:"size:100;not null;default:''"` // 稳定内部码，不存原始错误
	ProcessedAt       *time.Time  `json:"processedAt"`

	TenantID  uint            `json:"tenantId" gorm:"not null;default:1"`
	CreatedAt kernel.JSONTime `json:"createdAt"`
}

func (*OutboxEvent) TableName() string { return "tn_notification_outbox_events" }
