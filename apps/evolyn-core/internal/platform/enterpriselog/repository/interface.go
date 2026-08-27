// Package repository 企业日志域数据访问（000036）：login_logs / audit_logs
// 的只读查询（显式租户条件——两表为平台级表，无租户 Callback 语义）与
// enterprise_log_exports 任务存取。列表统一按 (created_at DESC, id DESC)
// 稳定排序；导出扫描走时间+ID 的 keyset 游标，避免深分页扫描
package repository

import (
	"context"
	"time"

	"evolyn/internal/platform/enterpriselog/model"

	"gorm.io/gorm"
)

// LogTimeRange 时间过滤（东八区自然日闭区间由服务层换算为半开区间）；
// 零值跳过对应边界
type LogTimeRange struct {
	Start   time.Time
	EndExcl time.Time
}

// LoginLogFilter 登录日志查询条件
type LoginLogFilter struct {
	TenantID uint
	MemberID uint
	Range    LogTimeRange
}

// AuditLogFilter 操作日志查询条件
type AuditLogFilter struct {
	TenantID     uint
	MemberID     uint
	CategoryCode string
	EventCode    string
	Range        LogTimeRange
}

// Repository 企业日志域仓储
type Repository interface {
	// ListLoginLogs 登录日志分页查询（稳定排序 created_at DESC, id DESC）；
	// 返回行带 JOIN 兜底的当前成员显示名（存量历史行快照缺失时用）
	ListLoginLogs(ctx context.Context, f LoginLogFilter, offset, limit int) ([]model.LoginLogRow, int64, error)
	// ScanLoginLogs 登录日志导出扫描：按 (created_at,id) keyset 游标分批回调，
	// 单批上限 batch；回调返回错误即中止
	ScanLoginLogs(ctx context.Context, f LoginLogFilter, batch int, fn func(rows []model.LoginLogRow) error) error
	// ListAuditLogs 操作日志分页查询（同登录日志口径与排序）
	ListAuditLogs(ctx context.Context, f AuditLogFilter, offset, limit int) ([]model.AuditLogRow, int64, error)
	// ScanAuditLogs 操作日志导出扫描（keyset 游标）
	ScanAuditLogs(ctx context.Context, f AuditLogFilter, batch int, fn func(rows []model.AuditLogRow) error) error
	// CreateExportTask 落一条导出任务（pending）
	CreateExportTask(ctx context.Context, task *model.ExportTask) error
	// GetExportTask 按 ID 取当前租户的任务；不存在/跨租户返回 ErrRecordNotFound
	GetExportTask(ctx context.Context, tenantID, id uint) (*model.ExportTask, error)
	// UpdateExportTask 更新任务（生成结果/状态/过期时间）
	UpdateExportTask(ctx context.Context, task *model.ExportTask) error
	// Migrate dev AutoMigrate（生产以 SQL 迁移为准）
	Migrate() error
}

type enterpriseLogRepository struct {
	db *gorm.DB
}

// NewRepository 企业日志域仓储工厂（ADR-007 域模块化）
func NewRepository(db *gorm.DB) Repository {
	return &enterpriseLogRepository{db: db}
}
