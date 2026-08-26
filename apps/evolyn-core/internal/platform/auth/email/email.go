// Package email 提供账号安全操作所需的邮箱验证码与短时身份凭证。
//
// 邮箱绑定不是普通资料更新：先通过当前手机号证明账号持有，再向新邮箱发送
// 一次性验证码。身份凭证、验证码、重发冷却和错误次数均存 Redis，服务端
// 重启或多实例部署时仍共享同一安全状态。
package email

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"evolyn/internal/platform/httpx"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

const (
	// DevFixedCode 仅用于开发与测试通道，生产环境不得启用。
	DevFixedCode = "666666"
	maxEmailLen  = 256
)

var (
	// ErrIdentityExpired 表示当前手机号验证已过期，须重新完成第一步。
	ErrIdentityExpired = httpx.NewBiz("AUTH_EMAIL_IDENTITY_EXPIRED", "身份验证已过期，请重新验证", http.StatusUnauthorized)
	// ErrEmailInvalid 统一拒绝不规范的邮箱地址，避免向错误地址发送邮件。
	ErrEmailInvalid = httpx.NewBiz("AUTH_EMAIL_INVALID", "邮箱格式不正确", http.StatusBadRequest)
	// ErrCodeInvalid 不区分验证码错误、过期和已消费，避免泄露验证码状态。
	ErrCodeInvalid = httpx.NewBiz("AUTH_EMAIL_CODE_INVALID", "邮箱验证码错误或已过期", http.StatusUnauthorized)
	// ErrTooManyTries 达到单验证码允许的最大试错次数，必须重新获取。
	ErrTooManyTries = httpx.NewBiz("AUTH_EMAIL_TOO_MANY_TRIES", "尝试次数过多，请重新获取验证码", http.StatusTooManyRequests)
	// ErrCooldown 拦截短时间内重复发送，控制邮件通道成本与骚扰风险。
	ErrCooldown = httpx.NewBiz("AUTH_EMAIL_COOLDOWN", "发送太频繁，请稍后再试", http.StatusTooManyRequests)
)

// Sender 是邮件通道抽象。生产可接 SMTP 或云邮件服务，业务状态机不随通道改变。
type Sender interface {
	Send(ctx context.Context, to, code string) error
}

// DevSender 本地开发通道仅记录脱敏收件地址与验证码，不向外部发送邮件。
type DevSender struct{}

func NewDevSender() *DevSender { return &DevSender{} }

func (d *DevSender) Send(_ context.Context, to, code string) error {
	logrus.Infof("[email:dev] send code to %s: %s", maskAddress(to), code)
	return nil
}

// Options 是验证码运行参数；零值采用与短信验证码一致的安全默认值。
type Options struct {
	CodeTTL     time.Duration
	Cooldown    time.Duration
	MaxTries    int
	IdentityTTL time.Duration
	DevEcho     bool
	FixedCode   string
}

func (o *Options) normalize() {
	if o.CodeTTL <= 0 {
		o.CodeTTL = 5 * time.Minute
	}
	if o.Cooldown <= 0 {
		o.Cooldown = time.Minute
	}
	if o.MaxTries <= 0 {
		o.MaxTries = 5
	}
	if o.IdentityTTL <= 0 {
		o.IdentityTTL = 5 * time.Minute
	}
}

// redisAPI 收窄到邮箱验证码所需方法，便于单测替身并避免泄露整个 Redis 客户端。
type redisAPI interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
}

// Service 管理邮箱绑定的两段安全证明：当前手机号证明生成 ticket，邮箱验证码
// 与 ticket、账号和目标邮箱三者绑定，不能跨账号或跨邮箱重放。
type Service struct {
	rdb    redisAPI
	sender Sender
	opts   Options
}

func NewService(rdb redisAPI, sender Sender, opts Options) *Service {
	opts.normalize()
	return &Service{rdb: rdb, sender: sender, opts: opts}
}

// IssueIdentityTicket 为已通过当前手机号验证的账号签发一次性短时凭证。
func (s *Service) IssueIdentityTicket(ctx context.Context, accountID uint) (string, error) {
	if accountID == 0 {
		return "", ErrIdentityExpired
	}
	ticket, err := randomToken()
	if err != nil {
		return "", err
	}
	if err := s.rdb.Set(ctx, identityKey(accountID), ticket, s.opts.IdentityTTL).Err(); err != nil {
		return "", fmt.Errorf("store email bind identity ticket: %w", err)
	}
	return ticket, nil
}

// SendCode 先确认身份凭证，再发送与该账号、邮箱绑定的验证码；成功时开发模式
// 可回显固定码，生产模式始终返回空字符串。
func (s *Service) SendCode(ctx context.Context, accountID uint, ticket, address string) (string, error) {
	address, err := NormalizeAddress(address)
	if err != nil {
		return "", err
	}
	if err := s.checkIdentityTicket(ctx, accountID, ticket); err != nil {
		return "", err
	}

	cooldown := cooldownKey(accountID, address)
	ok, err := s.rdb.SetNX(ctx, cooldown, 1, s.opts.Cooldown).Result()
	if err != nil {
		return "", fmt.Errorf("set email verification cooldown: %w", err)
	}
	if !ok {
		return "", ErrCooldown
	}

	code, err := s.pickCode()
	if err != nil {
		s.rdb.Del(ctx, cooldown)
		return "", err
	}
	if err := s.sender.Send(ctx, address, code); err != nil {
		s.rdb.Del(ctx, cooldown)
		return "", fmt.Errorf("send email verification code: %w", err)
	}
	if err := s.rdb.Set(ctx, codeKey(accountID, address), code, s.opts.CodeTTL).Err(); err != nil {
		// 邮件已经送出但无法验证时，删除冷却以便用户可以尽快重新获取。
		s.rdb.Del(ctx, cooldown, codeKey(accountID, address), triesKey(accountID, address))
		return "", fmt.Errorf("store email verification code: %w", err)
	}
	s.rdb.Del(ctx, triesKey(accountID, address))

	if s.opts.DevEcho {
		return code, nil
	}
	return "", nil
}

// VerifyCode 原子消费身份凭证和验证码，避免并发请求重复绑定邮箱。
func (s *Service) VerifyCode(ctx context.Context, accountID uint, ticket, address, code string) (string, error) {
	address, err := NormalizeAddress(address)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(ticket) == "" || strings.TrimSpace(code) == "" {
		return "", ErrCodeInvalid
	}

	res, err := s.rdb.Eval(ctx, verifyScript,
		[]string{codeKey(accountID, address), triesKey(accountID, address), identityKey(accountID)},
		code, ticket, s.opts.MaxTries, int(s.opts.CodeTTL.Seconds()),
	).Int64()
	if err != nil {
		return "", fmt.Errorf("verify email verification code: %w", err)
	}
	switch res {
	case 1:
		return address, nil
	case -3:
		return "", ErrIdentityExpired
	case 0, -1:
		return "", ErrCodeInvalid
	case -2:
		return "", ErrTooManyTries
	default:
		return "", fmt.Errorf("unexpected email verification result: %d", res)
	}
}

func (s *Service) checkIdentityTicket(ctx context.Context, accountID uint, ticket string) error {
	if accountID == 0 || strings.TrimSpace(ticket) == "" {
		return ErrIdentityExpired
	}
	stored, err := s.rdb.Get(ctx, identityKey(accountID)).Result()
	if err == redis.Nil {
		return ErrIdentityExpired
	}
	if err != nil {
		return fmt.Errorf("get email bind identity ticket: %w", err)
	}
	if stored != ticket {
		return ErrIdentityExpired
	}
	return nil
}

func (s *Service) pickCode() (string, error) {
	if s.opts.FixedCode != "" {
		return s.opts.FixedCode, nil
	}
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("generate email verification code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// NormalizeAddress 仅接受裸邮箱地址，统一小写后作为 Redis scope，避免大小写
// 差异绕过冷却或复用其他地址的验证码。
func NormalizeAddress(address string) (string, error) {
	address = strings.ToLower(strings.TrimSpace(address))
	if len(address) == 0 || len(address) > maxEmailLen {
		return "", ErrEmailInvalid
	}
	parsed, err := mail.ParseAddress(address)
	if err != nil || parsed.Address != address {
		return "", ErrEmailInvalid
	}
	return address, nil
}

func identityKey(accountID uint) string { return fmt.Sprintf("auth:email-bind:identity:%d", accountID) }

func codeKey(accountID uint, address string) string {
	return fmt.Sprintf("auth:email-bind:code:%d:%s", accountID, addressHash(address))
}

func triesKey(accountID uint, address string) string {
	return fmt.Sprintf("auth:email-bind:tries:%d:%s", accountID, addressHash(address))
}

func cooldownKey(accountID uint, address string) string {
	return fmt.Sprintf("auth:email-bind:cooldown:%d:%s", accountID, addressHash(address))
}

func addressHash(address string) string {
	sum := sha256.Sum256([]byte(address))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomToken() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("generate email bind identity ticket: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func maskAddress(address string) string {
	parts := strings.Split(address, "@")
	if len(parts) != 2 || parts[0] == "" {
		return "***"
	}
	local := []rune(parts[0])
	if len(local) == 1 {
		return "*@" + parts[1]
	}
	return string(local[:1]) + "***@" + parts[1]
}

// verifyScript 在 Redis 侧完成「身份凭证比对 → 验证码比较 → 消费/计数」，
// 防止两个并发 PUT 请求同时用同一验证码成功。
const verifyScript = `
local ticket = redis.call('GET', KEYS[3])
if not ticket or ticket ~= ARGV[2] then return -3 end
local stored = redis.call('GET', KEYS[1])
if not stored then return -1 end
if stored == ARGV[1] then
  redis.call('DEL', KEYS[1], KEYS[2], KEYS[3])
  return 1
end
local tries = redis.call('INCR', KEYS[2])
if tries == 1 then redis.call('EXPIRE', KEYS[2], ARGV[4]) end
if tries >= tonumber(ARGV[3]) then
  redis.call('DEL', KEYS[1], KEYS[2])
  return -2
end
return 0`
