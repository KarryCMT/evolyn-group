package service

import (
	"context"
	"testing"
	"time"

	kernel "evolyn/internal/model"
	secmodel "evolyn/internal/platform/auth/security/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"evolyn/internal/platform/auth/security/repository"
)

// ---- 会话服务测试桩 ----

type fakeTxRunner struct{ inTx bool }

func (f *fakeTxRunner) WithinTransaction(_ context.Context, fn func(ctx context.Context) error) error {
	f.inTx = true
	return fn(context.Background())
}

type fakeIssueSettings struct {
	repository.SettingsRepository
	locked   []uint
	settings *secmodel.SecuritySettings
}

func (f *fakeIssueSettings) Get(_ context.Context, accountID uint) (*secmodel.SecuritySettings, error) {
	if f.settings != nil {
		return f.settings, nil
	}
	return &secmodel.SecuritySettings{AccountID: accountID}, nil
}

func (f *fakeIssueSettings) LockAccountRow(_ context.Context, accountID uint) error {
	f.locked = append(f.locked, accountID)
	return nil
}

type fakeIssueSessions struct {
	repository.SessionRepository

	created []*secmodel.AccountSession
	revoked []string // sid:reason
	bumped  int
	store   map[string]*secmodel.AccountSession
	touched []string
}

func (f *fakeIssueSessions) Create(_ context.Context, s *secmodel.AccountSession) (*secmodel.AccountSession, error) {
	f.created = append(f.created, s)
	if f.store == nil {
		f.store = map[string]*secmodel.AccountSession{}
	}
	f.store[s.SID] = s
	return s, nil
}

func (f *fakeIssueSessions) GetBySID(_ context.Context, sid string) (*secmodel.AccountSession, error) {
	if s, ok := f.store[sid]; ok {
		return s, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeIssueSessions) Revoke(_ context.Context, sid, reason string) error {
	f.revoked = append(f.revoked, sid+":"+reason)
	if s, ok := f.store[sid]; ok {
		now := time.Now()
		s.RevokedAt = &now
		s.RevokeReason = &reason
	}
	return nil
}

func (f *fakeIssueSessions) RevokeOthers(_ context.Context, _ uint, _, reason string) (int64, error) {
	f.revoked = append(f.revoked, "others:"+reason)
	return 1, nil
}

func (f *fakeIssueSessions) BumpVersion(_ context.Context, sid string) (int64, error) {
	s := f.store[sid]
	s.TokenVersion++
	f.bumped++
	return s.TokenVersion, nil
}

func (f *fakeIssueSessions) TouchLastSeen(_ context.Context, sid string) error {
	f.touched = append(f.touched, sid)
	return nil
}

func newSessionSvc(settings *secmodel.SecuritySettings) (SessionService, *fakeIssueSettings, *fakeIssueSessions) {
	st := &fakeIssueSettings{settings: settings}
	ss := &fakeIssueSessions{store: map[string]*secmodel.AccountSession{}}
	return NewSessionService(&fakeTxRunner{}, st, ss), st, ss
}

// ---- 用例 ----

func TestIssueSingleSessionRevokesOthers(t *testing.T) {
	svc, settings, sessions := newSessionSvc(
		&secmodel.SecuritySettings{AccountID: 7, SingleSessionEnabled: true})

	session, err := svc.Issue(context.Background(), IssueRequest{AccountID: 7, AuthMethod: secmodel.AuthMethodPassword})
	require.NoError(t, err)

	// 账号行已锁、旧会话被挤出、新会话入库
	assert.Equal(t, []uint{7}, settings.locked, "签发前锁账号行")
	assert.Contains(t, sessions.revoked, "others:"+secmodel.RevokeReplaced)
	require.Len(t, sessions.created, 1)
	assert.Equal(t, session.SID, sessions.created[0].SID)
	assert.Equal(t, secmodel.AuthMethodPassword, session.AuthMethod)
	assert.NotEmpty(t, session.SID)
	assert.Equal(t, int64(1), session.TokenVersion)
}

func TestIssueMultiSessionKeepsOthers(t *testing.T) {
	svc, _, sessions := newSessionSvc(nil) // 缺省：单会话关闭

	_, err := svc.Issue(context.Background(), IssueRequest{AccountID: 7})
	require.NoError(t, err)
	assert.NotContains(t, sessions.revoked, "others:"+secmodel.RevokeReplaced)
}

func TestIssueUsesPreparedSID(t *testing.T) {
	svc, _, sessions := newSessionSvc(nil)

	session, err := svc.Issue(context.Background(), IssueRequest{
		SID:       "prepared-session-id",
		AccountID: 7,
	})
	require.NoError(t, err)
	assert.Equal(t, "prepared-session-id", session.SID)
	require.Len(t, sessions.created, 1)
	assert.Equal(t, "prepared-session-id", sessions.created[0].SID)
}

func TestValidateBranches(t *testing.T) {
	svc, _, sessions := newSessionSvc(nil)

	now := time.Now()
	replaced := secmodel.RevokeReplaced
	loggedOut := secmodel.RevokeLogout

	// 活跃会话 + 版本一致 → 通过
	live := &secmodel.AccountSession{SID: "live", AccountID: 7, TokenVersion: 2,
		LastSeenAt: kernel.JSONTime(now.Add(-time.Minute)), ExpiresAt: kernel.JSONTime(now.Add(time.Hour))}
	sessions.store["live"] = live
	require.NoError(t, svc.Validate(context.Background(), "live", 2))

	// 版本不一致（切换后旧令牌）→ STALE
	err := svc.Validate(context.Background(), "live", 1)
	assert.ErrorIs(t, err, ErrSessionStale)

	// 被挤出 → REPLACED
	sessions.store["kicked"] = &secmodel.AccountSession{SID: "kicked", TokenVersion: 1,
		RevokedAt: &now, RevokeReason: &replaced, ExpiresAt: kernel.JSONTime(now.Add(time.Hour))}
	assert.ErrorIs(t, svc.Validate(context.Background(), "kicked", 1), ErrSessionReplaced)

	// 其他原因撤销 → REVOKED
	sessions.store["out"] = &secmodel.AccountSession{SID: "out", TokenVersion: 1,
		RevokedAt: &now, RevokeReason: &loggedOut, ExpiresAt: kernel.JSONTime(now.Add(time.Hour))}
	assert.ErrorIs(t, svc.Validate(context.Background(), "out", 1), ErrSessionRevoked)

	// 过期 → EXPIRED
	sessions.store["old"] = &secmodel.AccountSession{SID: "old", TokenVersion: 1, ExpiresAt: kernel.JSONTime(now.Add(-time.Minute))}
	assert.ErrorIs(t, svc.Validate(context.Background(), "old", 1), ErrSessionExpired)

	// 查无会话行 → REVOKED（已清理）
	assert.ErrorIs(t, svc.Validate(context.Background(), "missing", 1), ErrSessionRevoked)

	// 空 sid（存量兼容期）→ 放行
	require.NoError(t, svc.Validate(context.Background(), "", 0))
}

func TestSwitchBumpReusesSID(t *testing.T) {
	svc, _, sessions := newSessionSvc(nil)
	now := time.Now()
	sessions.store["sid-1"] = &secmodel.AccountSession{SID: "sid-1", AccountID: 7, TokenVersion: 4,
		LastSeenAt: kernel.JSONTime(now), ExpiresAt: kernel.JSONTime(now.Add(time.Hour))}

	session, err := svc.SwitchBump(context.Background(), "sid-1")
	require.NoError(t, err)

	// 复用 sid、版本递增：租户切换不算新设备
	assert.Equal(t, "sid-1", session.SID)
	assert.Equal(t, int64(5), session.TokenVersion)
	assert.Equal(t, 1, sessions.bumped)
	assert.Empty(t, sessions.created, "切换不得新建会话")
}
