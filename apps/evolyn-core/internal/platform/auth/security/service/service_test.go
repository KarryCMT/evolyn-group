package service

import (
	"context"
	"testing"
	"time"

	"evolyn/internal/platform/auth/security/model"
	"evolyn/internal/platform/auth/security/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---- 测试桩：内嵌仓储接口零实现，仅覆写用例触及的方法 ----

type fakeSettingsRepo struct {
	repository.SettingsRepository
	settings *model.SecuritySettings
}

func (f *fakeSettingsRepo) Get(_ context.Context, accountID uint) (*model.SecuritySettings, error) {
	if f.settings != nil && f.settings.AccountID == accountID {
		return f.settings, nil
	}
	return &model.SecuritySettings{AccountID: accountID}, nil
}

type fakeFactorRepo struct {
	repository.FactorRepository
	factor *model.MFAFactor
}

func (f *fakeFactorRepo) GetActive(_ context.Context, accountID uint, _ string) (*model.MFAFactor, error) {
	if f.factor != nil && f.factor.AccountID == accountID {
		return f.factor, nil
	}
	return nil, gorm.ErrRecordNotFound
}

type fakeRecoveryRepo struct {
	repository.RecoveryRepository
	codes []model.RecoveryCode
}

func (f *fakeRecoveryRepo) ListAvailable(_ context.Context, accountID uint) ([]model.RecoveryCode, error) {
	out := make([]model.RecoveryCode, 0)
	for _, c := range f.codes {
		if c.AccountID == accountID {
			out = append(out, c)
		}
	}
	return out, nil
}

type fakeSessionRepo struct {
	repository.SessionRepository
	sessions []model.AccountSession
	revoked  []string
}

func (f *fakeSessionRepo) ListActiveByAccount(_ context.Context, accountID uint) ([]model.AccountSession, error) {
	out := make([]model.AccountSession, 0)
	for _, s := range f.sessions {
		if s.AccountID == accountID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeSessionRepo) GetBySID(_ context.Context, sid string) (*model.AccountSession, error) {
	for i := range f.sessions {
		if f.sessions[i].SID == sid {
			return &f.sessions[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeSessionRepo) Revoke(_ context.Context, sid, _ string) error {
	f.revoked = append(f.revoked, sid)
	return nil
}

// ---- 用例 ----

func newSvc(settings *model.SecuritySettings, factor *model.MFAFactor,
	codes []model.RecoveryCode, sessions []model.AccountSession) (SecurityService, *fakeSessionRepo) {
	sessionRepo := &fakeSessionRepo{sessions: sessions}
	return NewSecurityService(
		&fakeTxRunner{},
		&fakeSettingsRepo{settings: settings},
		&fakeFactorRepo{factor: factor},
		&fakeRecoveryRepo{codes: codes},
		sessionRepo,
		nil,
	), sessionRepo
}

func TestOverview(t *testing.T) {
	verified := time.Now()
	svc, _ := newSvc(
		&model.SecuritySettings{AccountID: 7, MFAEnabled: true, SingleSessionEnabled: true},
		&model.MFAFactor{AccountID: 7, VerifiedAt: &verified},
		[]model.RecoveryCode{{AccountID: 7}, {AccountID: 7}, {AccountID: 9}},
		[]model.AccountSession{{AccountID: 7}, {AccountID: 7}, {AccountID: 8}},
	)

	ov, err := svc.Overview(context.Background(), 7)
	require.NoError(t, err)
	assert.True(t, ov.MFAEnabled)
	assert.True(t, ov.SingleSessionEnabled)
	assert.True(t, ov.TotpEnrolled)
	assert.Equal(t, 2, ov.RecoveryCodesRemaining, "只统计本人未用恢复码")
	assert.Equal(t, 2, ov.ActiveSessions, "只统计本人活跃会话")
}

func TestOverviewDefaults(t *testing.T) {
	// 无设置行/无因子：全关、零余量（缺省语义）
	svc, _ := newSvc(nil, nil, nil, nil)
	ov, err := svc.Overview(context.Background(), 42)
	require.NoError(t, err)
	assert.False(t, ov.MFAEnabled)
	assert.False(t, ov.TotpEnrolled)
	assert.Equal(t, 0, ov.RecoveryCodesRemaining)
}

func TestRevokeSessionOwnership(t *testing.T) {
	sessions := []model.AccountSession{{AccountID: 7, SID: "mine"}, {AccountID: 9, SID: "others"}}
	svc, sessionRepo := newSvc(nil, nil, nil, sessions)

	// 他人会话：按不存在处理且不产生撤销（不泄露他人会话存在性）
	assert.Error(t, svc.RevokeSession(context.Background(), 7, "others"))
	assert.Empty(t, sessionRepo.revoked)

	// 本人会话：撤销成功
	require.NoError(t, svc.RevokeSession(context.Background(), 7, "mine"))
	assert.Equal(t, []string{"mine"}, sessionRepo.revoked)
}

func TestRevokeSessionNotFound(t *testing.T) {
	svc, _ := newSvc(nil, nil, nil, nil)
	assert.Error(t, svc.RevokeSession(context.Background(), 7, "missing"))
}
