// Package model 消息中心域数据模型：不可变逻辑消息、成员收件箱与出网 DTO。
package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	kernel "evolyn/internal/model"
)

// Message 不可变逻辑消息：模板渲染后的展示快照在物化时固化，多成员经收件箱
// 共享同一行；无软删，过期后由保留清理 Worker 成批硬删。
type Message struct {
	ID           uint        `json:"id" gorm:"autoIncrement;primaryKey"`
	EventID      string      `json:"eventId" gorm:"size:128;not null"`     // 生产者事件唯一 ID（Worker 幂等键之一）
	CategoryCode string      `json:"categoryCode" gorm:"size:64;not null"` // 稳定分类码
	EventCode    string      `json:"eventCode" gorm:"size:128;not null"`   // 稳定事件码
	Severity     string      `json:"severity" gorm:"size:16;not null;default:info"`
	Title        string      `json:"title" gorm:"size:200;not null;default:''"`
	Content      string      `json:"content" gorm:"size:2000;not null"`
	Action       JSONContent `json:"action" gorm:"type:jsonb;not null"`    // 受控跳转动作（稳定动作码+白名单参数）
	SourceRef    JSONContent `json:"sourceRef" gorm:"type:jsonb;not null"` // 追溯引用，不出网
	OccurredAt   time.Time   `json:"occurredAt" gorm:"not null"`           // 业务事件发生时间（排序第一因子）
	ExpiresAt    time.Time   `json:"expiresAt" gorm:"not null"`            // 读取与未读统计的有效期上界

	TenantID  uint            `json:"tenantId" gorm:"index;not null;default:1"`
	CreatedAt kernel.JSONTime `json:"createdAt"`
}

func (*Message) TableName() string { return "notification_messages" }

// MemberInbox 站内信扇出与已读状态：所有查询/更新必须显式携带
// tenant_id + member_id 双条件（SEC-NOTIFICATION-* 防横向越权）。
type MemberInbox struct {
	ID           uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	MessageID    uint            `json:"messageId" gorm:"not null"`
	MemberID     uint            `json:"memberId" gorm:"not null"`
	CategoryCode string          `json:"categoryCode" gorm:"size:64;not null"` // 扇出时固化的不可变冗余
	OccurredAt   time.Time       `json:"occurredAt" gorm:"not null"`           // 扇出时固化，稳定排序
	ReadAt       *time.Time      `json:"readAt"`                               // NULL=未读；重复标记不改写首次时间
	CreatedAt    kernel.JSONTime `json:"createdAt"`

	TenantID uint `json:"tenantId" gorm:"index;not null;default:1"`
}

func (*MemberInbox) TableName() string { return "notification_member_inboxes" }

// InboxRow 收件箱列表行（收件箱 JOIN 逻辑消息的读取投影）
type InboxRow struct {
	InboxID      uint
	ReadAt       *time.Time
	CategoryCode string
	EventCode    string
	Severity     string
	Title        string
	Content      string
	Action       JSONContent
	OccurredAt   time.Time
	CreatedAt    time.Time
}

// InboxQuery 收件箱游标分页查询条件（categoryId 必填；游标固化 occurred_at+id）
type InboxQuery struct {
	CategoryID string
	EventCode  string
	UnreadOnly bool
	Cursor     string // 不透明游标，空表示首页
	Limit      int
}

// CursorPayload 游标载荷：(occurred_at 纳秒, 收件箱 id)，排序 occurred_at DESC、id DESC
type CursorPayload struct {
	OccurredAtNano int64 `json:"o"`
	InboxID        uint  `json:"i"`
}

// InboxItemView 消息列表项（出网）
type InboxItemView struct {
	ID         uint            `json:"id"`         // 收件箱行 ID（已读定位值）
	CategoryID string          `json:"categoryId"` // 分类码
	EventCode  string          `json:"eventCode"`
	EventLabel string          `json:"eventLabel"` // 事件注册表中文展示名
	Severity   string          `json:"severity"`
	Title      string          `json:"title"`
	Content    string          `json:"content"` // 纯文本展示快照
	CreatedAt  kernel.JSONTime `json:"createdAt"`
	Read       bool            `json:"read"`
	ReadAt     kernel.JSONTime `json:"readAt"` // 零值输出空串
	Action     json.RawMessage `json:"action"` // 无动作时输出 {}
}

// InboxPageView 消息列表分页响应
type InboxPageView struct {
	Items           []InboxItemView `json:"items"`
	NextCursor      string          `json:"nextCursor"`
	HasMore         bool            `json:"hasMore"`
	RetentionMonths int             `json:"retentionMonths"` // 保留期（月），前端「保存最近 N 个月」提示
	ServerTime      kernel.JSONTime `json:"serverTime"`      // 服务端当前时间，批量已读 through 口令来源
}

// CategoryUnreadView 未读摘要中的分类计数
type CategoryUnreadView struct {
	CategoryID  string `json:"categoryId"`
	UnreadCount int64  `json:"unreadCount"`
}

// UnreadSummaryView 顶栏未读摘要（只返回未读数大于 0 的分类）
type UnreadSummaryView struct {
	UnreadTotal int64                `json:"unreadTotal"`
	Categories  []CategoryUnreadView `json:"categories"`
	GeneratedAt kernel.JSONTime      `json:"generatedAt"`
}

// ReadAllRequest 批量已读请求：categoryId 必填；through 为本次列表响应的
// serverTime，服务端只标记 occurred_at <= through 的旧消息，不误伤新到达
type ReadAllRequest struct {
	CategoryID string `json:"categoryId" binding:"required"`
	EventCode  string `json:"eventCode"`
	Through    string `json:"through"` // yyyy-MM-dd HH:mm:ss（东八区），空表示不设上界
}

// JSONContent JSONB 原样载体：动作/引用不经 map 往返，原样字节存取
type JSONContent json.RawMessage

// Value 实现 driver.Valuer：空值落 '{}'，其余字符串形态交给 pgx 写 jsonb
func (j JSONContent) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	return string(j), nil
}

// Scan 实现 sql.Scanner：接受 pgx 返回的 []byte / string
func (j *JSONContent) Scan(value interface{}) error {
	switch data := value.(type) {
	case nil:
		*j = JSONContent("null")
	case []byte:
		*j = append((*j)[:0], data...)
	case string:
		*j = JSONContent(data)
	default:
		return fmt.Errorf("notification: cannot scan JSONContent from %T", value)
	}
	return nil
}

// MarshalJSON 原样出网（json.RawMessage 语义）
func (j JSONContent) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON 原样收网
func (j *JSONContent) UnmarshalJSON(data []byte) error {
	*j = append((*j)[:0], data...)
	return nil
}
