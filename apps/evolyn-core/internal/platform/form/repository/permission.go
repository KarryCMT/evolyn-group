// 权限组仓储（表单权限 P1，ADR-007 域内小三层）：仅做持久化，一律经
// infrastructure.ResolveDB 取连接加入 ctx 传播事务。组行随 TenantBaseModel
// 软删；subjects 行无软删语义（授权关系无保留价值），组删除由 Service 同
// 事务显式硬删，外键 CASCADE 仅兜底物理清理路径。
package repository

import (
	"context"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/form/model"

	"gorm.io/gorm"
)

type permissionGroupRepository struct {
	db *gorm.DB
}

// NewPermissionGroupRepository 权限组仓储工厂
func NewPermissionGroupRepository(db *gorm.DB) PermissionGroupRepository {
	return &permissionGroupRepository{db: db}
}

// withContext 以请求 ctx 打开新会话：GORM 租户 Callback 自动注入过滤/回填；
// ctx 携带事务 session 时加入外层事务（FIX-020/021 统一事务边界）
func (r *permissionGroupRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

// PermissionGroupRepository 资产权限组仓储。
type PermissionGroupRepository interface {
	// Create 写入组行（TenantID 由调用方显式赋值，租户 Callback 兜底）
	Create(ctx context.Context, group *model.AssetPermissionGroup) (*model.AssetPermissionGroup, error)
	// GetByCode 按公开编码加载（ctx 租户过滤，软删行天然排除）
	GetByCode(ctx context.Context, code string) (*model.AssetPermissionGroup, error)
	// ListByAsset 资产下全部权限组（含禁用，不含软删）：配置列表与
	// switch-type 阻塞判定（§3.3 含禁用组）使用
	ListByAsset(ctx context.Context, assetType string, assetID uint) ([]model.AssetPermissionGroup, error)
	// CountByAsset 资产下权限组数量（含禁用，不含软删）：数量上限校验
	CountByAsset(ctx context.Context, assetType string, assetID uint) (int64, error)
	// ExistsByAssetIDs 批量判定资产是否存在任一权限组行（含禁用组；S5 收口
	// 事实源：存在任一行即进入授权模型）
	ExistsByAssetIDs(ctx context.Context, assetType string, assetIDs []uint) (map[uint]bool, error)
	// ListEnabledByAssetIDs 批量加载资产的启用组（判定器数据面）
	ListEnabledByAssetIDs(ctx context.Context, assetType string, assetIDs []uint) ([]model.AssetPermissionGroup, error)
	// UpdateWithRevision 整组乐观锁更新：revision 匹配才写入白名单字段并
	// 条件递增；0 行影响即口令过期（Service 转 FORM_PERMISSION_REVISION_CONFLICT）
	UpdateWithRevision(ctx context.Context, id uint, fromRevision int64, fields map[string]interface{}) (bool, error)
	// SoftDelete 软删组行（subjects 由调用方同事务硬删）
	SoftDelete(ctx context.Context, group *model.AssetPermissionGroup) error
	// ReplaceSubjects 整体替换组主体：同事务先硬删后批量写入（唯一约束
	// (group_id, subject_type, subject_id) 兜底重复项）
	ReplaceSubjects(ctx context.Context, groupID uint, subjects []model.AssetPermissionGroupSubject) error
	// DeleteSubjectsByGroupIDs 硬删组主体（组删除事务内调用）
	DeleteSubjectsByGroupIDs(ctx context.Context, groupIDs []uint) error
	// ListSubjectsByGroupIDs 批量加载组主体（配置读取与判定器主体反查）
	ListSubjectsByGroupIDs(ctx context.Context, groupIDs []uint) ([]model.AssetPermissionGroupSubject, error)
	// Migrate 开发/测试 AutoMigrate 路径（FIX-009：生产只走 SQL 迁移）
	Migrate() error
}

func (r *permissionGroupRepository) Create(ctx context.Context, group *model.AssetPermissionGroup) (*model.AssetPermissionGroup, error) {
	if err := r.withContext(ctx).Create(group).Error; err != nil {
		return nil, err
	}
	return group, nil
}

func (r *permissionGroupRepository) GetByCode(ctx context.Context, code string) (*model.AssetPermissionGroup, error) {
	group := new(model.AssetPermissionGroup)
	if err := r.withContext(ctx).Where("code = ?", code).First(group).Error; err != nil {
		return nil, err
	}
	return group, nil
}

func (r *permissionGroupRepository) ListByAsset(ctx context.Context, assetType string, assetID uint) ([]model.AssetPermissionGroup, error) {
	groups := make([]model.AssetPermissionGroup, 0)
	err := r.withContext(ctx).Model(&model.AssetPermissionGroup{}).
		Where("asset_type = ? AND asset_id = ?", assetType, assetID).
		Order("id ASC").
		Find(&groups).Error
	return groups, err
}

func (r *permissionGroupRepository) CountByAsset(ctx context.Context, assetType string, assetID uint) (int64, error) {
	var count int64
	err := r.withContext(ctx).Model(&model.AssetPermissionGroup{}).
		Where("asset_type = ? AND asset_id = ?", assetType, assetID).
		Count(&count).Error
	return count, err
}

func (r *permissionGroupRepository) ExistsByAssetIDs(ctx context.Context, assetType string, assetIDs []uint) (map[uint]bool, error) {
	existing := make(map[uint]bool, len(assetIDs))
	if len(assetIDs) == 0 {
		return existing, nil
	}
	rows := make([]struct {
		AssetID uint
	}, 0, len(assetIDs))
	if err := r.withContext(ctx).Model(&model.AssetPermissionGroup{}).
		Where("asset_type = ? AND asset_id IN ?", assetType, assetIDs).
		Select("DISTINCT asset_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		existing[row.AssetID] = true
	}
	return existing, nil
}

func (r *permissionGroupRepository) ListEnabledByAssetIDs(ctx context.Context, assetType string, assetIDs []uint) ([]model.AssetPermissionGroup, error) {
	if len(assetIDs) == 0 {
		return []model.AssetPermissionGroup{}, nil
	}
	groups := make([]model.AssetPermissionGroup, 0)
	err := r.withContext(ctx).Model(&model.AssetPermissionGroup{}).
		Where("asset_type = ? AND asset_id IN ? AND enabled = ?", assetType, assetIDs, true).
		Order("id ASC").
		Find(&groups).Error
	return groups, err
}

// UpdateWithRevision 乐观锁整组更新：fields 为白名单字段（name/description/
// enabled/operations/field_permissions/data_scope），revision 条件递增避免读改写竞态
func (r *permissionGroupRepository) UpdateWithRevision(
	ctx context.Context, id uint, fromRevision int64, fields map[string]interface{},
) (bool, error) {
	updates := make(map[string]interface{}, len(fields)+1)
	for key, value := range fields {
		updates[key] = value
	}
	updates["revision"] = fromRevision + 1
	result := r.withContext(ctx).Model(&model.AssetPermissionGroup{}).
		Where("id = ? AND revision = ?", id, fromRevision).
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *permissionGroupRepository) SoftDelete(ctx context.Context, group *model.AssetPermissionGroup) error {
	return r.withContext(ctx).Delete(group).Error
}

func (r *permissionGroupRepository) ReplaceSubjects(ctx context.Context, groupID uint, subjects []model.AssetPermissionGroupSubject) error {
	if err := r.withContext(ctx).Where("group_id = ?", groupID).
		Delete(&model.AssetPermissionGroupSubject{}).Error; err != nil {
		return err
	}
	if len(subjects) == 0 {
		return nil
	}
	// 主体行归属以入参组 ID 为准（请求 DTO 不携带内部组 ID，防伪造）
	stamped := make([]model.AssetPermissionGroupSubject, len(subjects))
	for i, subject := range subjects {
		subject.GroupID = groupID
		stamped[i] = subject
	}
	return r.withContext(ctx).Create(&stamped).Error
}

func (r *permissionGroupRepository) DeleteSubjectsByGroupIDs(ctx context.Context, groupIDs []uint) error {
	if len(groupIDs) == 0 {
		return nil
	}
	return r.withContext(ctx).Where("group_id IN ?", groupIDs).
		Delete(&model.AssetPermissionGroupSubject{}).Error
}

func (r *permissionGroupRepository) ListSubjectsByGroupIDs(ctx context.Context, groupIDs []uint) ([]model.AssetPermissionGroupSubject, error) {
	if len(groupIDs) == 0 {
		return []model.AssetPermissionGroupSubject{}, nil
	}
	subjects := make([]model.AssetPermissionGroupSubject, 0)
	err := r.withContext(ctx).Model(&model.AssetPermissionGroupSubject{}).
		Where("group_id IN ?", groupIDs).
		Order("id ASC").
		Find(&subjects).Error
	return subjects, err
}

// Migrate 开发/测试路径：AutoMigrate 建表（约束与索引以迁移链为准）
func (r *permissionGroupRepository) Migrate() error {
	return r.db.AutoMigrate(&model.AssetPermissionGroup{}, &model.AssetPermissionGroupSubject{})
}
