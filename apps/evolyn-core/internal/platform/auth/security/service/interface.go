package service

import (
	"context"

	"evolyn/internal/platform/auth/security/model"
)

// ctx 约定：数据方法统一以 ctx 为首参由 controller 透传；
// 本域为平台级表（无租户上下文）。

// SecurityService 账号安全服务（ADR-009 第 2 步：只读骨架 + 本人踢出会话）。
// 开关写入/TOTP/单会话挤出随第 3、4 步接入
type SecurityService interface {
	// Overview 安全概览：开关状态、TOTP 注册态、恢复码余量、活跃会话数
	Overview(ctx context.Context, accountID uint) (*SecurityOverview, error)
	// ListSessions 活跃会话列表（按最近活跃倒序）
	ListSessions(ctx context.Context, accountID uint) ([]model.AccountSession, error)
	// RevokeSession 本人踢出指定会话（校验归属），记安全流水（best-effort）
	RevokeSession(ctx context.Context, accountID uint, sid string) error
	// UpdateSingleSession 更新「禁止同时登录」开关；开启时保留当前会话并撤销其他设备。
	UpdateSingleSession(ctx context.Context, accountID uint, currentSID string, enabled bool) error
}

// SecurityOverview GET /accounts/me/security 响应体
type SecurityOverview struct {
	MFAEnabled             bool `json:"mfaEnabled"`
	SingleSessionEnabled   bool `json:"singleSessionEnabled"`
	TotpEnrolled           bool `json:"totpEnrolled"`           // 存在已验证的活跃 TOTP 因子
	RecoveryCodesRemaining int  `json:"recoveryCodesRemaining"` // 未使用恢复码数量
	ActiveSessions         int  `json:"activeSessions"`
}
