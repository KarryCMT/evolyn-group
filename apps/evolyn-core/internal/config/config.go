package config

import (
	"os"
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
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Migrate  bool   `yaml:"migrate"`    // AutoMigrate（开发/测试）
	Migrations bool  `yaml:"migrations"` // 版本化 SQL Migration（生产）
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

	if err := yaml.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}
