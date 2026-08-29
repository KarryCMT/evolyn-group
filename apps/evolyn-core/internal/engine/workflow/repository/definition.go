// Package repository Workflow 引擎仓储 SPI（Phase 0 契约草案）。
//
// 内核仅经本包接口感知持久化，不感知 GORM；行锁（SELECT ... FOR UPDATE）、
// 部分唯一索引、租户过滤均由 platform/workflow 仓储适配层承担
// （第 13 章：事务经既有 TxManager / ResolveDB 传播，实现侧一律通过
// infrastructure.ResolveDB(ctx, base) 取连接以加入调用方事务）。
// 接口签名随 Phase 1/2 落地细化，但「无第二套事务边界」与「跨租户
// 先加载再写」两条铁律不变。
package repository

import (
	"context"

	"evolyn/internal/engine/workflow/model"
)

// DefinitionRepository 流程定义与版本仓储。
type DefinitionRepository interface {
	// CreateDefinition 创建设计态定义（code 租户内唯一）
	CreateDefinition(ctx context.Context, def *model.Definition) error
	// FindDefinitionByCode 按稳定公开标识读取（code 出网、ID 不出网）
	FindDefinitionByCode(ctx context.Context, tenantID uint, code string) (*model.Definition, error)
	// ListDefinitions 租户内定义列表（排除软删）
	ListDefinitions(ctx context.Context, tenantID uint) ([]model.Definition, error)
	// SaveDraft 保存草稿（draft_revision 乐观锁；冲突返回稳定错误）
	SaveDraft(ctx context.Context, def *model.Definition) error
	// SoftDeleteDefinition 软删定义（仅允许无运行中实例；运行态历史保留）
	SoftDeleteDefinition(ctx context.Context, tenantID uint, code string) error
	// CreateVersion 发布落库：冻结 dsl_snapshot 并分配 version_no（事务内）
	CreateVersion(ctx context.Context, version *model.DefinitionVersion) error
	// FindVersion 读取指定版本快照
	FindVersion(ctx context.Context, tenantID, definitionID uint, versionNo int) (*model.DefinitionVersion, error)
	// FindVersionByID 按版本行 ID 读取快照（运行实例按 wf_instance
	// 冻结的 definition_version_id 定位，取节点配置与审批模式）
	FindVersionByID(ctx context.Context, tenantID, versionID uint) (*model.DefinitionVersion, error)
	// ListVersions 版本列表（version_no 降序）
	ListVersions(ctx context.Context, definitionID uint) ([]model.DefinitionVersion, error)
}
