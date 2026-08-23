package middleware

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware CORS 跨域中间件：AllowCredentials=true（Cookie 会话依赖）时
// 绝不能放行任意 Origin，否则任意站点可携凭证跨域调用 API（P1 整改）。
//
//   - allowedOrigins 生产域名白名单，Origin 精确匹配（含协议与端口）；
//   - devLoose 仅 debug 环境传入 true：白名单未命中时额外放行
//     localhost / 127.0.0.1 的任意端口（本地联调端口不固定），
//     非本机回环地址一律拒绝。
//
// release 环境的空白名单 fail-fast 由装配层（server）在启动时校验，此处不感知环境
func CORSMiddleware(allowedOrigins []string, devLoose bool) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOriginFunc:  buildAllowOriginFunc(allowedOrigins, devLoose),
		AllowMethods:     []string{"PUT", "PATCH", "GET", "DELETE", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Length", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
		AllowWebSockets:  true,
	})
}

// buildAllowOriginFunc 构造 Origin 判定函数（独立出来便于单测）：
// 白名单精确匹配优先；devLoose 额外放行本机回环任意端口
func buildAllowOriginFunc(allowedOrigins []string, devLoose bool) func(origin string) bool {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			allowed[trimmed] = struct{}{}
		}
	}
	return func(origin string) bool {
		if _, ok := allowed[origin]; ok {
			return true
		}
		return devLoose && isLoopbackOrigin(origin)
	}
}

// isLoopbackOrigin 判定 Origin 是否为本机回环地址（http/https 均认），
// host 必须形如 localhost[:port] 或 127.0.0.1[:port]，供 debug 环境宽松回落
func isLoopbackOrigin(origin string) bool {
	host := origin
	if i := strings.Index(host, "://"); i >= 0 {
		host = host[i+3:]
	}
	// 去掉端口（Origin 不含 path/userinfo，无需处理 "]/:" 等 IPv6 细节）
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i]
	}
	return host == "localhost" || host == "127.0.0.1"
}
