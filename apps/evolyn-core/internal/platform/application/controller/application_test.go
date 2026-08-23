package controller

import (
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRegisterRouteNoConflict 静态段 code 与参数段 :id 同层共存防回归：
// gin 的路由冲突 panic 发生在注册期（编译期发现不了），此处保证按 code
// 查询路由可与 GET /applications/:id 并存注册
func TestRegisterRouteNoConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// 仅验证路由树注册，handler 不会被调用，service 传 nil 安全
	NewApplicationController(nil).RegisterRoute(r.Group("/api/v1"))

	codeRouted := false
	for _, route := range r.Routes() {
		if route.Path == "/api/v1/applications/code/:code" {
			codeRouted = true
		}
	}
	if !codeRouted {
		t.Fatal("按 code 查询路由未注册")
	}
}
