package repository

import (
	"context"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/audit/model"

	"gorm.io/gorm"
)

type auditRepository struct {
	db *gorm.DB
}

// NewRepository 审计域仓储工厂（ADR-007 域模块化）
func NewRepository(db *gorm.DB) AuditRepository {
	return &auditRepository{db: db}
}

// Create 审计落库：显式不含租户 Callback 语义（audit_logs 的 tenant_id 由
// 调用方填充，平台级操作为 0）。取连接统一走 ResolveDB：审计通常在业务事务
// 提交后独立写入（失败策略见 audit/service），若调用方仍在事务内则随事务提交
func (r *auditRepository) Create(ctx context.Context, log *model.AuditLog) error {
	return infrastructure.ResolveDB(ctx, r.db).Create(log).Error
}

func (r *auditRepository) List(ctx context.Context, tenantID uint, offset, limit int) ([]model.AuditLog, error) {
	logs := make([]model.AuditLog, 0)
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).
		Order("id DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *auditRepository) Migrate() error {
	return r.db.AutoMigrate(&model.AuditLog{})
}
