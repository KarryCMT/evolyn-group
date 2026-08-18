package common

import (
	"context"

	"evolyn/internal/platform/iam/model"

	"evolyn/pkg/utils/request"
	"evolyn/pkg/utils/trace"

	"github.com/gin-gonic/gin"
)

type tenantIDKey struct{}

// NewTenantContext 把租户 ID 注入标准 context，供 GORM Callback、
// 引擎与 Worker 等脱离 gin 的数据路径读取（架构文档 26.3/26.4）
func NewTenantContext(ctx context.Context, tenantID uint) context.Context {
	return context.WithValue(ctx, tenantIDKey{}, tenantID)
}

// TenantIDFromContext 读取注入的租户 ID；ok 为 false 表示当前链路无租户上下文
func TenantIDFromContext(ctx context.Context) (uint, bool) {
	if ctx == nil {
		return 0, false
	}
	tenantID, ok := ctx.Value(tenantIDKey{}).(uint)
	return tenantID, ok
}

// SetTenant 把租户 ID 写入 gin 上下文（请求内快速取用）
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
