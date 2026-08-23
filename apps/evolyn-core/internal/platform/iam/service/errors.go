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

	// ErrCredentialsInvalid 密码登录凭证错误（ADR-008）：账号不存在与密码错误
	// 统一文案，不泄露账号存在性；401
	ErrCredentialsInvalid = httpx.NewBiz("AUTH_CREDENTIALS_INVALID", "账号或密码错误", http.StatusUnauthorized)

	// ErrAccountNotFound 短信登录手机号未注册（ADR-008）：验证码已通过，
	// 区别于凭证错误，可引导走注册；401
	ErrAccountNotFound = httpx.NewBiz("AUTH_ACCOUNT_NOT_FOUND", "该手机号未注册", http.StatusUnauthorized)

	// ErrNotMember 账号与目标租户无成员关系（ADR-008）：指定租户登录/切换租户；
	// 账号名/租户编码等细节经 Wrap 只入日志；403
	ErrNotMember = httpx.NewBiz("AUTH_NOT_MEMBER", "该账号不属于此租户", http.StatusForbidden)

	// ErrDuplicatePhone 手机号已被其他账号绑定（上线前整改 P2，换绑流程）：
	// 手机号全局唯一（部分唯一索引 uk_accounts_phone 兜底）；409
	ErrDuplicatePhone = httpx.NewBiz("DUPLICATE_PHONE", "该手机号已被其他账号绑定", http.StatusConflict)

	// ErrPhoneInvalid 手机号格式非法（与 sms 域 AUTH_PHONE_INVALID 同码同文案：
	// iam 不反向依赖认证域，就地复制口径）
	ErrPhoneInvalid = httpx.NewBiz("AUTH_PHONE_INVALID", "手机号格式不正确", http.StatusBadRequest)
)
