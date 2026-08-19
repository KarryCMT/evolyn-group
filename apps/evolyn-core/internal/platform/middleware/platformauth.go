package middleware

import (
	"net/http"

	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/iam/authorization"

	"github.com/gin-gonic/gin"
)

// PlatformAuthorizationMiddleware 平台运营域鉴权（FIX-008）：
// 仅挂在 /api/v1/platform 组，与租户域 AuthorizationMiddleware 互斥。
// 平台管理员 = 持有 cluster-admin 角色的成员（默认租户 root 组，纯函数
// 判定不触碰仓储，避免无租户上下文时按名查询组/角色的跨租户歧义）。
// 本域无 TenantMiddleware/TenantStatusMiddleware——平台域不依赖租户上下文
func PlatformAuthorizationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		member := ginctx.GetUser(c)
		if member == nil {
			httpx.ResponseFailed(c, http.StatusUnauthorized, nil)
			c.Abort()
			return
		}
		if !authorization.IsClusterAdmin(member) {
			httpx.ResponseFailed(c, http.StatusForbidden, nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
