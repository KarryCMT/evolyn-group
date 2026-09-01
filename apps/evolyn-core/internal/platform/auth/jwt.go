package auth

import (
	"fmt"
	"time"

	securitymodel "evolyn/internal/platform/auth/security/model"
	"evolyn/internal/platform/iam/model"

	"github.com/golang-jwt/jwt/v5"
)

const (
	Issuer = "evolyn"
)

// CustomClaims JWT 载荷（ADR-006 账号×成员拆分 + ADR-009 会话化）：
//   - AccountID/Name：平台账号（登录身份）
//   - MemberID/TenantID：本次会话绑定的成员关系与租户（切换租户即重签 token）
//   - SID/SessionTokenVersion：设备级会话（pf_account_sessions）；切换租户
//     复用 SID 递增版本重签。存量令牌无 sid（兼容期，中间件跳过会话校验）
//
// TenantID 为租户上下文的唯一来源，由租户中间件在 Authentication 之后提取（架构文档 26.4）
type CustomClaims struct {
	AccountID uint `json:"accountId"`
	MemberID  uint `json:"memberId"`
	TenantID  uint `json:"tenantId"`
	// SessionVersion 与 pf_accounts.session_version 对齐；密码更新后版本递增，
	// 认证中间件拒绝旧版本，保证找回密码会中止全部既有会话。
	SessionVersion uint64 `json:"sessionVersion"`
	Name           string `json:"name"`
	// SID 设备级会话标识（ADR-009）；空 = 存量兼容期令牌
	SID string `json:"sid,omitempty"`
	// SessionTokenVersion 会话令牌版本：租户切换重签时递增，旧令牌作废
	SessionTokenVersion int64 `json:"stv,omitempty"`
	jwt.RegisteredClaims
}

type JWTService struct {
	signKey        []byte
	issuer         string
	expireDuration time.Duration
}

func NewJWTService(secret string) *JWTService {
	return &JWTService{
		signKey:        []byte(secret),
		issuer:         Issuer,
		expireDuration: 7 * 24 * time.Hour,
	}
}

// CreateToken 按账号 + 登录成员 + 设备会话签发 token。
// session 为 nil 时签发无 sid 的兼容期令牌（仅测试/过渡路径使用）
func (s *JWTService) CreateToken(account *model.Account, member *model.User, session *securitymodel.AccountSession) (string, error) {
	if account == nil || member == nil {
		return "", fmt.Errorf("empty account or member")
	}
	jti, err := NewJti()
	if err != nil {
		return "", err
	}

	var (
		sid string
		stv int64
	)
	if session != nil {
		sid, stv = session.SID, session.TokenVersion
	}

	now := time.Now()
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		CustomClaims{
			AccountID:           account.ID,
			MemberID:            member.ID,
			TenantID:            member.TenantID,
			SessionVersion:      account.SessionVersion,
			Name:                account.Name,
			SID:                 sid,
			SessionTokenVersion: stv,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(s.expireDuration)),
				NotBefore: jwt.NewNumericDate(now.Add(-1000 * time.Second)),
				ID:        jti, // 随机会话标识：吊销粒度为单次会话（P2-8）
				Issuer:    s.issuer,
			},
		},
	)

	return token.SignedString(s.signKey)
}

// ParseToken 校验并返回会话 claims；成员/租户加载由调用方按 claims 完成
func (s *JWTService) ParseToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (i interface{}, err error) {
		return s.signKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invaild token")
	}

	return claims, nil
}
