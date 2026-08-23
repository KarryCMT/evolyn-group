package sms

import (
	"context"
	"errors"
	"fmt"
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

// Eval 实现短信域使用的 Lua 脚本：按 Go 等价逻辑模拟原子语义；真实并发
// 原子性由 Redis 保证。未知脚本直接失败，防止测试替身悄悄遗漏新脚本。
func (f *fakeRedis) Eval(_ context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	cmd := redis.NewCmd(context.Background())
	switch script {
	case reserveDailyScript:
		key := keys[0]
		limit, _ := strconv.ParseInt(toString(args[0]), 10, 64)
		current, _ := strconv.ParseInt(f.data[key], 10, 64)
		if current >= limit {
			cmd.SetVal(int64(0))
			return cmd
		}
		f.data[key] = strconv.FormatInt(current+1, 10)
		cmd.SetVal(int64(1))
		return cmd
	case releaseDailyScript:
		key := keys[0]
		current, _ := strconv.ParseInt(f.data[key], 10, 64)
		if current <= 1 {
			delete(f.data, key)
			cmd.SetVal(int64(0))
			return cmd
		}
		current--
		f.data[key] = strconv.FormatInt(current, 10)
		cmd.SetVal(current)
		return cmd
	case verifyScript:
		// 继续执行下方验证码消费语义。
	default:
		cmd.SetErr(fmt.Errorf("fakeRedis: unexpected script"))
		return cmd
	}

	codeKey, triesKey := keys[0], keys[1]
	code := toString(args[0])
	maxTries, _ := strconv.ParseInt(toString(args[1]), 10, 64)

	stored, ok := f.data[codeKey]
	if !ok {
		cmd.SetVal(int64(-1))
		return cmd
	}
	if stored == code {
		delete(f.data, codeKey)
		delete(f.data, triesKey)
		cmd.SetVal(int64(1))
		return cmd
	}

	tries := int64(0)
	if cur, ok := f.data[triesKey]; ok {
		tries, _ = strconv.ParseInt(cur, 10, 64)
	}
	tries++
	f.data[triesKey] = strconv.FormatInt(tries, 10)
	if tries >= maxTries {
		delete(f.data, codeKey)
		delete(f.data, triesKey)
		cmd.SetVal(int64(-2))
		return cmd
	}
	cmd.SetVal(int64(0))
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

	code, err := svc.Send(context.Background(), SceneLogin, "13800001111", "127.0.0.1")
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

	// reset 是合法场景（P1-3 密码找回），用真正未知的场景名断言拒绝
	if _, err := svc.Send(context.Background(), "bogus", "13800001111", "127.0.0.1"); !errors.Is(err, ErrScene) {
		t.Fatalf("unknown scene should be ErrScene, got %v", err)
	}
	if _, err := svc.Send(context.Background(), SceneReset, "13800001111", "127.0.0.1"); err != nil {
		t.Fatalf("reset scene should be valid, got %v", err)
	}
	if _, err := svc.Send(context.Background(), SceneLogin, "12345", "127.0.0.1"); !errors.Is(err, ErrPhone) {
		t.Fatalf("bad phone should be ErrPhone, got %v", err)
	}
}

func TestSecondsUntilTomorrow(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 8, 23, 23, 59, 30, 0, loc)
	if got := secondsUntilTomorrow(now); got != 30 {
		t.Fatalf("seconds until tomorrow = %d, want 30", got)
	}
}

// TestRegisterSceneIsolatedFromLogin 注册场景可用，且验证码与 login 场景
// 互不串用（scene 参与 Redis key）
func TestRegisterSceneIsolatedFromLogin(t *testing.T) {
	rdb, sender := newFakeRedis(), &fakeSender{}
	svc := newTestService(rdb, sender)

	loginCode, err := svc.Send(context.Background(), SceneLogin, "13800001111", "127.0.0.1")
	if err != nil {
		t.Fatalf("send login code: %v", err)
	}
	registerCode, err := svc.Send(context.Background(), SceneRegister, "13800001111", "127.0.0.1")
	if err != nil {
		t.Fatalf("send register code: %v", err)
	}

	// 注册码不能过 login 校验，登录码不能过 register 校验
	if err := svc.Verify(context.Background(), SceneLogin, "13800001111", registerCode); !errors.Is(err, ErrCodeInvalid) {
		t.Fatalf("register code should not pass login verify, got %v", err)
	}
	if err := svc.Verify(context.Background(), SceneRegister, "13800001111", loginCode); !errors.Is(err, ErrCodeInvalid) {
		t.Fatalf("login code should not pass register verify, got %v", err)
	}
	// 各自场景内校验通过
	if err := svc.Verify(context.Background(), SceneRegister, "13800001111", registerCode); err != nil {
		t.Fatalf("verify register code: %v", err)
	}
	if err := svc.Verify(context.Background(), SceneLogin, "13800001111", loginCode); err != nil {
		t.Fatalf("verify login code: %v", err)
	}
}

// TestFixedCode 开发/测试固定验证码：配置后 Send 恒返回该码且 Verify 通过；
// 未配置时维持 6 位随机码（前导零口径由正则覆盖）
func TestFixedCode(t *testing.T) {
	rdb := newFakeRedis()
	svc := NewService(rdb, &fakeSender{}, Options{MaxTries: 3, FixedCode: DevFixedCode})

	for range 2 {
		code, err := svc.Send(context.Background(), SceneRegister, "13800001111", "127.0.0.1")
		if err != nil {
			t.Fatalf("send: %v", err)
		}
		if code != DevFixedCode {
			t.Fatalf("code = %q, want fixed %q", code, DevFixedCode)
		}
		// 冷却期内重发被拒，需先清冷却键再发第二次（fakeRedis 不含 TTL 语义）
		delete(rdb.data, coolKey(SceneRegister, "13800001111"))
	}

	if err := svc.Verify(context.Background(), SceneRegister, "13800001111", DevFixedCode); err != nil {
		t.Fatalf("verify fixed code: %v", err)
	}
}

func TestSendCooldown(t *testing.T) {
	rdb := newFakeRedis()
	svc := newTestService(rdb, &fakeSender{})

	if _, err := svc.Send(context.Background(), SceneLogin, "13800001111", "127.0.0.1"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	if _, err := svc.Send(context.Background(), SceneLogin, "13800001111", "127.0.0.1"); !errors.Is(err, ErrCooldown) {
		t.Fatalf("second send within cooldown should be ErrCooldown, got %v", err)
	}
}

func TestVerifyWrongCodeAttempts(t *testing.T) {
	rdb := newFakeRedis()
	svc := NewService(rdb, &fakeSender{}, Options{MaxTries: 3})

	code, err := svc.Send(context.Background(), SceneLogin, "13800001111", "127.0.0.1")
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

// failSender 模拟通道发送失败（P2-6 回滚路径）
type failSender struct{}

func (f *failSender) Send(_ context.Context, _, _ string) error {
	return errors.New("provider down")
}

// TestVerifyAtomicConsume Lua 原子消费语义：命中即删（不可重放）、
// 不匹配计数、超限作废（P1-4）
func TestVerifyAtomicConsume(t *testing.T) {
	rdb, sender := newFakeRedis(), &fakeSender{}
	svc := NewService(rdb, sender, Options{MaxTries: 2, FixedCode: DevFixedCode})

	if _, err := svc.Send(context.Background(), SceneLogin, "13800001111", "127.0.0.1"); err != nil {
		t.Fatalf("send: %v", err)
	}

	// 错码两次 → 超限作废（MaxTries=2）
	if err := svc.Verify(context.Background(), SceneLogin, "13800001111", "000000"); !errors.Is(err, ErrCodeInvalid) {
		t.Fatalf("wrong code should be ErrCodeInvalid, got %v", err)
	}
	if err := svc.Verify(context.Background(), SceneLogin, "13800001111", "000000"); !errors.Is(err, ErrTooManyTries) {
		t.Fatalf("second wrong code should be ErrTooManyTries, got %v", err)
	}
	// 超限后正确码也已作废
	if err := svc.Verify(context.Background(), SceneLogin, "13800001111", DevFixedCode); !errors.Is(err, ErrCodeInvalid) {
		t.Fatalf("code should be invalidated after too many tries, got %v", err)
	}

	// 命中即删：同码二次校验失败（一次性）
	if _, err := svc.Send(context.Background(), SceneRegister, "13800001111", "127.0.0.1"); err != nil {
		t.Fatalf("send register: %v", err)
	}
	if err := svc.Verify(context.Background(), SceneRegister, "13800001111", DevFixedCode); err != nil {
		t.Fatalf("correct code should pass, got %v", err)
	}
	if err := svc.Verify(context.Background(), SceneRegister, "13800001111", DevFixedCode); !errors.Is(err, ErrCodeInvalid) {
		t.Fatalf("replayed code should fail, got %v", err)
	}
}

// TestSendRollbackCooldownOnFailure 通道发送失败时回滚冷却占位，
// 用户可立即重试（P2-6）
func TestSendRollbackCooldownOnFailure(t *testing.T) {
	rdb := newFakeRedis()
	svc := NewService(rdb, &failSender{}, Options{FixedCode: DevFixedCode})

	if _, err := svc.Send(context.Background(), SceneLogin, "13800001111", "127.0.0.1"); err == nil {
		t.Fatal("send should fail with failSender")
	}

	// 冷却已回滚：立即重发不被 ErrCooldown 拒绝（仍会因通道失败报错）
	if _, err := svc.Send(context.Background(), SceneLogin, "13800001111", "127.0.0.1"); errors.Is(err, ErrCooldown) {
		t.Fatal("cooldown should be rolled back after send failure")
	}
}

// TestSendDailyLimit 单手机号日限额：跨场景合计计数，超限拒绝（P2-7）
func TestSendDailyLimit(t *testing.T) {
	rdb, sender := newFakeRedis(), &fakeSender{}
	svc := NewService(rdb, sender, Options{DailyLimit: 2, FixedCode: DevFixedCode})

	phone := "13800002222"
	// 用不同场景避开 60 秒冷却：日限额按手机号跨场景合计
	for _, scene := range []string{SceneLogin, SceneRegister} {
		if _, err := svc.Send(context.Background(), scene, phone, "127.0.0.1"); err != nil {
			t.Fatalf("send %s: %v", scene, err)
		}
	}
	// 第三条（未用过的场景，无冷却冲突）被日限额拒绝
	if _, err := svc.Send(context.Background(), SceneReset, phone, "127.0.0.1"); !errors.Is(err, ErrDailyLimit) {
		t.Fatalf("third send should be ErrDailyLimit, got %v", err)
	}
	// 其他手机号不受影响
	if _, err := svc.Send(context.Background(), SceneRegister, "13800003333", "127.0.0.2"); err != nil {
		t.Fatalf("other phone should pass: %v", err)
	}
}

// TestSendIPDailyLimit 单 IP 日限额（上线前整改 P2）：跨手机号/场景合计，
// 防轮换手机号绕过手机号维度限额；不同 IP 互不影响
func TestSendIPDailyLimit(t *testing.T) {
	rdb, sender := newFakeRedis(), &fakeSender{}
	svc := NewService(rdb, sender, Options{IPDailyLimit: 2, FixedCode: DevFixedCode})

	// 同 IP 轮换手机号：前两条放行，第三条被 IP 限额拒绝
	if _, err := svc.Send(context.Background(), SceneLogin, "13800001111", "1.1.1.1"); err != nil {
		t.Fatalf("send 1: %v", err)
	}
	if _, err := svc.Send(context.Background(), SceneRegister, "13800002222", "1.1.1.1"); err != nil {
		t.Fatalf("send 2: %v", err)
	}
	if _, err := svc.Send(context.Background(), SceneReset, "13800003333", "1.1.1.1"); !errors.Is(err, ErrIPLimit) {
		t.Fatalf("third send from same ip should be ErrIPLimit, got %v", err)
	}
	// 其他 IP 不受影响
	if _, err := svc.Send(context.Background(), SceneLogin, "13800004444", "2.2.2.2"); err != nil {
		t.Fatalf("other ip should pass: %v", err)
	}
}

// TestSendCooldownReleasesIPQuota 冷却拒绝（未产生真实发送）归还 IP 预占
// 名额，不重复计数
func TestSendCooldownReleasesIPQuota(t *testing.T) {
	rdb := newFakeRedis()
	svc := NewService(rdb, &fakeSender{}, Options{FixedCode: DevFixedCode})

	if _, err := svc.Send(context.Background(), SceneLogin, "13800001111", "1.1.1.1"); err != nil {
		t.Fatalf("first send: %v", err)
	}
	if _, err := svc.Send(context.Background(), SceneLogin, "13800001111", "1.1.1.1"); !errors.Is(err, ErrCooldown) {
		t.Fatalf("second send within cooldown should be ErrCooldown, got %v", err)
	}
	if got := rdb.data[ipDailyKey("1.1.1.1")]; got != "1" {
		t.Fatalf("ip quota counter = %q, want 1 (cooldown rejection releases reservation)", got)
	}
}
