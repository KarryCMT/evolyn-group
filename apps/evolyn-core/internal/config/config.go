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
	Server      ServerConfig           `yaml:"server"`
	DB          DBConfig               `yaml:"db"`
	Redis       RedisConfig            `yaml:"redis"`
	Tenant      TenantRuntimeConfig    `yaml:"tenant"`
	OAuthConfig map[string]OAuthConfig `yaml:"oauth"`
	SMS         SMSConfig              `yaml:"sms"`
	Auth        AuthConfig             `yaml:"auth"`
	PKI         PKIConfig              `yaml:"pki"`
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

// PKIConfig 登录口令加密传输密钥：公钥经 GET /app/conf 下发前端，
// 私钥仅服务端解密用。PrivateKey 为直接配置的 PEM，PrivateKeyFile 为 PEM 文件路径，
// 两者只能配置其一；均留空时启动随机生成一把（仅开发/测试，重启即轮换）。
// 生产与多实例部署必须显式配置同一密钥对。
type PKIConfig struct {
	Algorithm      string `yaml:"algorithm"`      // 一期固定 rsa
	PrivateKey     string `yaml:"privateKey"`     // PEM 私钥（真实私钥不入库，经环境配置注入）
	PrivateKeyFile string `yaml:"privateKeyFile"` // PEM 私钥文件路径（相对配置文件所在目录）
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
