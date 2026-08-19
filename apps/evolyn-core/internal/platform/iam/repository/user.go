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

// CountByTenant 指定租户的有效成员数（软删行不计）：配额执行路径使用，
// 显式 Scope 而非依赖请求租户上下文（运营/定时任务可能无上下文）
func (u *userRepository) CountByTenant(ctx context.Context, tenantID uint) (int64, error) {
	var count int64
	err := u.withContext(ctx).Model(&model.User{}).
		Scopes(infrastructure.TenantScope(tenantID)).Count(&count).Error
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
