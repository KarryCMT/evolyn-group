package service

import (
	"context"
	"errors"
	"testing"

	auditservice "evolyn/internal/platform/audit/service"
	iammodel "evolyn/internal/platform/iam/model"
	iamservice "evolyn/internal/platform/iam/service"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantservice "evolyn/internal/platform/tenant/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- 注册编排测试桩：内嵌域服务接口零实现，仅覆写 Complete 链路触及的方法 ----

// fakeAccounts 账号服务桩：RegisterByPhone 返回预置账号与 created 标记，
// UpdateProfile/ListTenants/SwitchTenant 记录调用供编排断言
type fakeAccounts struct {
	iamservice.AccountService
	account     *iammodel.Account             // 注册/复用的账号（含 ID 与默认昵称）
	created     bool                          // 是否本次新建（false=已注册等价登录）
	memberships []iamservice.TenantMembership // 名下成员关系（IsOwner 驱动租户复用决策）
	member      *iammodel.User                // SwitchTenant 解析的 owner 成员
	updated     []*iammodel.Account           // UpdateProfile 收到的账号（画像断言）
	switched    []uint                        // SwitchTenant 收到的租户 ID
}

func (f *fakeAccounts) RegisterByPhone(_ context.Context, phone string) (*iammodel.Account, *iammodel.User, bool, error) {
	f.account.Phone = phone
	// 已注册路径返回登录优选成员（真实实现经 AuthByPhone→resolveLoginMember）；
	// 新账号路径该成员随后被 SwitchTenant 的 owner 成员取代
	return f.account, f.member, f.created, nil
}

func (f *fakeAccounts) UpdateProfile(_ context.Context, account *iammodel.Account) (*iammodel.Account, error) {
	f.updated = append(f.updated, account)
	return account, nil
}

func (f *fakeAccounts) ListTenants(context.Context, uint) ([]iamservice.TenantMembership, error) {
	return f.memberships, nil
}

func (f *fakeAccounts) SwitchTenant(_ context.Context, _, tenantID uint) (*iammodel.Account, *iammodel.User, error) {
	f.switched = append(f.switched, tenantID)
	return f.account, f.member, nil
}

// fakeTenants 租户服务桩：只覆写 SelfOpenInTx，记录开通调用并可注入失败
type fakeTenants struct {
	tenantservice.TenantService
	openedOwners []uint   // 每次开通收到的 owner 账号 ID
	openedNames  []string // 每次开通收到的企业名称
	tenant       *tenantmodel.Tenant
	err          error
}

func (f *fakeTenants) SelfOpenInTx(_ context.Context, ownerAccountID uint, name string, _ tenantmodel.OnboardingConfig) (*tenantmodel.Tenant, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.openedOwners = append(f.openedOwners, ownerAccountID)
	f.openedNames = append(f.openedNames, name)
	return f.tenant, nil
}

// fakeTx 事务桩：fn 失败置回滚标记（数据回滚语义由 infrastructure 的
// 真库集成测试验证，此处验证失败向编排调用方传播）
type fakeTx struct{ rolledBack bool }

func (f *fakeTx) WithinTransaction(_ context.Context, fn func(context.Context) error) error {
	if err := fn(context.Background()); err != nil {
		f.rolledBack = true
		return err
	}
	return nil
}

// fakeAudit 审计桩：验证「仅新开通租户在提交后补记一条流水」
type fakeAudit struct{ entries []auditservice.Entry }

func (f *fakeAudit) Record(_ context.Context, e auditservice.Entry) { f.entries = append(f.entries, e) }

// registrationFixtures 注册编排测试夹具集合（结构体返回：各用例按需取用，
// 避免 5 返回值多空位声明触发 dogsled）
type registrationFixtures struct {
	svc      *registrationService
	accounts *fakeAccounts
	tenants  *fakeTenants
	tx       *fakeTx
	audit    *fakeAudit
}

func newRegistrationFixtures() *registrationFixtures {
	accounts := &fakeAccounts{
		account: &iammodel.Account{ID: 7, Nickname: "138****1234"},
		member:  &iammodel.User{ID: 9, Nickname: "张三"},
	}
	tenants := &fakeTenants{
		tenant: &tenantmodel.Tenant{ID: 500, Code: "t-deadbeef", Name: "evolyn团队", Plan: tenantmodel.PlanFree},
	}
	tx := &fakeTx{}
	audit := &fakeAudit{}
	svc := NewRegistrationService(tx, accounts, tenants, audit).(*registrationService)
	return &registrationFixtures{svc: svc, accounts: accounts, tenants: tenants, tx: tx, audit: audit}
}

func registrationReq() *RegistrationRequest {
	return &RegistrationRequest{
		Phone:            "13800001234",
		Nickname:         "张三",
		Onboarding:       iammodel.AccountOnboarding{Role: "ceo", Channel: "zhihu"},
		TenantName:       "evolyn团队",
		TenantOnboarding: tenantmodel.OnboardingConfig{Demand: "低代码应用搭建", Industry: "互联网/软件"},
	}
}

// TestCompleteNewPhone 新手机号全链路：免密注册 → 画像落账号 → 事务内开通
// 租户 → 解析 owner 成员 → 提交后补记开通审计
func TestCompleteNewPhone(t *testing.T) {
	fx := newRegistrationFixtures()
	svc, accounts, tenants, audit := fx.svc, fx.accounts, fx.tenants, fx.audit
	// 名下仅有默认租户成员关系（非 owner）：应走开通而非复用
	accounts.memberships = []iamservice.TenantMembership{
		{TenantID: tenantmodel.DefaultTenantID, IsOwner: false},
	}
	accounts.created = true

	result, err := svc.Complete(context.Background(), registrationReq())
	require.NoError(t, err)

	assert.True(t, result.Created)
	assert.Same(t, accounts.member, result.Member, "令牌绑定新租户 owner 成员")
	assert.Equal(t, uint(500), accounts.switched[0], "成员解析指向新开通租户")

	// 账号画像：昵称与角色/渠道随注册一次性落库
	require.Len(t, accounts.updated, 1)
	assert.Equal(t, "张三", accounts.updated[0].Nickname)
	assert.Equal(t, iammodel.AccountOnboarding{Role: "ceo", Channel: "zhihu"}, accounts.updated[0].Onboarding)

	// 租户开通：企业名称与 owner 账号正确传递
	require.Len(t, tenants.openedOwners, 1)
	assert.Equal(t, uint(7), tenants.openedOwners[0])
	assert.Equal(t, "evolyn团队", tenants.openedNames[0])

	// 审计：提交成功后为新开通的租户补记一条开通流水
	require.Len(t, audit.entries, 1)
	entry := audit.entries[0]
	assert.Equal(t, "tenant", entry.Module)
	assert.Equal(t, "create", entry.Action)
	assert.Equal(t, uint(500), entry.TenantID)
}

// TestCompleteExistingAccountRestoresSession 已注册手机号再走注册（P1-1）：
// 仅恢复登录会话——不更新昵称/画像、不创建团队、不切换租户、不落审计
func TestCompleteExistingAccountRestoresSession(t *testing.T) {
	fx := newRegistrationFixtures()
	svc, accounts, tenants, audit := fx.svc, fx.accounts, fx.tenants, fx.audit
	accounts.created = false
	accounts.memberships = []iamservice.TenantMembership{
		{TenantID: tenantmodel.DefaultTenantID, IsOwner: false},
		{TenantID: 300, Code: "t-own", Name: "既有团队", IsOwner: true},
	}
	accounts.member.TenantID = 300

	result, err := svc.Complete(context.Background(), registrationReq())
	require.NoError(t, err)

	assert.False(t, result.Created)
	assert.Same(t, accounts.member, result.Member, "已注册路径直接用登录优选成员")
	assert.Empty(t, accounts.updated, "已注册不得覆盖昵称/画像")
	assert.Empty(t, accounts.switched, "已注册不做租户切换解析")
	assert.Empty(t, tenants.openedOwners, "已注册不得创建团队")
	assert.Empty(t, audit.entries, "恢复会话无业务变更，不落审计")
}

// TestCompleteExistingAccountWithoutOwnedTenantDoesNotOpen 已注册但名下
// 无自有租户（如仅被邀请的成员再走注册页）：按 P1-1 不再借注册向导补开
// 团队（「继续 onboarding/建团」将走受控接口），仅恢复会话
func TestCompleteExistingAccountWithoutOwnedTenantDoesNotOpen(t *testing.T) {
	fx := newRegistrationFixtures()
	svc, tenants := fx.svc, fx.tenants

	_, err := svc.Complete(context.Background(), registrationReq())
	require.NoError(t, err)
	assert.Empty(t, tenants.openedOwners, "已注册账号不得经注册向导开通租户")
}

// TestCompleteTenantOpenFailFailsWhole 新手机号注册时租户开通失败：错误
// 整体上抛（真实链路下账号/画像随事务回滚，不留半注册数据），且不落审计
func TestCompleteTenantOpenFailFailsWhole(t *testing.T) {
	fx := newRegistrationFixtures()
	svc, accounts, tenants, tx, audit := fx.svc, fx.accounts, fx.tenants, fx.tx, fx.audit
	accounts.created = true
	tenants.err = errors.New("db: insert tenants failed")

	_, err := svc.Complete(context.Background(), registrationReq())
	require.Error(t, err)
	assert.True(t, tx.rolledBack, "失败路径事务回滚")
	assert.Empty(t, audit.entries, "失败路径不落审计")
}

// TestCompleteEmptyNicknameKeepsDefault 新账号昵称空串：保留注册默认昵称
// （脱敏手机号），不用空值覆盖
func TestCompleteEmptyNicknameKeepsDefault(t *testing.T) {
	fx := newRegistrationFixtures()
	svc, accounts := fx.svc, fx.accounts
	accounts.created = true
	req := registrationReq()
	req.Nickname = ""

	_, err := svc.Complete(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, accounts.updated, 1)
	assert.Equal(t, "138****1234", accounts.updated[0].Nickname)
}
