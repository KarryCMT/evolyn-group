// Package repository 消息中心域数据访问（ADR-007 域内小三层）：仅做持久化，
// 一律经 infrastructure.ResolveDB 取连接加入 ctx 传播事务，不向 Service 暴露 GORM。
// 收件箱读写全部显式携带 tenant_id + member_id 双条件（横向越权防线不依赖
// ctx 租户 Callback 单点）。
package repository

import (
	"context"
	"time"

	"evolyn/internal/platform/notification/model"
)

// MessageRepository 逻辑消息与成员收件箱仓储。
type MessageRepository interface {
	// InsertIgnoreConflict 物化逻辑消息：(tenant_id, event_id) 冲突时返回
	// 已存在行 ID——Worker 重放依赖该幂等语义继续补扇出
	InsertIgnoreConflict(ctx context.Context, msg *model.Message) (uint, error)
	// InsertInboxesIgnoreConflict 扇出收件箱：(tenant_id, message_id, member_id)
	// 冲突吞并；单批过大时内部分批插入，避免长事务与大报文
	InsertInboxesIgnoreConflict(
		ctx context.Context, tenantID, messageID uint, categoryCode string, occurredAt time.Time, memberIDs []uint,
	) error
	// ListInbox 成员分类游标分页：JOIN 逻辑消息并排除 expires_at <= now()；
	// limit+1 探测 hasMore；hasCursor=false 时忽略 cursor
	ListInbox(
		ctx context.Context, tenantID, memberID uint, q model.InboxQuery,
		cursor model.CursorPayload, hasCursor bool,
	) ([]model.InboxRow, bool, error)
	// CountUnreadByCategory 分类未读统计（排除过期消息，即使清理滞后也不计数）
	CountUnreadByCategory(ctx context.Context, tenantID, memberID uint) ([]model.CategoryUnreadView, error)
	// GetInboxReadState 单条收件箱存在性与已读时间；exists=false 表示该成员
	// 收件箱中不存在（未读行的已读时间为 nil，不能与不存在混同）
	GetInboxReadState(ctx context.Context, tenantID, memberID, inboxID uint) (readAt *time.Time, exists bool, err error)
	// MarkInboxRead 幂等单条已读（仅未读行），返回影响行数（0=已读或不存在）
	MarkInboxRead(ctx context.Context, tenantID, memberID, inboxID uint) (int64, error)
	// MarkCategoryRead 批量已读：当前分类（可选事件）且 occurred_at <= through
	// 的未读行置 read_at；返回影响行数
	MarkCategoryRead(
		ctx context.Context, tenantID, memberID uint, categoryCode, eventCode string, through time.Time, hasThrough bool,
	) (int64, error)
	// DeleteExpiredInboxes 保留清理第一步：分批硬删过期消息的收件箱行，返回本批行数
	DeleteExpiredInboxes(ctx context.Context, batch int) (int64, error)
	// DeleteOrphanExpiredMessages 保留清理第二步：分批硬删已无收件箱引用的过期消息
	DeleteOrphanExpiredMessages(ctx context.Context, batch int) (int64, error)
	// Migrate 开发/测试 AutoMigrate 路径（FIX-009：生产只走 SQL 迁移）
	Migrate() error
}

// SettingRepository 租户通知设置聚合仓储（聚合根/偏好/接收规则/自定义联系人）。
type SettingRepository interface {
	// EnsureSetting 加载聚合根；不存在时幂等创建（读取侧兜底，兼容存量租户）
	EnsureSetting(ctx context.Context, tenantID uint) (*model.Setting, error)
	// BumpRevision 以旧 revision 为条件原子递增；false=口令过期（Service 转 409）
	BumpRevision(ctx context.Context, settingID uint, fromRevision int64) (bool, error)
	// ListPreferences 租户全部偏好覆盖行
	ListPreferences(ctx context.Context, tenantID uint) ([]model.Preference, error)
	// UpsertPreference 按 (tenant_id, event_code) upsert 覆盖行
	UpsertPreference(ctx context.Context, pref *model.Preference) error
	// ListPreferenceRecipients 批量加载偏好接收规则（键=preference_id）
	ListPreferenceRecipients(ctx context.Context, tenantID uint, preferenceIDs []uint) (map[uint][]model.PreferenceRecipient, error)
	// ReplaceRecipients 全量替换某偏好的接收规则（同事务先删后插）
	ReplaceRecipients(ctx context.Context, tenantID, preferenceID uint, items []model.PreferenceRecipient) error
	// ListCustomRecipients 自定义提醒对象列表（未删除）
	ListCustomRecipients(ctx context.Context, tenantID uint) ([]model.CustomRecipient, error)
	// GetCustomRecipient 单个联系人（未删除；不存在/跨租户即 NotFound）
	GetCustomRecipient(ctx context.Context, tenantID, id uint) (*model.CustomRecipient, error)
	// CountCustomRecipients 有效联系人数（上限校验）
	CountCustomRecipients(ctx context.Context, tenantID uint) (int64, error)
	// InsertCustomRecipient 新增联系人
	InsertCustomRecipient(ctx context.Context, recipient *model.CustomRecipient) (*model.CustomRecipient, error)
	// SoftDeleteCustomRecipient 软删联系人（保留关联历史）
	SoftDeleteCustomRecipient(ctx context.Context, tenantID, id uint) error
	// FindRecipientUsage 引用该联系人的偏好事件码清单（删除前校验）
	FindRecipientUsage(ctx context.Context, tenantID, recipientID uint) ([]string, error)
	// Migrate 开发/测试 AutoMigrate 路径
	Migrate() error
}

// OutboxRepository 事务 Outbox 仓储。
type OutboxRepository interface {
	// Insert 业务事务内写入事件；event_id 全局唯一冲突吞并（调用方幂等键重放）
	Insert(ctx context.Context, event *model.OutboxEvent) error
	// ClaimBatch 领取待处理事件：FOR UPDATE SKIP LOCKED 小批，多 Worker 安全；
	// 调用 ctx 必须无租户上下文（跨租户领取，租户 Callback 不应过滤）
	ClaimBatch(ctx context.Context, limit int) ([]model.OutboxEvent, error)
	// MarkDone 物化完成（processed_at 落位）
	MarkDone(ctx context.Context, id uint) error
	// MarkFailed 永久失败（超过重试上限或不可恢复错误），processed_at 落位
	MarkFailed(ctx context.Context, id uint, errorCode string) error
	// MarkRetry 暂时失败：attempt_count+1 并按退避推迟下次领取
	MarkRetry(ctx context.Context, id uint, errorCode string, nextAttemptAt time.Time) error
	// Migrate 开发/测试 AutoMigrate 路径
	Migrate() error
}
