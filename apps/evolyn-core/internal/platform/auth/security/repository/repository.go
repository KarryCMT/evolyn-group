// Package repository 账号安全与会话仓储（ADR-009 第 1 步）。
// 全平台级表无租户上下文，WithContext 仅为传播取消/超时
package repository

import (
	"context"
	"time"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/auth/security/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// resolve 经 ResolveDB 取连接：外层 TxManager 事务自动传播（FIX-020/021），
// 单会话登录「锁账号行 → 撤销他人 → 建新会话」依赖此口径保证原子性
func resolve(ctx context.Context, db *gorm.DB) *gorm.DB {
	return infrastructure.ResolveDB(ctx, db)
}

// SettingsRepository 安全开关读写
type SettingsRepository interface {
	// Get 缺行返回全关闭零值（不报错），开关语义默认关闭
	Get(ctx context.Context, accountID uint) (*model.SecuritySettings, error)
	Upsert(ctx context.Context, settings *model.SecuritySettings) error
	// LockAccountRow 事务内锁账号行：单会话登录的并发控制锚点——
	// 两个并发登录串行化，最终只留下后提交者的会话
	LockAccountRow(ctx context.Context, accountID uint) error
}

type settingsRepository struct{ db *gorm.DB }

func NewSettingsRepository(db *gorm.DB) SettingsRepository {
	return &settingsRepository{db: db}
}

func (r *settingsRepository) Get(ctx context.Context, accountID uint) (*model.SecuritySettings, error) {
	settings := &model.SecuritySettings{AccountID: accountID}
	if err := resolve(ctx, r.db).First(settings, "account_id = ?", accountID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return settings, nil // 缺省即全关
		}
		return nil, err
	}
	return settings, nil
}

func (r *settingsRepository) LockAccountRow(ctx context.Context, accountID uint) error {
	var id uint
	return resolve(ctx, r.db).Raw("SELECT id FROM accounts WHERE id = ? FOR UPDATE", accountID).Scan(&id).Error
}

func (r *settingsRepository) Upsert(ctx context.Context, settings *model.SecuritySettings) error {
	return resolve(ctx, r.db).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "account_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"mfa_enabled", "single_session_enabled", "updated_at"}),
	}).Create(settings).Error
}

// FactorRepository MFA 因子生命周期
type FactorRepository interface {
	Create(ctx context.Context, factor *model.MFAFactor) (*model.MFAFactor, error)
	// GetActive 同类型至多一个活跃因子（部分唯一索引保证）
	GetActive(ctx context.Context, accountID uint, factorType string) (*model.MFAFactor, error)
	// ConsumeCounter 以条件更新原子提交验证码计数器；若已有并发请求消费了
	// 相同或更晚的时间步，返回 false，从存储层杜绝 TOTP 重放。
	ConsumeCounter(ctx context.Context, id uint, counter int64) (bool, error)
	Disable(ctx context.Context, id uint) error
}

type factorRepository struct{ db *gorm.DB }

func NewFactorRepository(db *gorm.DB) FactorRepository {
	return &factorRepository{db: db}
}

func (r *factorRepository) Create(ctx context.Context, factor *model.MFAFactor) (*model.MFAFactor, error) {
	if err := resolve(ctx, r.db).Create(factor).Error; err != nil {
		return nil, err
	}
	return factor, nil
}

func (r *factorRepository) GetActive(ctx context.Context, accountID uint, factorType string) (*model.MFAFactor, error) {
	factor := new(model.MFAFactor)
	if err := resolve(ctx, r.db).
		Where("account_id = ? AND type = ? AND disabled_at IS NULL", accountID, factorType).
		First(factor).Error; err != nil {
		return nil, err
	}
	return factor, nil
}

func (r *factorRepository) ConsumeCounter(ctx context.Context, id uint, counter int64) (bool, error) {
	res := resolve(ctx, r.db).Model(&model.MFAFactor{}).
		Where("id = ? AND last_used_counter < ? AND disabled_at IS NULL", id, counter).
		Update("last_used_counter", counter)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

func (r *factorRepository) Disable(ctx context.Context, id uint) error {
	return resolve(ctx, r.db).Model(&model.MFAFactor{}).Where("id = ?", id).
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
	return resolve(ctx, r.db).Transaction(func(tx *gorm.DB) error {
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
	if err := resolve(ctx, r.db).
		Where("account_id = ? AND used_at IS NULL", accountID).
		Order("id").Find(&codes).Error; err != nil {
		return nil, err
	}
	return codes, nil
}

func (r *recoveryRepository) Consume(ctx context.Context, id uint) (bool, error) {
	res := resolve(ctx, r.db).Model(&model.RecoveryCode{}).
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
	if err := resolve(ctx, r.db).Create(session).Error; err != nil {
		return nil, err
	}
	return session, nil
}

func (r *sessionRepository) GetBySID(ctx context.Context, sid string) (*model.AccountSession, error) {
	session := new(model.AccountSession)
	if err := resolve(ctx, r.db).Where("sid = ?", sid).First(session).Error; err != nil {
		return nil, err
	}
	return session, nil
}

func (r *sessionRepository) TouchLastSeen(ctx context.Context, sid string) error {
	return resolve(ctx, r.db).Model(&model.AccountSession{}).Where("sid = ?", sid).
		Update("last_seen_at", time.Now()).Error
}

func (r *sessionRepository) BumpVersion(ctx context.Context, sid string) (int64, error) {
	session, err := r.GetBySID(ctx, sid)
	if err != nil {
		return 0, err
	}
	if err := resolve(ctx, r.db).Model(&model.AccountSession{}).Where("id = ?", session.ID).
		Update("token_version", session.TokenVersion+1).Error; err != nil {
		return 0, err
	}
	return session.TokenVersion + 1, nil
}

func (r *sessionRepository) Revoke(ctx context.Context, sid, reason string) error {
	return resolve(ctx, r.db).Model(&model.AccountSession{}).
		Where("sid = ? AND revoked_at IS NULL", sid).
		Updates(map[string]interface{}{"revoked_at": time.Now(), "revoke_reason": reason}).Error
}

func (r *sessionRepository) RevokeOthers(ctx context.Context, accountID uint, exceptSID, reason string) (int64, error) {
	res := resolve(ctx, r.db).Model(&model.AccountSession{}).
		Where("account_id = ? AND sid <> ? AND revoked_at IS NULL", accountID, exceptSID).
		Updates(map[string]interface{}{"revoked_at": time.Now(), "revoke_reason": reason})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

func (r *sessionRepository) ListActiveByAccount(ctx context.Context, accountID uint) ([]model.AccountSession, error) {
	sessions := make([]model.AccountSession, 0)
	if err := resolve(ctx, r.db).
		Where("account_id = ? AND revoked_at IS NULL", accountID).
		Order("last_seen_at DESC").Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *sessionRepository) DeleteExpiredBefore(ctx context.Context, before time.Time) (int64, error) {
	res := resolve(ctx, r.db).
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
	return resolve(ctx, r.db).Create(event).Error
}
