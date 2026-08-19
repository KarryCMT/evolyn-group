package repository

import (
	"context"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repositories iam 域仓储集合：聚合 account/user/group/rbac 四仓储与域级迁移/种子。
// 取代原全局 Repository 巨型聚合接口（ADR-007 域模块化），域间各自自治。
type Repositories struct {
	db         *gorm.DB
	account    AccountRepository
	user       UserRepository
	group      GroupRepository
	rbac       RBACRepository
	department DepartmentRepository
}

func NewRepositories(db *gorm.DB, rdb *infrastructure.RedisDB) *Repositories {
	return &Repositories{
		db:         db,
		account:    newAccountRepository(db, rdb),
		user:       newUserRepository(db, rdb),
		group:      newGroupRepository(db, rdb),
		rbac:       newRBACRepository(db, rdb),
		department: newDepartmentRepository(db, rdb),
	}
}

func (r *Repositories) Account() AccountRepository {
	return r.account
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

func (r *Repositories) Department() DepartmentRepository {
	return r.department
}

// Migrate iam 域表迁移：account/auth_infos → user → group → role/resource → department。
// AutoMigrate 仅开发/测试路径（FIX-009）：GORM 标签表达不了 PG 部分唯一索引，
// 此处用幂等 SQL 补齐，使开发库约束与 migrations 终态一致
//（FIX-002/003/004/017；外键约束只在 migrations 路径落地）
func (r *Repositories) Migrate() error {
	for _, m := range []interface{ Migrate() error }{r.account, r.user, r.group, r.rbac, r.department} {
		if err := m.Migrate(); err != nil {
			return err
		}
	}
	if err := r.dropLegacyUniqueIndexes(); err != nil {
		return err
	}
	return r.ensurePartialUniqueIndexes()
}

// ensurePartialUniqueIndexes 补齐软删除友好的租户内唯一索引（与迁移链同名同构）
func (r *Repositories) ensurePartialUniqueIndexes() error {
	for _, stmt := range []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_roles_tenant_name ON roles (tenant_id, name) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_groups_tenant_name ON groups (tenant_id, name) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_users_tenant_account ON users (tenant_id, account_id) WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_auth_identity ON auth_infos (auth_type, auth_id) WHERE deleted_at IS NULL`,
	} {
		if err := r.db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

// Init iam 域种子与存量回填：账号拆分回填最先执行，再平台基础资源与系统分组。
// 启动期无请求上下文，统一用 Background：无租户上下文时 Callback 无副作用，
// 种子数据按列默认值归属默认租户
func (r *Repositories) Init() error {
	ctx := context.Background()

	if err := r.backfillAccountSplit(); err != nil {
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
		// P3：运营域（管租户）与部门资源；platform 域仅 cluster-admin 通配可达
		{
			Name:  "platform",
			Scope: model.ClusterScope,
		},
		{
			Name:  "departments",
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

// backfillAccountSplit 账号×成员拆分的存量回填（ADR-006，幂等可重跑）：
//  1. 老用户行按同 ID 复制为平台账号（登录名/密码/邮箱/头像随迁）；
//  2. users.account_id 对齐自身 ID（同 ID 复制策略使成员与账号一一对应）；
//  3. auth_infos.account_id 从旧 user_id 回填；
//  4. 未设置 owner 的租户，owner 指向租内最小 ID 成员的账号。
//
// 旧列（users.name 等）不删除，代码不再声明即无业务影响；新库由 db.sql 保证干净
func (r *Repositories) backfillAccountSplit() error {
	// 同 ID 复制：避免 name/email 冲突重复导入
	if err := r.db.Exec(`
		INSERT INTO accounts (id, name, nickname, email, password, avatar, created_at, updated_at)
		SELECT u.id, u.name, u.name, u.email, u.password, u.avatar, u.created_at, u.updated_at
		FROM users u
		WHERE NOT EXISTS (SELECT 1 FROM accounts a WHERE a.id = u.id OR a.name = u.name)
	`).Error; err != nil {
		return err
	}

	if err := r.db.Exec(`
		UPDATE users SET account_id = id
		WHERE account_id = 0 AND EXISTS (SELECT 1 FROM accounts a WHERE a.id = users.id)
	`).Error; err != nil {
		return err
	}

	if err := r.db.Exec(`
		UPDATE auth_infos SET account_id = user_id
		WHERE (account_id = 0 OR account_id IS NULL)
		  AND EXISTS (SELECT 1 FROM accounts a WHERE a.id = auth_infos.user_id)
	`).Error; err != nil {
		return err
	}

	// FIX-016：owner 为可空外键，NULL = 未设置；无成员的租户保持 NULL
	//（子查询无行时自然写 NULL，不再落 0 哨兵）
	if err := r.db.Exec(`
		UPDATE tenants t SET owner_account_id = (
			SELECT u.account_id FROM users u
			WHERE u.tenant_id = t.id AND u.account_id > 0
			ORDER BY u.id LIMIT 1)
		WHERE owner_account_id IS NULL
	`).Error; err != nil {
		return err
	}

	return nil
}

// dropLegacyUniqueIndexes 移除 weave 时代的全局唯一索引：
// 账号×成员拆分与租户内种子化（每租户系统组/角色同名）要求 name 仅租户内唯一，
// 唯一性改由服务层校验。幂等，新库无旧索引时无操作
func (r *Repositories) dropLegacyUniqueIndexes() error {
	for _, idx := range []string{"idx_groups_name", "idx_roles_name"} {
		if err := r.db.Exec("DROP INDEX IF EXISTS " + idx).Error; err != nil {
			return err
		}
	}
	return nil
}
