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
	PKI         PKIConfig              `yaml:"pki"`
}

// SMSConfig 短信验证码运行参数：一期开发通道（provider=dev 仅打日志），
// 生产接真实服务商时扩展 provider 与密钥字段
type SMSConfig struct {
	Provider        string `yaml:"provider"`        // 通道：dev（默认，日志不外发）
	CodeTTLSeconds  int    `yaml:"codeTtlSeconds"`  // 验证码有效期秒（默认 300）
	CooldownSeconds int    `yaml:"cooldownSeconds"` // 重发冷却秒（默认 60）
	MaxTries        int    `yaml:"maxTries"`        // 单码最大试错次数（默认 5）
	DailyLimit      int    `yaml:"dailyLimit"`      // 单手机号自然日发送上限（默认 10，跨场景合计）
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
