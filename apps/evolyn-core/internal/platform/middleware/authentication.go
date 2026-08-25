package middleware

import (
	"evolyn/internal/contextx"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"fmt"
	"net/http"
	"strings"

	"evolyn/internal/platform/auth"
	securityservice "evolyn/internal/platform/auth/security/service"
	"evolyn/internal/platform/iam/repository"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ErrStaleSession 会话租户归属过期（ADR-008 稳定码）：签发后成员被移动到
// 其他租户，令牌中的 tenantId 与当前归属不一致，需重新登录
var (
	ErrStaleSession = httpx.NewBiz("AUTH_STALE_TENANT", "登录态已失效（租户归属变化），请重新登录", http.StatusUnauthorized)
	// ErrSessionInvalidated 密码找回/修改后账号的 session_version 已推进，旧 JWT
	// 不再可信，统一要求重新登录。
	ErrSessionInvalidated = httpx.NewBiz("AUTH_SESSION_INVALIDATED", "密码已更新，请重新登录", http.StatusUnauthorized)
	ErrTokenRevoked       = httpx.NewBiz("AUTH_TOKEN_REVOKED", "登录态已退出，请重新登录", http.StatusUnauthorized)
	// ErrRevokerUnavailable 吊销状态检查暂时不可用（fail-closed 模式下 Redis
	// 故障）：503 与 401（确已吊销）严格区分，客户端不应据此清除登录态
	ErrRevokerUnavailable = httpx.NewBiz("AUTH_REVOKE_CHECK_FAILED", "登录态校验暂时不可用，请稍后再试", http.StatusServiceUnavailable)
)

// AuthenticationMiddleware 会话认证：按 JWT claims 的 memberId 加载成员（ADR-006）。
// 此时尚未经过 TenantMiddleware（租户上下文来自本处加载成员的归属），
// ctx 无租户上下文，GetUserByID 按全局唯一 ID 查询
// AuthenticationMiddleware 会话认证。revoker 非空时校验令牌吊销状态
// （P2-8：登出拉黑 jti 至自然过期；可传 nil 跳过，供无 Redis 场景/测试）。
// sessionSvc 非空时校验设备会话（ADR-009：sid 未撤销且 token_version 一致，
// 被挤出返回 AUTH_SESSION_REPLACED）；存量无 sid 令牌跳过会话校验（兼容期）
func AuthenticationMiddleware(jwtService *auth.JWTService, userRepo repository.UserRepository, accountRepo repository.AccountRepository, revoker *auth.TokenRevoker, sessionSvc securityservice.SessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := getTokenFromAuthorizationHeader(c)
		if token == "" {
			token, _ = getTokenFromCookie(c)
		}

		claims, _ := jwtService.ParseToken(token)
		if claims != nil {
			if revoker != nil {
				revoked, err := revoker.Revoked(c.Request.Context(), claims.ID)
				if err != nil {
					if revoker.FailClosed() {
						// fail-closed：黑名单状态未知时拒绝请求，但用可区分的
						// 503 稳定码——不能用「已吊销」401（会触发前端清除仍有效的
						// 登录态，且 401 响应会顺带清 Cookie）
						httpx.ResponseFailed(c, http.StatusServiceUnavailable, ErrRevokerUnavailable)
						c.Abort()
						return
					}
					// fail-open：Redis 抖动不阻断请求（吊销是增强能力）
					logrus.Warnf("check token revocation (fail-open): %v", err)
				} else if revoked {
					httpx.ResponseFailed(c, http.StatusUnauthorized, ErrTokenRevoked)
					c.Abort()
					return
				}
			}
			if sessionSvc != nil && claims.SID != "" {
				if err := sessionSvc.Validate(c.Request.Context(), claims.SID, claims.SessionTokenVersion); err != nil {
					httpx.ResponseFailed(c, http.StatusUnauthorized, err)
					c.Abort()
					return
				}
			}
			account, err := accountRepo.GetByID(c.Request.Context(), claims.AccountID)
			if err != nil {
				httpx.ResponseFailed(c, http.StatusUnauthorized, ErrSessionInvalidated)
				c.Abort()
				return
			}
			if account.SessionVersion != claims.SessionVersion {
				httpx.ResponseFailed(c, http.StatusUnauthorized, ErrSessionInvalidated)
				c.Abort()
				return
			}
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
			if member.AccountId != claims.AccountID || member.TenantID != claims.TenantID {
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
