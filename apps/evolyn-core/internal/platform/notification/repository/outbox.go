package repository

import (
	"context"
	"time"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/notification/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type outboxRepository struct {
	db *gorm.DB
}

// NewOutboxRepository 事务 Outbox 仓储工厂（ADR-007 域模块化）
func NewOutboxRepository(db *gorm.DB) OutboxRepository {
	return &outboxRepository{db: db}
}

func (r *outboxRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *outboxRepository) Migrate() error {
	return r.db.AutoMigrate(&model.OutboxEvent{})
}

// Insert 业务事务内写入事件：event_id 全局唯一冲突吞并（幂等键重放不产生
// 第二条待处理事件）；RowsAffected==0 时调用方可记日志，不需要失败
func (r *outboxRepository) Insert(ctx context.Context, event *model.OutboxEvent) error {
	return r.withContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "event_id"}},
			DoNothing: true,
		}).Create(event).Error
}

// ClaimBatch 领取待处理事件：FOR UPDATE SKIP LOCKED 保证多 Worker 并发下
// 同一批次不重复领取；锁随调用方事务提交/回滚释放（回滚即回到 pending）。
// 到期判定带 1 秒宽限：应用节点与数据库时钟可能存在毫秒级偏移，事件至多
// 提前 1 秒领取（重试退避的粒度远大于此，幂等由行锁与 event_id 唯一约束保证）。
func (r *outboxRepository) ClaimBatch(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	events := make([]model.OutboxEvent, 0, limit)
	err := r.withContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("status = ? AND next_attempt_at <= now() + interval '1 second'", model.OutboxPending).
		Order("id ASC").
		Limit(limit).
		Find(&events).Error
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (r *outboxRepository) MarkDone(ctx context.Context, id uint) error {
	return r.withContext(ctx).
		Model(&model.OutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       model.OutboxDone,
			"processed_at": time.Now(),
		}).Error
}

func (r *outboxRepository) MarkFailed(ctx context.Context, id uint, errorCode string) error {
	return r.withContext(ctx).
		Model(&model.OutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":          model.OutboxFailed,
			"last_error_code": errorCode,
			"processed_at":    time.Now(),
		}).Error
}

func (r *outboxRepository) MarkRetry(ctx context.Context, id uint, errorCode string, nextAttemptAt time.Time) error {
	return r.withContext(ctx).
		Model(&model.OutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"attempt_count":   gorm.Expr("attempt_count + 1"),
			"last_error_code": errorCode,
			"next_attempt_at": nextAttemptAt,
		}).Error
}
