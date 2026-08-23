package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/iam/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- 密码强度统一校验（上线前整改 P2：8-64 位字母+数字+弱口令黑名单） ----

func TestValidatePasswordStrength(t *testing.T) {
	cases := []struct {
		name    string
		pw      string
		wantErr bool
	}{
		{"过短", "ab12cd", true},
		{"超长", strings.Repeat("a1", 33), true}, // 66 位
		{"纯字母", "abcdefgh", true},
		{"纯数字", "12345678", true},
		{"弱口令黑名单", "password1", true},
		{"弱口令黑名单大写", "ADMIN123", true},
		{"合规字母数字", "evolyn2026", false},
		{"含符号的合规密码", "hello@2026", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validatePasswordStrength(c.pw)
			if c.wantErr {
				assert.Error(t, err)
				// 强度不达标必须以稳定业务码出网（ADR-008），而非裸 error 文本
				var biz *httpx.BizError
				assert.True(t, errors.As(err, &biz))
				assert.Equal(t, httpx.CodeValidation, biz.Code)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestChangePasswordRejectsWeakPassword 弱新密码在改密路径被统一校验拦截
func TestChangePasswordRejectsWeakPassword(t *testing.T) {
	accounts := newPhoneAccountRepo(&model.Account{
		ID: 10, Name: "pwd-user", Phone: "13800001111",
		PasswordInitialized: boolPtr(true),
	})
	svc := newPhoneSvc(accounts, &phoneUserRepo{members: map[uint]*model.User{}})

	// 6 位旧口径密码不再放行（长度下限 8）
	assert.Error(t, svc.ChangePassword(context.Background(), 10, "", "abc123"))
	// 纯数字同样拒绝
	assert.Error(t, svc.ChangePassword(context.Background(), 10, "", "12345678"))
}

// ---- 换绑手机号（上线前整改 P2） ----

func TestChangePhoneHappyPath(t *testing.T) {
	accounts := newPhoneAccountRepo(&model.Account{
		ID: 10, Name: "u-1", Phone: "13800001111", PasswordInitialized: boolPtr(true),
	})
	svc := newPhoneSvc(accounts, &phoneUserRepo{members: map[uint]*model.User{}})

	updated, err := svc.ChangePhone(context.Background(), 10, "13800009999")
	require.NoError(t, err)
	assert.Equal(t, "13800009999", updated.Phone)
	assert.Equal(t, "13800009999", accounts.accounts[10].Phone)
}

func TestChangePhoneDuplicateRejected(t *testing.T) {
	accounts := newPhoneAccountRepo(
		&model.Account{ID: 10, Name: "u-1", Phone: "13800001111"},
		&model.Account{ID: 20, Name: "u-2", Phone: "13800002222"},
	)
	svc := newPhoneSvc(accounts, &phoneUserRepo{members: map[uint]*model.User{}})

	_, err := svc.ChangePhone(context.Background(), 10, "13800002222")
	assert.ErrorIs(t, err, ErrDuplicatePhone)
	// 原手机号未被改动
	assert.Equal(t, "13800001111", accounts.accounts[10].Phone)
}

func TestChangePhoneInvalidFormatRejected(t *testing.T) {
	accounts := newPhoneAccountRepo(&model.Account{ID: 10, Name: "u-1", Phone: "13800001111"})
	svc := newPhoneSvc(accounts, &phoneUserRepo{members: map[uint]*model.User{}})

	for _, bad := range []string{"12345", "1380000111", "abc"} {
		_, err := svc.ChangePhone(context.Background(), 10, bad)
		assert.ErrorIs(t, err, ErrPhoneInvalid, "phone %s should be rejected", bad)
	}
}

// TestEnsurePhoneAvailable 预检口径与 ChangePhone 一致（供控制器消费验证码前调用）
func TestEnsurePhoneAvailable(t *testing.T) {
	accounts := newPhoneAccountRepo(&model.Account{ID: 20, Name: "u-2", Phone: "13800002222"})
	svc := newPhoneSvc(accounts, &phoneUserRepo{members: map[uint]*model.User{}})

	assert.NoError(t, svc.EnsurePhoneAvailable(context.Background(), "13800003333"))
	assert.ErrorIs(t, svc.EnsurePhoneAvailable(context.Background(), "13800002222"), ErrDuplicatePhone)
	assert.ErrorIs(t, svc.EnsurePhoneAvailable(context.Background(), "123"), ErrPhoneInvalid)
}
