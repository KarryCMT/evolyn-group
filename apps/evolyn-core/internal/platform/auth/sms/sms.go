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
	"time"

	"evolyn/internal/platform/httpx"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// 验证码场景白名单：登录与注册两场景（找回密码复用同一套发送与校验）
const (
	SceneLogin    = "login"
	SceneRegister = "register"
)

var validScenes = map[string]struct{}{SceneLogin: {}, SceneRegister: {}}

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
}

// Options 服务参数：零值回落内置默认（见 normalize）
type Options struct {
	CodeTTL   time.Duration // 验证码有效期（默认 5 分钟）
	Cooldown  time.Duration // 重发冷却（默认 60 秒）
	MaxTries  int           // 单码最大试错次数（默认 5，超限作废需重发）
	DevEcho   bool          // 非生产联调：Send 返回后由调用方在响应中回显验证码
	FixedCode string        // 开发/测试固定验证码（如 666666）：非空时 Send 存储该码替代随机码，生产通道必须留空
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

// Send 发送验证码：场景/手机号校验 → 冷却闸（SetNX 原子占位）→ 生成 6 位
// 数字码 → 覆盖写入并清零试错 → 走通道发送。冷却期内重复请求直接拒绝
func (s *Service) Send(ctx context.Context, scene, phone string) (string, error) {
	if _, ok := validScenes[scene]; !ok {
		return "", ErrScene
	}
	if !phonePattern.MatchString(phone) {
		return "", ErrPhone
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

	// 新码覆盖旧码并清零试错计数（旧码立即失效）
	if _, err := s.rdb.Set(ctx, codeKey(scene, phone), code, s.opts.CodeTTL).Result(); err != nil {
		return "", fmt.Errorf("redis set code: %w", err)
	}
	s.rdb.Del(ctx, triesKey(scene, phone))

	if err := s.sender.Send(ctx, phone, code); err != nil {
		return "", fmt.Errorf("send sms: %w", err)
	}
	return code, nil
}

// Verify 校验验证码（一次性）：命中即删除防重放；未命中累计试错，
// 达上限作废当前码，需重新发送。冷却键不清理（发送节奏与校验解耦）
func (s *Service) Verify(ctx context.Context, scene, phone, code string) error {
	stored, err := s.rdb.Get(ctx, codeKey(scene, phone)).Result()
	if errors.Is(err, redis.Nil) {
		return ErrCodeInvalid
	}
	if err != nil {
		return fmt.Errorf("redis get code: %w", err)
	}

	if stored != code {
		tries, err := s.rdb.Incr(ctx, triesKey(scene, phone)).Result()
		if err != nil {
			return fmt.Errorf("redis incr tries: %w", err)
		}
		if tries == 1 {
			// 试错计数与验证码同生命周期，避免残留计数误伤后续新码
			s.rdb.Expire(ctx, triesKey(scene, phone), s.opts.CodeTTL)
		}
		if tries >= int64(s.opts.MaxTries) {
			s.rdb.Del(ctx, codeKey(scene, phone), triesKey(scene, phone))
			return ErrTooManyTries
		}
		return ErrCodeInvalid
	}

	s.rdb.Del(ctx, codeKey(scene, phone), triesKey(scene, phone))
	return nil
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
