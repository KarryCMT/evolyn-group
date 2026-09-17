package service

import (
	"context"
	"testing"
	"time"

	"evolyn/internal/infrastructure"
	securitymodel "evolyn/internal/platform/auth/security/model"
	securityrepository "evolyn/internal/platform/auth/security/repository"
	iammodel "evolyn/internal/platform/iam/model"
	"evolyn/internal/testsupport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 账号安全关键状态必须经真实 PostgreSQL 验证，避免单测桩掩盖事务、行锁或
// 条件更新语义。未配置 TEST_PG_DSN 时 testsupport 会自动跳过本文件的用例。
func newSessionIntegrationEnv(t *testing.T) (*gorm.DB, securityrepository.SessionRepository, SessionService, *iammodel.Account) {
	t.Helper()

	db := testsupport.NewPostgres(t)
	account := &iammodel.Account{Name: "security-integration-account"}
	require.NoError(t, db.Create(account).Error)

	sessions := securityrepository.NewSessionRepository(db)
	settings := securityrepository.NewSettingsRepository(db)
	tx := infrastructure.NewTxManager(db)
	return db, sessions, NewSessionService(tx, settings, sessions), account
}

// SEC-ACCOUNT-001：开启单会话后，新设备登录必须在同一事务中撤销旧设备；
// 旧令牌应稳定返回 AUTH_SESSION_REPLACED，而不是无声继续可用。
func TestAccountSecurityIntegrationSingleSessionReplacesOldSession(t *testing.T) {
	db, sessions, sessionService, account := newSessionIntegrationEnv(t)
	ctx := context.Background()

	_, err := sessionService.Issue(ctx, IssueRequest{
		SID: "security-old-session", AccountID: account.ID, AuthMethod: securitymodel.AuthMethodPassword,
	})
	require.NoError(t, err)

	// 同一数据库上的服务切换开关。单独构造以避免测试依赖 HTTP 层的 reauth 细节。
	securityService := NewSecurityService(
		infrastructure.NewTxManager(db), securityrepository.NewSettingsRepository(db), nil, nil, sessions, nil,
	)
	require.NoError(t, securityService.UpdateSingleSession(ctx, account.ID, "security-old-session", true))

	_, err = sessionService.Issue(ctx, IssueRequest{
		SID: "security-new-session", AccountID: account.ID, AuthMethod: securitymodel.AuthMethodPassword,
	})
	require.NoError(t, err)

	oldSession, err := sessions.GetBySID(ctx, "security-old-session")
	require.NoError(t, err)
	require.NotNil(t, oldSession.RevokedAt)
	require.NotNil(t, oldSession.RevokeReason)
	assert.Equal(t, securitymodel.RevokeReplaced, *oldSession.RevokeReason)
	assert.ErrorIs(t, sessionService.Validate(ctx, "security-old-session", oldSession.TokenVersion), ErrSessionReplaced)

	active, err := sessions.ListActiveByAccount(ctx, account.ID)
	require.NoError(t, err)
	require.Len(t, active, 1)
	assert.Equal(t, "security-new-session", active[0].SID)
}

// SEC-ACCOUNT-002：租户切换只更新既有会话的 token_version；不能创建另一台
// 设备会话，否则“设备列表”和单会话策略都会产生错误结果。
func TestAccountSecurityIntegrationTenantSwitchReusesSession(t *testing.T) {
	_, sessions, sessionService, account := newSessionIntegrationEnv(t)
	ctx := context.Background()

	issued, err := sessionService.Issue(ctx, IssueRequest{
		SID: "security-switch-session", AccountID: account.ID, AuthMethod: securitymodel.AuthMethodPassword,
	})
	require.NoError(t, err)

	switched, err := sessionService.SwitchBump(ctx, issued.SID)
	require.NoError(t, err)
	assert.Equal(t, issued.SID, switched.SID)
	assert.Equal(t, issued.TokenVersion+1, switched.TokenVersion)

	active, err := sessions.ListActiveByAccount(ctx, account.ID)
	require.NoError(t, err)
	require.Len(t, active, 1)
	assert.Equal(t, issued.SID, active[0].SID)
	assert.Equal(t, switched.TokenVersion, active[0].TokenVersion)
}

// SEC-ACCOUNT-003：TOTP 计数器是重放防线。相同时间步只能被数据库条件更新
// 成功一次，后续请求必须失败，即使两个请求落在不同进程中也一样。
func TestAccountSecurityIntegrationTOTPReplayIsRejected(t *testing.T) {
	db := testsupport.NewPostgres(t)
	account := &iammodel.Account{Name: "security-totp-account"}
	require.NoError(t, db.Create(account).Error)

	factors := securityrepository.NewFactorRepository(db)
	now := time.Now()
	factor, err := factors.Create(context.Background(), &securitymodel.MFAFactor{
		AccountID:        account.ID,
		Type:             securitymodel.FactorTypeTotp,
		SecretCiphertext: "test-only-ciphertext",
		KeyVersion:       1,
		VerifiedAt:       &now,
		LastUsedCounter:  100,
	})
	require.NoError(t, err)

	consumed, err := factors.ConsumeCounter(context.Background(), factor.ID, 101)
	require.NoError(t, err)
	assert.True(t, consumed)

	consumed, err = factors.ConsumeCounter(context.Background(), factor.ID, 101)
	require.NoError(t, err)
	assert.False(t, consumed)
}
