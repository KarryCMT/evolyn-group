// Package model 账号安全与会话体系模型（ADR-009）。全部平台级表
// （无 tenant_id），状态语义用显式时间列（revoked_at/used_at/
// disabled_at）而非 GORM 软删——「活跃集合」过滤是业务语义的一部分
package model

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"evolyn/internal/model"
)

// MFA 因子类型
const (
	FactorTypeTotp = "totp"
)

// 会话认证方式（第一步通过的证据类别）
const (
	AuthMethodPassword = "password"
	AuthMethodSMS      = "sms"
	AuthMethodOAuth    = "oauth"
	AuthMethodRegister = "register"
)

// 会话第二步验证方式（未启用 MFA 为空）
const (
	MFAMethodTotp     = "totp"
	MFAMethodRecovery = "recovery"
)

// 会话撤销原因（稳定码出网，前端可分支提示）
const (
	RevokeLogout          = "logout"           // 本人登出
	RevokeReplaced        = "replaced"         // 禁止同时登录被新会话挤出
	RevokePasswordChanged = "password_changed" // 密码变更撤销其他会话
	RevokePhoneChanged    = "phone_changed"    // 换绑手机
	RevokeMFAChanged      = "mfa_changed"      // MFA 启用/停用
	RevokeAdminRevoked    = "admin_revoked"    // 管理端强制下线
)

// SecuritySettings 账号安全开关（一行一账号，缺省即全关）
type SecuritySettings struct {
	AccountID            uint `json:"accountId" gorm:"primaryKey;autoIncrement:false"`
	MFAEnabled           bool `json:"mfaEnabled" gorm:"not null;default:false"`
	SingleSessionEnabled bool `json:"singleSessionEnabled" gorm:"not null;default:false"`

	UpdatedAt model.JSONTime `json:"updatedAt"`
}

func (*SecuritySettings) TableName() string {
	return "account_security_settings"
}

// MFAFactor 已验证的 MFA 因子；secret 仅密文（见 security service 加密），
// key_version 对应加密主密钥代次支持轮换
type MFAFactor struct {
	ID               uint           `json:"id" gorm:"autoIncrement;primaryKey"`
	AccountID        uint           `json:"accountId" gorm:"index;not null"`
	Type             string         `json:"type" gorm:"size:16;not null"`
	SecretCiphertext string         `json:"-" gorm:"size:1024;not null"` // 绝不出网
	KeyVersion       int            `json:"-" gorm:"not null;default:1"`
	VerifiedAt       *time.Time     `json:"verifiedAt"`
	LastUsedCounter  int64          `json:"-" gorm:"not null;default:0"` // TOTP 防重放窗口
	DisabledAt       *time.Time     `json:"disabledAt"`
	CreatedAt        model.JSONTime `json:"createdAt"`
	UpdatedAt        model.JSONTime `json:"updatedAt"`
}

func (*MFAFactor) TableName() string {
	return "account_mfa_factors"
}

// RecoveryCode 一次性恢复码：只存摘要（sha256），明文仅创建时展示一次
type RecoveryCode struct {
	ID         uint           `json:"id" gorm:"autoIncrement;primaryKey"`
	AccountID  uint           `json:"accountId" gorm:"index;not null"`
	CodeDigest string         `json:"-" gorm:"size:128;not null"`
	UsedAt     *time.Time     `json:"usedAt"`
	CreatedAt  model.JSONTime `json:"createdAt"`
}

func (*RecoveryCode) TableName() string {
	return "account_mfa_recovery_codes"
}

// AccountSession 设备级逻辑会话：sid 进 JWT，token_version 随租户切换重签递增
type AccountSession struct {
	ID uint `json:"id" gorm:"autoIncrement;primaryKey"`
	// SID 是既有数据库列 sid；显式声明以避免 GORM 将全大写缩写按 s_id 命名。
	SID          string         `json:"sid" gorm:"column:sid;size:64;not null;uniqueIndex"`
	AccountID    uint           `json:"accountId" gorm:"index;not null"`
	TokenVersion int64          `json:"tokenVersion" gorm:"not null;default:1"`
	AuthMethod   string         `json:"authMethod" gorm:"size:16;not null"`
	MFAMethod    *string        `json:"mfaMethod" gorm:"size:16"`
	CreatedAt    model.JSONTime `json:"createdAt"`
	LastSeenAt   model.JSONTime `json:"lastSeenAt"`
	// ExpiresAt 固定有效期（与 JWT 对齐）；RevokedAt 非空即失效
	ExpiresAt    model.JSONTime `json:"expiresAt" gorm:"not null"`
	RevokedAt    *time.Time     `json:"revokedAt"`
	RevokeReason *string        `json:"revokeReason" gorm:"size:32"`
	IP           string         `json:"ip" gorm:"size:45"`
	Location     string         `json:"location" gorm:"size:128"`
	UserAgent    string         `json:"userAgent" gorm:"size:512"`
}

func (*AccountSession) TableName() string {
	return "account_sessions"
}

// NewSID 会话公开标识：随机 16 字节 hex（32 字符），与 jti 同源口径
func NewSID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate sid: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// SecurityEvent 账号安全流水（追加写）：MFA 开关、恢复码、会话挤出等
type SecurityEvent struct {
	ID        uint           `json:"id" gorm:"autoIncrement;primaryKey"`
	AccountID uint           `json:"accountId" gorm:"index;not null"`
	EventType string         `json:"eventType" gorm:"size:32;not null"`
	SessionID string         `json:"sessionId" gorm:"size:64"`
	RequestID string         `json:"requestId" gorm:"size:64"`
	IP        string         `json:"ip" gorm:"size:45"`
	Metadata  EventMetadata  `json:"metadata" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt model.JSONTime `json:"createdAt"`
}

func (*SecurityEvent) TableName() string {
	return "account_security_events"
}

// EventMetadata 安全事件元数据（JSONB）
type EventMetadata map[string]string
