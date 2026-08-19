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

// DetachTenant 剥离租户上下文但保留其余键值（操作者/请求元数据/事务 session）。
// 开通租户等平台级写入路径用它替代 context.Background()（FIX-020）：
// 既避免运营者会话自带的租户上下文污染新租户数据，又让事务 session 继续
// 向下传播——直接换 Background 会把 ctx 携带的事务切断
func DetachTenant(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	// 以 nil 覆盖租户键：TenantIDFromContext 的类型断言随之失败即「无租户」
	return context.WithValue(ctx, tenantIDKey{}, nil)
}

// Actor 请求操作者（ADR-006 账号×成员拆分）：账号为登录身份，成员为租户内身份。
// 由 AuthenticationMiddleware 注入请求 context，业务审计等服务层从 ctx 读取
type Actor struct {
	AccountID uint
	MemberID  uint
}

type actorKey struct{}

// NewActorContext 注入操作者（未认证请求不注入）
func NewActorContext(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, actor)
}

// ActorFromContext 读取操作者；ok 为 false 表示无认证信息（如启动期/登录前路径）
func ActorFromContext(ctx context.Context) (Actor, bool) {
	if ctx == nil {
		return Actor{}, false
	}
	actor, ok := ctx.Value(actorKey{}).(Actor)
	return actor, ok
}

// RequestMeta 请求元数据（审计用）：来源 IP、UA 与请求标识。
// 由 RequestInfoMiddleware 注入，业务审计落库时从 ctx 读取
type RequestMeta struct {
	IP        string
	UserAgent string
	RequestID string
}

type requestMetaKey struct{}

// NewRequestMetaContext 注入请求元数据
func NewRequestMetaContext(ctx context.Context, meta RequestMeta) context.Context {
	return context.WithValue(ctx, requestMetaKey{}, meta)
}

// RequestMetaFromContext 读取请求元数据，未注入时返回零值
func RequestMetaFromContext(ctx context.Context) RequestMeta {
	if ctx == nil {
		return RequestMeta{}
	}
	meta, _ := ctx.Value(requestMetaKey{}).(RequestMeta)
	return meta
}
