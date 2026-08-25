package service

import (
	"context"
	"net/http"
	"time"

	kernel "evolyn/internal/model"
	secmodel "evolyn/internal/platform/auth/security/model"
	"evolyn/internal/platform/auth/security/repository"
	"evolyn/internal/platform/httpx"

	"errors"
	"gorm.io/gorm"
)

// 会话校验稳定错误码（ADR-008 出网；前端按 errCode 分支，
// AUTH_SESSION_REPLACED 触发「被新登录挤出」统一提示）
var (
	ErrSessionReplaced = httpx.NewBiz("AUTH_SESSION_REPLACED", "账号已在其他设备登录，当前会话已下线", http.StatusUnauthorized)
	ErrSessionRevoked  = httpx.NewBiz("AUTH_SESSION_REVOKED", "会话已失效，请重新登录", http.StatusUnauthorized)
	ErrSessionExpired  = httpx.NewBiz("AUTH_SESSION_EXPIRED", "会话已过期，请重新登录", http.StatusUnauthorized)
	ErrSessionStale    = httpx.NewBiz("AUTH_SESSION_STALE", "会话已更新，请使用最新令牌", http.StatusUnauthorized)
)

// touchThreshold LastSeenAt 刷新节流：验证时顺带触摸，但至多 5 分钟一次，
// 避免每请求一次写放大
const touchThreshold = 5 * time.Minute

// SessionTTL 会话有效期与 JWT 对齐（7 天）
const SessionTTL = 7 * 24 * time.Hour

// TxManager 会话签发的事务边界（infrastructure.TxManager 满足）
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// SessionService 会话签发与校验（ADR-009 第 3 步）：登录链路统一入口
type SessionService interface {
	// Issue 登录成功后签发会话：事务内锁账号行 → 单会话设置则撤销其他
	// 活跃会话 → 建新会话。返回携带 sid 的会话供 JWT 签发
	Issue(ctx context.Context, req IssueRequest) (*secmodel.AccountSession, error)
	// Validate 认证中间件校验：存在、未撤销、未过期、token_version 一致；
	// sid 为空（存量令牌兼容期）直接放行
	Validate(ctx context.Context, sid string, tokenVersion int64) error
	// Revoke 登出/管理端撤销
	Revoke(ctx context.Context, sid, reason string) error
	// SwitchBump 租户切换：复用 sid 递增 token_version（不算新设备），
	// 返回更新后的会话供重签
	SwitchBump(ctx context.Context, sid string) (*secmodel.AccountSession, error)
}

// IssueRequest 会话签发请求（登录链路的请求元数据）
type IssueRequest struct {
	AccountID  uint
	AuthMethod string // password / sms / oauth / register
	IP         string
	UserAgent  string
	Location   string
}

type sessionService struct {
	tx       TxManager
	settings repository.SettingsRepository
	sessions repository.SessionRepository
}

// NewSessionService 会话服务装配（server.go 调用）
func NewSessionService(tx TxManager, settings repository.SettingsRepository, sessions repository.SessionRepository) SessionService {
	return &sessionService{tx: tx, settings: settings, sessions: sessions}
}

func (s *sessionService) Issue(ctx context.Context, req IssueRequest) (*secmodel.AccountSession, error) {
	sid, err := secmodel.NewSID()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	session := &secmodel.AccountSession{
		SID:          sid,
		AccountID:    req.AccountID,
		TokenVersion: 1,
		AuthMethod:   req.AuthMethod,
		IP:           req.IP,
		UserAgent:    req.UserAgent,
		Location:     req.Location,
		CreatedAt:    kernel.JSONTime(now),
		LastSeenAt:   kernel.JSONTime(now),
		ExpiresAt:    kernel.JSONTime(now.Add(SessionTTL)),
	}

	// 多步写走统一事务（FIX-020/021 口径）：锁行 → 读设置 → 撤销他人 → 建会话。
	// 并发登录经行锁串行化，禁止同时登录时只留下后提交者的会话
	err = s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		if err := s.settings.LockAccountRow(tctx, req.AccountID); err != nil {
			return err
		}
		settings, err := s.settings.Get(tctx, req.AccountID)
		if err != nil {
			return err
		}
		if settings.SingleSessionEnabled {
			// 新会话尚未入库：撤销全部活跃会话（exceptSID 为空即全量）
			if _, err := s.sessions.RevokeOthers(tctx, req.AccountID, "", secmodel.RevokeReplaced); err != nil {
				return err
			}
		}
		_, err = s.sessions.Create(tctx, session)
		return err
	})
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (s *sessionService) Validate(ctx context.Context, sid string, tokenVersion int64) error {
	if sid == "" {
		// 存量无 sid 令牌兼容期放行（第 3 步上线后新签发令牌均带 sid）
		return nil
	}

	session, err := s.sessions.GetBySID(ctx, sid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 有 sid 的令牌查无会话行：已撤销并被清理
			return ErrSessionRevoked
		}
		return err
	}

	if session.RevokedAt != nil {
		if session.RevokeReason != nil && *session.RevokeReason == secmodel.RevokeReplaced {
			return ErrSessionReplaced
		}
		return ErrSessionRevoked
	}
	if time.Time(session.ExpiresAt).Before(time.Now()) {
		return ErrSessionExpired
	}
	if session.TokenVersion != tokenVersion {
		// 切换租户后旧令牌：客户端应已收到新令牌
		return ErrSessionStale
	}

	// 节流触摸：至多 5 分钟一次（best-effort）
	if time.Since(time.Time(session.LastSeenAt)) > touchThreshold {
		_ = s.sessions.TouchLastSeen(ctx, sid)
	}
	return nil
}

func (s *sessionService) Revoke(ctx context.Context, sid, reason string) error {
	return s.sessions.Revoke(ctx, sid, reason)
}

func (s *sessionService) SwitchBump(ctx context.Context, sid string) (*secmodel.AccountSession, error) {
	// 切换只做存活性校验（撤销/过期），不比对版本：切换者可能持本会话
	// 任一版本的令牌，递增后统一下发最新版
	if err := s.validateAlive(ctx, sid); err != nil {
		return nil, err
	}

	if _, err := s.sessions.BumpVersion(ctx, sid); err != nil {
		return nil, err
	}
	// 版本已由 BumpVersion 落库，重读最新态供重签
	return s.sessions.GetBySID(ctx, sid)
}

// validateAlive 会话存在性/撤销/过期校验（不比对令牌版本）
func (s *sessionService) validateAlive(ctx context.Context, sid string) error {
	if sid == "" {
		return ErrSessionRevoked
	}
	session, err := s.sessions.GetBySID(ctx, sid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSessionRevoked
		}
		return err
	}
	if session.RevokedAt != nil {
		if session.RevokeReason != nil && *session.RevokeReason == secmodel.RevokeReplaced {
			return ErrSessionReplaced
		}
		return ErrSessionRevoked
	}
	if time.Time(session.ExpiresAt).Before(time.Now()) {
		return ErrSessionExpired
	}
	return nil
}
