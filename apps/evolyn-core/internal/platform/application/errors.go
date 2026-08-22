// Package application 应用管理域（M2-A）：空白应用创建/查询/更新/软删与
// 配额占位。稳定业务错误码集中定义于本包（ADR-008），调用方按 errCode
// 分支；内部细节经 httpx.Wrap 只入日志。模板安装（M2-B）与异步实例化
// （M2-C）的错误码（TEMPLATE_* / APP_PROVISION_FAILED）随对应批次补充
package application

import (
	"net/http"

	"evolyn/internal/platform/httpx"
)

var (
	// ErrNameInvalid 应用名称不符合要求（去首尾空格后 1–128 字符）
	ErrNameInvalid = httpx.NewBiz("APP_NAME_INVALID", "应用名称不符合要求", http.StatusBadRequest)

	// ErrIconInvalid 图标键不在服务端稳定枚举内（不存前端组件名）
	ErrIconInvalid = httpx.NewBiz("APP_ICON_INVALID", "应用图标配置无效", http.StatusBadRequest)

	// ErrColorInvalid 颜色键不在服务端稳定枚举内（不存 CSS 字面值）
	ErrColorInvalid = httpx.NewBiz("APP_COLOR_INVALID", "应用颜色配置无效", http.StatusBadRequest)

	// ErrMemberInvalid 操作者不是当前租户成员（跨租户成员绑定拦截，§9.3）
	ErrMemberInvalid = httpx.NewBiz("APP_MEMBER_INVALID", "当前成员不属于该租户", http.StatusForbidden)

	// ErrForbidden 应用域操作越权（Service 内部经 ApplicationAccessEvaluator
	// 复核，与鉴权中间件共用 FORBIDDEN 稳定码，前端无需新分支）
	ErrForbidden = httpx.NewBiz(httpx.CodeForbidden, "没有执行该操作的权限", http.StatusForbidden)

	// ErrQueryInvalid 列表查询参数非法（status 过滤值不在枚举内等）
	ErrQueryInvalid = httpx.NewBiz("APP_QUERY_INVALID", "应用查询参数无效", http.StatusBadRequest)

	// ErrCursorInvalid 分页游标非法（非 base64/缺字段），客户端应刷新列表重取
	ErrCursorInvalid = httpx.NewBiz("APP_CURSOR_INVALID", "分页游标无效，请刷新列表后重试", http.StatusBadRequest)

	// ErrNotFound 应用不存在或无权访问（租户过滤后的 NotFound 统一口径）
	ErrNotFound = httpx.NewBiz("APP_NOT_FOUND", "应用不存在或无权访问", http.StatusNotFound)

	// ErrStatusInvalid 状态流转不合法（仅 active↔archived，§7.1）
	ErrStatusInvalid = httpx.NewBiz("APP_STATUS_INVALID", "当前应用状态不支持此操作", http.StatusConflict)

	// ErrProvisioning 实例化进行中（pending/running 不可编辑/删除/进设计器）
	ErrProvisioning = httpx.NewBiz("APP_PROVISIONING", "应用正在初始化，请稍后重试", http.StatusConflict)
)
