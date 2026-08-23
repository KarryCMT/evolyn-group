package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// fakeRevokerRedis 最小 Redis 替身：SetNX 占位 + Exists 计数
type fakeRevokerRedis struct {
	data map[string]string
}

func (f *fakeRevokerRedis) SetNX(_ context.Context, key string, value interface{}, _ time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(context.Background())
	if _, exists := f.data[key]; exists {
		cmd.SetVal(false)
	} else {
		f.data[key] = "1"
		cmd.SetVal(true)
	}
	return cmd
}

func (f *fakeRevokerRedis) Exists(_ context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(context.Background())
	var n int64
	for _, k := range keys {
		if _, ok := f.data[k]; ok {
			n++
		}
	}
	cmd.SetVal(n)
	return cmd
}

func TestTokenRevokerRevokeAndCheck(t *testing.T) {
	revoker := NewTokenRevoker(&fakeRevokerRedis{data: map[string]string{}}, false)

	// 未吊销
	revoked, err := revoker.Revoked(context.Background(), "jti-1")
	if err != nil || revoked {
		t.Fatalf("fresh jti should not be revoked (revoked=%v, err=%v)", revoked, err)
	}

	// 吊销后命中
	if err := revoker.Revoke(context.Background(), "jti-1", time.Hour); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	revoked, err = revoker.Revoked(context.Background(), "jti-1")
	if err != nil || !revoked {
		t.Fatalf("revoked jti should be rejected (revoked=%v, err=%v)", revoked, err)
	}

	// 空值与已过期令牌：直接视为成功/未吊销
	if err := revoker.Revoke(context.Background(), "", time.Hour); err != nil {
		t.Fatalf("empty jti revoke should be no-op, got %v", err)
	}
	if err := revoker.Revoke(context.Background(), "jti-2", -time.Minute); err != nil {
		t.Fatalf("expired revoke should be no-op, got %v", err)
	}
	revoked, err = revoker.Revoked(context.Background(), "jti-2")
	if err != nil || revoked {
		t.Fatalf("expired jti should not be revoked (revoked=%v, err=%v)", revoked, err)
	}
}

func TestNewJtiUniqueness(t *testing.T) {
	seen := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		jti, err := NewJti()
		if err != nil {
			t.Fatalf("new jti: %v", err)
		}
		if len(jti) != 32 {
			t.Fatalf("jti length = %d, want 32 (hex of 16 bytes)", len(jti))
		}
		seen[jti] = struct{}{}
	}
	if len(seen) != 100 {
		t.Fatalf("jti collisions: %d unique of 100", len(seen))
	}
}

// brokenRevokerRedis Redis 故障替身：所有命令返回错误
type brokenRevokerRedis struct{}

func (b *brokenRevokerRedis) SetNX(_ context.Context, _ string, _ interface{}, _ time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(context.Background())
	cmd.SetErr(errors.New("redis unavailable"))
	return cmd
}

func (b *brokenRevokerRedis) Exists(_ context.Context, _ ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(context.Background())
	cmd.SetErr(errors.New("redis unavailable"))
	return cmd
}

// TestTokenRevokerFailClosed Redis 异常时 Revoked 只报告原始状态（false+错误），
// fail-open/fail-closed 的拒绝决策归调用方（认证中间件以可区分的 503 出网）
func TestTokenRevokerFailClosed(t *testing.T) {
	failOpen := NewTokenRevoker(&brokenRevokerRedis{}, false)
	revoked, err := failOpen.Revoked(context.Background(), "jti-1")
	if revoked || err == nil {
		t.Fatalf("redis error should surface (revoked, err) = (%v, nil)", revoked)
	}
	if failOpen.FailClosed() {
		t.Fatal("fail-open mode should report FailClosed=false")
	}
	if err := failOpen.Revoke(context.Background(), "jti-1", time.Hour); err == nil {
		t.Fatal("revoke should surface redis error (caller decides degradation)")
	}

	failClosed := NewTokenRevoker(&brokenRevokerRedis{}, true)
	// 读侧不再吞错误伪装「已吊销」：状态未知就是未知，由调用方映射 503
	revoked, err = failClosed.Revoked(context.Background(), "jti-1")
	if revoked || err == nil {
		t.Fatalf("fail-closed should still surface raw state, got (%v, %v)", revoked, err)
	}
	if !failClosed.FailClosed() {
		t.Fatal("fail-closed mode should report FailClosed=true")
	}
}
