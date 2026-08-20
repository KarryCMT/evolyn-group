package middleware

import (
	"net/http"

	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantrepository "evolyn/internal/platform/tenant/repository"

	"github.com/gin-gonic/gin"
)

// 稳定错误码（FIX-007，ADR-008 起承载于 BizError）：前端据此区分冻结/注销
var (
	ErrTenantFrozen  = httpx.NewBiz("TENANT_FROZEN", "租户已被冻结，请联系管理员", http.StatusForbidden)
	ErrTenantDisable = httpx.NewBiz("TENANT_DISABLED", "租户已注销", http.StatusForbidden)
)

// TenantStatusMiddleware 租户状态请求级拦截（FIX-007，架构文档 26.2）：
// 位于 TenantMiddleware（解析/注入租户）之后、Authorization 之前——
// active 放行；frozen/deleted 直接拒绝，已签发 JWT 同样失效；
// 无租户上下文的请求（登录链路/平台域）不受影响
func TenantStatusMiddleware(tenantRepo tenantrepository.TenantRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := ginctx.GetTenant(c)
		if tenantID == 0 {
			c.Next()
			return
		}

		// 状态查询带 Redis 短缓存，状态变更写路径同步失效（见 tenant repository）
		status, err := tenantRepo.GetStatus(c.Request.Context(), tenantID)
		if err != nil {
			httpx.ResponseFailed(c, http.StatusForbidden, err)
			c.Abort()
			return
		}

		switch status {
		case tenantmodel.TenantActive:
			c.Next()
		case tenantmodel.TenantFrozen:
			httpx.ResponseFailed(c, http.StatusForbidden, ErrTenantFrozen)
			c.Abort()
		default:
			// deleted 及任何未知状态一律拒绝（默认拒绝原则）
			httpx.ResponseFailed(c, http.StatusForbidden, ErrTenantDisable)
			c.Abort()
		}
	}
}
