package service

import (
	"context"

	iammodel "evolyn/internal/platform/iam/model"
	tenantmodel "evolyn/internal/platform/tenant/model"
)

// TxManager 事务边界抽象（FIX-020，与 iam/tenant service 同口径）：具体
// 实现在 infrastructure（ctx 传播事务 session，嵌套调用加入外层事务），
// Service 只依赖最小接口，便于单测以快照/恢复模拟回滚
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// RegistrationRequest 注册向导最终提交（POST /auth/register）的全量数据：
// 三步纯前端采集（手机号+验证码 / 企业画像 / 账号画像）在「进入产品」时
// 一次性汇总上送，此前不产生任何服务端写副作用
type RegistrationRequest struct {
	Phone    string // 手机号（验证码已由控制器经 sms 域校验，即持有证明）
	Nickname string // 「怎么称呼你」：账号昵称；空串保留注册默认值（脱敏手机号）

	// Onboarding 账号画像：角色/了解渠道（「人」的属性挂账号，运营分析用）
	Onboarding iammodel.AccountOnboarding

	TenantName       string                       // 企业名称（必填，2-50 字符）
	TenantOnboarding tenantmodel.OnboardingConfig // 企业画像：需求/行业（选填，写入租户 Config）
	// PublicInviteToken 非空时加入对应租户，不再创建或复用自有租户。
	PublicInviteToken string
}

// RegistrationResult 注册完成结果：签发会话令牌所需的账号与新租户 owner 成员
type RegistrationResult struct {
	Account *iammodel.Account // 平台账号（登录身份）
	Member  *iammodel.User    // 新租户 owner 成员（令牌绑定此成员及其租户）
	Created bool              // 账号是否本次新建（false=已注册手机号等价短信登录）
}

// RegistrationService 注册编排服务（认证域）：注册向导最终提交「进入产品」
// 的单事务落库——注册账号/复用登录、账号画像、租户开通/复用、owner 成员
// 解析一步到位；向导中途放弃不留任何孤儿数据（无账号、无空壳租户）
type RegistrationService interface {
	// Complete 单事务完成注册。前置：验证码已由调用方校验。幂等：已注册
	// 手机号等价短信登录；名下已有自有租户则复用，不重复开通
	Complete(ctx context.Context, req *RegistrationRequest) (*RegistrationResult, error)
}

// MemberInvitationAccepter 是认证域接受公开成员邀请所需的最小能力，
// 由 iam 成员邀请服务实现，避免认证域依赖邀请存储细节。
type MemberInvitationAccepter interface {
	AcceptPublicLink(ctx context.Context, accountID uint, nickname, token string) (*iammodel.User, error)
}
