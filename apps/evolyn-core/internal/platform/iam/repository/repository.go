package repository

import (
	"context"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repositories iam 域仓储集合：聚合 user/group/rbac 三仓储与域级迁移/种子。
// 取代原全局 Repository 巨型聚合接口（ADR-007 域模块化），域间各自自治。
type Repositories struct {
	db    *gorm.DB
	user  UserRepository
	group GroupRepository
	rbac  RBACRepository
}

func NewRepositories(db *gorm.DB, rdb *infrastructure.RedisDB) *Repositories {
	return &Repositories{
		db:    db,
		user:  newUserRepository(db, rdb),
		group: newGroupRepository(db, rdb),
		rbac:  newRBACRepository(db, rdb),
	}
}

func (r *Repositories) User() UserRepository {
	return r.user
}

func (r *Repositories) Group() GroupRepository {
	return r.group
}

func (r *Repositories) RBAC() RBACRepository {
	return r.rbac
}

// Migrate iam 域表迁移：user/auth_infos → group → role/resource → department
func (r *Repositories) Migrate() error {
	for _, m := range []interface{ Migrate() error }{r.user, r.group, r.rbac} {
		if err := m.Migrate(); err != nil {
			return err
		}
	}
	// department 随 P3 建仓储，暂由域聚合统一迁移（departments 表 + department_users 关联）
	return r.db.AutoMigrate(&model.Department{})
}

// Init iam 域种子：平台基础资源与系统分组（原全局 repository.Init 的 iam 部分）。
// 启动期无请求上下文，统一用 Background：无租户上下文时 Callback 无副作用，
// 种子数据按列默认值归属默认租户
func (r *Repositories) Init() error {
	ctx := context.Background()

	// 平台基础资源注册：随功能演进而增删，K8s 相关资源已随功能剥离移除
	resources := []model.Resource{
		{
			Name:  model.GroupResource,
			Scope: model.ClusterScope,
		},
		{
			Name:  model.UserResource,
			Scope: model.ClusterScope,
		},
		{
			Name:  model.RoleResource,
			Scope: model.ClusterScope,
		},
		{
			Name:  model.AuthResource,
			Scope: model.ClusterScope,
		},
	}

	if err := r.rbac.CreateResources(ctx, resources, clause.OnConflict{DoNothing: true}); err != nil {
		return err
	}

	// create default group
	groups := []model.Group{
		{
			Name:     model.RootGroup,
			Kind:     model.SystemGroup,
			Describe: "system root group",
		},
		{
			Name:     model.AuthenticatedGroup,
			Kind:     model.SystemGroup,
			Describe: "system group contains all authenticated user",
		},
		{
			Name:     model.UnAuthenticatedGroup,
			Kind:     model.SystemGroup,
			Describe: "system group contains all unauthenticated user",
		},
	}
	if err := r.group.CreateGroups(ctx, groups, clause.OnConflict{DoNothing: true}); err != nil {
		return err
	}

	return nil
}
