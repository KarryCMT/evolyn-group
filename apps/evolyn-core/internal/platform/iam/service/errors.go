package service

import "errors"

// 稳定业务错误码（整改文档 FIX-002/003/006）：sentinel 错误文本即对外错误码，
// 细节通过 fmt.Errorf("%w: ...", ErrXxx) 附加，调用方以 errors.Is 判定
var (
	// ErrCrossTenantBinding 跨租户关系绑定被拒绝（FIX-006）：
	// Member/Group/Department/Role 两端必须属于同一租户
	ErrCrossTenantBinding = errors.New("CROSS_TENANT_BINDING_REJECTED")

	// ErrDuplicateName 租户内重名（FIX-002/003）：Role/Group 名称租户内唯一
	ErrDuplicateName = errors.New("DUPLICATE_NAME")

	// ErrDuplicateMember 同租户重复成员（FIX-004）：一个账号在一个租户仅一个有效成员
	ErrDuplicateMember = errors.New("DUPLICATE_MEMBER")
)
