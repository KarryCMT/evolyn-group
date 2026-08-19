package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"evolyn/internal/contextx"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"

	"evolyn/internal/utils/request"

	"github.com/gin-gonic/gin"
)

// RequestInfoMiddleware 解析资源请求信息并注入请求元数据：
// IP/UA/RequestID 随 ctx 下传，业务审计（FIX-013）等服务层从 ctx 读取
func RequestInfoMiddleware(resolver request.RequestInfoResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		ri, err := resolver.NewRequestInfo(c.Request)
		if err != nil {
			httpx.ResponseFailed(c, http.StatusBadRequest, err)
			c.Abort()
			return
		}

		ginctx.SetRequestInfo(c, ri)

		c.Request = c.Request.WithContext(contextx.NewRequestMetaContext(c.Request.Context(), contextx.RequestMeta{
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			RequestID: newRequestID(),
		}))

		c.Next()
	}
}

// newRequestID 8 字节随机十六进制：审计关联用的轻量请求标识
// （技术链路 trace 仍走 TraceMiddleware，两者职责不同）
func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
