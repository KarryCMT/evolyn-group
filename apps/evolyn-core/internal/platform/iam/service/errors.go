package service

import (
	"net/http"

	"evolyn/internal/platform/httpx"
)

// 稳定业务错误码（整改 FIX-002/003/004/006；ADR-008 起承载于 BizError）：
// 错误文本即对外错误码的历史口径由 Code 字段承接，调用方 errors.Is 判定不变；
// 细节通过 httpx.Wrap 附加（只入日志，不出网）
var (
	// ErrCrossTenantBinding 跨租户关系绑定被拒绝（FIX-006）：
	// Member/Group/Department/Role 两端必须属于同一租户
	ErrCrossTenantBinding = httpx.NewBiz("CROSS_TENANT_BINDING_REJECTED", "跨租户的关系绑定被拒绝", http.StatusForbidden)

	// ErrDuplicateName 租户内重名（FIX-002/003）：Role/Group 名称租户内唯一
	ErrDuplicateName = httpx.NewBiz("DUPLICATE_NAME", "名称已存在", http.StatusConflict)

	// ErrDuplicateMember 同租户重复成员（FIX-004）：一个账号在一个租户仅一个有效成员
	ErrDuplicateMember = httpx.NewBiz("DUPLICATE_MEMBER", "该账号已是本租户成员", http.StatusConflict)
)
