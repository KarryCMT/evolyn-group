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

	// ErrVersionNotFound 指定发布版本不存在（版本以 version_no 标识，历史版本均可读）
	ErrVersionNotFound = httpx.NewBiz("WORKFLOW_VERSION_NOT_FOUND", "流程版本不存在", http.StatusNotFound)

	// ErrForbidden 流程域操作越权（与鉴权中间件共用 FORBIDDEN 稳定码）
	ErrForbidden = httpx.NewBiz(httpx.CodeForbidden, "没有执行该操作的权限", http.StatusForbidden)
)
