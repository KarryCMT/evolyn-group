package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// fakeGuardRedis 内存版 Redis 替身：覆盖 LoginGuard 用到的最小命令语义
// （忽略 TTL 过期，单测不等待；Eval 按 Go 等价逻辑模拟 failScript 原子语义）
type fakeGuardRedis struct {
	data map[string]string
}

func (f *fakeGuardRedis) Exists(_ context.Context, keys ...string) *redis.IntCmd {
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

func (f *fakeGuardRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(context.Background())
	var n int64
	for _, k := range keys {
		if _, ok := f.data[k]; ok {
			delete(f.data, k)
			n++
		}
	}
	cmd.SetVal(n)
	return cmd
}

func (f *fakeGuardRedis) Eval(_ context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	cmd := redis.NewCmd(context.Background())
	if script != failScript {
		cmd.SetErr(fmt.Errorf("fakeGuardRedis: unexpected script"))
		return cmd
	}
	failKey, lockKey := keys[0], keys[1]
	maxFails, _ := strconv.ParseInt(guardArgString(args[1]), 10, 64)

	current := int64(0)
	if cur, ok := f.data[failKey]; ok {
		current, _ = strconv.ParseInt(cur, 10, 64)
	}
	current++
	if current >= maxFails {
		f.data[lockKey] = "1"
		delete(f.data, failKey)
		cmd.SetVal(int64(1))
		return cmd
	}
	f.data[failKey] = strconv.FormatInt(current, 10)
	cmd.SetVal(int64(0))
	return cmd
}

// guardArgString go-redis 会把 Go 值原样传给 Eval，替身按类型转字符串
func guardArgString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		return ""
	}
}

// TestLoginGuardLocksAfterMaxFails 连续失败达上限锁定；锁定期内 Locked
// 为真；成功登录（Reset）打断累计则永不锁定
func TestLoginGuardLocksAfterMaxFails(t *testing.T) {
	rdb := &fakeGuardRedis{data: map[string]string{}}
	guard := NewLoginGuard(rdb, LoginGuardOptions{MaxFails: 3, LockDuration: 15 * time.Minute})
	ctx := context.Background()

	// 前两次失败：不锁定
	guard.RecordFailure(ctx, "13800001111")
	guard.RecordFailure(ctx, "13800001111")
	if guard.Locked(ctx, "13800001111") {
		t.Fatal("should not lock before max fails")
	}
	// 第三次失败达上限：锁定并清计数
	guard.RecordFailure(ctx, "13800001111")
	if !guard.Locked(ctx, "13800001111") {
		t.Fatal("should be locked after max fails")
	}
	// 其他标识不受影响
	if guard.Locked(ctx, "someone-else") {
		t.Fatal("other identity should not be locked")
	}
}

// TestLoginGuardResetInterruptsStreak 成功登录清零失败计数：
// 失败-成功-失败的循环永不达到锁定阈值
func TestLoginGuardResetInterruptsStreak(t *testing.T) {
	rdb := &fakeGuardRedis{data: map[string]string{}}
	guard := NewLoginGuard(rdb, LoginGuardOptions{MaxFails: 2, LockDuration: time.Minute})
	ctx := context.Background()

	for range 5 {
		guard.RecordFailure(ctx, "alice")
		guard.Reset(ctx, "alice")
	}
	if guard.Locked(ctx, "alice") {
		t.Fatal("interrupted streaks should never lock")
	}
}

// TestLoginGuardKeysAreHashed 上线前复查 P2：登录标识（手机号/账号名）
// 必须散列后进 Redis key，明文 PII 不落 key
func TestLoginGuardKeysAreHashed(t *testing.T) {
	rdb := &fakeGuardRedis{data: map[string]string{}}
	guard := NewLoginGuard(rdb, LoginGuardOptions{MaxFails: 2, LockDuration: time.Minute})
	ctx := context.Background()

	guard.RecordFailure(ctx, "13800001111")

	expected := sha256.Sum256([]byte("13800001111"))
	hashed := hex.EncodeToString(expected[:])
	if _, ok := rdb.data["evolyn:auth:fail:"+hashed]; !ok {
		t.Fatal("fail key should be built on hashed identifier")
	}
	for key := range rdb.data {
		if key == "evolyn:auth:fail:13800001111" {
			t.Fatal("plaintext identifier leaked into redis key")
		}
	}
}

// TestLoginGuardKeysUseHMACWhenSecretSet 加固项：配置独立密钥后标识用
// HMAC-SHA-256 散列——与无密钥散列结果不同（防字典反查），且同一标识
// 跨调用稳定（多实例共享同一把密钥时计数落同一 key）
func TestLoginGuardKeysUseHMACWhenSecretSet(t *testing.T) {
	rdb := &fakeGuardRedis{data: map[string]string{}}
	guard := NewLoginGuard(rdb, LoginGuardOptions{MaxFails: 2, LockDuration: time.Minute, Secret: "guard-secret"})
	ctx := context.Background()

	guard.RecordFailure(ctx, "13800001111")
	guard.RecordFailure(ctx, "13800001111")
	if !guard.Locked(ctx, "13800001111") {
		t.Fatal("guard with secret should still lock normally")
	}

	plain := sha256.Sum256([]byte("13800001111"))
	plainHex := hex.EncodeToString(plain[:])
	mac := hmac.New(sha256.New, []byte("guard-secret"))
	mac.Write([]byte("13800001111"))
	hmacHex := hex.EncodeToString(mac.Sum(nil))
	if hmacHex == plainHex {
		t.Fatal("hmac digest should differ from keyless sha256")
	}
	if _, ok := rdb.data["evolyn:auth:lock:"+hmacHex]; !ok {
		t.Fatal("lock key should be built on hmac digest")
	}
	for key := range rdb.data {
		if strings.HasSuffix(key, plainHex) {
			t.Fatal("keyless digest leaked while secret configured")
		}
	}
}

// TestLoginGuardDefaults 零值参数回落默认 5 次 / 15 分钟
func TestLoginGuardDefaults(t *testing.T) {
	guard := NewLoginGuard(&fakeGuardRedis{data: map[string]string{}}, LoginGuardOptions{})
	if guard.opts.MaxFails != 5 {
		t.Fatalf("default MaxFails = %d, want 5", guard.opts.MaxFails)
	}
	if guard.opts.LockDuration != 15*time.Minute {
		t.Fatalf("default LockDuration = %v, want 15m", guard.opts.LockDuration)
	}
}
