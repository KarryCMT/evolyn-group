// Package workflow 流程引擎平台适配层（ADR-012）——Definition Engine（Phase 1）：
// 流程定义 CRUD、草稿保存（draft_revision 乐观锁）、DSL 严格校验与 Expr 预编译、
// 不可变发布快照与版本查询。稳定业务错误码集中定义于本包（ADR-008），
// 调用方按 errCode 分支；内部细节经 httpx.Wrap 只入日志。
package workflow

import (
	"net/http"

	"evolyn/internal/platform/httpx"
)

var (
	// ErrWorkflowNotFound 流程定义不存在或无权访问（租户过滤后的 NotFound 统一口径）
	ErrWorkflowNotFound = httpx.NewBiz("WORKFLOW_NOT_FOUND", "流程不存在或无权访问", http.StatusNotFound)

	// ErrInstanceNotFound 流程实例不存在或无权访问
	ErrInstanceNotFound = httpx.NewBiz("WORKFLOW_INSTANCE_NOT_FOUND", "流程实例不存在或无权访问", http.StatusNotFound)

	// ErrTaskNotFound 任务不存在或无权访问
	ErrTaskNotFound = httpx.NewBiz("WORKFLOW_TASK_NOT_FOUND", "任务不存在或无权访问", http.StatusNotFound)

	// ErrWorkflowNameInvalid 流程名称不符合要求（trim 后 1–128 字符）
	ErrWorkflowNameInvalid = httpx.NewBiz("WORKFLOW_NAME_INVALID", "流程名称不符合要求", http.StatusBadRequest)

	// ErrWorkflowDescriptionInvalid 流程描述不符合要求（≤512 字符）
	ErrWorkflowDescriptionInvalid = httpx.NewBiz("WORKFLOW_DESCRIPTION_INVALID", "流程描述不符合要求", http.StatusBadRequest)

	// ErrRevisionConflict 草稿修订号过期（他人已保存），客户端刷新后重试
	ErrRevisionConflict = httpx.NewBiz("WORKFLOW_REVISION_CONFLICT", "流程已被他人更新，请刷新后重试", http.StatusConflict)

	// ErrDefinitionInvalid DSL 严格校验失败；data 携带 issues:[{path,code,message}]
	//（path 为 DSL 文档内定位，code 为校验器稳定错误码，message 为中文说明）
	ErrDefinitionInvalid = httpx.NewBiz("WORKFLOW_DEFINITION_INVALID", "流程定义不符合 DSL v1 协议", http.StatusBadRequest)

	// ErrWorkflowCodeInvalid 路由编码不符合 wf_ 前缀约定（客户端错误）
	ErrWorkflowCodeInvalid = httpx.NewBiz("WORKFLOW_CODE_INVALID", "无效的流程编码", http.StatusBadRequest)

	// ErrWorkflowFormCodeInvalid 绑定表单编码不符合 form_ 前缀约定（000060 表单绑定）
	ErrWorkflowFormCodeInvalid = httpx.NewBiz("WORKFLOW_FORM_CODE_INVALID", "无效的绑定表单编码", http.StatusBadRequest)

	// ErrWorkflowFormAlreadyBound 同一表单已经绑定了有效流程定义。流程设计器
	// 可据此重新读取绑定定义，避免数据库唯一索引错误被脱敏为内部错误。
	ErrWorkflowFormAlreadyBound = httpx.NewBiz("WORKFLOW_FORM_ALREADY_BOUND", "该表单已绑定流程定义，请刷新后重试", http.StatusConflict)

	// ErrVersionNotFound 指定发布版本不存在（版本以 version_no 标识，历史版本均可读）
	ErrVersionNotFound = httpx.NewBiz("WORKFLOW_VERSION_NOT_FOUND", "流程版本不存在", http.StatusNotFound)

	// ErrFormVersionInvalid 表单绑定无效：表单不存在/跨租户/版本号不存在
	ErrFormVersionInvalid = httpx.NewBiz("WORKFLOW_FORM_VERSION_INVALID", "表单或表单版本无效", http.StatusBadRequest)

	// ErrForbidden 流程域操作越权（与鉴权中间件共用 FORBIDDEN 稳定码）
	ErrForbidden = httpx.NewBiz(httpx.CodeForbidden, "没有执行该操作的权限", http.StatusForbidden)

	// ---- 运行态（Phase 2，第 20.5 章冻结码段） ----

	// ErrNotPublished 流程尚未发布可执行版本（发起校验）
	ErrNotPublished = httpx.NewBiz("WORKFLOW_VERSION_NOT_PUBLISHED", "流程尚未发布", http.StatusBadRequest)

	// ErrInstanceAlreadyRunning 同一业务键已存在运行中实例（业务幂等，第 14.1 章）
	ErrInstanceAlreadyRunning = httpx.NewBiz("WORKFLOW_INSTANCE_ALREADY_RUNNING", "该业务已存在进行中的流程", http.StatusConflict)

	// ErrInstanceNotRunning 实例不在运行中（终态实例不可再操作）
	ErrInstanceNotRunning = httpx.NewBiz("WORKFLOW_INSTANCE_NOT_RUNNING", "流程已结束，无法继续操作", http.StatusConflict)

	// ErrTaskNotPending 任务不在待办状态（已被处理/取消/转办；双击防护，第 13.2 章）
	ErrTaskNotPending = httpx.NewBiz("WORKFLOW_TASK_NOT_PENDING", "任务已被处理，请刷新后重试", http.StatusConflict)

	// ErrTaskForbidden 当前操作者不是该任务的参与人（实例级校验，第 21 章）
	ErrTaskForbidden = httpx.NewBiz("WORKFLOW_TASK_FORBIDDEN", "没有执行该任务的权限", http.StatusForbidden)

	// ---- 表单联动与表达式（Phase 3，第 20.5 章冻结码段） ----

	// ErrExpressionInvalid 条件表达式运行期求值失败（发布期已预编译拦截
	// 语法/白名单问题，运行期失败属数据形态异常）
	ErrExpressionInvalid = httpx.NewBiz("WORKFLOW_EXPRESSION_INVALID", "条件表达式执行失败", http.StatusBadRequest)

	// ErrAssigneeNotFound 审批人解析为空且无可用兜底（第 17 章补充语义，
	// 禁止静默跳过节点）
	ErrAssigneeNotFound = httpx.NewBiz("WORKFLOW_ASSIGNEE_NOT_FOUND", "无法解析审批人，请联系管理员配置审批人", http.StatusConflict)

	// ErrFormFieldForbidden 审批编辑携带了节点字段权限未授权的字段，
	// 或实例未绑定表单却携带编辑值（第 15.4 章审批提交协议）
	ErrFormFieldForbidden = httpx.NewBiz("WORKFLOW_FORM_FIELD_FORBIDDEN", "存在本节点不允许修改的表单字段", http.StatusForbidden)

	// ErrActionNotAllowed 当前状态下不允许执行该动作（如撤回窗口已关闭：
	// 实例已存在完成的人工审批任务、非退回状态下重提交，第 10.4 章冻结规则）
	ErrActionNotAllowed = httpx.NewBiz("WORKFLOW_ACTION_NOT_ALLOWED", "当前状态不允许执行该操作", http.StatusConflict)
)
