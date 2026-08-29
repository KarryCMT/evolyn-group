// Package repository 表单资产域数据访问（ADR-007 域内小三层）：仅做持久化，
// 一律经 infrastructure.ResolveDB 取连接加入 ctx 传播事务，不向 Service 暴露 GORM。
package repository

import (
	"context"

	"evolyn/internal/platform/form/model"
)

// ListParams 应用内表单列表查询（游标按 id 倒序，新表单靠前）。
type ListParams struct {
	ApplicationID uint
	Limit         int
	HasCursor     bool
	AfterID       uint // 游标行 ID（取 id 严格小于该值的下一页）
}

// FormMenuTarget 是表单仓储供应用菜单读侧消费的最小投影。
type FormMenuTarget struct {
	Code     string
	FormType model.FormType
}

// FormRepository 表单资产仓储。
type FormRepository interface {
	// Create 创建表单（TenantID 由调用方显式赋值，租户 Callback 兜底）
	Create(ctx context.Context, form *model.Form) (*model.Form, error)
	// GetByCode 按公开编码加载（ctx 携带租户时自动过滤，跨租户编码即 NotFound）
	GetByCode(ctx context.Context, code string) (*model.Form, error)
	// List 应用内游标分页；返回当页数据与是否还有下一页（limit+1 探测）
	List(ctx context.Context, params ListParams) ([]model.Form, bool, error)
	// UpdateName 改名（白名单字段）
	UpdateName(ctx context.Context, id uint, name string) error
	// UpdateFormType 切换表单类型（ADR-011：服务层校验枚举与类型变化后写入）
	UpdateFormType(ctx context.Context, id uint, formType model.FormType) error
	// UpdateDraft 草稿乐观锁保存：draft_revision 匹配才写入并条件递增；
	// 0 行影响即口令过期（Service 转 FORM_REVISION_CONFLICT）
	UpdateDraft(ctx context.Context, id uint, fromRevision int64, content model.JSONContent) (bool, error)
	// MarkPublished 发布事务内回写最新发布指针（latest_version_id + published_version）；
	// 只追加指针，不触碰草稿与历史快照
	MarkPublished(ctx context.Context, id uint, versionID uint, versionNo int) error
	// SoftDelete 软删（配额随计数口径自然释放；发布版本行保留）
	SoftDelete(ctx context.Context, form *model.Form) error
	// CountBillableFormsByTenant 计费表单数（deleted_at IS NULL 全量）
	CountBillableFormsByTenant(ctx context.Context, tenantID uint) (int64, error)
	// ExistingFormTargets 返回 ids 中存在且未软删表单的菜单目标投影（M2-资产-1
	// 菜单读侧存在性判定与 target 投影；租户过滤由 ctx 承载）
	ExistingFormTargets(ctx context.Context, ids []uint) (map[uint]FormMenuTarget, error)
	// Migrate 开发/测试 AutoMigrate 路径（FIX-009：生产只走 SQL 迁移）
	Migrate() error
}

// FormVersionRepository 发布版本仓储（不可变：只创建与读取）。
type FormVersionRepository interface {
	// Create 写入快照（同一发布事务内回填 schema_revision=行 id，见 SetSchemaRevision）
	Create(ctx context.Context, version *model.FormVersion) (*model.FormVersion, error)
	// SetSchemaRevision 创建事务内回填修订口令（= 行 id；此后的确无更新路径）
	SetSchemaRevision(ctx context.Context, id uint, revision int64) error
	// GetByID 按行 ID 加载（ctx 租户过滤兜底）
	GetByID(ctx context.Context, id uint) (*model.FormVersion, error)
	// MaxVersionNo 表单内最大发布号（发布递增用；无版本返回 0）
	MaxVersionNo(ctx context.Context, formID uint) (int, error)
	// GetByFormAndVersionNo 按 (form_id, version_no) 定位历史版本（提交受理）
	GetByFormAndVersionNo(ctx context.Context, formID uint, versionNo int) (*model.FormVersion, error)
	// Migrate 开发/测试 AutoMigrate 路径
	Migrate() error
}

// FormRecordRepository 记录仓储（P2：追加写）。
type FormRecordRepository interface {
	Create(ctx context.Context, record *model.FormRecord) (*model.FormRecord, error)
	// Migrate 开发/测试 AutoMigrate 路径
	Migrate() error
}
