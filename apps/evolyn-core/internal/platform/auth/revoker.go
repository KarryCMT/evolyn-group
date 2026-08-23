package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// TokenRevoker 令牌吊销（P2-8）：登出即把 jti 拉黑到自然过期。
// JWT 本身无状态、固定 7 天有效期，仅清 Cookie 无法让已泄露的 Bearer
// 失效；黑名单 TTL 取令牌剩余有效期，到期自动出清不占存储。
// 未携带 jti 的旧令牌无法吊销（历史 token 量小，接受过渡）
type TokenRevoker struct {
	rdb revokerRedis
	// failClosed Redis 异常时的降级策略（上线前整改 P2）：
	// false=放行（默认，可用性优先，吊销是增强能力）；true=视为已吊销
	// 拒绝请求（对已泄露令牌立即失效要求更高的部署开启）
	failClosed bool
}

// revokerRedis 本能力用到的 Redis 最小接口（*redis.Client 天然满足）
type revokerRedis interface {
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
}

func NewTokenRevoker(rdb revokerRedis, failClosed bool) *TokenRevoker {
	return &TokenRevoker{rdb: rdb, failClosed: failClosed}
}

func blacklistKey(jti string) string { return "evolyn:jwt:bl:" + jti }

// Revoke 拉黑令牌：TTL 为令牌剩余有效期（<=0 视为已过期直接成功）
func (r *TokenRevoker) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	if jti == "" || ttl <= 0 {
		return nil
	}
	if err := r.rdb.SetNX(ctx, blacklistKey(jti), 1, ttl).Err(); err != nil {
		return fmt.Errorf("redis setnx blacklist: %w", err)
	}
	return nil
}

// Revoked 查询令牌是否已被吊销，返回黑名单的原始状态与 Redis 错误。
// 降级决策归调用方（上线前复查 P2）：
//   - err != nil 且 failClosed：拒绝请求，但必须以可区分的 503 稳定码
//     （AUTH_REVOKE_CHECK_FAILED，见认证中间件）而非「已吊销」401 出网，
//     避免客户端把「暂时查不了」误当「令牌确已吊销」清掉仍有效的登录态；
//   - err != nil 且 fail-open：放行并记录日志（吊销是增强能力）。
func (r *TokenRevoker) Revoked(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}
	n, err := r.rdb.Exists(ctx, blacklistKey(jti)).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists blacklist: %w", err)
	}
	return n > 0, nil
}

// FailClosed 是否处于 fail-closed 模式（Redis 异常时拒绝而非放行）：
// 登出等写侧据此决定「吊销写失败是否如实报错」
func (r *TokenRevoker) FailClosed() bool { return r.failClosed }

// NewJti 生成随机会话标识（hex 32 字符）：同一成员多次登录各自独立，
// 吊销互不误伤（原 jti=成员 ID 的写法会导致吊销连带所有历史会话）
func NewJti() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate jti: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
