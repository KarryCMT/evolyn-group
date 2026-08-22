package auth

import (
	"context"
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
	revoker := NewTokenRevoker(&fakeRevokerRedis{data: map[string]string{}})

	// 未吊销
	if revoker.Revoked(context.Background(), "jti-1") {
		t.Fatal("fresh jti should not be revoked")
	}

	// 吊销后命中
	if err := revoker.Revoke(context.Background(), "jti-1", time.Hour); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if !revoker.Revoked(context.Background(), "jti-1") {
		t.Fatal("revoked jti should be rejected")
	}

	// 空值与已过期令牌：直接视为成功/未吊销
	if err := revoker.Revoke(context.Background(), "", time.Hour); err != nil {
		t.Fatalf("empty jti revoke should be no-op, got %v", err)
	}
	if err := revoker.Revoke(context.Background(), "jti-2", -time.Minute); err != nil {
		t.Fatalf("expired revoke should be no-op, got %v", err)
	}
	if revoker.Revoked(context.Background(), "jti-2") {
		t.Fatal("expired jti should not be revoked")
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
