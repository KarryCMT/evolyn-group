package repository

import (
	"context"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
)

type roleGroupRepository struct{ db *gorm.DB }

func newRoleGroupRepository(db *gorm.DB) RoleGroupRepository { return &roleGroupRepository{db: db} }

func (r *roleGroupRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *roleGroupRepository) List(ctx context.Context) ([]model.RoleGroup, error) {
	groups := make([]model.RoleGroup, 0)
	if err := r.withContext(ctx).Order("sort").Order("id").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (r *roleGroupRepository) GetByID(ctx context.Context, id uint) (*model.RoleGroup, error) {
	group := new(model.RoleGroup)
	if err := r.withContext(ctx).First(group, id).Error; err != nil {
		return nil, err
	}
	return group, nil
}

func (r *roleGroupRepository) GetByName(ctx context.Context, name string) (*model.RoleGroup, error) {
	group := new(model.RoleGroup)
	if err := r.withContext(ctx).Where("name = ?", name).First(group).Error; err != nil {
		return nil, err
	}
	return group, nil
}

func (r *roleGroupRepository) Create(ctx context.Context, group *model.RoleGroup) (*model.RoleGroup, error) {
	err := r.withContext(ctx).Create(group).Error
	return group, err
}

func (r *roleGroupRepository) Update(ctx context.Context, group *model.RoleGroup) (*model.RoleGroup, error) {
	err := r.withContext(ctx).Select("name", "sort").Updates(group).Error
	return group, err
}

func (r *roleGroupRepository) Delete(ctx context.Context, group *model.RoleGroup) error {
	return r.withContext(ctx).Delete(group).Error
}

func (r *roleGroupRepository) Migrate() error { return r.db.AutoMigrate(&model.RoleGroup{}) }
