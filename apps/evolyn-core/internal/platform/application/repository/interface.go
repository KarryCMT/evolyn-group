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

// MenuRepository 应用菜单仓储（M2-菜单-1 只读骨架；分组管理/重排写路径
// 随 M2-菜单-3 落地，资产域创建时的事务内节点维护随 M2-资产-1 落地）
type MenuRepository interface {
	// GetSnapshot 按租户与应用编码读取「应用元信息 + 未软删菜单节点」的
	// 一致性快照；应用不存在/跨租户返回 gorm.ErrRecordNotFound
	GetSnapshot(ctx context.Context, tenantID uint, code string) (*MenuSnapshot, error)
	// Migrate 开发/测试 AutoMigrate 路径（生产只走 SQL 迁移）
	Migrate() error
}
