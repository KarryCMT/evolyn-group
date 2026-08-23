// Package sms 短信验证码域（认证域子能力）：验证码生成/存储/校验与发送
// 通道抽象。存储走 Redis（TTL + 重发冷却 + 试错上限），通道一期仅开发
// 日志实现，生产接真实服务商时实现 Sender 即可，其余链路不变
package sms

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"time"

	"evolyn/internal/platform/httpx"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// 验证码场景白名单：登录/注册/找回密码重置/换绑手机号（P1-3）
const (
	SceneLogin    = "login"
	SceneRegister = "register"
	SceneReset    = "reset"
	// SceneRebind 换绑手机号：旧手机号验证「原身份持有」、新手机号验证
	// 「新号持有」，两个码同场景隔离于登录/注册（rebind 场景要求已登录，
	// 由控制器把关，防匿名向任意号码滥发）
	SceneRebind = "rebind"
)

var validScenes = map[string]struct{}{SceneLogin: {}, SceneRegister: {}, SceneReset: {}, SceneRebind: {}}

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
	// ErrIPLimit 单 IP 日限额（上线前整改 P2）：手机号日限额可被轮换手机号
	// 绕过，IP 维度兜底短信成本风控
	ErrIPLimit = httpx.NewBiz("AUTH_SMS_IP_LIMIT", "当前网络今日发送次数已达上限，请明天再试", http.StatusTooManyRequests)
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
	CodeTTL      time.Duration // 验证码有效期（默认 5 分钟）
	Cooldown     time.Duration // 重发冷却（默认 60 秒）
	MaxTries     int           // 单码最大试错次数（默认 5，超限作废需重发）
	DailyLimit   int           // 单手机号每日发送上限（默认 10，P2-7 防刷）
	IPDailyLimit int           // 单 IP 每日发送上限（默认 30，跨手机号/场景合计；防轮换手机号绕过手机号限额）
	DevEcho      bool          // 非生产联调：Send 返回后由调用方在响应中回显验证码
	FixedCode    string        // 开发/测试固定验证码（如 666666）：非空时 Send 存储该码替代随机码，生产通道必须留空
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
	if o.IPDailyLimit <= 0 {
		o.IPDailyLimit = 30
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

// ipDailyKey 单 IP 日发送计数（跨手机号/场景合计）：手机号日限额可被轮换
// 手机号绕过，IP 维度兜底短信成本风控。ip 为空（理论不应发生，gin ClientIP
// 恒有回落）时归入 unknown 桶，保持「无 IP 不放行」的保守口径
func ipDailyKey(ip string) string {
	if ip == "" {
		ip = "unknown"
	}
	return fmt.Sprintf("evolyn:sms:ipdaily:%s:%s", ip, time.Now().Format("20060102"))
}

// reserveDailyScript 原子预占当天的发送名额。预占发生在调用短信服务商前，
// 从根源避免并发请求都通过旧的 GET 计数检查而突破日限额。
// 返回值：1=预占成功，0=已达日上限。
const reserveDailyScript = `
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
if current >= tonumber(ARGV[1]) then return 0 end
local count = redis.call('INCR', KEYS[1])
if count == 1 then redis.call('EXPIRE', KEYS[1], ARGV[2]) end
return 1`

// releaseDailyScript 通道尚未真正发送时归还预占名额。发送已成功但 Redis
// 落码失败时不归还，避免攻击者利用存储异常绕过真实短信成本限制。
const releaseDailyScript = `
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
if current <= 1 then
  redis.call('DEL', KEYS[1])
  return 0
end
return redis.call('DECR', KEYS[1])`

// secondsUntilTomorrow 返回当前自然日剩余秒数。计数跨场景共享，并在下一天
// 零点自动过期，避免原 25 小时 TTL 让上限跨自然日延后恢复。
func secondsUntilTomorrow(now time.Time) int64 {
	nextDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	seconds := int64(nextDay.Sub(now).Seconds())
	if seconds < 1 {
		return 1
	}
	return seconds
}

// reserveDaily 原子预占日额度（手机号/IP 维度共用），Redis 故障时明确失败，
// 不允许在防刷状态未知时发送。返回 (是否预占成功, error)
func (s *Service) reserveDaily(ctx context.Context, key string, limit int) (bool, error) {
	res, err := s.rdb.Eval(ctx, reserveDailyScript, []string{key}, limit, secondsUntilTomorrow(time.Now())).Int64()
	if err != nil {
		return false, fmt.Errorf("redis reserve daily: %w", err)
	}
	if res == 0 {
		return false, nil
	}
	if res != 1 {
		return false, fmt.Errorf("unexpected reserve daily result: %d", res)
	}
	return true, nil
}

// releaseDaily 仅用于短信通道调用失败等“没有产生实际发送”的分支归还预占
// 名额；释放失败宁可保留该名额消耗，也不能放宽日限额
func (s *Service) releaseDaily(ctx context.Context, key string) {
	if _, err := s.rdb.Eval(ctx, releaseDailyScript, []string{key}).Int64(); err != nil {
		logrus.Warnf("release sms daily quota: %v", err)
	}
}

// Send 发送验证码：场景/手机号校验 → IP 日限额预占（成本风控第一道闸，
// 被拒请求不占用单手机号的冷却与额度）→ 冷却闸（SetNX 原子占位）→ 原子
// 预占手机号日限额 → 生成 6 位数字码 → 走通道发送 → 落码。
// 未产生真实发送的分支回滚冷却并归还两维度日额度；通道已成功但落码失败时
// 取消冷却并清理旧码，允许用户立即重试且不会误用上一次验证码（两维度日
// 额度保留为已消耗，真实短信成本不能被存储异常绕过）
func (s *Service) Send(ctx context.Context, scene, phone, ip string) (string, error) {
	if _, ok := validScenes[scene]; !ok {
		return "", ErrScene
	}
	if !phonePattern.MatchString(phone) {
		return "", ErrPhone
	}

	ipOK, err := s.reserveDaily(ctx, ipDailyKey(ip), s.opts.IPDailyLimit)
	if err != nil {
		return "", err
	}
	if !ipOK {
		return "", ErrIPLimit
	}

	ok, err := s.rdb.SetNX(ctx, coolKey(scene, phone), 1, s.opts.Cooldown).Result()
	if err != nil {
		s.releaseDaily(ctx, ipDailyKey(ip))
		return "", fmt.Errorf("redis setnx cooldown: %w", err)
	}
	if !ok {
		s.releaseDaily(ctx, ipDailyKey(ip))
		return "", ErrCooldown
	}
	phoneOK, err := s.reserveDaily(ctx, dailyKey(phone), s.opts.DailyLimit)
	if err != nil {
		s.rdb.Del(ctx, coolKey(scene, phone))
		s.releaseDaily(ctx, ipDailyKey(ip))
		return "", err
	}
	if !phoneOK {
		s.rdb.Del(ctx, coolKey(scene, phone))
		s.releaseDaily(ctx, ipDailyKey(ip))
		return "", ErrDailyLimit
	}

	code, err := s.pickCode()
	if err != nil {
		s.rdb.Del(ctx, coolKey(scene, phone))
		s.releaseDaily(ctx, dailyKey(phone))
		s.releaseDaily(ctx, ipDailyKey(ip))
		return "", err
	}

	// 先发送后落码（P2-6）：通道失败时回滚冷却占位，用户可立即重试，
	// 也不会出现「已覆盖旧码但新码未送达」的中间态
	if err := s.sender.Send(ctx, phone, code); err != nil {
		s.rdb.Del(ctx, coolKey(scene, phone))
		s.releaseDaily(ctx, dailyKey(phone))
		s.releaseDaily(ctx, ipDailyKey(ip))
		return "", fmt.Errorf("send sms: %w", err)
	}

	// 新码覆盖旧码并清零试错计数（旧码立即失效）
	if _, err := s.rdb.Set(ctx, codeKey(scene, phone), code, s.opts.CodeTTL).Result(); err != nil {
		// 发送已发生但新码无法验证：清掉冷却及可能残留的旧码，让用户能立刻
		// 重试；手机号与 IP 两维度日额度均保留为已消耗
		s.rdb.Del(ctx, coolKey(scene, phone), codeKey(scene, phone), triesKey(scene, phone))
		return "", fmt.Errorf("redis set code: %w", err)
	}
	s.rdb.Del(ctx, triesKey(scene, phone))

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
