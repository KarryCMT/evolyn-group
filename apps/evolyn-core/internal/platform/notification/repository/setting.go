package repository

import (
	"context"
	"time"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/notification/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type settingRepository struct {
	db *gorm.DB
}

// NewSettingRepository 通知设置聚合仓储工厂（ADR-007 域模块化）
func NewSettingRepository(db *gorm.DB) SettingRepository {
	return &settingRepository{db: db}
}

func (r *settingRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *settingRepository) Migrate() error {
	return r.db.AutoMigrate(
		&model.Setting{}, &model.Preference{}, &model.PreferenceRecipient{}, &model.CustomRecipient{},
	)
}

// EnsureSetting 加载聚合根；不存在时幂等创建（ON CONFLICT DO NOTHING 兼容
// 部分唯一索引，并发补齐只有一行胜出），随后重读保证返回带 ID 的行
func (r *settingRepository) EnsureSetting(ctx context.Context, tenantID uint) (*model.Setting, error) {
	setting, err := r.getSetting(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if setting != nil {
		return setting, nil
	}
	// 嵌入字段不能以字面量初始化，显式赋值（显式指定租户归属，bctx 无租户
	// 上下文时避免落到列默认值）
	created := &model.Setting{}
	created.TenantID = tenantID
	if err := r.withContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(created).Error; err != nil {
		return nil, err
	}
	setting, err = r.getSetting(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return setting, nil
}

func (r *settingRepository) getSetting(ctx context.Context, tenantID uint) (*model.Setting, error) {
	setting := new(model.Setting)
	err := r.withContext(ctx).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Take(setting).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return setting, nil
}

// BumpRevision 以旧 revision 为条件原子递增（乐观锁收口，零行影响=口令过期）
func (r *settingRepository) BumpRevision(ctx context.Context, settingID uint, fromRevision int64) (bool, error) {
	result := r.withContext(ctx).
		Model(&model.Setting{}).
		Where("id = ? AND revision = ? AND deleted_at IS NULL", settingID, fromRevision).
		Updates(map[string]interface{}{
			"revision":   gorm.Expr("revision + 1"),
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *settingRepository) ListPreferences(ctx context.Context, tenantID uint) ([]model.Preference, error) {
	preferences := make([]model.Preference, 0)
	err := r.withContext(ctx).
		Where("tenant_id = ?", tenantID).
		Find(&preferences).Error
	return preferences, err
}

// UpsertPreference 按 (tenant_id, event_code) 唯一索引 upsert 覆盖行；
// 冲突更新路径不回填主键，按唯一键回读保证调用方拿到带 ID 的行
func (r *settingRepository) UpsertPreference(ctx context.Context, pref *model.Preference) error {
	err := r.withContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "tenant_id"}, {Name: "event_code"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"system_enabled", "email_enabled", "sms_enabled", "recipients_overridden", "updated_at",
			}),
		}).Create(pref).Error
	if err != nil {
		return err
	}
	if pref.ID == 0 {
		return r.withContext(ctx).
			Where("tenant_id = ? AND event_code = ?", pref.TenantID, pref.EventCode).
			Take(pref).Error
	}
	return nil
}

func (r *settingRepository) ListPreferenceRecipients(
	ctx context.Context, tenantID uint, preferenceIDs []uint,
) (map[uint][]model.PreferenceRecipient, error) {
	result := make(map[uint][]model.PreferenceRecipient, len(preferenceIDs))
	if len(preferenceIDs) == 0 {
		return result, nil
	}
	rows := make([]model.PreferenceRecipient, 0)
	err := r.withContext(ctx).
		Where("tenant_id = ? AND preference_id IN ?", tenantID, preferenceIDs).
		Order("id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.PreferenceID] = append(result[row.PreferenceID], row)
	}
	return result, nil
}

// ReplaceRecipients 全量替换某偏好的接收规则（先删后插，同事务原子）
func (r *settingRepository) ReplaceRecipients(
	ctx context.Context, tenantID, preferenceID uint, items []model.PreferenceRecipient,
) error {
	if err := r.withContext(ctx).
		Where("preference_id = ? AND tenant_id = ?", preferenceID, tenantID).
		Delete(&model.PreferenceRecipient{}).Error; err != nil {
		return err
	}
	for i := range items {
		items[i].TenantID = tenantID
		items[i].PreferenceID = preferenceID
	}
	if len(items) == 0 {
		return nil
	}
	return r.withContext(ctx).Create(&items).Error
}

func (r *settingRepository) ListCustomRecipients(ctx context.Context, tenantID uint) ([]model.CustomRecipient, error) {
	recipients := make([]model.CustomRecipient, 0)
	err := r.withContext(ctx).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Order("id ASC").
		Find(&recipients).Error
	return recipients, err
}

func (r *settingRepository) GetCustomRecipient(ctx context.Context, tenantID, id uint) (*model.CustomRecipient, error) {
	recipient := new(model.CustomRecipient)
	err := r.withContext(ctx).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		First(recipient, id).Error
	if err != nil {
		return nil, err
	}
	return recipient, nil
}

func (r *settingRepository) CountCustomRecipients(ctx context.Context, tenantID uint) (int64, error) {
	var count int64
	err := r.withContext(ctx).
		Model(&model.CustomRecipient{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Count(&count).Error
	return count, err
}

func (r *settingRepository) InsertCustomRecipient(
	ctx context.Context, recipient *model.CustomRecipient,
) (*model.CustomRecipient, error) {
	if err := r.withContext(ctx).Create(recipient).Error; err != nil {
		return nil, err
	}
	return recipient, nil
}

func (r *settingRepository) SoftDeleteCustomRecipient(ctx context.Context, tenantID, id uint) error {
	return r.withContext(ctx).
		Model(&model.CustomRecipient{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Update("deleted_at", time.Now()).Error
}

// FindRecipientUsage 引用该联系人的偏好事件码（删除前的在用校验）
func (r *settingRepository) FindRecipientUsage(ctx context.Context, tenantID, recipientID uint) ([]string, error) {
	codes := make([]string, 0)
	err := r.withContext(ctx).Raw(`
		SELECT DISTINCT p.event_code
		FROM tenant_notification_preference_recipients pr
		JOIN tenant_notification_preferences p ON p.id = pr.preference_id
		WHERE pr.tenant_id = ? AND pr.custom_recipient_id = ?
		ORDER BY p.event_code`, tenantID, recipientID).
		Scan(&codes).Error
	if err != nil {
		return nil, err
	}
	return codes, nil
}
