package email

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// memoryRedis 只实现邮箱绑定服务使用到的原子语义；用于验证验证码、身份凭证
// 及重试状态是否确实绑定在同一个账号和邮箱作用域内。
type memoryRedis struct {
	values map[string]string
	tries  map[string]int
}

func newMemoryRedis() *memoryRedis {
	return &memoryRedis{values: map[string]string{}, tries: map[string]int{}}
}

func (m *memoryRedis) Get(_ context.Context, key string) *redis.StringCmd {
	value, ok := m.values[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(value, nil)
}

func (m *memoryRedis) Set(_ context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	m.values[key] = fmt.Sprint(value)
	return redis.NewStatusResult("OK", nil)
}

func (m *memoryRedis) SetNX(_ context.Context, key string, value interface{}, _ time.Duration) *redis.BoolCmd {
	if _, ok := m.values[key]; ok {
		return redis.NewBoolResult(false, nil)
	}
	m.values[key] = fmt.Sprint(value)
	return redis.NewBoolResult(true, nil)
}

func (m *memoryRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	var deleted int64
	for _, key := range keys {
		if _, ok := m.values[key]; ok {
			delete(m.values, key)
			deleted++
		}
		delete(m.tries, key)
	}
	return redis.NewIntResult(deleted, nil)
}

func (m *memoryRedis) Eval(_ context.Context, _ string, keys []string, args ...interface{}) *redis.Cmd {
	code, ticket := fmt.Sprint(args[0]), fmt.Sprint(args[1])
	maxTries, _ := strconv.Atoi(fmt.Sprint(args[2]))
	if m.values[keys[2]] != ticket {
		return redis.NewCmdResult(int64(-3), nil)
	}
	stored, ok := m.values[keys[0]]
	if !ok {
		return redis.NewCmdResult(int64(-1), nil)
	}
	if stored == code {
		delete(m.values, keys[0])
		delete(m.values, keys[2])
		delete(m.tries, keys[1])
		return redis.NewCmdResult(int64(1), nil)
	}
	m.tries[keys[1]]++
	if m.tries[keys[1]] >= maxTries {
		delete(m.values, keys[0])
		delete(m.tries, keys[1])
		return redis.NewCmdResult(int64(-2), nil)
	}
	return redis.NewCmdResult(int64(0), nil)
}

type captureSender struct {
	to   string
	code string
}

func (s *captureSender) Send(_ context.Context, to, code string) error {
	s.to, s.code = to, code
	return nil
}

func TestEmailBindingVerificationConsumesTicketAndCode(t *testing.T) {
	ctx := context.Background()
	rdb := newMemoryRedis()
	sender := new(captureSender)
	svc := NewService(rdb, sender, Options{FixedCode: DevFixedCode, DevEcho: true})

	ticket, err := svc.IssueIdentityTicket(ctx, 42)
	require.NoError(t, err)

	code, err := svc.SendCode(ctx, 42, ticket, "User@Example.COM")
	require.NoError(t, err)
	require.Equal(t, DevFixedCode, code)
	require.Equal(t, "user@example.com", sender.to)
	require.Equal(t, DevFixedCode, sender.code)

	_, err = svc.VerifyCode(ctx, 42, ticket, "user@example.com", "000000")
	require.ErrorIs(t, err, ErrCodeInvalid)

	address, err := svc.VerifyCode(ctx, 42, ticket, "user@example.com", code)
	require.NoError(t, err)
	require.Equal(t, "user@example.com", address)

	_, err = svc.VerifyCode(ctx, 42, ticket, "user@example.com", code)
	require.ErrorIs(t, err, ErrIdentityExpired)
}

func TestNormalizeAddressRejectsDisplayName(t *testing.T) {
	_, err := NormalizeAddress("灵衍云 <user@example.com>")
	require.ErrorIs(t, err, ErrEmailInvalid)
}
