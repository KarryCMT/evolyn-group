package service

import (
	"context"
	"strconv"

	auditservice "evolyn/internal/platform/audit/service"
	iamservice "evolyn/internal/platform/iam/service"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantservice "evolyn/internal/platform/tenant/service"
)

// registrationService 注册编排实现：组合 iam 账号服务与 tenant 租户服务，
// 全流程共享一个数据库事务（FIX-020：子服务的 WithinTransaction 嵌套调用
// 加入本事务，任一步失败整体回滚，不留「有账号无租户」的半注册状态）
type registrationService struct {
	tx       TxManager
	accounts iamservice.AccountService
	tenants  tenantservice.TenantService
	invites  MemberInvitationAccepter
	audit    auditservice.Recorder
}

// NewRegistrationService 注册编排服务装配（server.go 调用）
func NewRegistrationService(
	tx TxManager,
	accounts iamservice.AccountService,
	tenants tenantservice.TenantService,
	audit auditservice.Recorder,
	invites ...MemberInvitationAccepter,
) RegistrationService {
	var invitationAccepter MemberInvitationAccepter
	if len(invites) > 0 {
		invitationAccepter = invites[0]
	}
	return &registrationService{tx: tx, accounts: accounts, tenants: tenants, invites: invitationAccepter, audit: audit}
}

// Complete 注册向导最终提交「进入产品」的单事务编排：
//  1. 账号：免密注册（服务端随机登录名/密码 + 默认租户成员，
//     PasswordInitialized=false，密码由用户后续在个人中心首设）；
//     已注册手机号等价短信登录（created=false），重试天然幂等
//  2. 账号画像：昵称/onboarding 先落账号侧（本链路无成员上下文，默认
//     租户成员保留脱敏手机号昵称）；owner 成员随租户开通创建时继承该昵称
//  3. 租户：名下已有自有租户则复用（重试幂等，不重复占配额），否则经
//     SelfOpenInTx 在本事务内开通（不开独立事务、不记审计）
//  4. 成员：解析新租户 owner 成员，供控制器签发绑定新租户的令牌
//
// P1-1 注册语义：已注册手机号（created=false）仅恢复登录会话——
// 不更新昵称/画像、不创建团队（向导新填的企业信息与画像被忽略），
// 避免覆盖既有资料与账号/成员昵称漂移；「继续 onboarding」需求
// 将来单独设计受控接口
func (s *registrationService) Complete(ctx context.Context, req *RegistrationRequest) (*RegistrationResult, error) {
	var (
		result       *RegistrationResult
		openedTenant *tenantmodel.Tenant // 仅记录本次新开通的租户（复用路径无业务变更，不审计）
	)
	err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		account, member, created, err := s.accounts.RegisterByPhone(tctx, req.Phone)
		if err != nil {
			return err
		}

		if !created && req.PublicInviteToken == "" {
			// 已注册：恢复会话即止（RegisterByPhone 已按登录成员优选解析）
			result = &RegistrationResult{Account: account, Member: member, Created: false}
			return nil
		}

		// 昵称为空串保留注册默认值（脱敏手机号），不覆盖。
		// 已有账号经公开链接加入时同样不覆盖平台级昵称，仅使用邀请昵称
		// 写目标租户成员。
		if created && req.Nickname != "" {
			account.Nickname = req.Nickname
		}
		if created {
			account.Onboarding = req.Onboarding
			if _, err := s.accounts.UpdateProfile(tctx, account); err != nil {
				return err
			}
		}

		if req.PublicInviteToken != "" {
			if s.invites == nil {
				return iamservice.ErrMemberInvitationInvalid
			}
			member, err = s.invites.AcceptPublicLink(tctx, account.ID, req.Nickname, req.PublicInviteToken)
			if err != nil {
				return err
			}
			result = &RegistrationResult{Account: account, Member: member, Created: created}
			return nil
		}

		tenant, opened, err := s.resolveTenant(tctx, account.ID, req)
		if err != nil {
			return err
		}
		if opened {
			openedTenant = tenant
		}

		// 只读解析：取账号在新租户的成员身份（注册即绑定新租户，免 switch 一跳）
		_, member, err = s.accounts.SwitchTenant(tctx, account.ID, tenant.ID)
		if err != nil {
			return err
		}

		result = &RegistrationResult{Account: account, Member: member, Created: created}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 审计在事务提交成功后独立写入（best-effort，FIX-020 决策）：补记
	// SelfOpenInTx 留空的开通流水（口径与 tenantService.Open 一致）
	if s.audit != nil && openedTenant != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "tenant", Action: "create", ResourceType: "tenant",
			ResourceID: strconv.FormatUint(uint64(openedTenant.ID), 10),
			TenantID:   openedTenant.ID,
			After:      map[string]any{"code": openedTenant.Code, "name": openedTenant.Name, "plan": openedTenant.Plan},
		})
	}
	return result, nil
}

// resolveTenant 租户决策：账号名下已有自有租户则复用（向导重试幂等），
// 否则在本事务内自助开通。复用路径仅回填后续所需的租户 ID（签发与审计
// 都不再需要完整租户实体）
func (s *registrationService) resolveTenant(ctx context.Context, accountID uint, req *RegistrationRequest) (*tenantmodel.Tenant, bool, error) {
	memberships, err := s.accounts.ListTenants(ctx, accountID)
	if err != nil {
		return nil, false, err
	}
	for _, m := range memberships {
		if m.IsOwner {
			return &tenantmodel.Tenant{ID: m.TenantID}, false, nil
		}
	}

	tenant, err := s.tenants.SelfOpenInTx(ctx, accountID, req.TenantName, req.TenantOnboarding)
	if err != nil {
		return nil, false, err
	}
	return tenant, true, nil
}
