// Package workbench 自定义工作台域（000078 企业级配置定版）：企业工作台
// 配置的存取。企业管理员经 /workbench 保存、全员经 GET 读取共用文档
// （tn_workbenches，每租户一行）；租户开通事务内种子默认布局。
// 稳定业务错误码集中定义于本包（ADR-008），调用方按 errCode 分支；内部细节
// 经 httpx.Wrap 只入日志
package workbench

import (
	"net/http"

	"evolyn/internal/platform/httpx"
)

var (
	// ErrDocumentInvalid 工作台文档结构校验失败（version/widgets/卡片类型/坐标约束等）
	ErrDocumentInvalid = httpx.NewBiz("WORKBENCH_DOCUMENT_INVALID", "工作台配置结构无效", http.StatusBadRequest)

	// ErrRevisionConflict 乐观锁冲突：工作台已被其他窗口或管理员保存
	ErrRevisionConflict = httpx.NewBiz("WORKBENCH_REVISION_CONFLICT", "工作台已被其他窗口保存，请刷新后重试", http.StatusConflict)
)
