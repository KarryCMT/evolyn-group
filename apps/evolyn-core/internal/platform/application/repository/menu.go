package repository

import (
	"context"
	"encoding/json"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/application/model"

	"gorm.io/gorm"
)

type menuRepository struct {
	db *gorm.DB
}

// NewMenuRepository 菜单仓储工厂（ADR-007 域内小三层）
func NewMenuRepository(db *gorm.DB) MenuRepository {
	return &menuRepository{db: db}
}

// menuSnapshotRow GetSnapshot 单条 SQL 的扫描载体：entries 为节点 JSON
// 数组（snake_case 列名，经 menuEntryJSON 转模型）；Scan 目标不是 GORM
// model（Statement.Schema 为 nil），租户 Callback 不会注入条件——
// SQL 内已显式携带 tenant_id，这是刻意设计而非遗漏
type menuSnapshotRow struct {
	AppID              uint
	AppCode            string
	AppStatus          string
	AppProvisionStatus string
	MenuRevision       int64
	Entries            []byte
}

// menuEntryJSON to_jsonb(e) 输出的节点列（列名 snake_case），仅取模型
// 需要的字段，其余（config 等）首期出网不消费、留空即可
type menuEntryJSON struct {
	ID            uint    `json:"id"`
	ApplicationID uint    `json:"application_id"`
	Code          string  `json:"code"`
	ParentEntryID *uint   `json:"parent_entry_id"`
	EntryType     string  `json:"entry_type"`
	Name          string  `json:"name"`
	Icon          string  `json:"icon"`
	Color         string  `json:"color"`
	TargetType    *string `json:"target_type"`
	TargetID      *uint   `json:"target_id"`
	SortOrder     int64   `json:"sort_order"`
}

// GetSnapshot 以单条 SQL 同时读取应用行（含 menu_revision）与未软删菜单
// 节点：单语句即单快照，天然满足「修订号与节点必须同快照」的并发约束
// （方案 §5.2）。节点在库内即按 (sort_order, code) 聚合有序，Service 不再
// 依赖数据库以外的排序来源；无匹配行（跨租户 code/软删应用）时 AppID
// 为零值，向上返回 ErrRecordNotFound 与 GetByCode 同口径
func (r *menuRepository) GetSnapshot(ctx context.Context, tenantID uint, code string) (*MenuSnapshot, error) {
	const query = `
SELECT a.id AS app_id,
       a.code AS app_code,
       a.status AS app_status,
       a.provision_status AS app_provision_status,
       a.menu_revision AS menu_revision,
       COALESCE(
           jsonb_agg(to_jsonb(e) ORDER BY e.sort_order ASC, e.code ASC)
               FILTER (WHERE e.id IS NOT NULL),
           '[]'::jsonb
       ) AS entries
FROM applications a
LEFT JOIN application_menu_entries e
    ON e.application_id = a.id
   AND e.tenant_id = a.tenant_id
   AND e.deleted_at IS NULL
WHERE a.tenant_id = ?
  AND a.code = ?
  AND a.deleted_at IS NULL
GROUP BY a.id, a.code, a.status, a.provision_status, a.menu_revision`

	var row menuSnapshotRow
	if err := infrastructure.ResolveDB(ctx, r.db).Raw(query, tenantID, code).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.AppID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	var raw []menuEntryJSON
	if len(row.Entries) > 0 {
		if err := json.Unmarshal(row.Entries, &raw); err != nil {
			return nil, err
		}
	}

	entries := make([]model.MenuEntry, 0, len(raw))
	for _, e := range raw {
		entries = append(entries, model.MenuEntry{
			ID:            e.ID,
			TenantID:      tenantID,
			ApplicationID: e.ApplicationID,
			Code:          e.Code,
			ParentEntryID: e.ParentEntryID,
			EntryType:     e.EntryType,
			Name:          e.Name,
			Icon:          e.Icon,
			Color:         e.Color,
			TargetType:    e.TargetType,
			TargetID:      e.TargetID,
			SortOrder:     e.SortOrder,
		})
	}

	return &MenuSnapshot{
		ApplicationID:   row.AppID,
		ApplicationCode: row.AppCode,
		Status:          row.AppStatus,
		ProvisionStatus: row.AppProvisionStatus,
		MenuRevision:    row.MenuRevision,
		Entries:         entries,
	}, nil
}

// Migrate 开发/测试路径：AutoMigrate 建表 + 补齐 GORM 标签表达不了的
// 部分索引（与迁移链 000016 同名同构）；applications.menu_revision 列
// 由 Application 模型 AutoMigrate 自动补齐
func (r *menuRepository) Migrate() error {
	if err := r.db.AutoMigrate(&model.MenuEntry{}); err != nil {
		return err
	}
	statements := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_application_menu_entries_tenant_code
			ON application_menu_entries (tenant_id, code) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_application_menu_entries_app_parent_sort
			ON application_menu_entries (tenant_id, application_id, parent_entry_id, sort_order, code)
			WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_application_menu_entries_app_target
			ON application_menu_entries (tenant_id, application_id, target_type, target_id)
			WHERE deleted_at IS NULL AND target_id IS NOT NULL`,
	}
	for _, stmt := range statements {
		if err := r.db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

// compile-time 接口实现断言（损坏时在编译期暴露，而非运行期装配失败）
var _ MenuRepository = (*menuRepository)(nil)
