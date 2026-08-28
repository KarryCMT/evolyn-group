package repository

import (
	"context"
	"time"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/notification/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// inboxInsertBatch 收件箱扇出单批上限：大租户分批插入，避免长事务与大报文
const inboxInsertBatch = 500

type messageRepository struct {
	db *gorm.DB
}

// NewMessageRepository 消息与收件箱仓储工厂（ADR-007 域模块化）
func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *messageRepository) Migrate() error {
	return r.db.AutoMigrate(&model.Message{}, &model.MemberInbox{})
}

// InsertIgnoreConflict 物化逻辑消息：冲突（Worker 重放）时按幂等键回查行 ID。
// 不依赖 RETURNING，兼容 GORM 通用路径。
func (r *messageRepository) InsertIgnoreConflict(ctx context.Context, msg *model.Message) (uint, error) {
	result := r.withContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(msg)
	if result.Error != nil {
		return 0, result.Error
	}
	if result.RowsAffected > 0 {
		return msg.ID, nil
	}
	// (tenant_id, event_id) 已存在：物化曾被中断，回查行 ID 继续补扇出
	var existing model.Message
	if err := r.withContext(ctx).
		Where("tenant_id = ? AND event_id = ?", msg.TenantID, msg.EventID).
		Take(&existing).Error; err != nil {
		return 0, err
	}
	return existing.ID, nil
}

// InsertInboxesIgnoreConflict 扇出收件箱：单条 SQL 批插 + 冲突吞并，超过单批
// 上限自动分段（文档 12 章：不在业务事务内对成千上万成员循环插入）
func (r *messageRepository) InsertInboxesIgnoreConflict(
	ctx context.Context, tenantID, messageID uint, categoryCode string, occurredAt time.Time, memberIDs []uint,
) error {
	for start := 0; start < len(memberIDs); start += inboxInsertBatch {
		end := start + inboxInsertBatch
		if end > len(memberIDs) {
			end = len(memberIDs)
		}
		rows := make([]model.MemberInbox, 0, end-start)
		for _, memberID := range memberIDs[start:end] {
			rows = append(rows, model.MemberInbox{
				TenantID:     tenantID,
				MessageID:    messageID,
				MemberID:     memberID,
				CategoryCode: categoryCode,
				OccurredAt:   occurredAt,
			})
		}
		if err := r.withContext(ctx).
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&rows).Error; err != nil {
			return err
		}
	}
	return nil
}

// ListInbox 成员分类游标分页：排序 occurred_at DESC、id DESC；游标以 PG 行值
// 比较 (occurred_at, id) < (?, ?) 表达，同时间多条不重不漏；limit+1 探测 hasMore
func (r *messageRepository) ListInbox(
	ctx context.Context, tenantID, memberID uint, q model.InboxQuery,
	cursor model.CursorPayload, hasCursor bool,
) ([]model.InboxRow, bool, error) {
	query := r.withContext(ctx).
		Table("notification_member_inboxes AS i").
		Select(`
			i.id AS inbox_id,
			i.read_at,
			i.category_code,
			i.occurred_at,
			m.event_code,
			m.severity,
			m.title,
			m.content,
			m.action,
			m.created_at`).
		Joins("JOIN notification_messages m ON m.id = i.message_id").
		Where("i.tenant_id = ? AND i.member_id = ? AND i.category_code = ?", tenantID, memberID, q.CategoryID).
		Where("m.expires_at > now()")
	if q.EventCode != "" {
		query = query.Where("m.event_code = ?", q.EventCode)
	}
	if q.UnreadOnly {
		query = query.Where("i.read_at IS NULL")
	}
	if hasCursor {
		cursorTime := time.Unix(0, cursor.OccurredAtNano)
		query = query.Where("(i.occurred_at, i.id) < (?, ?)", cursorTime, cursor.InboxID)
	}
	rows := make([]model.InboxRow, 0, q.Limit+1)
	if err := query.
		Order("i.occurred_at DESC, i.id DESC").
		Limit(q.Limit + 1).
		Scan(&rows).Error; err != nil {
		return nil, false, err
	}
	if len(rows) > q.Limit {
		return rows[:q.Limit], true, nil
	}
	return rows, false, nil
}

// CountUnreadByCategory 未读分类统计：SQL 内排除过期消息，不依赖清理任务准时性
func (r *messageRepository) CountUnreadByCategory(
	ctx context.Context, tenantID, memberID uint,
) ([]model.CategoryUnreadView, error) {
	rows := make([]model.CategoryUnreadView, 0)
	err := r.withContext(ctx).
		Table("notification_member_inboxes AS i").
		Select("i.category_code AS category_id, COUNT(*) AS unread_count").
		Joins("JOIN notification_messages m ON m.id = i.message_id").
		Where("i.tenant_id = ? AND i.member_id = ? AND i.read_at IS NULL", tenantID, memberID).
		Where("m.expires_at > now()").
		Group("i.category_code").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// GetInboxReadState 单条收件箱存在性与已读时间：未读行返回 (nil, true, nil)，
// 不存在返回 (nil, false, nil)——二者语义不同，不能混同
func (r *messageRepository) GetInboxReadState(
	ctx context.Context, tenantID, memberID, inboxID uint,
) (*time.Time, bool, error) {
	var row struct {
		ReadAt *time.Time
	}
	err := r.withContext(ctx).
		Table("notification_member_inboxes").
		Select("read_at").
		Where("tenant_id = ? AND member_id = ? AND id = ?", tenantID, memberID, inboxID).
		Take(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	return row.ReadAt, true, nil
}

// MarkInboxRead 幂等单条已读：仅未读行命中；已读行 0 影响但语义为成功
// （首次 read_at 不被改写，Service 依据 GetInboxReadState 区分不存在）
func (r *messageRepository) MarkInboxRead(ctx context.Context, tenantID, memberID, inboxID uint) (int64, error) {
	result := r.withContext(ctx).
		Model(&model.MemberInbox{}).
		Where("tenant_id = ? AND member_id = ? AND id = ? AND read_at IS NULL", tenantID, memberID, inboxID).
		Update("read_at", time.Now())
	return result.RowsAffected, result.Error
}

// MarkCategoryRead 批量已读：分类 + 可选事件 + through 上界（仅旧消息），
// 不误伤操作同时新到达的消息
func (r *messageRepository) MarkCategoryRead(
	ctx context.Context, tenantID, memberID uint, categoryCode, eventCode string, through time.Time, hasThrough bool,
) (int64, error) {
	query := r.withContext(ctx).
		Model(&model.MemberInbox{}).
		Where("tenant_id = ? AND member_id = ? AND category_code = ? AND read_at IS NULL", tenantID, memberID, categoryCode)
	if eventCode != "" {
		query = query.Where(
			"message_id IN (SELECT id FROM notification_messages WHERE event_code = ?)", eventCode)
	}
	if hasThrough {
		query = query.Where("occurred_at <= ?", through)
	}
	result := query.Update("read_at", time.Now())
	return result.RowsAffected, result.Error
}

// DeleteExpiredInboxes 保留清理第一步：子查询限定本批收件箱行再删（PG DELETE
// 不支持 LIMIT）；返回本批实际删除行数，0 表示本轮清理完成
func (r *messageRepository) DeleteExpiredInboxes(ctx context.Context, batch int) (int64, error) {
	result := r.withContext(ctx).Exec(`
		DELETE FROM notification_member_inboxes
		WHERE id IN (
		    SELECT i.id
		    FROM notification_member_inboxes i
		    JOIN notification_messages m ON m.id = i.message_id
		    WHERE m.expires_at <= now()
		    LIMIT ?
		)`, batch)
	return result.RowsAffected, result.Error
}

// DeleteOrphanExpiredMessages 保留清理第二步：删已无收件箱引用的过期消息
func (r *messageRepository) DeleteOrphanExpiredMessages(ctx context.Context, batch int) (int64, error) {
	result := r.withContext(ctx).Exec(`
		DELETE FROM notification_messages
		WHERE id IN (
		    SELECT m.id
		    FROM notification_messages m
		    WHERE m.expires_at <= now()
		      AND NOT EXISTS (
		          SELECT 1 FROM notification_member_inboxes i WHERE i.message_id = m.id
		      )
		    LIMIT ?
		)`, batch)
	return result.RowsAffected, result.Error
}
