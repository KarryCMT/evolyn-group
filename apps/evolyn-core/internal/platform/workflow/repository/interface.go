// Package repository 流程引擎平台层数据访问（ADR-007 域内小三层）：仅做持久化，
// 一律经 infrastructure.ResolveDB 取连接加入 ctx 传播事务，不向 Service 暴露 GORM。
// 行锁/乐观锁语义在方法注释中明确；租户过滤依赖 ctx 携带租户上下文
// （GORM 租户 Callback），跨租户编码即 NotFound。
package repository

import (
	"context"

	"evolyn/internal/platform/workflow/model"
)

// ListParams 租户内定义列表查询（游标按 id 倒序，新定义靠前）。
type ListParams struct {
	Limit     int
	Cursor    string
	HasCursor bool
	AfterID   uint // 游标行 ID（取 id 严格小于该值的下一页）
}

// DefinitionRepository 流程定义仓储。
type DefinitionRepository interface {
	// Create 创建定义（TenantID 由调用方显式赋值，租户 Callback 兜底；
	// code 租户内唯一索引兜底，冲突经唯一索引报错由 Service 转写）
	Create(ctx context.Context, def *model.WfDefinition) (*model.WfDefinition, error)
	// GetByCode 按公开编码加载（未软删行）
	GetByCode(ctx context.Context, code string) (*model.WfDefinition, error)
	// List 租户内游标分页；返回当页数据与是否还有下一页（limit+1 探测）
	List(ctx context.Context, params ListParams) ([]model.WfDefinition, bool, error)
	// UpdateMeta 白名单更新名称/描述
	UpdateMeta(ctx context.Context, id uint, name, description string) error
	// SaveDraft 草稿乐观锁保存：draft_revision 匹配才写入并条件递增；
	// 0 行影响即口令过期（Service 转 WORKFLOW_REVISION_CONFLICT）
	SaveDraft(ctx context.Context, id uint, fromRevision int64, content model.DSLContent) (bool, error)
	// MarkPublished 发布事务内回写最新发布指针（latest_version_id +
	// published_version）；只追加指针，不触碰草稿与历史快照
	MarkPublished(ctx context.Context, id uint, versionID uint, versionNo int) error
	// SoftDelete 软删（发布版本行保留；运行中实例守卫自 Phase 2 接入）
	SoftDelete(ctx context.Context, def *model.WfDefinition) error
	// Migrate 开发/测试 AutoMigrate 路径（FIX-009：生产只走 SQL 迁移）
	Migrate() error
}

// VersionRepository 发布快照仓储（追加写，无更新/删除路径）。
type VersionRepository interface {
	// MaxVersionNo 当前最大发布号（发布事务内取 max+1，(definition_id,
	// version_no) 唯一约束兜底并发发布）
	MaxVersionNo(ctx context.Context, definitionID uint) (int, error)
	// Create 写入不可变快照（发布事务内）
	Create(ctx context.Context, version *model.WfDefinitionVersion) (*model.WfDefinitionVersion, error)
	// GetByDefinitionAndVersionNo 按版本号读取（历史版本均可读）
	GetByDefinitionAndVersionNo(ctx context.Context, definitionID uint, versionNo int) (*model.WfDefinitionVersion, error)
	// ListByDefinition 版本列表（version_no 降序）
	ListByDefinition(ctx context.Context, definitionID uint) ([]model.WfDefinitionVersion, error)
	// Migrate 开发/测试 AutoMigrate 路径（FIX-009：生产只走 SQL 迁移）
	Migrate() error
}
