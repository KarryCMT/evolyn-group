package controller

import "github.com/gin-gonic/gin"

type Controller interface {
	Name() string
	RegisterRoute(*gin.RouterGroup)
}

// PlatformController 平台运营域控制器（FIX-008）：实现本接口且 Platform()
// 返回 true 的控制器注册到 /api/v1/platform 组——该组只挂
// Authentication + PlatformAuthorization，无 TenantMiddleware/TenantStatus；
// 其余控制器注册到租户域组（Authentication + Tenant + TenantStatus +
// Authorization）。两个权限域不可串用
type PlatformController interface {
	Controller
	Platform() bool
}
