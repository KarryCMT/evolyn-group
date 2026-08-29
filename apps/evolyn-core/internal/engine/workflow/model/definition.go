package model

// Definition 流程定义（设计态逻辑流程，如「报销审批」）。
// 内核视角的纯值对象；持久化（draft_content JSONB + draft_revision 乐观锁）
// 由 platform/workflow 仓储适配层承担。
type Definition struct {
	// ID 内部自增主键，不出网；对外一律使用 Code（对齐表单域 form_code 先例）
	ID uint
	// TenantID 归属租户
	TenantID uint
	// Code 稳定公开标识：路由/API/菜单 target 一律用 code
	Code string
	// Name 展示名
	Name string
	// Description 描述
	Description string
	// Draft 当前草稿 DSL 文档
	Draft Document
	// DraftRevision 草稿乐观锁版本号（对齐表单域 draft_revision 协议）
	DraftRevision int64
	// Status 设计态状态
	Status DefinitionStatus
}

// DefinitionStatus 设计态状态。软删仅允许删除无运行中实例的定义；
// 运行态历史（instance/operation）不随设计态删除。
type DefinitionStatus string

const (
	DefinitionStatusDraft     DefinitionStatus = "DRAFT"
	DefinitionStatusPublished DefinitionStatus = "PUBLISHED"
	DefinitionStatusDeleted   DefinitionStatus = "DELETED"
)

// DefinitionVersion 一次发布后冻结的可执行版本（不可变）。
// DSL 整体冻结进 dsl_snapshot，运行实例只绑定版本，永不自动升级。
type DefinitionVersion struct {
	ID uint
	// DefinitionID 所属定义（内部外键，不出网）
	DefinitionID uint
	// VersionNo 定义内递增版本号（对外版本标识）
	VersionNo int
	// Snapshot 发布时冻结的 DSL 全文档（唯一事实源，含 Node/Edge/Config）
	Snapshot Document
}
