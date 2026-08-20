package middleware

import (
	"evolyn/internal/platform/httpx"
	"net/http"

	"evolyn/internal/utils/ratelimit"

	"github.com/gin-gonic/gin"
)

// ErrRateLimited 频控命中（ADR-008）：复用通用码 RATE_LIMITED，
// 命中的限流器/key 细节经 Wrap 只入日志不出网
var ErrRateLimited = httpx.NewBiz(httpx.CodeRateLimited, "请求过于频繁，请稍后再试", http.StatusTooManyRequests)

func RateLimitMiddleware(configs []ratelimit.LimitConfig) (gin.HandlerFunc, error) {
	var limiters []*ratelimit.RateLimiter
	for i := range configs {
		limiter, err := ratelimit.NewRateLimiter(&configs[i])
		if err != nil {
			return nil, err
		}
		limiters = append(limiters, limiter)
	}

	return func(c *gin.Context) {
		for _, limiter := range limiters {
			if err := limiter.Accept(c); err != nil {
				// 频控命中：稳定码/安全文案出网，limiter 的 key/策略细节只入日志（ADR-008）；
				// 必须 Abort——写完 429 后继续放行会让后续 handler 照常执行
				httpx.ResponseFailed(c, http.StatusTooManyRequests, httpx.Wrap(ErrRateLimited, err))
				c.Abort()
				return
			}
		}

		c.Next()
	}, nil
}
