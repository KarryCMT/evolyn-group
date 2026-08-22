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
}

// revokerRedis 本能力用到的 Redis 最小接口（*redis.Client 天然满足）
type revokerRedis interface {
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
}

func NewTokenRevoker(rdb revokerRedis) *TokenRevoker {
	return &TokenRevoker{rdb: rdb}
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

// Revoked 令牌是否已被吊销；Redis 异常时放行（吊销是增强能力，
// 不应因 Redis 抖动阻断全部请求，且异常已由调用方记录）
func (r *TokenRevoker) Revoked(ctx context.Context, jti string) bool {
	if jti == "" {
		return false
	}
	n, err := r.rdb.Exists(ctx, blacklistKey(jti)).Result()
	if err != nil {
		return false
	}
	return n > 0
}

// NewJti 生成随机会话标识（hex 32 字符）：同一成员多次登录各自独立，
// 吊销互不误伤（原 jti=成员 ID 的写法会导致吊销连带所有历史会话）
func NewJti() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate jti: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
