package authentication

import (
	"fmt"
	"strconv"
	"time"

	"evolyn/internal/model"

	"github.com/golang-jwt/jwt/v5"
)

const (
	Issuer = "evolyn"
)

// CustomClaims JWT 载荷；TenantID 为租户上下文的唯一来源，
// 由租户中间件在 Authentication 之后提取（架构文档 26.4）
type CustomClaims struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	TenantID uint   `json:"tenantId"`
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

func (s *JWTService) CreateToken(user *model.User) (string, error) {
	if user == nil {
		return "", fmt.Errorf("empty user")
	}
	now := time.Now()
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		CustomClaims{
			Name:     user.Name,
			ID:       user.ID,
			TenantID: user.TenantID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(s.expireDuration)),
				NotBefore: jwt.NewNumericDate(now.Add(-1000 * time.Second)),
				ID:        strconv.Itoa(int(user.ID)),
				Issuer:    s.issuer,
			},
		},
	)

	return token.SignedString(s.signKey)
}

func (s *JWTService) ParseToken(tokenString string) (*model.User, error) {
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

	user := &model.User{
		ID:   claims.ID,
		Name: claims.Name,
		BaseModel: model.BaseModel{
			TenantID: claims.TenantID,
		},
	}

	return user, nil
}
