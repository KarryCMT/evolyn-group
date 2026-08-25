package auth

import (
	"testing"
	"time"

	secmodel "evolyn/internal/platform/auth/security/model"

	"evolyn/internal/platform/iam/model"

	"github.com/stretchr/testify/assert"
)

func newSession() (*model.Account, *model.User) {
	account := &model.Account{ID: 9, Name: "someone"}
	member := &model.User{ID: 7, AccountId: 9, Nickname: "someone"}
	member.TenantID = 3
	return account, member
}

func TestCreateToken(t *testing.T) {
	service := NewJWTService("test")

	testCases := []struct {
		name        string
		account     *model.Account
		member      *model.User
		expectedErr bool
	}{
		{
			name:        "account is nil",
			member:      &model.User{ID: 1},
			expectedErr: true,
		},
		{
			name:        "member is nil",
			account:     &model.Account{ID: 1, Name: "someone"},
			expectedErr: true,
		},
		{
			name:        "create token success",
			account:     &model.Account{ID: 9, Name: "someone"},
			member:      &model.User{ID: 7, AccountId: 9},
			expectedErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := service.CreateToken(tc.account, tc.member, nil)
			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.Empty(t, err)
				assert.NotEmpty(t, token)
			}
		})
	}
}

func TestParseToken(t *testing.T) {
	testCases := []struct {
		name        string
		token       string
		expiresAt   time.Duration
		expectedErr bool
	}{
		{
			name:        "invaild token",
			token:       "some-token",
			expectedErr: true,
		},
		{
			name:        "token expiration",
			expiresAt:   -24 * time.Hour,
			expectedErr: true,
		},
		{
			name:        "parse token success",
			expectedErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewJWTService("test")
			if tc.expiresAt != 0 {
				service.expireDuration = tc.expiresAt
			}

			if tc.token == "" {
				account, member := newSession()
				token, err := service.CreateToken(account, member, nil)
				assert.Empty(t, err)
				tc.token = token
			}

			claims, err := service.ParseToken(tc.token)
			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.Empty(t, err)
				assert.NotNil(t, claims)
			}
		})
	}
}

func TestTokenTenantRoundtrip(t *testing.T) {
	service := NewJWTService("test")

	account, member := newSession()
	account.SessionVersion = 3
	token, err := service.CreateToken(account, member, &secmodel.AccountSession{SID: "abc123", TokenVersion: 3})
	assert.Empty(t, err)

	claims, err := service.ParseToken(token)
	assert.Empty(t, err)
	// 会话五元组完整往返：账号/成员/租户/会话版本/登录名（ADR-006）。
	assert.Equal(t, uint(9), claims.AccountID)
	assert.Equal(t, uint(7), claims.MemberID)
	assert.Equal(t, uint(3), claims.TenantID)
	assert.Equal(t, uint64(3), claims.SessionVersion)
	assert.Equal(t, "someone", claims.Name)
	// 设备会话维度（ADR-009）：sid 与令牌版本完整往返
	assert.Equal(t, "abc123", claims.SID)
	assert.Equal(t, int64(3), claims.SessionTokenVersion)
}
