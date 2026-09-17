package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"evolyn/internal/infrastructure"
	securitymodel "evolyn/internal/platform/auth/security/model"
	securityrepository "evolyn/internal/platform/auth/security/repository"
	iammodel "evolyn/internal/platform/iam/model"
	iamrepository "evolyn/internal/platform/iam/repository"
	iamservice "evolyn/internal/platform/iam/service"
	"evolyn/internal/testsupport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
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

// SEC-ACCOUNT-002：两个设备同时完成登录时，账号行锁必须将会话签发串行化，
// 最终只能保留一个活跃会话。该用例刻意并发调用真实仓储，不能由内存桩替代。
func TestAccountSecurityIntegrationConcurrentSingleSessionLoginKeepsOneSession(t *testing.T) {
	db, sessions, sessionService, account := newSessionIntegrationEnv(t)
	ctx := context.Background()

	securityService := NewSecurityService(
		infrastructure.NewTxManager(db), securityrepository.NewSettingsRepository(db), nil, nil, sessions, nil,
	)
	require.NoError(t, securityService.UpdateSingleSession(ctx, account.ID, "", true))

	type issueResult struct {
		sid string
		err error
	}
	results := make(chan issueResult, 2)
	start := make(chan struct{})
	var wait sync.WaitGroup
	for _, sid := range []string{"security-concurrent-one", "security-concurrent-two"} {
		wait.Add(1)
		go func(sid string) {
			defer wait.Done()
			<-start
			_, err := sessionService.Issue(ctx, IssueRequest{
				SID: sid, AccountID: account.ID, AuthMethod: securitymodel.AuthMethodPassword,
			})
			results <- issueResult{sid: sid, err: err}
		}(sid)
	}
	close(start)
	wait.Wait()
	close(results)

	for result := range results {
		require.NoErrorf(t, result.err, "issue session %s", result.sid)
	}
	active, err := sessions.ListActiveByAccount(ctx, account.ID)
	require.NoError(t, err)
	require.Len(t, active, 1)

	validCount := 0
	replacedCount := 0
	for _, sid := range []string{"security-concurrent-one", "security-concurrent-two"} {
		err := sessionService.Validate(ctx, sid, 1)
		if err == nil {
			validCount++
		} else if errors.Is(err, ErrSessionReplaced) {
			replacedCount++
		} else {
			t.Fatalf("validate session %s: %v", sid, err)
		}
	}
	assert.Equal(t, 1, validCount)
	assert.Equal(t, 1, replacedCount)
}

// SEC-ACCOUNT-003：租户切换只更新既有会话的 token_version；不能创建另一台
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

// SEC-ACCOUNT-004：TOTP 计数器是重放防线。相同时间步只能被数据库条件更新
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

// SEC-ACCOUNT-005：密码变更是凭据级安全事件。除推进 account.session_version
// 使旧 JWT 立即失效外，还必须撤销除当前设备外的 SID，保证设备会话列表与
// 认证实际状态一致。
func TestAccountSecurityIntegrationPasswordChangeRevokesOtherSessions(t *testing.T) {
	db := testsupport.NewPostgres(t)
	ctx := context.Background()

	hashed, err := bcrypt.GenerateFromPassword([]byte("right-old"), bcrypt.DefaultCost)
	require.NoError(t, err)
	account := &iammodel.Account{Name: "security-password-account", Password: string(hashed)}
	require.NoError(t, db.Create(account).Error)

	sessions := securityrepository.NewSessionRepository(db)
	settings := securityrepository.NewSettingsRepository(db)
	tx := infrastructure.NewTxManager(db)
	sessionService := NewSessionService(tx, settings, sessions)
	_, err = sessionService.Issue(ctx, IssueRequest{
		SID: "security-password-current", AccountID: account.ID, AuthMethod: securitymodel.AuthMethodPassword,
	})
	require.NoError(t, err)
	_, err = sessionService.Issue(ctx, IssueRequest{
		SID: "security-password-other", AccountID: account.ID, AuthMethod: securitymodel.AuthMethodPassword,
	})
	require.NoError(t, err)

	iamRepos := iamrepository.NewRepositories(db, testsupport.DisabledRedis())
	accountService := iamservice.NewAccountService(tx, iamRepos.Account(), nil, nil, nil, nil, sessions)
	require.NoError(t, accountService.ChangePassword(
		ctx, account.ID, "security-password-current", "right-old", "newpass123",
	))

	current, err := sessions.GetBySID(ctx, "security-password-current")
	require.NoError(t, err)
	assert.Nil(t, current.RevokedAt)

	other, err := sessions.GetBySID(ctx, "security-password-other")
	require.NoError(t, err)
	require.NotNil(t, other.RevokedAt)
	require.NotNil(t, other.RevokeReason)
	assert.Equal(t, securitymodel.RevokePasswordChanged, *other.RevokeReason)

	updated, err := iamRepos.Account().GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, updated.SessionVersion)
}
