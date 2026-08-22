package repository

import (
	"context"
	"time"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/auth/loginlog/model"

	"gorm.io/gorm"
)

type loginLogRepository struct {
	db *gorm.DB
}

// NewRepository 登录日志域仓储工厂（ADR-007 域模块化）
func NewRepository(db *gorm.DB) LoginLogRepository {
	return &loginLogRepository{db: db}
}

// Create 登录日志落库：显式不含租户 Callback 语义（account_id 由调用方填充，
// 平台级查询不走租户过滤）。取连接统一走 ResolveDB：登录记录无外层事务，
// 正常随连接直写；若调用方处于事务 ctx 则随事务提交
func (r *loginLogRepository) Create(ctx context.Context, log *model.LoginLog) error {
	return infrastructure.ResolveDB(ctx, r.db).Create(log).Error
}

// ListByAccount 账号维度倒序分页 + 总数；时间过滤为零值跳过。
// 不经 ResolveDB 是刻意为之：查询只读且永不参与登录写事务
func (r *loginLogRepository) ListByAccount(ctx context.Context, accountID uint, start, endExcl time.Time, offset, limit int) ([]model.LoginLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.LoginLog{}).Where("account_id = ?", accountID)
	if !start.IsZero() {
		query = query.Where("created_at >= ?", start)
	}
	if !endExcl.IsZero() {
		query = query.Where("created_at < ?", endExcl)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	logs := make([]model.LoginLog, 0, limit)
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

func (r *loginLogRepository) Migrate() error {
	return r.db.AutoMigrate(&model.LoginLog{})
}
