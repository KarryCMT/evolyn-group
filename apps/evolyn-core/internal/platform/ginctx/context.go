package ginctx

import (
	"evolyn/internal/platform/auth"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/utils/request"
	"evolyn/internal/utils/trace"

	"github.com/gin-gonic/gin"
)

// gin 上下文键：请求内快速取用（原 pkg/common 拆分而来，ADR-007）
const (
	UserContextKey        = `user`
	SessionContextKey     = `session`
	TenantContextKey      = `tenant`
	TraceContextKey       = `trace`
	RequestInfoContextKey = `requestInfo`
)

// SetSession 保存会话 claims（AuthenticationMiddleware 注入；租户切换等
// 需要账号身份的接口从会话取 AccountID，ADR-006）
func SetSession(c *gin.Context, claims *auth.CustomClaims) {
	if c == nil || claims == nil {
		return
	}
	c.Set(SessionContextKey, claims)
}

// GetSession 读取会话 claims，未认证请求返回 nil
func GetSession(c *gin.Context) *auth.CustomClaims {
	if c == nil {
		return nil
	}

	val, ok := c.Get(SessionContextKey)
	if !ok {
		return nil
	}

	claims, ok := val.(*auth.CustomClaims)
	if !ok {
		return nil
	}

	return claims
}

// SetTenant 把租户 ID 写入 gin 上下文（请求内快速取用）；
// 标准 context 通道见 internal/contextx
func SetTenant(c *gin.Context, tenantID uint) {
	if c == nil || tenantID == 0 {
		return
	}
	c.Set(TenantContextKey, tenantID)
}

// GetTenant 读取 gin 上下文中的租户 ID，无则返回 0
func GetTenant(c *gin.Context) uint {
	if c == nil {
		return 0
	}
	val, ok := c.Get(TenantContextKey)
	if !ok {
		return 0
	}
	tenantID, ok := val.(uint)
	if !ok {
		return 0
	}
	return tenantID
}

func SetTrace(c *gin.Context, t *trace.Trace) {
	if c == nil || t == nil {
		return
	}

	c.Set(TraceContextKey, t)
}

func GetTrace(c *gin.Context) *trace.Trace {
	if c == nil {
		return nil
	}

	val, ok := c.Get(TraceContextKey)
	if !ok {
		return nil
	}

	trace, ok := val.(*trace.Trace)
	if !ok {
		return nil
	}

	return trace
}

func TraceStep(c *gin.Context, msg string, fields ...trace.Field) {
	trace := GetTrace(c)
	if trace != nil {
		trace.Step(msg, fields...)
	}
}

func SetUser(c *gin.Context, user *model.User) {
	if c == nil || user == nil {
		return
	}

	c.Set(UserContextKey, user)
}

func GetUser(c *gin.Context) *model.User {
	if c == nil {
		return nil
	}

	val, ok := c.Get(UserContextKey)
	if !ok {
		return nil
	}

	user, ok := val.(*model.User)
	if !ok {
		return nil
	}

	return user
}

func SetRequestInfo(c *gin.Context, ri *request.RequestInfo) {
	if c == nil || ri == nil {
		return
	}

	c.Set(RequestInfoContextKey, ri)
}

func GetRequestInfo(c *gin.Context) *request.RequestInfo {
	if c == nil {
		return nil
	}

	val, ok := c.Get(RequestInfoContextKey)
	if !ok {
		return nil
	}

	ri, ok := val.(*request.RequestInfo)
	if !ok {
		return nil
	}

	return ri
}
