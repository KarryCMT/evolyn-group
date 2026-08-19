package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

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
