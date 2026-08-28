package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"evolyn/internal/utils/ratelimit"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Server       ServerConfig           `yaml:"server"`
	DB           DBConfig               `yaml:"db"`
	Redis        RedisConfig            `yaml:"redis"`
	Tenant       TenantRuntimeConfig    `yaml:"tenant"`
	OAuthConfig  map[string]OAuthConfig `yaml:"oauth"`
	SMS          SMSConfig              `yaml:"sms"`
	Email        EmailConfig            `yaml:"email"`
	Auth         AuthConfig             `yaml:"auth"`
	PKI          PKIConfig              `yaml:"pki"`
	Security     SecurityConfig         `yaml:"security"`
	Storage      StorageConfig          `yaml:"storage"`
	Notification NotificationConfig     `yaml:"notification"`
}

// AuthConfig 认证域运行参数（登录失败锁定与令牌吊销降级策略）。
// 零值回落内置默认（见 normalize），生产可按风控口径调整
type AuthConfig struct {
	LoginMaxFails    int `yaml:"loginMaxFails"`    // 窗口内连续登录失败上限（默认 5，达到即锁定）
	LoginLockMinutes int `yaml:"loginLockMinutes"` // 锁定时长分钟（默认 15；失败计窗口与锁定时长同值）
	// RevokeFailClosed 令牌吊销检查的 Redis 异常降级策略：false（默认）= 放行，
	// 可用性优先（吊销是增强能力）；true = 视为已吊销并拒绝请求，已泄露令牌
	// 的立即失效优先，代价是 Redis 故障期间全部携带 jti 的请求 401
	RevokeFailClosed bool `yaml:"revokeFailClosed"`
	// LoginGuardSecret 登录失败计数标识散列的独立密钥：非空时 HMAC-SHA-256
	// 防字典反查；留空回退无密钥 SHA-256（release 启动告警）。多实例必须
	// 共享同一把；示例模板占位值（CHANGE_ME）在归一化时视为未配置
	LoginGuardSecret string `yaml:"loginGuardSecret"`
}

// normalize 零值回落默认值：失败上限 5 次、锁定 15 分钟；
// 占位密钥归一化为未配置——示例模板的 CHANGE_ME 被原样复制到生产时，
// 若视为「已配置」会绕过 release 告警，而 HMAC 密钥实际公开可预测，
// 防字典反查静默失效
func (a *AuthConfig) normalize() {
	if a.LoginMaxFails <= 0 {
		a.LoginMaxFails = 5
	}
	if a.LoginLockMinutes <= 0 {
		a.LoginLockMinutes = 15
	}
	if isPlaceholderSecret(a.LoginGuardSecret) {
		a.LoginGuardSecret = ""
	}
}

// isPlaceholderSecret 判定是否为示例模板占位值：本仓库约定为 CHANGE_ME
// （忽略大小写与首尾空白），见 app.example.yaml
func isPlaceholderSecret(secret string) bool {
	return strings.EqualFold(strings.TrimSpace(secret), "CHANGE_ME")
}

// SMSConfig 短信验证码运行参数：一期开发通道（provider=dev 仅打日志），
// 生产接真实服务商时扩展 provider 与密钥字段
type SMSConfig struct {
	Provider        string `yaml:"provider"`        // 通道：dev（默认，日志不外发）
	CodeTTLSeconds  int    `yaml:"codeTtlSeconds"`  // 验证码有效期秒（默认 300）
	CooldownSeconds int    `yaml:"cooldownSeconds"` // 重发冷却秒（默认 60）
	MaxTries        int    `yaml:"maxTries"`        // 单码最大试错次数（默认 5）
	DailyLimit      int    `yaml:"dailyLimit"`      // 单手机号自然日发送上限（默认 10，跨场景合计）
	IPDailyLimit    int    `yaml:"ipDailyLimit"`    // 单 IP 自然日发送上限（默认 30，跨手机号/场景合计；防轮换手机号刷短信成本）
	// DevEcho 响应中回显验证码：仅本地联调可开，生产必须关闭
	DevEcho bool `yaml:"devEcho"`
}

// EmailConfig 是邮箱绑定验证码的通道与风控参数。开发环境使用 dev 通道，仅
// 记录脱敏地址和固定码；生产使用 smtp，凭据由未提交的部署配置注入。
type EmailConfig struct {
	Provider        string `yaml:"provider"`
	CodeTTLSeconds  int    `yaml:"codeTtlSeconds"`
	CooldownSeconds int    `yaml:"cooldownSeconds"`
	MaxTries        int    `yaml:"maxTries"`
	IdentityTTL     int    `yaml:"identityTtlSeconds"`
	DevEcho         bool   `yaml:"devEcho"`
	SMTPHost        string `yaml:"smtpHost"`
	SMTPPort        int    `yaml:"smtpPort"`
	SMTPUsername    string `yaml:"smtpUsername"`
	SMTPPassword    string `yaml:"smtpPassword"`
	SMTPFrom        string `yaml:"smtpFrom"`
	// SMTPImplicitTLS 用于 465 等服务端要求客户端在连接建立时即 TLS 握手的端口；
	// 留空时服务会将 465 自动识别为隐式 TLS，其余端口优先使用 STARTTLS。
	SMTPImplicitTLS bool `yaml:"smtpImplicitTls"`
}

// PKIConfig 登录口令加密传输密钥：公钥经 GET /app/conf 下发前端，
// 私钥仅服务端解密用。PrivateKey 为直接配置的 PEM，PrivateKeyFile 为 PEM 文件路径，
// 两者只能配置其一；均留空时启动随机生成一把（仅开发/测试，重启即轮换）。
// 生产与多实例部署必须显式配置同一密钥对。
type PKIConfig struct {
	Algorithm      string `yaml:"algorithm"`      // 一期固定 rsa
	PrivateKey     string `yaml:"privateKey"`     // PEM 私钥（真实私钥不入库，经环境配置注入）
	PrivateKeyFile string `yaml:"privateKeyFile"` // PEM 私钥文件路径（相对配置文件所在目录）
}

// SecurityConfig 是账号安全域的运行配置。TOTP 主密钥只允许由部署配置注入，
// 禁止写入数据库或提交到仓库。
type SecurityConfig struct {
	TOTP TOTPConfig `yaml:"totp"`
}

// TOTPConfig 支持多个在用主密钥：新因子用 CurrentKeyVersion 加密，历史因子
// 按自身 key_version 解密，从而使轮换不影响既有用户。
type TOTPConfig struct {
	CurrentKeyVersion int            `yaml:"currentKeyVersion"`
	MasterKeys        map[int]string `yaml:"masterKeys"`
}

// Validate 仅在启用 TOTP 能力时调用。配置必须包含当前版本，且每一版本均为
// 正整数，具体密钥格式由 totp.NewKeyring 校验。
func (t TOTPConfig) Validate() error {
	if t.CurrentKeyVersion <= 0 {
		return fmt.Errorf("security.totp.currentKeyVersion 必须大于 0")
	}
	if len(t.MasterKeys) == 0 {
		return fmt.Errorf("security.totp.masterKeys 不能为空")
	}
	if _, ok := t.MasterKeys[t.CurrentKeyVersion]; !ok {
		return fmt.Errorf("security.totp.masterKeys 缺少当前密钥版本 %d", t.CurrentKeyVersion)
	}
	for version, key := range t.MasterKeys {
		if version <= 0 || strings.TrimSpace(key) == "" {
			return fmt.Errorf("security.totp.masterKeys 包含无效密钥版本")
		}
	}
	return nil
}

type ServerConfig struct {
	ENV                    string                  `yaml:"env"`
	Address                string                  `yaml:"address"`
	Port                   int                     `yaml:"port"`
	GracefulShutdownPeriod int                     `yaml:"gracefulShutdownPeriod"`
	LimitConfigs           []ratelimit.LimitConfig `yaml:"rateLimits"`
	JWTSecret              string                  `yaml:"jwtSecret"`
	// AllowedOrigins CORS 允许携带凭证的来源白名单（精确匹配，含协议与端口）。
	// release 环境空白名单拒绝启动（fail-fast）；debug 空白名单回落放行
	// localhost/127.0.0.1 任意端口（本地联调）
	AllowedOrigins []string `yaml:"allowedOrigins"`
}

// DBConfig 数据库连接与 Schema 管理策略（FIX-009）：
// migrations（版本化 SQL，生产唯一来源）与 migrate（GORM AutoMigrate，
// 仅开发/测试）互斥，二者同开视为配置错误
type DBConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	Name       string `yaml:"name"`
	User       string `yaml:"user"`
	Password   string `yaml:"password"`
	Migrate    bool   `yaml:"migrate"`    // AutoMigrate（开发/测试）
	Migrations bool   `yaml:"migrations"` // 版本化 SQL Migration（生产）
}

// TenantRuntimeConfig 租户域运行参数（FIX-012）：注销数据保留期与清理周期
type TenantRuntimeConfig struct {
	RetentionDays        int `yaml:"retentionDays"`        // 注销后数据保留天数（默认 30）
	PurgeIntervalSeconds int `yaml:"purgeIntervalSeconds"` // Purge Worker 扫描周期秒（默认 3600）
}

// Retention 保留期换算（零/负值回落默认 30 天）
func (t TenantRuntimeConfig) Retention() time.Duration {
	if t.RetentionDays <= 0 {
		return 30 * 24 * time.Hour
	}
	return time.Duration(t.RetentionDays) * 24 * time.Hour
}

// PurgeInterval 清理周期换算（零/负值回落默认 1 小时）
func (t TenantRuntimeConfig) PurgeInterval() time.Duration {
	if t.PurgeIntervalSeconds <= 0 {
		return time.Hour
	}
	return time.Duration(t.PurgeIntervalSeconds) * time.Second
}

type RedisConfig struct {
	Enable   bool   `yaml:"enable"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
}

// StorageConfig 是 RustFS 的 S3 兼容连接配置。Endpoint 供服务端执行
// Stat/Delete 等数据面操作；ExternalEndpoint 专供预签名 URL，必须是浏览器
// 实际可访问的地址。密钥只允许经未提交的环境配置注入。
type StorageConfig struct {
	Enabled                      bool   `yaml:"enabled"`
	Endpoint                     string `yaml:"endpoint"`
	ExternalEndpoint             string `yaml:"externalEndpoint"`
	AccessKey                    string `yaml:"accessKey"`
	SecretKey                    string `yaml:"secretKey"`
	UseSSL                       bool   `yaml:"useSSL"`
	Bucket                       string `yaml:"bucket"`
	Prefix                       string `yaml:"prefix"`
	PresignTTLSeconds            int    `yaml:"presignTtlSeconds"`
	MaxUploadBytes               int64  `yaml:"maxUploadBytes"`
	UploadCleanupIntervalSeconds int    `yaml:"uploadCleanupIntervalSeconds"`
}

// PresignTTL 预签名有效期缺省 15 分钟，避免客户端拿到长期对象写权限。
func (s StorageConfig) PresignTTL() time.Duration {
	if s.PresignTTLSeconds <= 0 {
		return 15 * time.Minute
	}
	return time.Duration(s.PresignTTLSeconds) * time.Second
}

// UploadCleanupInterval 上传会话清理的扫描周期，零值回落 5 分钟。
func (s StorageConfig) UploadCleanupInterval() time.Duration {
	if s.UploadCleanupIntervalSeconds <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(s.UploadCleanupIntervalSeconds) * time.Second
}

// NotificationConfig 消息中心域运行参数（消息中心 P1/P2）：Outbox 物化与
// 过期清理的 Worker 节奏、保留期口径和自定义提醒对象租户上限。全部零值
// 回落内置默认，不配置即按默认运行
type NotificationConfig struct {
	OutboxIntervalSeconds    int `yaml:"outboxIntervalSeconds"`    // Outbox Worker 轮询间隔秒（默认 5）
	OutboxBatchSize          int `yaml:"outboxBatchSize"`          // Outbox 单轮领取事件数（默认 20，上限 200）
	RetentionIntervalSeconds int `yaml:"retentionIntervalSeconds"` // 过期清理扫描周期秒（默认 21600 即 6 小时）
	RetentionBatchSize       int `yaml:"retentionBatchSize"`       // 清理每批行数（默认 1000，上限 5000）
	RetentionMonths          int `yaml:"retentionMonths"`          // 消息保留期月数（默认 6；页面「保存最近六个月」口径）
	CustomRecipientLimit     int `yaml:"customRecipientLimit"`     // 租户自定义提醒对象上限（默认 200）
}

// OutboxInterval Outbox 轮询间隔（零/负值回落 5 秒）
func (n NotificationConfig) OutboxInterval() time.Duration {
	if n.OutboxIntervalSeconds <= 0 {
		return 5 * time.Second
	}
	return time.Duration(n.OutboxIntervalSeconds) * time.Second
}

// OutboxBatch Outbox 单轮领取批量（回落 20，钳制上限 200 防长事务）
func (n NotificationConfig) OutboxBatch() int {
	if n.OutboxBatchSize <= 0 {
		return 20
	}
	if n.OutboxBatchSize > 200 {
		return 200
	}
	return n.OutboxBatchSize
}

// RetentionInterval 过期清理周期（回落 6 小时）
func (n NotificationConfig) RetentionInterval() time.Duration {
	if n.RetentionIntervalSeconds <= 0 {
		return 6 * time.Hour
	}
	return time.Duration(n.RetentionIntervalSeconds) * time.Second
}

// RetentionBatch 清理单批行数（回落 1000，钳制上限 5000）
func (n NotificationConfig) RetentionBatch() int {
	if n.RetentionBatchSize <= 0 {
		return 1000
	}
	if n.RetentionBatchSize > 5000 {
		return 5000
	}
	return n.RetentionBatchSize
}

// RetentionMonthsValue 消息保留期月数（回落 6）
func (n NotificationConfig) RetentionMonthsValue() int {
	if n.RetentionMonths <= 0 {
		return 6
	}
	return n.RetentionMonths
}

// RecipientLimit 自定义提醒对象上限（回落 200）
func (n NotificationConfig) RecipientLimit() int {
	if n.CustomRecipientLimit <= 0 {
		return 200
	}
	return n.CustomRecipientLimit
}

func (s *StorageConfig) normalize() error {
	s.Endpoint = strings.TrimSpace(s.Endpoint)
	s.ExternalEndpoint = strings.TrimSpace(s.ExternalEndpoint)
	s.Bucket = strings.TrimSpace(s.Bucket)
	s.Prefix = strings.Trim(strings.TrimSpace(s.Prefix), "/")
	if !s.Enabled {
		return nil
	}
	if s.Endpoint == "" || s.AccessKey == "" || s.SecretKey == "" || s.Bucket == "" {
		return fmt.Errorf("storage 启用时 endpoint/accessKey/secretKey/bucket 均不能为空")
	}
	if s.MaxUploadBytes <= 0 {
		return fmt.Errorf("storage 启用时 maxUploadBytes 必须大于 0")
	}
	return nil
}

type OAuthConfig struct {
	AuthType     string `yaml:"authType"`
	ClientId     string `yaml:"clientId"`
	ClientSecret string `yaml:"clientSecret"`
}

// normalizeOrigins CORS 白名单归一化：去首尾空白并过滤空项
func normalizeOrigins(origins []string) []string {
	normalized := make([]string, 0, len(origins))
	for _, origin := range origins {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return normalized
}

func Parse(appConfig string) (*Config, error) {
	config := &Config{}

	file, err := os.Open(appConfig)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if err := yaml.NewDecoder(file).Decode(config); err != nil {
		return nil, err
	}
	// 认证域运行参数零值回落默认（失败锁定阈值等）
	config.Auth.normalize()
	// CORS 白名单归一化：去首尾空白、丢弃空串项（形如 ["", " "] 的配置等价
	// 于未配置，release 的空白名单 fail-fast 才不会被空项绕过）
	config.Server.AllowedOrigins = normalizeOrigins(config.Server.AllowedOrigins)
	if err := config.Storage.normalize(); err != nil {
		return nil, err
	}

	if config.PKI.PrivateKey != "" && config.PKI.PrivateKeyFile != "" {
		return nil, fmt.Errorf("pki.privateKey and pki.privateKeyFile cannot both be configured")
	}
	if config.PKI.PrivateKeyFile != "" {
		privateKeyPath := config.PKI.PrivateKeyFile
		if !filepath.IsAbs(privateKeyPath) {
			privateKeyPath = filepath.Join(filepath.Dir(appConfig), privateKeyPath)
		}
		privateKey, err := os.ReadFile(privateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read pki private key file: %w", err)
		}
		if strings.TrimSpace(string(privateKey)) == "" {
			return nil, fmt.Errorf("pki private key file is empty")
		}
		config.PKI.PrivateKey = string(privateKey)
	}

	return config, nil
}
