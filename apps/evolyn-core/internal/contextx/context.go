package contextx

import "context"

// 租户上下文契约（架构文档 26.3/26.4）：标准 context 注入/读取，
// 供 GORM Callback、引擎与 Worker 等脱离 gin 的数据路径使用。
// 独立成包以保持零依赖：infrastructure 与 platform 均可引用。
type tenantIDKey struct{}

// NewTenantContext 把租户 ID 注入标准 context
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
