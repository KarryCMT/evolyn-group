package middleware

import (
	"evolyn/internal/contextx"
	"evolyn/internal/platform/ginctx"

	"github.com/gin-gonic/gin"
)

// TenantMiddleware 租户上下文中间件（架构文档 26.4）：
// 位于 Authentication 之后、Authorization 之前，从已认证用户提取租户 ID，
// 同时写入 gin.Context（请求内取用）与 request context（GORM Callback /
// 引擎等数据路径取用）。未认证请求不带租户上下文，行为与现状一致。
func TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := ginctx.GetUser(c)
		if user == nil || user.TenantID == 0 {
			c.Next()
			return
		}

		ginctx.SetTenant(c, user.TenantID)
		c.Request = c.Request.WithContext(contextx.NewTenantContext(c.Request.Context(), user.TenantID))

		c.Next()
	}
}
