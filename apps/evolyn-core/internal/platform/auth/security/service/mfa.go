package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"evolyn/internal/platform/auth/security/model"
	"evolyn/internal/platform/auth/security/repository"
	"evolyn/internal/platform/auth/security/totp"
	"evolyn/internal/platform/httpx"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// MFA 业务错误均为稳定码；验证码、挑战与恢复码均使用同一模糊文案，避免
// 向攻击者透露账号是否绑定 MFA、挑战是否存在等内部状态。
var (
	ErrMFAUnavailable      = httpx.NewBiz("AUTH_MFA_UNAVAILABLE", "登录二次验证暂不可用，请稍后再试", http.StatusServiceUnavailable)
	ErrMFAInvalid          = httpx.NewBiz("AUTH_MFA_INVALID", "验证信息错误或已过期", http.StatusUnauthorized)
	ErrMFAChallengeExpired = httpx.NewBiz("AUTH_MFA_CHALLENGE_EXPIRED", "验证已过期，请重新登录", http.StatusUnauthorized)
	ErrMFAChallengeTries   = httpx.NewBiz("AUTH_MFA_CHALLENGE_TRIES_EXCEEDED", "尝试次数过多，请重新登录", http.StatusTooManyRequests)
	ErrMFAReauthRequired   = httpx.NewBiz("AUTH_REAUTH_REQUIRED", "请先完成身份验证", http.StatusUnauthorized)
	// ErrMFAReauthLoginRequired 用于存量 JWT：它们没有设备会话 SID，无法安全绑定
	// 一次性 reauthToken，必须重新登录以签发带 SID 的新令牌。
	ErrMFAReauthLoginRequired = httpx.NewBiz("AUTH_REAUTH_LOGIN_REQUIRED", "当前登录态不支持此安全操作，请重新登录", http.StatusUnauthorized)
)

const (
	mfaChallengeTTL   = 5 * time.Minute
	reauthTTL         = 5 * time.Minute
	mfaChallengeTries = 5
	recoveryCodeCount = 10
)

// redisMFAClient 收敛 MFA 所需 Redis 操作；挑战状态必须持久于 Redis，Redis
// 不可用时拒绝进入 MFA 流程，不能回退为无二次验证登录。
type redisMFAClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
}

// MFAService 负责 TOTP 因子、恢复码、短时 challenge 与 reauthToken。账号/成员
// 解析仍留在认证控制器，防止 security 子域反向依赖 IAM 业务服务。
type MFAService interface {
	Enabled(ctx context.Context, accountID uint) (bool, error)
	Enroll(ctx context.Context, accountID uint, issuer, account string) (*TOTPEnrollment, error)
	ConfirmEnrollment(ctx context.Context, accountID uint, currentSID, enrollmentID, code string) ([]string, error)
	VerifyCode(ctx context.Context, accountID uint, method, code string) (string, error)
	Disable(ctx context.Context, accountID uint, currentSID string) error
	CreateLoginChallenge(ctx context.Context, input LoginChallengeInput) (string, error)
	ConsumeLoginChallenge(ctx context.Context, challengeID, method, code string) (*LoginChallenge, string, error)
	CreateReauthToken(ctx context.Context, accountID uint, sid, method, code string) (string, error)
	IssueReauthToken(ctx context.Context, accountID uint, sid string) (string, error)
	RequireReauth(ctx context.Context, accountID uint, sid, token string) error
}

type mfaService struct {
	tx       TxManager
	settings repository.SettingsRepository
	factors  repository.FactorRepository
	recovery repository.RecoveryRepository
	sessions repository.SessionRepository
	events   repository.EventRepository
	keyring  *totp.Keyring
	rdb      redisMFAClient
}

// NewMFAService 在 server 装配时仅于有效的 TOTP 密钥配置和 Redis 均可用时调用。
func NewMFAService(tx TxManager, settings repository.SettingsRepository, factors repository.FactorRepository,
	recovery repository.RecoveryRepository, sessions repository.SessionRepository, events repository.EventRepository,
	keyring *totp.Keyring, rdb redisMFAClient) MFAService {
	return &mfaService{tx: tx, settings: settings, factors: factors, recovery: recovery, sessions: sessions, events: events, keyring: keyring, rdb: rdb}
}

// TOTPEnrollment 是一次绑定向导的短时响应。Secret 仅存在 Redis；前端应将
// otpauthURL 渲染成二维码，不得把 secret 保存到本地存储或日志。
type TOTPEnrollment struct {
	EnrollmentID string `json:"enrollmentId"`
	OtpauthURL   string `json:"otpauthUrl"`
}

type pendingEnrollment struct {
	AccountID uint   `json:"accountId"`
	Secret    string `json:"secret"`
}

// LoginChallengeInput 是第一步登录成功后的最小上下文。账号与成员对象不写入
// Redis，验证通过后由认证域重新加载，避免短时缓存存放画像与认证凭证。
type LoginChallengeInput struct {
	AccountID  uint   `json:"accountId"`
	TenantID   uint   `json:"tenantId"`
	AuthMethod string `json:"authMethod"`
	SetCookie  bool   `json:"setCookie"`
}

type LoginChallenge struct {
	LoginChallengeInput
}

type reauthState struct {
	AccountID uint   `json:"accountId"`
	SID       string `json:"sid"`
}

func (s *mfaService) Enabled(ctx context.Context, accountID uint) (bool, error) {
	settings, err := s.settings.Get(ctx, accountID)
	if err != nil {
		return false, err
	}
	return settings.MFAEnabled, nil
}

func (s *mfaService) Enroll(ctx context.Context, accountID uint, issuer, account string) (*TOTPEnrollment, error) {
	if !s.available() {
		return nil, ErrMFAUnavailable
	}
	if factor, err := s.factors.GetActive(ctx, accountID, model.FactorTypeTotp); err == nil && factor.VerifiedAt != nil {
		return nil, httpx.NewBiz("AUTH_MFA_ALREADY_ENABLED", "登录二次验证已启用", http.StatusConflict)
	} else if err != nil && !isNotFound(err) {
		return nil, err
	}

	secret, err := totp.GenerateSecret()
	if err != nil {
		return nil, err
	}
	id, err := randomToken()
	if err != nil {
		return nil, err
	}
	pending, err := json.Marshal(pendingEnrollment{AccountID: accountID, Secret: secret})
	if err != nil {
		return nil, err
	}
	if err := s.rdb.Set(ctx, enrollmentKey(id), pending, mfaChallengeTTL).Err(); err != nil {
		return nil, fmt.Errorf("store mfa enrollment: %w", err)
	}
	return &TOTPEnrollment{EnrollmentID: id, OtpauthURL: totp.OtpauthURL(issuer, account, secret)}, nil
}

func (s *mfaService) ConfirmEnrollment(ctx context.Context, accountID uint, currentSID, enrollmentID, code string) ([]string, error) {
	if !s.available() {
		return nil, ErrMFAUnavailable
	}
	pending, err := s.loadEnrollment(ctx, enrollmentID)
	if err != nil {
		return nil, err
	}
	if pending.AccountID != accountID {
		return nil, ErrMFAInvalid
	}
	counter, ok, err := totp.Verify(pending.Secret, code, totp.At(time.Now().Unix()), 0)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrMFAInvalid
	}

	ciphertext, version, err := s.keyring.Encrypt(pending.Secret)
	if err != nil {
		return nil, err
	}
	codes, digests, err := newRecoveryCodes()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		if err := s.settings.LockAccountRow(tctx, accountID); err != nil {
			return err
		}
		if factor, err := s.factors.GetActive(tctx, accountID, model.FactorTypeTotp); err == nil && factor.VerifiedAt != nil {
			return httpx.NewBiz("AUTH_MFA_ALREADY_ENABLED", "登录二次验证已启用", http.StatusConflict)
		} else if err != nil && !isNotFound(err) {
			return err
		}
		if _, err := s.factors.Create(tctx, &model.MFAFactor{AccountID: accountID, Type: model.FactorTypeTotp, SecretCiphertext: ciphertext, KeyVersion: version, VerifiedAt: &now, LastUsedCounter: counter}); err != nil {
			return err
		}
		if err := s.recovery.Replace(tctx, accountID, digests); err != nil {
			return err
		}
		settings, err := s.settings.Get(tctx, accountID)
		if err != nil {
			return err
		}
		settings.MFAEnabled = true
		if err := s.settings.Upsert(tctx, settings); err != nil {
			return err
		}
		// 启用 MFA 属于高风险变更：保留操作者当前会话，立即撤销其他设备。
		_, err = s.sessions.RevokeOthers(tctx, accountID, currentSID, model.RevokeMFAChanged)
		return err
	}); err != nil {
		return nil, err
	}
	// 绑定成功后才消费向导；即使 Redis 删除短暂失败，重复提交也会被唯一活跃因子拦截。
	_ = s.rdb.Del(ctx, enrollmentKey(enrollmentID)).Err()
	s.appendEvent(ctx, accountID, "mfa_enabled", "")
	return codes, nil
}

func (s *mfaService) VerifyCode(ctx context.Context, accountID uint, method, code string) (string, error) {
	switch method {
	case model.MFAMethodTotp:
		return s.verifyTOTP(ctx, accountID, code)
	case model.MFAMethodRecovery:
		return s.verifyRecovery(ctx, accountID, code)
	default:
		return "", ErrMFAInvalid
	}
}

func (s *mfaService) Disable(ctx context.Context, accountID uint, currentSID string) error {
	return s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		factor, err := s.factors.GetActive(tctx, accountID, model.FactorTypeTotp)
		if err != nil {
			if isNotFound(err) {
				return ErrMFAInvalid
			}
			return err
		}
		if err := s.factors.Disable(tctx, factor.ID); err != nil {
			return err
		}
		if err := s.recovery.Replace(tctx, accountID, nil); err != nil {
			return err
		}
		settings, err := s.settings.Get(tctx, accountID)
		if err != nil {
			return err
		}
		settings.MFAEnabled = false
		if err := s.settings.Upsert(tctx, settings); err != nil {
			return err
		}
		_, err = s.sessions.RevokeOthers(tctx, accountID, currentSID, model.RevokeMFAChanged)
		return err
	})
}

func (s *mfaService) CreateLoginChallenge(ctx context.Context, input LoginChallengeInput) (string, error) {
	if !s.available() {
		return "", ErrMFAUnavailable
	}
	id, err := randomToken()
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(LoginChallenge{LoginChallengeInput: input})
	if err != nil {
		return "", err
	}
	if err := s.rdb.Set(ctx, challengeKey(id), payload, mfaChallengeTTL).Err(); err != nil {
		return "", fmt.Errorf("store mfa challenge: %w", err)
	}
	return id, nil
}

func (s *mfaService) ConsumeLoginChallenge(ctx context.Context, challengeID, method, code string) (*LoginChallenge, string, error) {
	challenge, err := s.loadChallenge(ctx, challengeID)
	if err != nil {
		return nil, "", err
	}
	mfaMethod, err := s.VerifyCode(ctx, challenge.AccountID, method, code)
	if err != nil {
		if s.recordChallengeFailure(ctx, challengeID) {
			return nil, "", ErrMFAChallengeTries
		}
		return nil, "", err
	}
	// 验证码消费由数据库条件更新保证至多一次；随后 GETDEL 保证只有胜出的
	// 请求能获得第一步登录上下文，从而至多签发一条会话。
	res, err := s.rdb.Eval(ctx, getDeleteScript, []string{challengeKey(challengeID), challengeTriesKey(challengeID)}).Text()
	if err != nil {
		return nil, "", fmt.Errorf("consume mfa challenge: %w", err)
	}
	if res == "" {
		return nil, "", ErrMFAChallengeExpired
	}
	return challenge, mfaMethod, nil
}

func (s *mfaService) CreateReauthToken(ctx context.Context, accountID uint, sid, method, code string) (string, error) {
	if _, err := s.VerifyCode(ctx, accountID, method, code); err != nil {
		return "", err
	}
	return s.IssueReauthToken(ctx, accountID, sid)
}

func (s *mfaService) IssueReauthToken(ctx context.Context, accountID uint, sid string) (string, error) {
	if !s.available() {
		return "", ErrMFAUnavailable
	}
	id, err := randomToken()
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(reauthState{AccountID: accountID, SID: sid})
	if err != nil {
		return "", err
	}
	if err := s.rdb.Set(ctx, reauthKey(id), payload, reauthTTL).Err(); err != nil {
		return "", fmt.Errorf("store reauth token: %w", err)
	}
	return id, nil
}

func (s *mfaService) RequireReauth(ctx context.Context, accountID uint, sid, token string) error {
	if !s.available() || token == "" {
		return ErrMFAReauthRequired
	}
	raw, err := s.rdb.Eval(ctx, getDeleteScript, []string{reauthKey(token)}).Text()
	if err != nil || raw == "" {
		return ErrMFAReauthRequired
	}
	state := new(reauthState)
	if err := json.Unmarshal([]byte(raw), state); err != nil || state.AccountID != accountID || state.SID != sid {
		return ErrMFAReauthRequired
	}
	return nil
}

func (s *mfaService) verifyTOTP(ctx context.Context, accountID uint, code string) (string, error) {
	factor, err := s.factors.GetActive(ctx, accountID, model.FactorTypeTotp)
	if err != nil || factor.VerifiedAt == nil {
		return "", ErrMFAInvalid
	}
	secret, err := s.keyring.Decrypt(factor.KeyVersion, factor.SecretCiphertext)
	if err != nil {
		return "", err
	}
	counter, ok, err := totp.Verify(secret, code, totp.At(time.Now().Unix()), factor.LastUsedCounter)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrMFAInvalid
	}
	consumed, err := s.factors.ConsumeCounter(ctx, factor.ID, counter)
	if err != nil {
		return "", err
	}
	if !consumed {
		return "", ErrMFAInvalid
	}
	return model.MFAMethodTotp, nil
}

func (s *mfaService) verifyRecovery(ctx context.Context, accountID uint, code string) (string, error) {
	digest := digestRecoveryCode(code)
	codes, err := s.recovery.ListAvailable(ctx, accountID)
	if err != nil {
		return "", err
	}
	for _, candidate := range codes {
		if subtle.ConstantTimeCompare([]byte(candidate.CodeDigest), []byte(digest)) != 1 {
			continue
		}
		consumed, err := s.recovery.Consume(ctx, candidate.ID)
		if err != nil {
			return "", err
		}
		if consumed {
			s.appendEvent(ctx, accountID, "mfa_recovery_used", "")
			return model.MFAMethodRecovery, nil
		}
		return "", ErrMFAInvalid
	}
	return "", ErrMFAInvalid
}

func (s *mfaService) available() bool { return s != nil && s.keyring != nil && s.rdb != nil }

func (s *mfaService) loadEnrollment(ctx context.Context, id string) (*pendingEnrollment, error) {
	raw, err := s.rdb.Get(ctx, enrollmentKey(id)).Bytes()
	if err != nil || len(raw) == 0 {
		return nil, ErrMFAChallengeExpired
	}
	pending := new(pendingEnrollment)
	if err := json.Unmarshal(raw, pending); err != nil {
		return nil, ErrMFAChallengeExpired
	}
	return pending, nil
}

func (s *mfaService) loadChallenge(ctx context.Context, id string) (*LoginChallenge, error) {
	raw, err := s.rdb.Get(ctx, challengeKey(id)).Bytes()
	if err != nil || len(raw) == 0 {
		return nil, ErrMFAChallengeExpired
	}
	challenge := new(LoginChallenge)
	if err := json.Unmarshal(raw, challenge); err != nil {
		return nil, ErrMFAChallengeExpired
	}
	return challenge, nil
}

func (s *mfaService) recordChallengeFailure(ctx context.Context, id string) bool {
	// Lua 与 SMS 验证码同口径：失败次数的递增、首次 TTL 对齐和超限后的
	// challenge 删除必须是一个原子操作，不能用 GET 后 Set 造成并发绕过。
	result, err := s.rdb.Eval(ctx, challengeFailureScript,
		[]string{challengeKey(id), challengeTriesKey(id)}, mfaChallengeTries).Int64()
	return err == nil && result == 1
}

func (s *mfaService) appendEvent(ctx context.Context, accountID uint, eventType, sid string) {
	if s.events != nil {
		_ = s.events.Append(ctx, &model.SecurityEvent{AccountID: accountID, EventType: eventType, SessionID: sid, Metadata: model.EventMetadata{}})
	}
}

func isNotFound(err error) bool { return err == gorm.ErrRecordNotFound }

func enrollmentKey(id string) string { return "evolyn:mfa:enrollment:" + id }
func challengeKey(id string) string  { return "evolyn:mfa:challenge:" + id }
func challengeTriesKey(id string) string {
	return "evolyn:mfa:challenge-tries:" + id
}
func reauthKey(id string) string { return "evolyn:mfa:reauth:" + id }

const getDeleteScript = `
local val = redis.call('GET', KEYS[1])
if not val then return '' end
redis.call('DEL', unpack(KEYS))
return val`

const challengeFailureScript = `
local challenge = redis.call('GET', KEYS[1])
if not challenge then return -1 end
local count = redis.call('INCR', KEYS[2])
if count == 1 then
  local ttl = redis.call('TTL', KEYS[1])
  if ttl > 0 then redis.call('EXPIRE', KEYS[2], ttl) end
end
if count >= tonumber(ARGV[1]) then
  redis.call('DEL', KEYS[1], KEYS[2])
  return 1
end
return 0`

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func newRecoveryCodes() ([]string, []string, error) {
	codes := make([]string, 0, recoveryCodeCount)
	digests := make([]string, 0, recoveryCodeCount)
	for range recoveryCodeCount {
		buf := make([]byte, 10)
		if _, err := rand.Read(buf); err != nil {
			return nil, nil, err
		}
		code := strings.ToUpper(hex.EncodeToString(buf))
		codes = append(codes, code)
		digests = append(digests, digestRecoveryCode(code))
	}
	return codes, digests, nil
}

func digestRecoveryCode(code string) string {
	sum := sha256.Sum256([]byte(strings.ToUpper(strings.TrimSpace(code))))
	return hex.EncodeToString(sum[:])
}
