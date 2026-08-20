package middleware

import (
	"evolyn/internal/contextx"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"fmt"
	"net/http"
	"strings"

	"evolyn/internal/platform/auth"
	"evolyn/internal/platform/iam/repository"

	"github.com/gin-gonic/gin"
)

// ErrStaleSession 会话租户归属过期（ADR-008 稳定码）：签发后成员被移动到
// 其他租户，令牌中的 tenantId 与当前归属不一致，需重新登录
var ErrStaleSession = httpx.NewBiz("AUTH_STALE_TENANT", "登录态已失效（租户归属变化），请重新登录", http.StatusUnauthorized)

// AuthenticationMiddleware 会话认证：按 JWT claims 的 memberId 加载成员（ADR-006）。
// 此时尚未经过 TenantMiddleware（租户上下文来自本处加载成员的归属），
// ctx 无租户上下文，GetUserByID 按全局唯一 ID 查询
func AuthenticationMiddleware(jwtService *auth.JWTService, userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := getTokenFromAuthorizationHeader(c)
		if token == "" {
			token, _ = getTokenFromCookie(c)
		}

		claims, _ := jwtService.ParseToken(token)
		if claims != nil {
			ginctx.SetSession(c, claims)
			// 操作者（账号+成员）随 ctx 下传，业务审计等服务层从 ctx 读取（FIX-013）
			c.Request = c.Request.WithContext(contextx.NewActorContext(c.Request.Context(), contextx.Actor{
				AccountID: claims.AccountID,
				MemberID:  claims.MemberID,
			}))
			member, err := userRepo.GetUserByID(c.Request.Context(), claims.MemberID)
			if err != nil {
				httpx.ResponseFailed(c, http.StatusInternalServerError, fmt.Errorf("failed to get user"))
				c.Abort()
				return
			}
			// 会话签发后成员被移动到其他租户等异常：拒绝过期会话
			if member.TenantID != claims.TenantID {
				httpx.ResponseFailed(c, http.StatusUnauthorized, ErrStaleSession)
				c.Abort()
				return
			}
			ginctx.SetUser(c, member)
		}

		c.Next()
	}
}

func getTokenFromCookie(c *gin.Context) (string, error) {
	return c.Cookie("token")
}

func getTokenFromAuthorizationHeader(c *gin.Context) (string, error) {
	auth := c.Request.Header.Get("Authorization")
	if auth == "" {
		return "", nil
	}

	token := strings.Fields(auth)
	if len(token) != 2 || strings.ToLower(token[0]) != "bearer" || token[1] == "" {
		return "", fmt.Errorf("authorization header invaild")
	}

	return token[1], nil
}
