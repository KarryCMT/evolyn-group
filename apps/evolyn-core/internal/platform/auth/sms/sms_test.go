package sms

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// fakeRedis 内存版 Redis 替身：忽略 TTL 过期（单测不等待），覆盖命令语义
type fakeRedis struct {
	data map[string]string
}

func newFakeRedis() *fakeRedis { return &fakeRedis{data: map[string]string{}} }

func (f *fakeRedis) Get(_ context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(context.Background())
	if v, ok := f.data[key]; ok {
		cmd.SetVal(v)
	} else {
		cmd.SetErr(redis.Nil)
	}
	return cmd
}

func (f *fakeRedis) Set(_ context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	f.data[key] = toString(value)
	return redis.NewStatusCmd(context.Background())
}

func (f *fakeRedis) SetNX(_ context.Context, key string, value interface{}, _ time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(context.Background())
	if _, exists := f.data[key]; exists {
		cmd.SetVal(false)
	} else {
		f.data[key] = toString(value)
		cmd.SetVal(true)
	}
	return cmd
}

func (f *fakeRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
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

func (f *fakeRedis) Incr(_ context.Context, key string) *redis.IntCmd {
	cmd := redis.NewIntCmd(context.Background())
	v := int64(0)
	if cur, ok := f.data[key]; ok {
		v, _ = strconv.ParseInt(cur, 10, 64)
	}
	v++
	f.data[key] = strconv.FormatInt(v, 10)
	cmd.SetVal(v)
	return cmd
}

func (f *fakeRedis) Expire(_ context.Context, _ string, _ time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(context.Background())
	cmd.SetVal(true)
	return cmd
}

func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case int64:
		return strconv.FormatInt(t, 10)
	case int:
		return strconv.Itoa(t)
	default:
		return ""
	}
}

// fakeSender 记录发送调用的通道替身
type fakeSender struct{ calls int }

func (f *fakeSender) Send(_ context.Context, _, _ string) error {
	f.calls++
	return nil
}

func newTestService(rdb redisAPI, sender Sender) *Service {
	return NewService(rdb, sender, Options{MaxTries: 3})
}

func TestSendAndVerifyHappyPath(t *testing.T) {
	rdb, sender := newFakeRedis(), &fakeSender{}
	svc := newTestService(rdb, sender)

	code, err := svc.Send(context.Background(), SceneLogin, "13800001111")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if !regexp.MustCompile(`^\d{6}$`).MatchString(code) {
		t.Fatalf("code %q is not 6 digits", code)
	}
	if sender.calls != 1 {
		t.Fatalf("sender called %d times, want 1", sender.calls)
	}

	if err := svc.Verify(context.Background(), SceneLogin, "13800001111", code); err != nil {
		t.Fatalf("verify: %v", err)
	}

	// 一次性：命中后立即失效，重放被拒
	if err := svc.Verify(context.Background(), SceneLogin, "13800001111", code); !errors.Is(err, ErrCodeInvalid) {
		t.Fatalf("replay verify should fail with ErrCodeInvalid, got %v", err)
	}
}

func TestSendSceneAndPhoneValidation(t *testing.T) {
	svc := newTestService(newFakeRedis(), &fakeSender{})

	if _, err := svc.Send(context.Background(), "register", "13800001111"); !errors.Is(err, ErrScene) {
		t.Fatalf("unknown scene should be ErrScene, got %v", err)
	}
	if _, err := svc.Send(context.Background(), SceneLogin, "12345"); !errors.Is(err, ErrPhone) {
		t.Fatalf("bad phone should be ErrPhone, got %v", err)
	}
}

func TestSendCooldown(t *testing.T) {
	rdb := newFakeRedis()
	svc := newTestService(rdb, &fakeSender{})

	if _, err := svc.Send(context.Background(), SceneLogin, "13800001111"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	if _, err := svc.Send(context.Background(), SceneLogin, "13800001111"); !errors.Is(err, ErrCooldown) {
		t.Fatalf("second send within cooldown should be ErrCooldown, got %v", err)
	}
}

func TestVerifyWrongCodeAttempts(t *testing.T) {
	rdb := newFakeRedis()
	svc := NewService(rdb, &fakeSender{}, Options{MaxTries: 3})

	code, err := svc.Send(context.Background(), SceneLogin, "13800001111")
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	// 前两次错码：ErrCodeInvalid
	for range 2 {
		if err := svc.Verify(context.Background(), SceneLogin, "13800001111", "000000"); !errors.Is(err, ErrCodeInvalid) {
			t.Fatalf("wrong code should be ErrCodeInvalid, got %v", err)
		}
	}
	// 第三次错码达上限：作废并返回 ErrTooManyTries
	if err := svc.Verify(context.Background(), SceneLogin, "13800001111", "000000"); !errors.Is(err, ErrTooManyTries) {
		t.Fatalf("third wrong code should be ErrTooManyTries, got %v", err)
	}
	// 正确码也已作废
	if err := svc.Verify(context.Background(), SceneLogin, "13800001111", code); !errors.Is(err, ErrCodeInvalid) {
		t.Fatalf("code should be invalidated after max tries, got %v", err)
	}
}

func TestVerifyMissingCode(t *testing.T) {
	svc := newTestService(newFakeRedis(), &fakeSender{})
	if err := svc.Verify(context.Background(), SceneLogin, "13800001111", "123456"); !errors.Is(err, ErrCodeInvalid) {
		t.Fatalf("missing code should be ErrCodeInvalid, got %v", err)
	}
}
