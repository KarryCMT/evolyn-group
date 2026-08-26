package repository

import (
	"context"
	"strconv"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/iam/model"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var (
	// memberCreateField 成员创建仅写归属与展示字段（ADR-006：登录身份在账号侧）。
	// tenant_id 必须在列：显式指定租户的创建路径（如租户开通建 owner 成员）
	// 依赖本列写入，Create Select 过滤会丢弃 Callback 注入值之外的字段
	memberCreateField = []string{"account_id", "nickname", "tenant_id"}
)

type userRepository struct {
	db  *gorm.DB
	rdb *infrastructure.RedisDB
}

func newUserRepository(db *gorm.DB, rdb *infrastructure.RedisDB) UserRepository {
	return &userRepository{
		db:  db,
		rdb: rdb,
	}
}

// withContext 以请求 ctx 打开新会话：GORM 租户 Callback 从
// Statement.Context 读取租户并自动注入过滤/回填，租户对业务代码透明。
// ctx 携带事务 session 时加入外层事务（FIX-020/021 统一事务边界）
func (u *userRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, u.db)
}

func (u *userRepository) List(ctx context.Context) (model.Users, error) {
	users := make(model.Users, 0)
	if err := u.withContext(ctx).Preload(model.GroupAssociation).Preload("Roles").Preload(model.DepartmentAssociation).Order("id").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// ListPage 组织成员分页查询。先按 members 主表计数和分页，再 Preload 多对多
// 部门/角色，避免多关联 JOIN 令分页重复或 total 失真。
func (u *userRepository) ListPage(ctx context.Context, params model.MemberListQuery) (model.Users, int64, error) {
	// JOIN accounts 仅用于过滤有效账号与关键词匹配；必须限定主查询为 users.*。
	// 否则 PostgreSQL 返回的 accounts 列会与 users 的同名列混在一起，GORM 扫描时
	// 可能将 members.account_id 覆盖为零值，进而使创建者资料显示为空。
	query := u.withContext(ctx).Model(&model.User{}).Select("users.*").
		Joins("JOIN accounts ON accounts.id = users.account_id AND accounts.deleted_at IS NULL")

	if params.DepartmentID > 0 {
		query = query.Joins("JOIN department_users ON department_users.user_id = users.id").
			Where("department_users.department_id = ?", params.DepartmentID)
	}
	if params.RoleID > 0 {
		// 按角色筛选只连接关系表，成员完整角色列表仍由 Preload 返回，供组织页
		// 在同一行展示成员拥有的全部角色。
		query = query.Joins("JOIN user_roles ON user_roles.user_id = users.id").
			Where("user_roles.role_id = ?", params.RoleID)
	}
	if params.Keyword != "" {
		keyword := "%" + params.Keyword + "%"
		query = query.Where(
			"users.nickname ILIKE ? OR accounts.nickname ILIKE ? OR accounts.name ILIKE ? OR accounts.phone ILIKE ? OR accounts.email ILIKE ?",
			keyword, keyword, keyword, keyword, keyword,
		)
	}
	switch params.Status {
	case model.MemberStatusActive, model.MemberStatusDisabled, model.MemberStatusResigned:
		query = query.Where("users.status = ?", params.Status)
	default:
		query = query.Where("users.status IN ?", []string{model.MemberStatusActive, model.MemberStatusDisabled})
	}

	var total int64
	// Distinct 会把当前链的 Select 改为 users.id。计数使用独立会话，避免后续
	// Find 只取回 id，导致创建者的 account_id、昵称等字段全部成为零值。
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Distinct("users.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	users := make(model.Users, 0)
	if err := query.Preload("Account").Preload("Roles").Preload(model.DepartmentAssociation).
		Order("users.id").Offset((params.Page - 1) * params.PageSize).Limit(params.PageSize).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// ListByAccount 账号的全部成员关系（登录后租户列表/默认成员解析）。
// 登录链路可能尚未注入租户上下文，此处显式按账号查全量成员关系
func (u *userRepository) ListByAccount(ctx context.Context, accountID uint) (model.Users, error) {
	users := make(model.Users, 0)
	if err := u.withContext(ctx).Preload(model.GroupAssociation).Preload("Roles").Where("account_id = ?", accountID).Order("id").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// GetByAccountAndTenant 精确定位账号在指定租户的成员（租户切换链路）
func (u *userRepository) GetByAccountAndTenant(ctx context.Context, accountID, tenantID uint) (*model.User, error) {
	user := new(model.User)
	if err := u.withContext(ctx).Preload(model.GroupAssociation).Preload(model.DepartmentAssociation).Preload("Roles").
		Where("account_id = ? and tenant_id = ?", accountID, tenantID).First(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// CountByTenant 指定租户的有效成员数（软删与离职成员均不计）：配额执行路径使用，
// 显式 Scope 而非依赖请求租户上下文（运营/定时任务可能无上下文）
func (u *userRepository) CountByTenant(ctx context.Context, tenantID uint) (int64, error) {
	var count int64
	err := u.withContext(ctx).Model(&model.User{}).
		Scopes(infrastructure.TenantScope(tenantID)).
		Where("status IN ?", []string{model.MemberStatusActive, model.MemberStatusDisabled}).
		Count(&count).Error
	return count, err
}

func (u *userRepository) Create(ctx context.Context, member *model.User) (*model.User, error) {
	if err := u.withContext(ctx).Select(memberCreateField).Create(member).Error; err != nil {
		return nil, err
	}

	u.setCacheUser(member)

	return member, nil
}

func (u *userRepository) Update(ctx context.Context, member *model.User) (*model.User, error) {
	if err := u.withContext(ctx).Model(&model.User{}).Where("id = ?", member.ID).
		Select("nickname").Updates(member).Error; err != nil {
		return nil, err
	}

	u.rdb.HDel(member.CacheKey(), strconv.Itoa(int(member.ID)))

	return member, nil
}

// UpdateStatus 仅修改成员租户内状态及离职时间；账号资料不受该操作影响。
func (u *userRepository) UpdateStatus(ctx context.Context, member *model.User) (*model.User, error) {
	if err := u.withContext(ctx).Model(&model.User{}).Where("id = ?", member.ID).
		Select("status", "resigned_at").Updates(member).Error; err != nil {
		return nil, err
	}
	u.rdb.HDel(member.CacheKey(), strconv.Itoa(int(member.ID)))
	return member, nil
}

func (u *userRepository) Delete(ctx context.Context, member *model.User) error {
	if err := u.withContext(ctx).Delete(member).Error; err != nil {
		return err
	}
	u.rdb.HDel(member.CacheKey(), strconv.Itoa(int(member.ID)))
	return nil
}

// GetUserByID 按成员 ID 加载（认证中间件按 JWT memberId 取成员；
// 此时请求 ctx 尚无租户上下文，随后由 TenantMiddleware 注入）
func (u *userRepository) GetUserByID(ctx context.Context, id uint) (*model.User, error) {
	// TODO HSet not support expire, avoid roles and groups inconsistent
	// if user := u.getCacheUser(id); user != nil {
	// 	return user, nil
	// }

	user := new(model.User)
	if err := u.withContext(ctx).Preload(model.GroupAssociation).Preload("Groups.Roles").Preload("Roles").First(user, id).Error; err != nil {
		return nil, err
	}

	if err := u.setCacheUser(user); err != nil {
		logrus.Errorf("failed to set user: %v", err)
	}

	return user, nil
}

func (u *userRepository) AddRole(ctx context.Context, role *model.Role, user *model.User) error {
	return u.withContext(ctx).Model(user).Association("Roles").Append(role)
}

func (u *userRepository) DelRole(ctx context.Context, role *model.Role, user *model.User) error {
	return u.withContext(ctx).Model(user).Association("Roles").Delete(role)
}

func (u *userRepository) GetGroups(ctx context.Context, user *model.User) ([]model.Group, error) {
	groups := make([]model.Group, 0)
	err := u.withContext(ctx).Model(user).Association(model.GroupAssociation).Find(&groups)
	return groups, err
}

// PurgeByAccount 平台账号注销时硬删其所有租户成员身份。关联表没有独立
// 生命周期，必须先于 users 删除；调用方负责在外层事务中先校验账号不再
// 是任何租户的创建人。
func (u *userRepository) PurgeByAccount(ctx context.Context, accountID uint) error {
	db := u.withContext(ctx)
	for _, stmt := range []string{
		`DELETE FROM department_users WHERE user_id IN (SELECT id FROM users WHERE account_id = ?)`,
		`DELETE FROM user_groups WHERE user_id IN (SELECT id FROM users WHERE account_id = ?)`,
		`DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE account_id = ?)`,
		`DELETE FROM users WHERE account_id = ?`,
	} {
		if err := db.Exec(stmt, accountID).Error; err != nil {
			return err
		}
	}
	return nil
}

func (u *userRepository) Migrate() error {
	return u.db.AutoMigrate(&model.User{})
}

func (u *userRepository) setCacheUser(user *model.User) error {
	if user == nil {
		return nil
	}

	return u.rdb.HSet(user.CacheKey(), strconv.Itoa(int(user.ID)), user)
}

func (u *userRepository) getCacheUser(id uint) *model.User {
	user := new(model.User)
	key := user.CacheKey()
	field := strconv.Itoa(int(id))
	if err := u.rdb.HGet(key, field, user); err != nil {
		if err != infrastructure.RedisDisableError {
			logrus.Warnf("failed to hget field %s from key %s, %v", field, key, err)
		}
		return nil
	}

	return user
}
