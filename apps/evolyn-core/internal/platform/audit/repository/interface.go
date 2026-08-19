package repository

import (
	"context"

	"evolyn/internal/platform/audit/model"
)

// AuditRepository 审计日志数据访问：追加写 + 只读查询
type AuditRepository interface {
	Create(ctx context.Context, log *model.AuditLog) error
	// List 按租户倒序分页（运营/合规查询用；一期无 API 暴露）
	List(ctx context.Context, tenantID uint, offset, limit int) ([]model.AuditLog, error)
	Migrate() error
}
