// Package tenantproduct 产品中心域（一期）：平台内置产品在租户内的启停与
// 可用范围配置。稳定业务错误码集中定义于本包（ADR-008），前端按 errCode
// 分支（packages/utils/src/request/errorCodes.ts 对齐维护），内部细节经
// httpx.Wrap 只入日志
package tenantproduct

import (
	"net/http"

	"evolyn/internal/platform/httpx"
)

var (
	// ErrProductNotFound 产品不存在、未初始化或不属于当前租户（文档 6.5）
	ErrProductNotFound = httpx.NewBiz("TENANT_PRODUCT_NOT_FOUND", "产品不存在或未初始化", http.StatusNotFound)

	// ErrScopeInvalid 范围模式非法，或 all 模式携带部门/成员 ID
	ErrScopeInvalid = httpx.NewBiz("TENANT_PRODUCT_SCOPE_INVALID", "产品可用范围参数无效", http.StatusBadRequest)

	// ErrScopeEmpty partial 模式未选择任何部门和成员（不允许退化为无人可用）
	ErrScopeEmpty = httpx.NewBiz("TENANT_PRODUCT_SCOPE_EMPTY", "请至少选择一个部门或成员", http.StatusBadRequest)

	// ErrMemberInvalid 成员不存在、不属于当前租户或不是有效成员
	ErrMemberInvalid = httpx.NewBiz("TENANT_PRODUCT_MEMBER_INVALID", "所选成员无效或不可用", http.StatusBadRequest)

	// ErrDepartmentInvalid 部门不存在、不属于当前租户或不可用于范围
	ErrDepartmentInvalid = httpx.NewBiz("TENANT_PRODUCT_DEPARTMENT_INVALID", "所选部门无效或不可用", http.StatusBadRequest)

	// ErrRevisionConflict 乐观锁版本过期（并发更新冲突）
	ErrRevisionConflict = httpx.NewBiz("TENANT_PRODUCT_REVISION_CONFLICT", "配置已更新，请刷新后重试", http.StatusConflict)

	// ErrProductDisabled 访问时产品被租户停用（预留给产品受保护入口，
	// 访问判定器 CanAccess 以 bool 返回，不出网错误）
	ErrProductDisabled = httpx.NewBiz("TENANT_PRODUCT_DISABLED", "产品已被租户停用", http.StatusForbidden)

	// ErrAccessDenied 当前成员不在产品可用范围内（预留给产品受保护入口）
	ErrAccessDenied = httpx.NewBiz("TENANT_PRODUCT_ACCESS_DENIED", "当前成员不在产品可用范围内", http.StatusForbidden)
)
