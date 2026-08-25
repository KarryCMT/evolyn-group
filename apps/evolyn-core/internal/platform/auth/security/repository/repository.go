// Package repository 账号安全与会话仓储（ADR-009 第 1 步）。
// 全平台级表无租户上下文，WithContext 仅为传播取消/超时
package repository

import (
	"context"
	"time"

	"evolyn/internal/platform/auth/security/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SettingsRepository 安全开关读写
type SettingsRepository interface {
	// Get 缺行返回全关闭零值（不报错），开关语义默认关闭
	Get(ctx context.Context, accountID uint) (*model.SecuritySettings, error)
	Upsert(ctx context.Context, settings *model.SecuritySettings) error
}

type settingsRepository struct{ db *gorm.DB }

func NewSettingsRepository(db *gorm.DB) SettingsRepository {
	return &settingsRepository{db: db}
}

func (r *settingsRepository) Get(ctx context.Context, accountID uint) (*model.SecuritySettings, error) {
	settings := &model.SecuritySettings{AccountID: accountID}
	if err := r.db.WithContext(ctx).First(settings, "account_id = ?", accountID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return settings, nil // 缺省即全关
		}
		return nil, err
	}
	return settings, nil
}

func (r *settingsRepository) Upsert(ctx context.Context, settings *model.SecuritySettings) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "account_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"mfa_enabled", "single_session_enabled", "updated_at"}),
	}).Create(settings).Error
}

// FactorRepository MFA 因子生命周期
type FactorRepository interface {
	Create(ctx context.Context, factor *model.MFAFactor) (*model.MFAFactor, error)
	// GetActive 同类型至多一个活跃因子（部分唯一索引保证）
	GetActive(ctx context.Context, accountID uint, factorType string) (*model.MFAFactor, error)
	UpdateCounter(ctx context.Context, id uint, counter int64) error
	Disable(ctx context.Context, id uint) error
}

type factorRepository struct{ db *gorm.DB }

func NewFactorRepository(db *gorm.DB) FactorRepository {
	return &factorRepository{db: db}
}

func (r *factorRepository) Create(ctx context.Context, factor *model.MFAFactor) (*model.MFAFactor, error) {
	if err := r.db.WithContext(ctx).Create(factor).Error; err != nil {
		return nil, err
	}
	return factor, nil
}

func (r *factorRepository) GetActive(ctx context.Context, accountID uint, factorType string) (*model.MFAFactor, error) {
	factor := new(model.MFAFactor)
	if err := r.db.WithContext(ctx).
		Where("account_id = ? AND type = ? AND disabled_at IS NULL", accountID, factorType).
		First(factor).Error; err != nil {
		return nil, err
	}
	return factor, nil
}

func (r *factorRepository) UpdateCounter(ctx context.Context, id uint, counter int64) error {
	return r.db.WithContext(ctx).Model(&model.MFAFactor{}).Where("id = ?", id).
		Update("last_used_counter", counter).Error
}

func (r *factorRepository) Disable(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&model.MFAFactor{}).Where("id = ?", id).
		Update("disabled_at", time.Now()).Error
}

// RecoveryRepository 恢复码：只存摘要，消费即置 used_at
type RecoveryRepository interface {
	// Replace 整批替换（重新生成恢复码时旧码全部作废删除）
	Replace(ctx context.Context, accountID uint, digests []string) error
	ListAvailable(ctx context.Context, accountID uint) ([]model.RecoveryCode, error)
	// Consume 原子消费：仅当未使用时置 used_at，返回是否成功
	Consume(ctx context.Context, id uint) (bool, error)
}

type recoveryRepository struct{ db *gorm.DB }

func NewRecoveryRepository(db *gorm.DB) RecoveryRepository {
	return &recoveryRepository{db: db}
}

func (r *recoveryRepository) Replace(ctx context.Context, accountID uint, digests []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("account_id = ?", accountID).Delete(&model.RecoveryCode{}).Error; err != nil {
			return err
		}
		if len(digests) == 0 {
			return nil
		}
		codes := make([]model.RecoveryCode, 0, len(digests))
		for _, d := range digests {
			codes = append(codes, model.RecoveryCode{AccountID: accountID, CodeDigest: d})
		}
		return tx.Create(&codes).Error
	})
}

func (r *recoveryRepository) ListAvailable(ctx context.Context, accountID uint) ([]model.RecoveryCode, error) {
	codes := make([]model.RecoveryCode, 0)
	if err := r.db.WithContext(ctx).
		Where("account_id = ? AND used_at IS NULL", accountID).
		Order("id").Find(&codes).Error; err != nil {
		return nil, err
	}
	return codes, nil
}

func (r *recoveryRepository) Consume(ctx context.Context, id uint) (bool, error) {
	res := r.db.WithContext(ctx).Model(&model.RecoveryCode{}).
		Where("id = ? AND used_at IS NULL", id).
		Update("used_at", time.Now())
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// SessionRepository 设备级会话：创建/查询/撤销/挤出/清理
type SessionRepository interface {
	Create(ctx context.Context, session *model.AccountSession) (*model.AccountSession, error)
	GetBySID(ctx context.Context, sid string) (*model.AccountSession, error)
	TouchLastSeen(ctx context.Context, sid string) error
	// BumpVersion 租户切换重签时递增（复用 sid，不算新设备）
	BumpVersion(ctx context.Context, sid string) (int64, error)
	Revoke(ctx context.Context, sid, reason string) error
	// RevokeOthers 禁止同时登录：撤销账号其余活跃会话（事务内配合账号行锁使用）
	RevokeOthers(ctx context.Context, accountID uint, exceptSID, reason string) (int64, error)
	ListActiveByAccount(ctx context.Context, accountID uint) ([]model.AccountSession, error)
	// DeleteExpiredBefore 清理 Worker：删除已过期且已撤销的历史会话
	DeleteExpiredBefore(ctx context.Context, before time.Time) (int64, error)
}

type sessionRepository struct{ db *gorm.DB }

func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, session *model.AccountSession) (*model.AccountSession, error) {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return nil, err
	}
	return session, nil
}

func (r *sessionRepository) GetBySID(ctx context.Context, sid string) (*model.AccountSession, error) {
	session := new(model.AccountSession)
	if err := r.db.WithContext(ctx).Where("sid = ?", sid).First(session).Error; err != nil {
		return nil, err
	}
	return session, nil
}

func (r *sessionRepository) TouchLastSeen(ctx context.Context, sid string) error {
	return r.db.WithContext(ctx).Model(&model.AccountSession{}).Where("sid = ?", sid).
		Update("last_seen_at", time.Now()).Error
}

func (r *sessionRepository) BumpVersion(ctx context.Context, sid string) (int64, error) {
	session, err := r.GetBySID(ctx, sid)
	if err != nil {
		return 0, err
	}
	if err := r.db.WithContext(ctx).Model(&model.AccountSession{}).Where("id = ?", session.ID).
		Update("token_version", session.TokenVersion+1).Error; err != nil {
		return 0, err
	}
	return session.TokenVersion + 1, nil
}

func (r *sessionRepository) Revoke(ctx context.Context, sid, reason string) error {
	return r.db.WithContext(ctx).Model(&model.AccountSession{}).
		Where("sid = ? AND revoked_at IS NULL", sid).
		Updates(map[string]interface{}{"revoked_at": time.Now(), "revoke_reason": reason}).Error
}

func (r *sessionRepository) RevokeOthers(ctx context.Context, accountID uint, exceptSID, reason string) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.AccountSession{}).
		Where("account_id = ? AND sid <> ? AND revoked_at IS NULL", accountID, exceptSID).
		Updates(map[string]interface{}{"revoked_at": time.Now(), "revoke_reason": reason})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

func (r *sessionRepository) ListActiveByAccount(ctx context.Context, accountID uint) ([]model.AccountSession, error) {
	sessions := make([]model.AccountSession, 0)
	if err := r.db.WithContext(ctx).
		Where("account_id = ? AND revoked_at IS NULL", accountID).
		Order("last_seen_at DESC").Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *sessionRepository) DeleteExpiredBefore(ctx context.Context, before time.Time) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("expires_at < ? OR (revoked_at IS NOT NULL AND revoked_at < ?)", before, before).
		Delete(&model.AccountSession{})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

// EventRepository 安全流水追加写（best-effort，读侧运营查询）
type EventRepository interface {
	Append(ctx context.Context, event *model.SecurityEvent) error
}

type eventRepository struct{ db *gorm.DB }

func NewEventRepository(db *gorm.DB) EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) Append(ctx context.Context, event *model.SecurityEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}
