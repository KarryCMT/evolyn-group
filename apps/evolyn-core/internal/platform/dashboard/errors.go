// Package dashboard 定义业务仪表盘资产域的稳定业务错误。
package dashboard

import (
	"net/http"

	"evolyn/internal/platform/httpx"
)

var (
	ErrNotFound            = httpx.NewBiz("DASHBOARD_NOT_FOUND", "仪表盘不存在或无权访问", http.StatusNotFound)
	ErrNameInvalid         = httpx.NewBiz("DASHBOARD_NAME_INVALID", "仪表盘名称不符合要求", http.StatusBadRequest)
	ErrAppearanceInvalid   = httpx.NewBiz("DASHBOARD_APPEARANCE_INVALID", "仪表盘图标或颜色配置无效", http.StatusBadRequest)
	ErrAppInvalid          = httpx.NewBiz("DASHBOARD_APP_INVALID", "应用不存在或不可用", http.StatusBadRequest)
	ErrSchemaInvalid       = httpx.NewBiz("DASHBOARD_SCHEMA_INVALID", "仪表盘内容不符合保存协议", http.StatusBadRequest)
	ErrDraftConflict       = httpx.NewBiz("DASHBOARD_DRAFT_CONFLICT", "仪表盘已被他人更新，请刷新后重试", http.StatusConflict)
	ErrIdempotencyConflict = httpx.NewBiz("DASHBOARD_IDEMPOTENCY_CONFLICT", "创建请求标识已用于不同内容", http.StatusConflict)
	ErrRequestInvalid      = httpx.NewBiz("DASHBOARD_REQUEST_INVALID", "仪表盘请求参数不符合要求", http.StatusBadRequest)
	ErrForbidden           = httpx.NewBiz(httpx.CodeForbidden, "没有执行该操作的权限", http.StatusForbidden)
)
