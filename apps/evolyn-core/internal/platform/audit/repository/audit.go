package repository

import (
	"context"

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
// 调用方填充，平台级操作为 0），普通 WithContext 即可
func (r *auditRepository) Create(ctx context.Context, log *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
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
