package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	iammodel "evolyn/internal/platform/iam/model"
	tenantmodel "evolyn/internal/platform/tenant/model"
)

// stubTx 单测事务桩：不执行事务函数，直接返回数据库不可用错误，
// 用于让「参数合法」的用例止步于 Open 之前，从而只验证 SelfOpen 的参数门禁
type stubTx struct{}

func (stubTx) WithinTransaction(_ context.Context, _ func(ctx context.Context) error) error {
	return errors.New("stub: database unavailable in unit test")
}

// TestSelfOpenNameValidation 自助开通的名称与 owner 校验：名称 2-50 字符
// （去首尾空白）、owner 必填；非法参数在触达事务前直接拒绝
func TestSelfOpenNameValidation(t *testing.T) {
	svc := &tenantService{tx: stubTx{}}
	ctx := context.Background()

	cases := []struct {
		name  string
		valid bool
	}{
		{"", false},
		{"  ", false},
		{"我", false},                     // 单字符过短
		{" evolyn团队 ", true},             // 去空白后合法
		{strings.Repeat("团", 51), false}, // 51 字符过长
		{strings.Repeat("团", 50), true},
	}

	for _, tc := range cases {
		_, err := svc.SelfOpen(ctx, 1, tc.name, tenantmodel.OnboardingConfig{})
		if tc.valid {
			// 合法名称应通过参数校验：错误只可能来自数据库桩，而非名称/owner 门禁
			if err != nil && (strings.Contains(err.Error(), "tenant name") || strings.Contains(err.Error(), "owner")) {
				t.Errorf("name %q should pass validation, got: %v", tc.name, err)
			}
		} else if err == nil || !strings.Contains(err.Error(), "tenant name") {
			t.Errorf("name %q should be rejected by name validation, got err=%v", tc.name, err)
		}
	}

	// owner 账号缺失：名称合法也应拒绝
	if _, err := svc.SelfOpen(ctx, 0, "evolyn团队", tenantmodel.OnboardingConfig{}); err == nil || !strings.Contains(err.Error(), "owner") {
		t.Errorf("empty owner account should be rejected, got err=%v", err)
	}

	// SelfOpenInTx 与 SelfOpen 同一参数门禁（事务内组合通道同样先拒无效参数）
	if _, err := svc.SelfOpenInTx(ctx, 1, "我", tenantmodel.OnboardingConfig{}); err == nil || !strings.Contains(err.Error(), "tenant name") {
		t.Errorf("SelfOpenInTx should reject invalid name, got err=%v", err)
	}
	if _, err := svc.SelfOpenInTx(ctx, 0, "evolyn团队", tenantmodel.OnboardingConfig{}); err == nil || !strings.Contains(err.Error(), "owner") {
		t.Errorf("SelfOpenInTx should reject empty owner, got err=%v", err)
	}
}

// TestSelfOpenInTxOpensWithoutAudit 事务内自助开通：开通结果与 SelfOpen
// 同构（免费版/企业画像/owner 成员继承账号昵称），但**不记审计**——审计
// 由调用方（注册编排）在外层事务提交后补记，避免回滚留下假流水
func TestSelfOpenInTxOpensWithoutAudit(t *testing.T) {
	store, tenantRepo, iam, audit := newOpenFixtures()
	// 预置 owner 账号：SelfOpenInTx 语义是「账号由调用方先行注册」，
	// 昵称已由编排方先行落库（注册向导「怎么称呼你」）
	owner := &iammodel.Account{ID: store.nextID(), Name: "u-abcd1234", Nickname: "张三"}
	store.accounts[owner.ID] = owner
	svc := newOpenService(store, openRollbackTx{store}, tenantRepo, iam, audit)

	tenant, err := svc.SelfOpenInTx(context.Background(), owner.ID, " evolyn团队 ", tenantmodel.OnboardingConfig{Industry: "互联网/软件"})
	if err != nil {
		t.Fatalf("SelfOpenInTx should succeed, got: %v", err)
	}

	if tenant.Name != "evolyn团队" {
		t.Errorf("tenant name should be trimmed, got %q", tenant.Name)
	}
	if tenant.Plan != tenantmodel.PlanFree {
		t.Errorf("plan should default to free, got %q", tenant.Plan)
	}
	if tenant.Config.Onboarding.Industry != "互联网/软件" {
		t.Errorf("onboarding profile should be written into tenant config, got %+v", tenant.Config.Onboarding)
	}
	if tenant.OwnerAccountId == nil || *tenant.OwnerAccountId != owner.ID {
		t.Errorf("owner account should be %d, got %v", owner.ID, tenant.OwnerAccountId)
	}
	if audit.entries != 0 {
		t.Errorf("SelfOpenInTx must not record audit (caller records after commit), got %d entries", audit.entries)
	}
	if len(store.users) != 1 {
		t.Fatalf("owner member should be created, got %d members", len(store.users))
	}
	for _, member := range store.users {
		if member.Nickname != "张三" {
			t.Errorf("owner member should inherit account nickname, got %q", member.Nickname)
		}
		if member.TenantID != tenant.ID {
			t.Errorf("owner member should belong to the new tenant %d, got %d", tenant.ID, member.TenantID)
		}
	}
}

// TestGenerateTenantCode 编码格式与唯一性：t- 前缀 + 8 位 hex，批量生成不重复
func TestGenerateTenantCode(t *testing.T) {
	pattern := regexp.MustCompile(`^t-[0-9a-f]{8}$`)

	const count = 1000
	seen := make(map[string]struct{}, count)
	for range count {
		code := generateTenantCode()
		if !pattern.MatchString(code) {
			t.Fatalf("code %q does not match t-xxxxxxxx pattern", code)
		}
		if _, dup := seen[code]; dup {
			t.Fatalf("duplicate code generated: %s", code)
		}
		seen[code] = struct{}{}
	}
}
