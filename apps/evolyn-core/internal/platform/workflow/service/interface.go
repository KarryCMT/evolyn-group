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

// ApplicationDirectory 应用目录窄端口（000064 产品日志）：流程定义经
// form_code 绑定表单，再由表单归属应用；装配层以 form+application 仓储
// 适配，workflow 域不直接依赖 form/application 域
type ApplicationDirectory interface {
	// ApplicationByFormCode 按表单编码解析所属应用视图（ctx 租户过滤：
	// 跨租户/不存在即 notFound）
	ApplicationByFormCode(ctx context.Context, formCode string) (app ApplicationView, notFound bool, err error)
}

// ApplicationView 应用只读视图（workflow 域关心的最小字段）：审计事件的
// 应用维度快照源
type ApplicationView struct {
	ID   uint
	Code string
	Name string
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

	// ---- Phase 4：完整人工任务与审批中心（第 10/20.3/20.4 章） ----

	// RejectTask 驳回任务（V1 terminate 语义：实例整体终止）
	RejectTask(ctx context.Context, member *iammodel.User, req *model.RejectTaskRequest) (*model.ActionTaskResult, error)
	// ReturnTask 退回发起人（实例保持 RUNNING，进入发起人修改等待态）
	ReturnTask(ctx context.Context, member *iammodel.User, req *model.ReturnTaskRequest) (*model.ActionTaskResult, error)
	// TransferTask 转办任务（原任务 TRANSFERRED，新任务另建，历史链可追溯）
	TransferTask(ctx context.Context, member *iammodel.User, req *model.TransferTaskRequest) (*model.ActionTaskResult, error)
	// WithdrawInstance 发起人撤回（撤回窗口：尚无已完成人工审批任务）
	WithdrawInstance(ctx context.Context, member *iammodel.User, instanceID uint, req *model.InstanceActionRequest) (*model.ActionTaskResult, error)
	// TerminateInstance 管理员终止（独立权限，不受撤回窗口限制）
	TerminateInstance(ctx context.Context, member *iammodel.User, instanceID uint, req *model.InstanceActionRequest) (*model.ActionTaskResult, error)
	// ResubmitInstance 发起人重新提交（流程从退回节点继续）
	ResubmitInstance(ctx context.Context, member *iammodel.User, instanceID uint, req *model.ResubmitInstanceRequest) (*model.ActionTaskResult, error)
	// ListTasks 审批中心任务查询：我的待办/我的已办/抄送我的
	ListTasks(ctx context.Context, member *iammodel.User, query model.ListTasksQuery) (*model.TaskPage, error)
	// ListInstances 审批中心实例查询：我发起的
	ListInstances(ctx context.Context, member *iammodel.User, query model.ListInstancesQuery) (*model.InstancePage, error)
	// GetTask 任务详情上下文（表单快照/数据 + 字段权限 + 允许动作 + 时间线）
	GetTask(ctx context.Context, member *iammodel.User, taskID uint) (*model.TaskDetail, error)
}
