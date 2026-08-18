package auth

import (
	"fmt"
	"strconv"
	"time"

	"evolyn/internal/platform/iam/model"

	"github.com/golang-jwt/jwt/v5"
)

const (
	Issuer = "evolyn"
)

// CustomClaims JWT 载荷（ADR-006 账号×成员拆分）：
//   - AccountID/Name：平台账号（登录身份）
//   - MemberID/TenantID：本次会话绑定的成员关系与租户（切换租户即重签 token）
//
// TenantID 为租户上下文的唯一来源，由租户中间件在 Authentication 之后提取（架构文档 26.4）
type CustomClaims struct {
	AccountID uint   `json:"accountId"`
	MemberID  uint   `json:"memberId"`
	TenantID  uint   `json:"tenantId"`
	Name      string `json:"name"`
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

// CreateToken 按账号 + 登录成员签发会话 token
func (s *JWTService) CreateToken(account *model.Account, member *model.User) (string, error) {
	if account == nil || member == nil {
		return "", fmt.Errorf("empty account or member")
	}
	now := time.Now()
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		CustomClaims{
			AccountID: account.ID,
			MemberID:  member.ID,
			TenantID:  member.TenantID,
			Name:      account.Name,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(s.expireDuration)),
				NotBefore: jwt.NewNumericDate(now.Add(-1000 * time.Second)),
				ID:        strconv.Itoa(int(member.ID)),
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
