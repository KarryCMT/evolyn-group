// Package service 流程定义域服务（Phase 1 Definition Engine，ADR-012）：
// 定义 CRUD、草稿保存（乐观锁）、DSL 严格校验与 Expr 预编译、不可变发布。
// 写路径在 Service 内按权限集复核（与鉴权中间件同口径，POST/PUT URL 门
// 之外做第二道校验）；DSL 校验以引擎内核严格校验器为唯一事实源。
package service

import (
	"context"

	iammodel "evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/workflow/model"
)

// AccessEvaluator 权限集窄端口（装配层由 application 域 RBAC 评估器适配）。
type AccessEvaluator interface {
	Permissions(ctx context.Context, member *iammodel.User) map[string]bool
}

// DefinitionService 流程定义域服务接口。
type DefinitionService interface {
	// Create 创建流程定义（草稿初始化为最小合法 DSL：start → end）
	Create(ctx context.Context, member *iammodel.User, req *model.CreateWorkflowRequest) (*model.WorkflowDetail, error)
	// List 租户内定义游标分页
	List(ctx context.Context, member *iammodel.User, query model.ListWorkflowsQuery) (*model.WorkflowPage, error)
	// Get 按公开编码读取定义详情（含草稿全文与修订口令）
	Get(ctx context.Context, member *iammodel.User, code string) (*model.WorkflowDetail, error)
	// Update 白名单更新名称/描述
	Update(ctx context.Context, member *iammodel.User, code string, req *model.UpdateWorkflowRequest) (*model.WorkflowDetail, error)
	// SaveDraft 保存草稿：严格校验 + draft_revision 乐观锁条件更新
	SaveDraft(ctx context.Context, member *iammodel.User, code string, req *model.SaveDraftRequest) (*model.SaveDraftResult, error)
	// Delete 软删定义（发布版本保留；运行中实例守卫自 Phase 2 接入）
	Delete(ctx context.Context, member *iammodel.User, code string) error
	// Publish 发布：口令复核 → DSL 严格校验 → Expr 预编译 → 事务内冻结快照
	Publish(ctx context.Context, member *iammodel.User, code string, req *model.PublishRequest) (*model.PublishResult, error)
	// ListVersions 版本列表（version_no 降序）
	ListVersions(ctx context.Context, member *iammodel.User, code string) ([]model.VersionSummary, error)
	// GetVersion 指定版本详情（含冻结快照全文；历史版本可读）
	GetVersion(ctx context.Context, member *iammodel.User, code string, versionNo int) (*model.VersionDetail, error)
}

// RuntimeService 最小流程运行时服务接口（Phase 2：发起/详情/同意）。
type RuntimeService interface {
	// Start 发起流程实例（业务幂等 + 请求幂等，事务内推进至审批挂起或完成）
	Start(ctx context.Context, member *iammodel.User, req *model.StartInstanceRequest) (*model.InstanceDetail, error)
	// GetInstance 实例详情（绑定关系 + 节点/任务/操作时间线）
	GetInstance(ctx context.Context, member *iammodel.User, instanceID uint) (*model.InstanceDetail, error)
	// Approve 审批同意（行锁 + 参与人校验 + 节点完成判定 + 同事务推进）
	Approve(ctx context.Context, member *iammodel.User, req *model.ApproveTaskRequest) (*model.ApproveTaskResult, error)
}
