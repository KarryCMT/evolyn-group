package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/application/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	Hidden        bool    `json:"hidden"`
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
			Hidden:        e.Hidden,
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

// ---- M2-资产-1：表单资产节点维护（全部在调用方事务内执行） ----

// newMenuEntryCode 生成菜单节点编码：menu_ + 16 位随机 hex。租户内唯一由
// uk_application_menu_entries_tenant_code 部分唯一索引兜底（软删释放）。
func newMenuEntryCode() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "menu_" + hex.EncodeToString(buf), nil
}

// MaxSortOrder 指定父节点（parentEntryID 为 nil 即根级）下最大排序值；
// 无节点返回 0。新增子节点取返回值 + 1024，与根级口径一致。
func (r *menuRepository) MaxSortOrder(ctx context.Context, applicationID uint, parentEntryID *uint) (int64, error) {
	var maxSort int64
	query := infrastructure.ResolveDB(ctx, r.db).Model(&model.MenuEntry{}).
		Where("application_id = ?", applicationID)
	if parentEntryID == nil {
		query = query.Where("parent_entry_id IS NULL")
	} else {
		query = query.Where("parent_entry_id = ?", *parentEntryID)
	}
	err := query.Select("COALESCE(MAX(sort_order), 0)").Scan(&maxSort).Error
	return maxSort, err
}

// FindByCode 按节点编码查未软删节点（挂载父分组定位）；不存在返回
// gorm.ErrRecordNotFound。租户过滤由 GORM 租户 Callback 注入。
func (r *menuRepository) FindByCode(ctx context.Context, applicationID uint, code string) (*model.MenuEntry, error) {
	var entry model.MenuEntry
	err := infrastructure.ResolveDB(ctx, r.db).
		Where("application_id = ? AND code = ?", applicationID, code).
		First(&entry).Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// CreateFormEntry 在表单创建事务内插入 form 资产节点（根级或指定分组下、
// target 指向表单 ID）；code 冲突由唯一索引兜底，随机空间下可忽略。
func (r *menuRepository) CreateFormEntry(ctx context.Context, entry *model.MenuEntry) (*model.MenuEntry, error) {
	code, err := newMenuEntryCode()
	if err != nil {
		return nil, err
	}
	entry.Code = code
	if err := infrastructure.ResolveDB(ctx, r.db).Create(entry).Error; err != nil {
		return nil, err
	}
	return entry, nil
}

// CreateGroupEntry 创建菜单分组；group 不携带 target，数据库 CHECK 约束与
// 服务层类型常量共同保证分组不会伪装成资产节点。
func (r *menuRepository) CreateGroupEntry(ctx context.Context, entry *model.MenuEntry) (*model.MenuEntry, error) {
	code, err := newMenuEntryCode()
	if err != nil {
		return nil, err
	}
	entry.Code = code
	if err := infrastructure.ResolveDB(ctx, r.db).Create(entry).Error; err != nil {
		return nil, err
	}
	return entry, nil
}

// UpdateNameByFormTarget 表单改名事务内同步节点展示名。
func (r *menuRepository) UpdateNameByFormTarget(ctx context.Context, applicationID, formID uint, name string) error {
	return infrastructure.ResolveDB(ctx, r.db).Model(&model.MenuEntry{}).
		Where("application_id = ? AND entry_type = ? AND target_id = ?", applicationID, model.MenuEntryTypeForm, formID).
		Update("name", name).Error
}

// UpdateAppearanceByFormTarget 表单图标/颜色修改事务内同步节点展示属性
// （ADR-011：展示属性以资产域为事实源；空串清空，出网投影为 null）。
func (r *menuRepository) UpdateAppearanceByFormTarget(ctx context.Context, applicationID, formID uint, icon, color string) error {
	return infrastructure.ResolveDB(ctx, r.db).Model(&model.MenuEntry{}).
		Where("application_id = ? AND entry_type = ? AND target_id = ?", applicationID, model.MenuEntryTypeForm, formID).
		Updates(map[string]interface{}{"icon": icon, "color": color}).Error
}

// SoftDeleteByFormTarget 表单删除事务内软删节点。
func (r *menuRepository) SoftDeleteByFormTarget(ctx context.Context, applicationID, formID uint) error {
	return infrastructure.ResolveDB(ctx, r.db).
		Where("application_id = ? AND entry_type = ? AND target_id = ?", applicationID, model.MenuEntryTypeForm, formID).
		Delete(&model.MenuEntry{}).Error
}

// BumpMenuRevision 菜单写入的并发口令：同事务条件递增（SQL 表达式避免
// 读改写竞态，口径同 applications.menu_revision 约定）。
func (r *menuRepository) BumpMenuRevision(ctx context.Context, applicationID uint) error {
	return infrastructure.ResolveDB(ctx, r.db).Model(&model.Application{}).
		Where("id = ?", applicationID).
		Update("menu_revision", gorm.Expr("menu_revision + 1")).Error
}

// BumpMenuRevisionFrom 通过 WHERE menu_revision = baseRevision 原子占用下一
// 个修订号。所有菜单管理写路径先执行本方法，因此成功更新应用行后也获得
// 行锁，后续节点校验和写入不会与另一菜单写事务交错。
func (r *menuRepository) BumpMenuRevisionFrom(ctx context.Context, applicationID uint, baseRevision int64) (bool, error) {
	result := infrastructure.ResolveDB(ctx, r.db).Model(&model.Application{}).
		Where("id = ? AND menu_revision = ?", applicationID, baseRevision).
		Update("menu_revision", gorm.Expr("menu_revision + 1"))
	return result.RowsAffected == 1, result.Error
}

// UpdateEntryFields 节点白名单字段更新（fields 由 Service 组装，仓储不做
// 二次裁剪；先经 BumpMenuRevisionFrom 占用修订号后再调用，同事务执行）。
func (r *menuRepository) UpdateEntryFields(ctx context.Context, applicationID, entryID uint, fields map[string]interface{}) error {
	return infrastructure.ResolveDB(ctx, r.db).Model(&model.MenuEntry{}).
		Where("application_id = ? AND id = ?", applicationID, entryID).
		Updates(fields).Error
}

// ---- ADR-011：菜单节点个人收藏（个人状态，不递增菜单修订号） ----

// CreateFavorite 写入收藏行：(member_id, entry_id) 唯一约束幂等，重复收藏
// 静默成功（OnConflict DoNothing）。
func (r *menuRepository) CreateFavorite(ctx context.Context, fav *model.MenuEntryFavorite) error {
	return infrastructure.ResolveDB(ctx, r.db).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(fav).Error
}

// DeleteFavoriteByCode 取消收藏：按成员 + 节点编码定位（节点编码租户内
// 唯一）。目标行不存在同样按成功处理（幂等），返回是否实际删除了行。
func (r *menuRepository) DeleteFavoriteByCode(ctx context.Context, tenantID, memberID uint, entryCode string) (bool, error) {
	result := infrastructure.ResolveDB(ctx, r.db).
		Where("tenant_id = ? AND member_id = ? AND entry_id IN (?)", tenantID, memberID,
			infrastructure.ResolveDB(ctx, r.db).Model(&model.MenuEntry{}).
				Select("id").Where("code = ?", entryCode)).
		Delete(&model.MenuEntryFavorite{})
	return result.RowsAffected > 0, result.Error
}

// FavoriteEntryIDs 当前成员在指定应用内已收藏的节点 ID 集合（菜单读取时
// 投影 Favorited 状态用）。
func (r *menuRepository) FavoriteEntryIDs(ctx context.Context, tenantID, memberID, applicationID uint) (map[uint]bool, error) {
	var ids []uint
	err := infrastructure.ResolveDB(ctx, r.db).Model(&model.MenuEntryFavorite{}).
		Where("tenant_id = ? AND member_id = ? AND application_id = ?", tenantID, memberID, applicationID).
		Pluck("entry_id", &ids).Error
	if err != nil {
		return nil, err
	}
	set := make(map[uint]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set, nil
}

// DeleteFavoritesByFormTarget 表单软删事务内硬删其菜单节点的关联收藏行
// （个人状态无保留价值；不清理会残留指向软删节点的幽灵收藏）。
func (r *menuRepository) DeleteFavoritesByFormTarget(ctx context.Context, applicationID, formID uint) error {
	return infrastructure.ResolveDB(ctx, r.db).
		Where("application_id = ? AND entry_id IN (?)", applicationID,
			infrastructure.ResolveDB(ctx, r.db).Model(&model.MenuEntry{}).
				Select("id").
				Where("application_id = ? AND entry_type = ? AND target_id = ?", applicationID, model.MenuEntryTypeForm, formID)).
		Delete(&model.MenuEntryFavorite{}).Error
}

// ---- ADR-011：表单引用视图（form 域窄端口消费的跨应用反查） ----

// FormMenuReference 引用视图行：表单被哪个应用的哪个菜单节点引用。
type FormMenuReference struct {
	ApplicationCode string
	ApplicationName string
	EntryCode       string
	EntryName       string
	EntryType       string
	ParentEntryCode *string
}

// ListFormMenuReferences 跨应用反查引用指定表单的未软删菜单节点
// （含应用名快照与父节点编码投影）。租户过滤显式携带（raw SQL 不经租户
// Callback），引用视图本身是只读诊断信息，不做可见性裁剪（调用方须持
// forms:get）。
func (r *menuRepository) ListFormMenuReferences(ctx context.Context, tenantID, formID uint) ([]FormMenuReference, error) {
	const query = `
SELECT a.code AS application_code,
       a.name AS application_name,
       e.code AS entry_code,
       e.name AS entry_name,
       e.entry_type AS entry_type,
       p.code AS parent_entry_code
FROM application_menu_entries e
INNER JOIN applications a
    ON a.id = e.application_id
   AND a.tenant_id = e.tenant_id
   AND a.deleted_at IS NULL
LEFT JOIN application_menu_entries p
    ON p.id = e.parent_entry_id
   AND p.deleted_at IS NULL
WHERE e.tenant_id = ?
  AND e.entry_type = 'form'
  AND e.target_id = ?
  AND e.deleted_at IS NULL
ORDER BY a.code ASC, e.sort_order ASC, e.code ASC`
	var rows []FormMenuReference
	err := infrastructure.ResolveDB(ctx, r.db).Raw(query, tenantID, formID).Scan(&rows).Error
	return rows, err
}

// Migrate 开发/测试路径：AutoMigrate 建表 + 补齐 GORM 标签表达不了的
// 部分索引（与迁移链 000016 同名同构）；applications.menu_revision 列
// 由 Application 模型 AutoMigrate 自动补齐
func (r *menuRepository) Migrate() error {
	if err := r.db.AutoMigrate(&model.MenuEntry{}, &model.MenuEntryFavorite{}); err != nil {
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
