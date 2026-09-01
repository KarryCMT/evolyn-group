package repository

import (
	"context"
	"errors"
	"time"

	"evolyn/internal/platform/enterpriselog/model"

	"gorm.io/gorm"
)

// 显示名兜底链（JOIN tn_users/accounts）：写时快照 → 成员昵称 → 账号昵称 →
// 登录名。JOIN 限定同租户有效成员（软删/跨租户成员不参与兜底，保留快照值）
const (
	loginLogSelect = `
pf_login_logs.id, pf_login_logs.member_id, pf_login_logs.client, pf_login_logs.ip,
pf_login_logs.location, pf_login_logs.created_at, pf_login_logs.actor_name_snapshot,` +
		"\n" + `COALESCE(
	NULLIF(pf_login_logs.actor_name_snapshot, ''),
	NULLIF(u.nickname, ''),
	NULLIF(a.nickname, ''),
	a.name, '') AS display_name`

	loginLogJoins = `
LEFT JOIN tn_users u ON u.id = pf_login_logs.member_id AND u.tenant_id = pf_login_logs.tenant_id AND u.deleted_at IS NULL
LEFT JOIN pf_accounts a ON a.id = u.account_id AND a.deleted_at IS NULL`

	auditLogSelect = `
tn_audit_logs.id, tn_audit_logs.member_id, tn_audit_logs.event_code, tn_audit_logs.category_code,
tn_audit_logs.summary, tn_audit_logs.ip, tn_audit_logs.created_at, tn_audit_logs.actor_name_snapshot,` +
		"\n" + `COALESCE(
	NULLIF(tn_audit_logs.actor_name_snapshot, ''),
	NULLIF(u.nickname, ''),
	NULLIF(a.nickname, ''),
	a.name, '') AS display_name`

	auditLogJoins = `
LEFT JOIN tn_users u ON u.id = tn_audit_logs.member_id AND u.tenant_id = tn_audit_logs.tenant_id AND u.deleted_at IS NULL
LEFT JOIN pf_accounts a ON a.id = u.account_id AND a.deleted_at IS NULL`
)

// applyLoginLogFilter 登录日志公共过滤（计数与列表共用，避免 JOIN 参与 count）
func applyLoginLogFilter(q *gorm.DB, f LoginLogFilter) *gorm.DB {
	q = q.Where("pf_login_logs.tenant_id = ?", f.TenantID)
	if f.MemberID != 0 {
		q = q.Where("pf_login_logs.member_id = ?", f.MemberID)
	}
	if !f.Range.Start.IsZero() {
		q = q.Where("pf_login_logs.created_at >= ?", f.Range.Start)
	}
	if !f.Range.EndExcl.IsZero() {
		q = q.Where("pf_login_logs.created_at < ?", f.Range.EndExcl)
	}
	return q
}

// applyAuditLogFilter 操作日志公共过滤
func applyAuditLogFilter(q *gorm.DB, f AuditLogFilter) *gorm.DB {
	q = q.Where("tn_audit_logs.tenant_id = ?", f.TenantID)
	if f.MemberID != 0 {
		q = q.Where("tn_audit_logs.member_id = ?", f.MemberID)
	}
	if f.CategoryCode != "" {
		q = q.Where("tn_audit_logs.category_code = ?", f.CategoryCode)
	}
	if f.EventCode != "" {
		q = q.Where("tn_audit_logs.event_code = ?", f.EventCode)
	}
	if !f.Range.Start.IsZero() {
		q = q.Where("tn_audit_logs.created_at >= ?", f.Range.Start)
	}
	if !f.Range.EndExcl.IsZero() {
		q = q.Where("tn_audit_logs.created_at < ?", f.Range.EndExcl)
	}
	return q
}

func (r *enterpriseLogRepository) ListLoginLogs(ctx context.Context, f LoginLogFilter, offset, limit int) ([]model.LoginLogRow, int64, error) {
	var total int64
	if err := applyLoginLogFilter(r.db.WithContext(ctx).Table("pf_login_logs"), f).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]model.LoginLogRow, 0, limit)
	if err := applyLoginLogFilter(
		r.db.WithContext(ctx).Table("pf_login_logs").Select(loginLogSelect).Joins(loginLogJoins), f,
	).Order("pf_login_logs.created_at DESC, pf_login_logs.id DESC").
		Offset(offset).Limit(limit).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// ScanLoginLogs keyset 游标扫描：(created_at, id) 严格小于游标值的下一批，
// 行比较谓词可完整命中 (tenant_id, created_at DESC, id DESC) 索引
func (r *enterpriseLogRepository) ScanLoginLogs(ctx context.Context, f LoginLogFilter, batch int, fn func(rows []model.LoginLogRow) error) error {
	if batch <= 0 {
		batch = 500
	}
	var (
		lastCreatedAt time.Time
		lastID        uint
	)
	for {
		q := applyLoginLogFilter(
			r.db.WithContext(ctx).Table("pf_login_logs").Select(loginLogSelect).Joins(loginLogJoins), f,
		)
		if lastID != 0 {
			q = q.Where("(pf_login_logs.created_at, pf_login_logs.id) < (?, ?)", lastCreatedAt, lastID)
		}
		rows := make([]model.LoginLogRow, 0, batch)
		if err := q.Order("pf_login_logs.created_at DESC, pf_login_logs.id DESC").Limit(batch).Scan(&rows).Error; err != nil {
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

func (r *enterpriseLogRepository) ListAuditLogs(ctx context.Context, f AuditLogFilter, offset, limit int) ([]model.AuditLogRow, int64, error) {
	var total int64
	if err := applyAuditLogFilter(r.db.WithContext(ctx).Table("tn_audit_logs"), f).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]model.AuditLogRow, 0, limit)
	if err := applyAuditLogFilter(
		r.db.WithContext(ctx).Table("tn_audit_logs").Select(auditLogSelect).Joins(auditLogJoins), f,
	).Order("tn_audit_logs.created_at DESC, tn_audit_logs.id DESC").
		Offset(offset).Limit(limit).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *enterpriseLogRepository) ScanAuditLogs(ctx context.Context, f AuditLogFilter, batch int, fn func(rows []model.AuditLogRow) error) error {
	if batch <= 0 {
		batch = 500
	}
	var (
		lastCreatedAt time.Time
		lastID        uint
	)
	for {
		q := applyAuditLogFilter(
			r.db.WithContext(ctx).Table("tn_audit_logs").Select(auditLogSelect).Joins(auditLogJoins), f,
		)
		if lastID != 0 {
			q = q.Where("(tn_audit_logs.created_at, tn_audit_logs.id) < (?, ?)", lastCreatedAt, lastID)
		}
		rows := make([]model.AuditLogRow, 0, batch)
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
func (r *enterpriseLogRepository) CreateExportTask(ctx context.Context, task *model.ExportTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// GetExportTask 租户归属复核后的任务读取：跨租户 ID 与不存在同义
// （ErrRecordNotFound，调用方映射业务错误）
func (r *enterpriseLogRepository) GetExportTask(ctx context.Context, tenantID, id uint) (*model.ExportTask, error) {
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
func (r *enterpriseLogRepository) UpdateExportTask(ctx context.Context, task *model.ExportTask) error {
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

func (r *enterpriseLogRepository) Migrate() error {
	return r.db.AutoMigrate(&model.ExportTask{})
}
