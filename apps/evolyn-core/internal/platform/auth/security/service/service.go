package service

import (
	"context"
	"errors"

	"evolyn/internal/platform/auth/security/model"
	"evolyn/internal/platform/auth/security/repository"
	"evolyn/internal/platform/httpx"

	"gorm.io/gorm"
)

type securityService struct {
	settings repository.SettingsRepository
	factors  repository.FactorRepository
	recovery repository.RecoveryRepository
	sessions repository.SessionRepository
	events   repository.EventRepository
}

// NewSecurityService 安全服务装配（server.go 调用；events 允许 nil 便于测试）
func NewSecurityService(
	settings repository.SettingsRepository,
	factors repository.FactorRepository,
	recovery repository.RecoveryRepository,
	sessions repository.SessionRepository,
	events repository.EventRepository,
) SecurityService {
	return &securityService{
		settings: settings,
		factors:  factors,
		recovery: recovery,
		sessions: sessions,
		events:   events,
	}
}

func (s *securityService) Overview(ctx context.Context, accountID uint) (*SecurityOverview, error) {
	settings, err := s.settings.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// TOTP 注册态：存在活跃且已验证的因子
	totpEnrolled := false
	if factor, err := s.factors.GetActive(ctx, accountID, model.FactorTypeTotp); err == nil {
		totpEnrolled = factor.VerifiedAt != nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	codes, err := s.recovery.ListAvailable(ctx, accountID)
	if err != nil {
		return nil, err
	}

	sessions, err := s.sessions.ListActiveByAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	return &SecurityOverview{
		MFAEnabled:             settings.MFAEnabled,
		SingleSessionEnabled:   settings.SingleSessionEnabled,
		TotpEnrolled:           totpEnrolled,
		RecoveryCodesRemaining: len(codes),
		ActiveSessions:         len(sessions),
	}, nil
}

func (s *securityService) ListSessions(ctx context.Context, accountID uint) ([]model.AccountSession, error) {
	return s.sessions.ListActiveByAccount(ctx, accountID)
}

// RevokeSession 本人踢出会话：先取会话校验归属（防越权撤销他人会话），
// 再原子撤销（revoked_at IS NULL 条件保证幂等）；成功后补安全流水
func (s *securityService) RevokeSession(ctx context.Context, accountID uint, sid string) error {
	session, err := s.sessions.GetBySID(ctx, sid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpx.ErrNotFound("会话不存在")
		}
		return err
	}
	if session.AccountID != accountID {
		// 归属不符按不存在处理，不向调用方泄露他人会话是否存在
		return httpx.ErrNotFound("会话不存在")
	}

	if err := s.sessions.Revoke(ctx, sid, model.RevokeLogout); err != nil {
		return err
	}

	if s.events != nil {
		_ = s.events.Append(ctx, &model.SecurityEvent{
			AccountID: accountID,
			EventType: "session_revoked",
			SessionID: sid,
			Metadata:  model.EventMetadata{"by": "self"},
		})
	}
	return nil
}
