package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// LoginGuardOptions 登录失败锁定参数：零值回落默认（5 次 / 15 分钟）
type LoginGuardOptions struct {
	MaxFails     int           // 任意窗口内连续失败上限（默认 5）
	LockDuration time.Duration // 滑动窗口宽度与锁定时长（默认 15 分钟，两者同值）
	// Secret 标识散列独立密钥（auth.loginGuardSecret）：非空时用 HMAC-SHA-256
	// 防字典反查——手机号/常见用户名原像空间小，无密钥散列可被彩虹表穷举。
	// 留空回退无密钥 SHA-256（保持可用性，生产建议配置；多实例必须共享同一把）
	Secret string
}

// failScript 原子「计数 → 刷新窗口 → 达标落锁」（上线前复查 P2：原实现
// INCR/EXPIRE/SET 三步分离，并发失败可在窗口内漂移计数绕过阈值）。
// 每次失败都刷新计数 TTL——真滑动窗口语义：任意 MaxFails 次失败落在
// 任意一个 LockDuration 窗口内即锁定。返回 1 表示本次失败触发锁定
const failScript = `
local fails = redis.call('INCR', KEYS[1])
redis.call('EXPIRE', KEYS[1], ARGV[1])
if fails >= tonumber(ARGV[2]) then
  redis.call('SET', KEYS[2], 1, 'EX', ARGV[1])
  redis.call('DEL', KEYS[1])
  return 1
end
return 0`

// LoginGuard 密码登录失败锁定（上线前整改 P2）：以「登录标识」（登录名或
// 手机号，由控制器从请求取）为维度在 Redis 计连续失败，滑动窗口内累计达
// MaxFails 即锁定同一时长；成功登录清零计数。不存在的账号同样计数：既防
// 按名/号枚举探测，也让对任意标识的在线爆破同样被锁。
// Redis 异常时 fail-open（失败计数是增强能力，可用性优先，异常已记录
// 日志；与令牌吊销黑名单的默认降级口径一致）。
// 登录标识经散列后才进 Redis key：key 可能出现在监控面板、备份与慢日志中，
// 明文手机号/账号名属 PII 不落 key；配置独立密钥（Options.Secret）时用
// HMAC-SHA-256，无密钥散列可被字典反查（上线后加固项）。
// 本包不可 import httpx（ginctx→auth→httpx 存在环），锁定状态的对外
// 业务码（AUTH_LOGIN_LOCKED）由 auth/controller 定义并映射
type LoginGuard struct {
	rdb    guardRedis
	opts   LoginGuardOptions
	secret string
}

// guardRedis 本能力用到的 Redis 最小接口（*redis.Client 天然满足，单测可替身）
type guardRedis interface {
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	// Eval 执行 Lua 脚本（失败计数原子语义用，签名同 go-redis 原生）
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
}

func NewLoginGuard(rdb guardRedis, opts LoginGuardOptions) *LoginGuard {
	if opts.MaxFails <= 0 {
		opts.MaxFails = 5
	}
	if opts.LockDuration <= 0 {
		opts.LockDuration = 15 * time.Minute
	}
	return &LoginGuard{rdb: rdb, opts: opts, secret: opts.Secret}
}

// hashIdent 登录标识单向散列（hex 64 字符）后再进 Redis key：
// 配置了独立密钥用 HMAC-SHA-256（防彩虹表/字典反查），否则回退无密钥
// SHA-256。同一部署内密钥必须一致，否则各实例对同一标识会落到不同 key
func hashIdent(secret, ident string) string {
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(ident))
		return hex.EncodeToString(mac.Sum(nil))
	}
	sum := sha256.Sum256([]byte(ident))
	return hex.EncodeToString(sum[:])
}

func (g *LoginGuard) loginFailKey(ident string) string {
	return "evolyn:auth:fail:" + hashIdent(g.secret, ident)
}

func (g *LoginGuard) loginLockKey(ident string) string {
	return "evolyn:auth:lock:" + hashIdent(g.secret, ident)
}

// Locked 登录标识是否处于锁定期（控制器命中后以 AUTH_LOGIN_LOCKED 业务码
// 拒绝）。Redis 异常时返回 false（fail-open，见类型注释）
func (g *LoginGuard) Locked(ctx context.Context, ident string) bool {
	if ident == "" {
		return false
	}
	n, err := g.rdb.Exists(ctx, g.loginLockKey(ident)).Result()
	if err != nil {
		logrus.Warnf("login guard check lock: %v", err)
		return false
	}
	return n > 0
}

// RecordFailure 原子记录一次登录失败：滑动窗口计数（每次失败刷新 TTL），
// 达上限落锁定标记并清计数（锁定期满后从零重新计）。
// Redis 异常时放弃本次计数（fail-open），不能因计数故障阻断登录链路
func (g *LoginGuard) RecordFailure(ctx context.Context, ident string) {
	if ident == "" {
		return
	}
	res, err := g.rdb.Eval(ctx, failScript,
		[]string{g.loginFailKey(ident), g.loginLockKey(ident)},
		int64(g.opts.LockDuration.Seconds()), g.opts.MaxFails,
	).Int64()
	if err != nil {
		logrus.Warnf("login guard record failure: %v", err)
		return
	}
	if res == 1 {
		// 触发锁定是安全事件：留 warn 级线索（标识已散列，不可反查明文）
		logrus.Warn("login guard: identifier locked after repeated failures")
	}
}

// Reset 登录成功后清零失败计数（「连续」失败语义：成功即打断累计）
func (g *LoginGuard) Reset(ctx context.Context, ident string) {
	if ident == "" {
		return
	}
	g.rdb.Del(ctx, g.loginFailKey(ident))
}
