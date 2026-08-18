package middleware

import (
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"net/http"

	"evolyn/internal/utils/request"

	"github.com/gin-gonic/gin"
)

func RequestInfoMiddleware(resolver request.RequestInfoResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		ri, err := resolver.NewRequestInfo(c.Request)
		if err != nil {
			httpx.ResponseFailed(c, http.StatusBadRequest, err)
			c.Abort()
			return
		}

		ginctx.SetRequestInfo(c, ri)

		c.Next()
	}
}
