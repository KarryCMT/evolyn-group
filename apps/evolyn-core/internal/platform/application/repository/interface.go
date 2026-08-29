// Package repository 应用管理域数据访问（ADR-007 域内小三层）：
// 仅做持久化，一律经 infrastructure.ResolveDB 取连接加入 ctx 传播事务，
// 不向 Service 暴露 GORM *DB
package repository

import (
	"context"

	"evolyn/internal/platform/application/model"
)

// ListParams 列表查询参数（游标已由 Service 解码为排序定位值）：
// 定位语义为「(sort_order, id) 排在游标行之后」，与列表序
// sort_order ASC, id DESC 对应
type ListParams struct {
	Keyword     string
	Status      string
	Limit       int
	HasCursor   bool
	AfterSortID int64 // 游标行的 sort_order
	AfterID     uint  // 游标行的应用 ID
}

// ApplicationRepository 应用实例仓储
type ApplicationRepository interface {
	// Create 创建应用实例（TenantID 由调用方显式赋值，租户 Callback 兜底）
	Create(ctx context.Context, app *model.Application) (*model.Application, error)
	// CreateInstallation 创建安装记录（创建事务内与应用同批写入）
	CreateInstallation(ctx context.Context, inst *model.Installation) error
	// GetByID 按 ID 加载（ctx 携带租户时自动过滤，跨租户 ID 即 NotFound）
	GetByID(ctx context.Context, id uint) (*model.Application, error)
	// GetByCode 按应用编码加载（code 租户内唯一；租户过滤语义同 GetByID）
	GetByCode(ctx context.Context, code string) (*model.Application, error)
	// List 游标分页查询；返回当页数据与是否还有下一页（limit+1 探测）
	List(ctx context.Context, params ListParams) ([]model.Application, bool, error)
	// UpdateFields 白名单字段更新（先加载校验后调用，租户过滤随 ctx）
	UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error
	// SoftDelete 软删（仅写 deleted_at；配额随计数口径自然释放）
	SoftDelete(ctx context.Context, app *model.Application) error
	// CountBillableByTenant 计费应用数（deleted_at IS NULL 全量，§10）：
	// 含 archived 与 pending/running/failed，软删后才释放名额
	CountBillableByTenant(ctx context.Context, tenantID uint) (int64, error)
	// Migrate 开发/测试 AutoMigrate 路径（FIX-009：生产只走 SQL 迁移）
	Migrate() error
}

// MenuSnapshot 应用与菜单节点的同快照读取结果：menuRevision 与 entries
// 来自单条 SQL 的同一语句快照（方案 §5.2：禁止 Read Committed 两读拼接，
// 否则客户端可能取得「旧树 + 新修订号」在全量重排时漏检冲突）
type MenuSnapshot struct {
	ApplicationID   uint
	ApplicationCode string
	Status          string
	ProvisionStatus string
	MenuRevision    int64
	Entries         []model.MenuEntry
}

// MenuRepository 应用菜单仓储。所有写方法都经 ResolveDB 加入调用方事务；
// 菜单管理写入先条件推进修订号，再在同一事务内修改节点。
type MenuRepository interface {
	// GetSnapshot 按租户与应用编码读取「应用元信息 + 未软删菜单节点」的
	// 一致性快照；应用不存在/跨租户返回 gorm.ErrRecordNotFound
	GetSnapshot(ctx context.Context, tenantID uint, code string) (*MenuSnapshot, error)
	// CreateGroupEntry 创建不绑定资产的 group 节点；code 由仓储生成。
	CreateGroupEntry(ctx context.Context, entry *model.MenuEntry) (*model.MenuEntry, error)
	// CreateFormEntry 在表单创建事务内插入 form 资产节点：code 服务端生成，
	// sort_order 取同父（根级或指定分组）最大值 + 1024（首个 1024）
	CreateFormEntry(ctx context.Context, entry *model.MenuEntry) (*model.MenuEntry, error)
	// UpdateNameByFormTarget 按应用与表单目标同步节点名（表单改名事务内）
	UpdateNameByFormTarget(ctx context.Context, applicationID, formID uint, name string) error
	// UpdateAppearanceByFormTarget 按应用与表单目标同步节点图标/颜色
	//（ADR-011：展示属性以资产域为事实源，空串清空）
	UpdateAppearanceByFormTarget(ctx context.Context, applicationID, formID uint, icon, color string) error
	// SoftDeleteByFormTarget 软删表单目标节点（表单删除事务内）
	SoftDeleteByFormTarget(ctx context.Context, applicationID, formID uint) error
	// MaxSortOrder 指定父节点（nil 即根级）下最大排序值；无节点返回 0
	MaxSortOrder(ctx context.Context, applicationID uint, parentEntryID *uint) (int64, error)
	// FindByCode 按节点编码查未软删节点（表单挂载父分组定位用）
	FindByCode(ctx context.Context, applicationID uint, code string) (*model.MenuEntry, error)
	// BumpMenuRevision 同事务递增应用菜单修订号（菜单写入的并发口令）
	BumpMenuRevision(ctx context.Context, applicationID uint) error
	// BumpMenuRevisionFrom 仅当当前修订号等于 baseRevision 时递增；false
	// 表示并发写入已抢先提交，调用方应返回 APP_MENU_VERSION_CONFLICT。
	BumpMenuRevisionFrom(ctx context.Context, applicationID uint, baseRevision int64) (bool, error)
	// UpdateEntryFields 节点白名单字段更新（fields 由 Service 组装；
	// 须先经 BumpMenuRevisionFrom 占用修订号后同事务调用）
	UpdateEntryFields(ctx context.Context, applicationID, entryID uint, fields map[string]interface{}) error
	// CreateFavorite 写入成员收藏（(member_id, entry_id) 唯一幂等）
	CreateFavorite(ctx context.Context, fav *model.MenuEntryFavorite) error
	// DeleteFavoriteByCode 按成员 + 节点编码取消收藏（幂等）；返回是否实际删除
	DeleteFavoriteByCode(ctx context.Context, tenantID, memberID uint, entryCode string) (bool, error)
	// FavoriteEntryIDs 当前成员在指定应用内已收藏的节点 ID 集合
	FavoriteEntryIDs(ctx context.Context, tenantID, memberID, applicationID uint) (map[uint]bool, error)
	// DeleteFavoritesByFormTarget 表单软删事务内硬删其菜单节点的关联收藏行
	DeleteFavoritesByFormTarget(ctx context.Context, applicationID, formID uint) error
	// ListFormMenuReferences 跨应用反查引用指定表单的未软删菜单节点
	//（引用视图只读诊断，租户条件显式携带）
	ListFormMenuReferences(ctx context.Context, tenantID, formID uint) ([]FormMenuReference, error)
	// Migrate 开发/测试 AutoMigrate 路径（生产只走 SQL 迁移）
	Migrate() error
}
