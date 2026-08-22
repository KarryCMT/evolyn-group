// Package sms 短信验证码域（认证域子能力）：验证码生成/存储/校验与发送
// 通道抽象。存储走 Redis（TTL + 重发冷却 + 试错上限），通道一期仅开发
// 日志实现，生产接真实服务商时实现 Sender 即可，其余链路不变
package sms

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"evolyn/internal/platform/httpx"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// 验证码场景白名单：登录/注册/找回密码重置（P1-3）
const (
	SceneLogin    = "login"
	SceneRegister = "register"
	SceneReset    = "reset"
)

var validScenes = map[string]struct{}{SceneLogin: {}, SceneRegister: {}, SceneReset: {}}

// DevFixedCode 开发/测试环境固定验证码（6 位，与随机码位数口径一致）；
// 仅在 provider=dev 时经 Options.FixedCode 启用，生产通道不受影响
const DevFixedCode = "666666"

// 业务错误（ADR-008：BizError 稳定码，ResponseFailed 自动映射状态码/文案）
var (
	ErrScene        = httpx.NewBiz("AUTH_SMS_SCENE_INVALID", "不支持的验证码场景", http.StatusBadRequest)
	ErrPhone        = httpx.NewBiz("AUTH_PHONE_INVALID", "手机号格式不正确", http.StatusBadRequest)
	ErrCooldown     = httpx.NewBiz("AUTH_COOLDOWN", "发送太频繁，请稍后再试", http.StatusTooManyRequests)
	ErrCodeInvalid  = httpx.NewBiz("AUTH_SMS_INVALID", "验证码错误或已过期", http.StatusUnauthorized)
	ErrTooManyTries = httpx.NewBiz("AUTH_SMS_TOO_MANY_TRIES", "尝试次数过多，请重新获取验证码", http.StatusTooManyRequests)
	ErrDailyLimit   = httpx.NewBiz("AUTH_SMS_DAILY_LIMIT", "今日发送次数已达上限，请明天再试", http.StatusTooManyRequests)
)

var phonePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)

// Sender 短信发送通道抽象：生产实现对接阿里云/腾讯云等服务商
type Sender interface {
	Send(ctx context.Context, phone, code string) error
}

// DevSender 开发通道：仅输出日志不外发，联调时配合 DevEcho 回显验证码
type DevSender struct{}

func NewDevSender() *DevSender { return &DevSender{} }

func (d *DevSender) Send(_ context.Context, phone, code string) error {
	logrus.Infof("[sms:dev] send code to %s", phone)
	return nil
}

// redisAPI 本域用到的 Redis 最小接口（*redis.Client 天然满足，单测可替身）
type redisAPI interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Incr(ctx context.Context, key string) *redis.IntCmd
	Expire(ctx context.Context, key string, ttl time.Duration) *redis.BoolCmd
	// Eval 执行 Lua 脚本（验证码原子消费用，签名同 go-redis 原生）
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
}

// verifyScript 原子「比较 → 消费/计数」（P1-4，消除 GET 后 DEL 的并发竞态：
// 两个并发请求不可能同时读到同一正确验证码）。返回值约定：
//
//	 1  命中并删除（码与试错计数）
//	 0  不匹配且未超限（试错计数已 +1）
//	-1  无码（未发送或已过期/已消费）
//	-2  试错超限（码与计数已清理，需重新发送）
const verifyScript = `
local stored = redis.call('GET', KEYS[1])
if not stored then return -1 end
if stored == ARGV[1] then
  redis.call('DEL', KEYS[1], KEYS[2])
  return 1
end
local tries = redis.call('INCR', KEYS[2])
if tries == 1 then redis.call('EXPIRE', KEYS[2], ARGV[3]) end
if tries >= tonumber(ARGV[2]) then
  redis.call('DEL', KEYS[1], KEYS[2])
  return -2
end
return 0`

// Options 服务参数：零值回落内置默认（见 normalize）
type Options struct {
	CodeTTL    time.Duration // 验证码有效期（默认 5 分钟）
	Cooldown   time.Duration // 重发冷却（默认 60 秒）
	MaxTries   int           // 单码最大试错次数（默认 5，超限作废需重发）
	DailyLimit int           // 单手机号每日发送上限（默认 10，P2-7 防刷）
	DevEcho    bool          // 非生产联调：Send 返回后由调用方在响应中回显验证码
	FixedCode  string        // 开发/测试固定验证码（如 666666）：非空时 Send 存储该码替代随机码，生产通道必须留空
}

func (o *Options) normalize() {
	if o.CodeTTL <= 0 {
		o.CodeTTL = 5 * time.Minute
	}
	if o.Cooldown <= 0 {
		o.Cooldown = 60 * time.Second
	}
	if o.MaxTries <= 0 {
		o.MaxTries = 5
	}
	if o.DailyLimit <= 0 {
		o.DailyLimit = 10
	}
}

type Service struct {
	rdb    redisAPI
	sender Sender
	opts   Options
}

func NewService(rdb redisAPI, sender Sender, opts Options) *Service {
	opts.normalize()
	return &Service{rdb: rdb, sender: sender, opts: opts}
}

// EchoEnabled 是否在响应中回显验证码（仅本地联调配置可开）
func (s *Service) EchoEnabled() bool { return s.opts.DevEcho }

func codeKey(scene, phone string) string  { return fmt.Sprintf("evolyn:sms:code:%s:%s", scene, phone) }
func coolKey(scene, phone string) string  { return fmt.Sprintf("evolyn:sms:cool:%s:%s", scene, phone) }
func triesKey(scene, phone string) string { return fmt.Sprintf("evolyn:sms:tries:%s:%s", scene, phone) }

// dailyKey 单手机号日发送计数（跨场景合计，P2-7 防刷）
func dailyKey(phone string) string {
	return fmt.Sprintf("evolyn:sms:daily:%s:%s", phone, time.Now().Format("20060102"))
}

// Send 发送验证码：场景/手机号校验 → 日限额 → 冷却闸（SetNX 原子占位）→
// 生成 6 位数字码 → 先走通道发送（失败回滚冷却键，无状态残留）→
// 成功后落码并清零试错、累加日计数。冷却期内重复请求直接拒绝
func (s *Service) Send(ctx context.Context, scene, phone string) (string, error) {
	if _, ok := validScenes[scene]; !ok {
		return "", ErrScene
	}
	if !phonePattern.MatchString(phone) {
		return "", ErrPhone
	}

	// 日限额（P2-7）：已达上限直接拒绝，未达上限先不计数（发送成功才计数）
	if sent, err := s.rdb.Get(ctx, dailyKey(phone)).Result(); err == nil {
		if n, _ := strconv.ParseInt(sent, 10, 64); n >= int64(s.opts.DailyLimit) {
			return "", ErrDailyLimit
		}
	} else if !errors.Is(err, redis.Nil) {
		return "", fmt.Errorf("redis get daily: %w", err)
	}

	ok, err := s.rdb.SetNX(ctx, coolKey(scene, phone), 1, s.opts.Cooldown).Result()
	if err != nil {
		return "", fmt.Errorf("redis setnx cooldown: %w", err)
	}
	if !ok {
		return "", ErrCooldown
	}

	code, err := s.pickCode()
	if err != nil {
		return "", err
	}

	// 先发送后落码（P2-6）：通道失败时回滚冷却占位，用户可立即重试，
	// 也不会出现「已覆盖旧码但新码未送达」的中间态
	if err := s.sender.Send(ctx, phone, code); err != nil {
		s.rdb.Del(ctx, coolKey(scene, phone))
		return "", fmt.Errorf("send sms: %w", err)
	}

	// 新码覆盖旧码并清零试错计数（旧码立即失效）
	if _, err := s.rdb.Set(ctx, codeKey(scene, phone), code, s.opts.CodeTTL).Result(); err != nil {
		return "", fmt.Errorf("redis set code: %w", err)
	}
	s.rdb.Del(ctx, triesKey(scene, phone))

	// 发送成功后累加日计数，首次设置 25 小时过期（覆盖跨日边界余量）
	if n, err := s.rdb.Incr(ctx, dailyKey(phone)).Result(); err == nil && n == 1 {
		s.rdb.Expire(ctx, dailyKey(phone), 25*time.Hour)
	}

	return code, nil
}

// Verify 校验验证码（一次性）：Lua 原子完成「比较 → 删除/计数」（P1-4，
// 消除先 GET 后 DEL 的并发竞态——并发请求不可能同时通过同一验证码）。
// 未命中累计试错，达上限作废当前码需重发；冷却键不清理（发送节奏与校验解耦）
func (s *Service) Verify(ctx context.Context, scene, phone, code string) error {
	res, err := s.rdb.Eval(ctx, verifyScript,
		[]string{codeKey(scene, phone), triesKey(scene, phone)},
		code, s.opts.MaxTries, int(s.opts.CodeTTL.Seconds()),
	).Int64()
	if err != nil {
		return fmt.Errorf("redis eval verify: %w", err)
	}

	switch res {
	case 1:
		return nil
	case 0, -1: // 不匹配未超限 / 无码（未发送或已消费）
		return ErrCodeInvalid
	case -2:
		return ErrTooManyTries
	default:
		return fmt.Errorf("unexpected verify result: %d", res)
	}
}

// pickCode 选码：开发/测试配置了 FixedCode 时直接采用（跳过随机生成），
// 否则走 6 位随机数字码
func (s *Service) pickCode() (string, error) {
	if s.opts.FixedCode != "" {
		return s.opts.FixedCode, nil
	}
	return generateCode()
}

// generateCode 6 位数字验证码（保留前导零）
func generateCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
