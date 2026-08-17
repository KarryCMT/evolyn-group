package repository

import (
	"context"

	"evolyn/internal/database"
	"evolyn/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func NewRepository(db *gorm.DB, rdb *database.RedisDB) Repository {
	r := &repository{
		db:     db,
		rdb:    rdb,
		user:   newUserRepository(db, rdb),
		group:  newGroupRepository(db, rdb),
		rbac:   newRBACRepository(db, rdb),
		tenant: newTenantRepository(db, rdb),
	}

	// tenants 表最先迁移，保证业务模型加 tenant_id 列时默认租户已就绪
	r.migrants = getMigrants(
		r.tenant,
		r.user,
		r.group,
		r.rbac,
	)

	return r
}

func getMigrants(objs ...interface{}) []Migrant {
	var migrants []Migrant
	for _, obj := range objs {
		if m, ok := obj.(Migrant); ok {
			migrants = append(migrants, m)
		}
	}
	return migrants
}

type repository struct {
	user     UserRepository
	group    GroupRepository
	rbac     RBACRepository
	tenant   TenantRepository
	db       *gorm.DB
	rdb      *database.RedisDB
	migrants []Migrant
}

func (r *repository) User() UserRepository {
	return r.user
}

func (r *repository) Group() GroupRepository {
	return r.group
}

func (r *repository) RBAC() RBACRepository {
	return r.rbac
}

func (r *repository) Tenant() TenantRepository {
	return r.tenant
}

func (r *repository) Close() error {
	db, _ := r.db.DB()
	if db != nil {
		if err := db.Close(); err != nil {
			return err
		}
	}

	if r.rdb != nil {
		if err := r.rdb.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) Ping(ctx context.Context) error {
	db, err := r.db.DB()
	if err != nil {
		return err
	}
	if err = db.PingContext(ctx); err != nil {
		return err
	}

	if r.rdb == nil {
		return nil
	}
	if _, err := r.rdb.Ping(ctx).Result(); err != nil {
		return err
	}

	return nil
}

func (r *repository) Migrate() error {
	for _, m := range r.migrants {
		if err := m.Migrate(); err != nil {
			return err
		}
	}
	return nil
}

func (r *repository) Init() error {
	// 默认租户必须最先种子化：单租户/存量数据的归属兜底（见架构文档 26.8 P0）
	if err := r.Tenant().SeedDefaultTenant(); err != nil {
		return err
	}

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

	if err := r.RBAC().CreateResources(resources, clause.OnConflict{DoNothing: true}); err != nil {
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
	if err := r.Group().CreateGroups(groups, clause.OnConflict{DoNothing: true}); err != nil {
		return err
	}

	return nil
}
