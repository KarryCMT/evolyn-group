package repository

import (
	"context"

	"evolyn/internal/database"
	"evolyn/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	groupUpdateFields = []string{"Describe", "Roles", "UpdaterId"}
)

type groupRepository struct {
	db  *gorm.DB
	rdb *database.RedisDB
}

func newGroupRepository(db *gorm.DB, rdb *database.RedisDB) GroupRepository {
	return &groupRepository{
		db:  db,
		rdb: rdb,
	}
}

// withContext 以请求 ctx 打开新会话，租户过滤由 GORM Callback 自动注入
func (g *groupRepository) withContext(ctx context.Context) *gorm.DB {
	return g.db.WithContext(ctx)
}

func (g *groupRepository) List(ctx context.Context) ([]model.Group, error) {
	groups := make([]model.Group, 0)
	if err := g.withContext(ctx).Order("name").Preload("Roles").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (g *groupRepository) Create(ctx context.Context, user *model.User, group *model.Group) (*model.Group, error) {
	group.CreatorId = user.ID
	group.Users = []model.User{*user}
	err := g.withContext(ctx).Create(group).Error
	return group, err
}

func (g *groupRepository) CreateGroups(ctx context.Context, groups []model.Group, conds ...clause.Expression) error {
	return g.withContext(ctx).Clauses(conds...).Create(groups).Error
}

func (g *groupRepository) GetUsers(ctx context.Context, group *model.Group) (model.Users, error) {
	users := make(model.Users, 0)
	err := g.withContext(ctx).Model(group).Association(model.UserAssociation).Find(&users)
	return users, err
}

func (g *groupRepository) AddUser(ctx context.Context, user *model.User, group *model.Group) error {
	return g.withContext(ctx).Model(group).Association(model.UserAssociation).Append(user)
}

func (g *groupRepository) DelUser(ctx context.Context, user *model.User, group *model.Group) error {
	return g.withContext(ctx).Model(group).Association(model.UserAssociation).Delete(user)
}

func (g *groupRepository) AddRole(ctx context.Context, role *model.Role, group *model.Group) error {
	var err error
	if group.ID == 0 {
		group, err = g.GetGroupByName(ctx, group.Name)
	}
	if err != nil {
		return err
	}
	return g.withContext(ctx).Model(group).Association("Roles").Append(role)
}

func (g *groupRepository) DelRole(ctx context.Context, role *model.Role, group *model.Group) error {
	var err error
	if group.ID == 0 {
		group, err = g.GetGroupByName(ctx, group.Name)
	}
	if err != nil {
		return err
	}
	return g.withContext(ctx).Model(group).Association("Roles").Delete(role)
}

func (g *groupRepository) GetGroupByID(ctx context.Context, id uint) (*model.Group, error) {
	group := new(model.Group)
	if err := g.withContext(ctx).Preload("Users").Preload("Roles").First(group, id).Error; err != nil {
		return nil, err
	}

	return group, nil
}

func (g *groupRepository) GetGroupByName(ctx context.Context, name string) (*model.Group, error) {
	group := new(model.Group)
	if err := g.withContext(ctx).Preload("Users").Preload("Roles").Where("name = ?", name).First(group).Error; err != nil {
		return nil, err
	}

	return group, nil
}

func (g *groupRepository) Update(ctx context.Context, group *model.Group) (*model.Group, error) {
	err := g.withContext(ctx).Model(group).Select(groupUpdateFields).Updates(group).Error
	return group, err
}

func (g *groupRepository) Delete(ctx context.Context, id uint) error {
	return g.withContext(ctx).Delete(&model.Group{}, id).Error
}

func (g *groupRepository) RoleBinding(ctx context.Context, role *model.Role, group *model.Group) error {
	return g.withContext(ctx).Model(group).Association("Roles").Append(role)
}

func (g *groupRepository) Migrate() error {
	return g.db.AutoMigrate(&model.Group{})
}
