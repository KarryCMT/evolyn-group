package repository

import (
	"context"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type rbacRepository struct {
	db  *gorm.DB
	rdb *infrastructure.RedisDB
}

func newRBACRepository(db *gorm.DB, rdb *infrastructure.RedisDB) RBACRepository {
	return &rbacRepository{
		db:  db,
		rdb: rdb,
	}
}

// withContext 以请求 ctx 打开新会话，租户过滤由 GORM Callback 自动注入
func (rbac *rbacRepository) withContext(ctx context.Context) *gorm.DB {
	return rbac.db.WithContext(ctx)
}

func (rbac *rbacRepository) List(ctx context.Context) ([]model.Role, error) {
	roles := make([]model.Role, 0)
	if err := rbac.withContext(ctx).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (rbac *rbacRepository) ListResources(ctx context.Context) ([]model.Resource, error) {
	resources := make([]model.Resource, 0)
	if err := rbac.withContext(ctx).Order("name").Find(&resources).Error; err != nil {
		return nil, err
	}
	return resources, nil
}

func (rbac *rbacRepository) Create(ctx context.Context, role *model.Role) (*model.Role, error) {
	err := rbac.withContext(ctx).Create(role).Error
	return role, err
}

func (rbac *rbacRepository) CreateResource(ctx context.Context, resource *model.Resource) (*model.Resource, error) {
	err := rbac.withContext(ctx).Create(resource).Error
	return resource, err
}

func (rbac *rbacRepository) CreateResources(ctx context.Context, resources []model.Resource, conds ...clause.Expression) error {
	err := rbac.withContext(ctx).Clauses(conds...).Create(resources).Error
	return err
}

func (rbac *rbacRepository) GetRoleByID(ctx context.Context, id int) (*model.Role, error) {
	role := &model.Role{}
	err := rbac.withContext(ctx).First(role, id).Error
	return role, err
}

func (rbac *rbacRepository) GetResource(ctx context.Context, id int) (*model.Resource, error) {
	res := &model.Resource{}
	err := rbac.withContext(ctx).First(res, id).Error
	return res, err
}

func (rbac *rbacRepository) GetRoleByName(ctx context.Context, name string) (*model.Role, error) {
	role := new(model.Role)
	if err := rbac.withContext(ctx).Where("name = ?", name).First(role).Error; err != nil {
		return nil, err
	}

	return role, nil
}

func (rbac *rbacRepository) Update(ctx context.Context, role *model.Role) (*model.Role, error) {
	err := rbac.withContext(ctx).Updates(role).Error
	return role, err
}

func (rbac *rbacRepository) Delete(ctx context.Context, id uint) error {
	return rbac.withContext(ctx).Delete(&model.Role{}, id).Error
}

func (rbac *rbacRepository) DeleteResource(ctx context.Context, id uint) error {
	return rbac.withContext(ctx).Delete(&model.Resource{}, id).Error
}

func (rbac *rbacRepository) Migrate() error {
	return rbac.db.AutoMigrate(&model.Role{}, &model.Resource{})
}
