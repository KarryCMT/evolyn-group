package repository

import (
	"context"
	"errors"
	"strings"

	"evolyn/internal/platform/productlog/model"

	"gorm.io/gorm"
)

// 显示名兜底链（JOIN tn_users/accounts）：写时快照 → 成员昵称 → 账号昵称 →
// 登录名。JOIN 限定同租户有效成员（软删/跨租户成员不参与兜底，保留快照值）
const productLogSelect = `
tn_audit_logs.id, tn_audit_logs.member_id, tn_audit_logs.event_code, tn_audit_logs.category_code,
tn_audit_logs.actor_name_snapshot, tn_audit_logs.target_name_snapshot, tn_audit_logs.summary,
tn_audit_logs.application_name_snapshot, tn_audit_logs.ip, tn_audit_logs.created_at,` +
	"\n" + `COALESCE(
	NULLIF(tn_audit_logs.actor_name_snapshot, ''),
	NULLIF(u.nickname, ''),
	NULLIF(a.nickname, ''),
	a.name, '') AS display_name`

const productLogJoins = `
LEFT JOIN tn_users u ON u.id = tn_audit_logs.member_id AND u.tenant_id = tn_audit_logs.tenant_id AND u.deleted_at IS NULL
LEFT JOIN pf_accounts a ON a.id = u.account_id AND a.deleted_at IS NULL`

// likePattern 关键词 LIKE 模式：转义 %/_ 通配符后包裹（关键词按字面匹配）
func likePattern(keyword string) string {
	escaped := strings.ReplaceAll(keyword, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "%", "\\%")
	escaped = strings.ReplaceAll(escaped, "_", "\\_")
	return "%" + escaped + "%"
}

// applyProductLogFilter 产品日志公共过滤（计数与列表共用，避免 JOIN 参与
// count）：分类白名单必填——产品日志只读产品分类，企业治理行为不进本目录
func applyProductLogFilter(q *gorm.DB, f ProductLogFilter) *gorm.DB {
	q = q.Where("tn_audit_logs.tenant_id = ?", f.TenantID).
		Where("tn_audit_logs.category_code IN ?", f.Categories)
	if f.MemberID != 0 {
		q = q.Where("tn_audit_logs.member_id = ?", f.MemberID)
	}
	if f.CategoryCode != "" {
		q = q.Where("tn_audit_logs.category_code = ?", f.CategoryCode)
	}
	if f.EventCode != "" {
		q = q.Where("tn_audit_logs.event_code = ?", f.EventCode)
	}
	if f.ApplicationID != 0 {
		q = q.Where("tn_audit_logs.application_id = ?", f.ApplicationID)
	}
	if f.Keyword != "" {
		// 关键词仅匹配受控展示字段（应用名快照/操作对象/摘要），不查
		// before/after 原始快照
		pattern := likePattern(f.Keyword)
		q = q.Where(
			"tn_audit_logs.application_name_snapshot ILIKE ? OR tn_audit_logs.target_name_snapshot ILIKE ? OR tn_audit_logs.summary ILIKE ?",
			pattern, pattern, pattern,
		)
	}
	if !f.Range.Start.IsZero() {
		q = q.Where("tn_audit_logs.created_at >= ?", f.Range.Start)
	}
	if !f.Range.EndExcl.IsZero() {
		q = q.Where("tn_audit_logs.created_at < ?", f.Range.EndExcl)
	}
	return q
}

func (r *productLogRepository) ListProductLogs(ctx context.Context, f ProductLogFilter, offset, limit int) ([]model.ProductLogRow, int64, error) {
	var total int64
	if err := applyProductLogFilter(r.db.WithContext(ctx).Table("tn_audit_logs"), f).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]model.ProductLogRow, 0, limit)
	if err := applyProductLogFilter(
		r.db.WithContext(ctx).Table("tn_audit_logs").Select(productLogSelect).Joins(productLogJoins), f,
	).Order("tn_audit_logs.created_at DESC, tn_audit_logs.id DESC").
		Offset(offset).Limit(limit).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// ScanProductLogs keyset 游标扫描：(created_at, id) 严格小于游标值的下一批，
// 行比较谓词可完整命中 (tenant_id, category_code, created_at DESC, id DESC)
// 或 (tenant_id, application_id, created_at DESC, id DESC) 索引
func (r *productLogRepository) ScanProductLogs(ctx context.Context, f ProductLogFilter, batch int, fn func(rows []model.ProductLogRow) error) error {
	if batch <= 0 {
		batch = 500
	}
	var (
		lastCreatedAt interface{}
		lastID        uint
	)
	for {
		q := applyProductLogFilter(
			r.db.WithContext(ctx).Table("tn_audit_logs").Select(productLogSelect).Joins(productLogJoins), f,
		)
		if lastID != 0 {
			q = q.Where("(tn_audit_logs.created_at, tn_audit_logs.id) < (?, ?)", lastCreatedAt, lastID)
		}
		rows := make([]model.ProductLogRow, 0, batch)
		if err := q.Order("tn_audit_logs.created_at DESC, tn_audit_logs.id DESC").Limit(batch).Scan(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		if err := fn(rows); err != nil {
			return err
		}
		if len(rows) < batch {
			return nil
		}
		lastCreatedAt, lastID = rows[len(rows)-1].CreatedAt, rows[len(rows)-1].ID
	}
}

// CreateExportTask 落导出任务（显式字段写入，无租户 Callback 依赖）
func (r *productLogRepository) CreateExportTask(ctx context.Context, task *model.ExportTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// GetExportTask 租户归属复核后的任务读取：跨租户 ID 与不存在同义
// （ErrRecordNotFound，调用方映射业务错误）
func (r *productLogRepository) GetExportTask(ctx context.Context, tenantID, id uint) (*model.ExportTask, error) {
	task := new(model.ExportTask)
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return task, nil
}

// UpdateExportTask 按 ID 更新任务结果字段（生成结果/状态/过期时间）
func (r *productLogRepository) UpdateExportTask(ctx context.Context, task *model.ExportTask) error {
	return r.db.WithContext(ctx).Model(&model.ExportTask{}).
		Where("id = ? AND tenant_id = ?", task.ID, task.TenantID).
		Updates(map[string]interface{}{
			"status":     task.Status,
			"file_name":  task.FileName,
			"file_data":  task.FileData,
			"total":      task.Total,
			"expires_at": task.ExpiresAt,
		}).Error
}

func (r *productLogRepository) Migrate() error {
	return r.db.AutoMigrate(&model.ExportTask{})
}
