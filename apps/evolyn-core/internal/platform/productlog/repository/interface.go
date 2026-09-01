// Package repository 产品日志域数据访问（000064）：tn_audit_logs 的产品
// 分类只读查询（显式租户条件——平台级表，无租户 Callback 语义）与
// tn_product_log_exports 任务存取。列表统一按 (created_at DESC, id DESC)
// 稳定排序；导出扫描走时间+ID 的 keyset 游标，避免深分页扫描。
// 应用维度查询始终以前置 tenant_id 为首列（000064 索引口径），禁止仅凭
// application_id 查询审计日志
package repository

import (
	"context"
	"time"

	"evolyn/internal/platform/productlog/model"

	"gorm.io/gorm"
)

// LogTimeRange 时间过滤（东八区自然日闭区间由服务层换算为半开区间）；
// 零值跳过对应边界
type LogTimeRange struct {
	Start   time.Time
	EndExcl time.Time
}

// ProductLogFilter 产品日志查询条件：Categories 为产品分类白名单（必填，
// 与企业日志查询范围互斥），由服务层经 audit 注册表下发
type ProductLogFilter struct {
	TenantID      uint
	MemberID      uint
	CategoryCode  string
	EventCode     string
	ApplicationID uint
	Keyword       string
	Range         LogTimeRange
	Categories    []string
}

// Repository 产品日志域仓储
type Repository interface {
	// ListProductLogs 产品日志分页查询（稳定排序 created_at DESC, id DESC）；
	// 返回行带 JOIN 兜底的当前成员显示名（存量历史行快照缺失时用）
	ListProductLogs(ctx context.Context, f ProductLogFilter, offset, limit int) ([]model.ProductLogRow, int64, error)
	// ScanProductLogs 导出扫描：按 (created_at,id) keyset 游标分批回调，
	// 单批上限 batch；回调返回错误即中止
	ScanProductLogs(ctx context.Context, f ProductLogFilter, batch int, fn func(rows []model.ProductLogRow) error) error
	// CreateExportTask 落一条导出任务（pending）
	CreateExportTask(ctx context.Context, task *model.ExportTask) error
	// GetExportTask 按 ID 取当前租户的任务；不存在/跨租户返回 ErrRecordNotFound
	GetExportTask(ctx context.Context, tenantID, id uint) (*model.ExportTask, error)
	// UpdateExportTask 更新任务（生成结果/状态/过期时间）
	UpdateExportTask(ctx context.Context, task *model.ExportTask) error
	// Migrate dev AutoMigrate（生产以 SQL 迁移为准）
	Migrate() error
}

type productLogRepository struct {
	db *gorm.DB
}

// NewRepository 产品日志域仓储工厂（ADR-007 域模块化）
func NewRepository(db *gorm.DB) Repository {
	return &productLogRepository{db: db}
}
