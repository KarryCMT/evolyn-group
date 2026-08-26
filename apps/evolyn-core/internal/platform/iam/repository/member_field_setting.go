package repository

import (
	"context"
	"errors"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type memberFieldSettingRepository struct{ db *gorm.DB }

func newMemberFieldSettingRepository(db *gorm.DB) MemberFieldSettingRepository {
	return &memberFieldSettingRepository{db: db}
}

func (r *memberFieldSettingRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *memberFieldSettingRepository) ListByTenant(ctx context.Context) ([]model.MemberFieldSetting, error) {
	settings := make([]model.MemberFieldSetting, 0)
	// 按注册表顺序无法在 SQL 侧表达，读取后由 Service 按注册表重排
	if err := r.withContext(ctx).Order("field_key").Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *memberFieldSettingRepository) GetByFieldKey(ctx context.Context, fieldKey string) (*model.MemberFieldSetting, error) {
	setting := new(model.MemberFieldSetting)
	if err := r.withContext(ctx).Where("field_key = ?", fieldKey).First(setting).Error; err != nil {
		return nil, err
	}
	return setting, nil
}

// CreateBatch seed 路径写入：ON CONFLICT DO NOTHING 保证幂等（租户开通
// 事务与读取侧兜底重入安全）。Select 显式列出全部列——bool 列带 default
// tag 时 GORM 会跳过零值 false 让数据库默认 true 生效，必须强制包含
func (r *memberFieldSettingRepository) CreateBatch(ctx context.Context, settings []model.MemberFieldSetting) error {
	if len(settings) == 0 {
		return nil
	}
	return r.withContext(ctx).
		Select("TenantID", "FieldKey", "PersonalVisible", "PersonalEditable", "CardVisible", "Revision").
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&settings).Error
}

// UpdateWithRevision 乐观锁更新：仅当行存在且 revision 与请求一致时写入，
// 写入同时 revision = revision + 1。返回 false 表示版本冲突或行不存在，
// 由 Service 区分业务错误（文档 5.1：revision 过期返回配置冲突）
func (r *memberFieldSettingRepository) UpdateWithRevision(ctx context.Context, id uint, revision int64, updates map[string]interface{}) (bool, error) {
	result := r.withContext(ctx).Model(&model.MemberFieldSetting{}).
		Where("id = ? AND revision = ?", id, revision).
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *memberFieldSettingRepository) BumpRevision(ctx context.Context) error {
	return r.withContext(ctx).Model(&model.MemberFieldSetting{}).
		Where("1 = 1").
		Update("revision", gorm.Expr("revision + 1")).Error
}

func (r *memberFieldSettingRepository) Migrate() error {
	if err := r.db.AutoMigrate(&model.MemberFieldSetting{}); err != nil {
		return err
	}
	// AutoMigrate 表达不了部分唯一索引（FIX-009 口径），幂等 SQL 补齐与
	// migrations 终态一致
	for _, stmt := range []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_member_field_settings_tenant_field ON tenant_member_field_settings (tenant_id, field_key) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_member_field_settings_tenant_updated ON tenant_member_field_settings (tenant_id, updated_at) WHERE deleted_at IS NULL`,
	} {
		if err := r.db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

type memberProfileRepository struct{ db *gorm.DB }

func newMemberProfileRepository(db *gorm.DB) MemberProfileRepository {
	return &memberProfileRepository{db: db}
}

func (r *memberProfileRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *memberProfileRepository) GetByMember(ctx context.Context, memberID uint) (*model.MemberProfile, error) {
	profile := new(model.MemberProfile)
	if err := r.withContext(ctx).Where("member_id = ?", memberID).First(profile).Error; err != nil {
		return nil, err
	}
	return profile, nil
}

// Upsert 档案写入：存在则按白名单整体替换 identifier/attributes（attributes
// 由 Service 层合并后整体下传，此处不做增量合并，保证与请求语义一致）
func (r *memberProfileRepository) Upsert(ctx context.Context, profile *model.MemberProfile) (*model.MemberProfile, error) {
	existing, err := r.GetByMember(ctx, profile.MemberID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil && err == nil {
		// 先加载再写：跨租户 member_id 由 Callback 过滤为 NotFound，
		// 不存在「租户过滤 0 行影响却返回成功」的盲写路径（整改口径）
		if err := r.withContext(ctx).Model(&model.MemberProfile{}).
			Where("id = ?", existing.ID).
			Select("identifier", "attributes").
			Updates(map[string]interface{}{
				"identifier": profile.Identifier,
				"attributes": profile.Attributes,
			}).Error; err != nil {
			return nil, err
		}
		profile.ID = existing.ID
		return profile, nil
	}
	if err := r.withContext(ctx).Create(profile).Error; err != nil {
		return nil, err
	}
	return profile, nil
}

func (r *memberProfileRepository) IdentifierExists(ctx context.Context, identifier string, excludeMemberID uint) (bool, error) {
	var count int64
	if err := r.withContext(ctx).Model(&model.MemberProfile{}).
		Where("identifier = ? AND member_id <> ? AND identifier <> ''", identifier, excludeMemberID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *memberProfileRepository) Migrate() error {
	if err := r.db.AutoMigrate(&model.MemberProfile{}); err != nil {
		return err
	}
	for _, stmt := range []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_member_profiles_tenant_member ON member_profiles (tenant_id, member_id) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_member_profiles_tenant_identifier ON member_profiles (tenant_id, identifier) WHERE identifier <> '' AND deleted_at IS NULL`,
	} {
		if err := r.db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
