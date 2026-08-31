package service

import (
	"context"
	"errors"
	"testing"

	"evolyn/internal/contextx"
	kernel "evolyn/internal/model"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"
	tenantservice "evolyn/internal/platform/tenant/service"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ---- 轻量仓储桩：内嵌接口零实现，仅覆写用到的方法（未覆写方法被调用会 panic，测试即失败）----

type fakeUserRepo struct {
	repository.UserRepository
	users           map[uint]*model.User
	created         []*model.User
	addRoleBindings []uint // 成功绑定的 roleID 记录
}

func (f *fakeUserRepo) GetUserByID(ctx context.Context, id uint) (*model.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeUserRepo) GetByAccountAndTenant(ctx context.Context, accountID, tenantID uint) (*model.User, error) {
	for _, u := range f.users {
		if u.AccountId == accountID && u.TenantID == tenantID {
			return u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeUserRepo) Create(ctx context.Context, member *model.User) (*model.User, error) {
	member.ID = uint(100 + len(f.users))
	f.users[member.ID] = member
	f.created = append(f.created, member)
	return member, nil
}

func (f *fakeUserRepo) AddRole(ctx context.Context, role *model.Role, user *model.User) error {
	f.addRoleBindings = append(f.addRoleBindings, role.ID)
	return nil
}

type fakeRBACRepo struct {
	repository.RBACRepository
	roles map[uint]*model.Role
}

func (f *fakeRBACRepo) GetRoleByID(ctx context.Context, id int) (*model.Role, error) {
	if r, ok := f.roles[uint(id)]; ok {
		return r, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeRBACRepo) GetRoleByName(ctx context.Context, name string) (*model.Role, error) {
	// 模拟 GORM 租户 Callback：ctx 携带租户时按租户过滤
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	for _, r := range f.roles {
		if r.Name != name {
			continue
		}
		if ok && r.TenantID != tenantID {
			continue
		}
		return r, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeRBACRepo) Create(ctx context.Context, role *model.Role) (*model.Role, error) {
	role.ID = uint(200 + len(f.roles))
	f.roles[role.ID] = role
	return role, nil
}

type fakeGroupRepo struct {
	repository.GroupRepository
	groups     map[uint]*model.Group
	addUserIDs []uint // 成功绑定的成员记录
}

func (f *fakeGroupRepo) GetGroupByID(ctx context.Context, id uint) (*model.Group, error) {
	if g, ok := f.groups[id]; ok {
		return g, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeGroupRepo) GetGroupByName(ctx context.Context, name string) (*model.Group, error) {
	// 模拟 GORM 租户 Callback：ctx 携带租户时按租户过滤
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	for _, g := range f.groups {
		if g.Name != name {
			continue
		}
		if ok && g.TenantID != tenantID {
			continue
		}
		return g, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeGroupRepo) Create(ctx context.Context, user *model.User, group *model.Group) (*model.Group, error) {
	group.ID = uint(200 + len(f.groups))
	f.groups[group.ID] = group
	return group, nil
}

func (f *fakeGroupRepo) AddUser(ctx context.Context, user *model.User, group *model.Group) error {
	f.addUserIDs = append(f.addUserIDs, user.ID)
	return nil
}

func (f *fakeGroupRepo) AddRole(ctx context.Context, role *model.Role, group *model.Group) error {
	return nil
}

type fakeAccountRepo struct {
	repository.AccountRepository
	accounts map[uint]*model.Account
}

func (f *fakeAccountRepo) GetByID(ctx context.Context, id uint) (*model.Account, error) {
	if a, ok := f.accounts[id]; ok {
		return a, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeAccountRepo) GetByName(ctx context.Context, name string) (*model.Account, error) {
	for _, a := range f.accounts {
		if a.Name == name {
			return a, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

type fakeAudit struct{}

func (fakeAudit) Record(context.Context, auditservice.Entry) {}

// fakeQuota 可配置放行/超限的配额桩
type fakeQuota struct {
	exceeded bool
}

func (f fakeQuota) Check(ctx context.Context, tenantID uint, key string) error {
	if f.exceeded {
		return tenantservice.ErrQuotaExceeded
	}
	return nil
}

func (f fakeQuota) Usage(ctx context.Context, tenantID uint, key string) (int64, error) {
	return 0, nil
}

// CheckAndReserve 桩：直接透传 fn（单测不覆盖并发语义，真实链路见应用域集成测试）
func (f fakeQuota) CheckAndReserve(ctx context.Context, tenantID uint, key string, fn func(ctx context.Context) error) error {
	if f.exceeded {
		return tenantservice.ErrQuotaExceeded
	}
	return fn(ctx)
}

// bindingFixtures 租户关系绑定测试夹具集合（结构体返回：各用例按需取用，
// 避免 4 返回值多空位声明触发 dogsled）
type bindingFixtures struct {
	users    *fakeUserRepo
	roles    *fakeRBACRepo
	groups   *fakeGroupRepo
	accounts *fakeAccountRepo
}

// newBindingFixtures 双租户（1/2）测试夹具：成员 1 与角色 5 同属租户 1；
// 成员 2、角色 6、分组 7 属租户 2
func newBindingFixtures() *bindingFixtures {
	users := &fakeUserRepo{users: map[uint]*model.User{
		1: {ID: 1, AccountId: 10, Nickname: "member-a", TenantBaseModel: kernel.TenantBaseModel{TenantID: 1}},
		2: {ID: 2, AccountId: 20, Nickname: "member-b", TenantBaseModel: kernel.TenantBaseModel{TenantID: 2}},
	}}
	roles := &fakeRBACRepo{roles: map[uint]*model.Role{
		5: {ID: 5, Name: "role-t1", TenantBaseModel: kernel.TenantBaseModel{TenantID: 1}},
		6: {ID: 6, Name: "role-t2", TenantBaseModel: kernel.TenantBaseModel{TenantID: 2}},
	}}
	groups := &fakeGroupRepo{groups: map[uint]*model.Group{
		7: {ID: 7, Name: "group-t2", TenantBaseModel: kernel.TenantBaseModel{TenantID: 2}},
		8: {ID: 8, Name: "group-t1", TenantBaseModel: kernel.TenantBaseModel{TenantID: 1}},
	}}
	accounts := &fakeAccountRepo{accounts: map[uint]*model.Account{
		10: {ID: 10, Name: "acc-a", Nickname: "acc-a"},
		30: {ID: 30, Name: "acc-new", Nickname: "acc-new"},
	}}
	return &bindingFixtures{users: users, roles: roles, groups: groups, accounts: accounts}
}

func tenantCtx(t *testing.T, tenantID uint) context.Context {
	t.Helper()
	return contextx.NewTenantContext(context.Background(), tenantID)
}

// ---- FIX-006：跨租户关系绑定必须失败 ----

func TestAddRoleCrossTenantRejected(t *testing.T) {
	fx := newBindingFixtures()
	users, roles := fx.users, fx.roles
	svc := NewUserService(passThroughTx{}, users, &fakeAccountRepo{}, roles, nil, fakeQuota{}, fakeAudit{})

	// 租户 1 的成员 1 绑定租户 2 的角色 6：必须拒绝且不落关系表
	err := svc.AddRole(tenantCtx(t, 1), "1", "6")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrCrossTenantBinding))
	assert.Empty(t, users.addRoleBindings)

	// 同租户绑定放行
	err = svc.AddRole(tenantCtx(t, 1), "1", "5")
	assert.NoError(t, err)
	assert.Equal(t, []uint{5}, users.addRoleBindings)
}

func TestGroupAddUserCrossTenantRejected(t *testing.T) {
	fx := newBindingFixtures()
	users, roles, groups := fx.users, fx.roles, fx.groups
	svc := NewGroupService(groups, users, roles, fakeAudit{})

	// 租户 1 成员 1 加入租户 2 分组 7：拒绝且不落绑定
	err := svc.AddUser(tenantCtx(t, 1), &model.User{ID: 1}, "7")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrCrossTenantBinding))
	assert.Empty(t, groups.addUserIDs)

	// 同租户分组放行
	err = svc.AddUser(tenantCtx(t, 1), &model.User{ID: 1}, "8")
	assert.NoError(t, err)
	assert.Equal(t, []uint{1}, groups.addUserIDs)
}

func TestGroupAddRoleCrossTenantRejected(t *testing.T) {
	fx := newBindingFixtures()
	users, roles, groups := fx.users, fx.roles, fx.groups
	svc := NewGroupService(groups, users, roles, fakeAudit{})

	// 租户 1 分组 8 绑定租户 2 角色 6：拒绝
	err := svc.AddRole(tenantCtx(t, 1), "8", "6")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrCrossTenantBinding))

	err = svc.AddRole(tenantCtx(t, 1), "8", "5")
	assert.NoError(t, err)
}

// ---- FIX-002/003：租户内重名预检 ----

func TestRoleCreateDuplicateNameRejected(t *testing.T) {
	fx := newBindingFixtures()
	roles := fx.roles
	svc := NewRBACService(roles, fakeAudit{})

	// 租户 1 已有 role-t1，再建同名：拒绝
	_, err := svc.Create(tenantCtx(t, 1), &model.Role{Name: "role-t1"})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrDuplicateName))

	// 租户 2 建同名：放行（各租户命名空间隔离）
	_, err = svc.Create(tenantCtx(t, 2), &model.Role{Name: "role-t1"})
	assert.NoError(t, err)
}

func TestGroupCreateDuplicateNameRejected(t *testing.T) {
	fx := newBindingFixtures()
	users, roles, groups := fx.users, fx.roles, fx.groups
	svc := NewGroupService(groups, users, roles, fakeAudit{})

	// 租户 2 已有 group-t2，再建同名：拒绝
	_, err := svc.Create(tenantCtx(t, 2), &model.User{ID: 2}, &model.Group{Name: "group-t2"})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrDuplicateName))

	// 租户 1 建同名：放行
	_, err = svc.Create(tenantCtx(t, 1), &model.User{ID: 1}, &model.Group{Name: "group-t2"})
	assert.NoError(t, err)
}

// ---- FIX-010：拉人入租户闭环 ----

func TestAddMemberDuplicateRejected(t *testing.T) {
	fx := newBindingFixtures()
	users, accounts := fx.users, fx.accounts
	svc := NewUserService(passThroughTx{}, users, accounts, nil, nil, fakeQuota{}, fakeAudit{})

	// 账号 10 在租户 1 已有成员（成员 1）：拒绝重复加入
	_, err := svc.AddMember(tenantCtx(t, 1), &AddMemberRequest{AccountID: 10})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrDuplicateMember))
	assert.Empty(t, users.created)
}

func TestAddMemberQuotaExceeded(t *testing.T) {
	fx := newBindingFixtures()
	users, accounts := fx.users, fx.accounts
	svc := NewUserService(passThroughTx{}, users, accounts, nil, nil, fakeQuota{exceeded: true}, fakeAudit{})

	_, err := svc.AddMember(tenantCtx(t, 1), &AddMemberRequest{AccountID: 30})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, tenantservice.ErrQuotaExceeded))
	assert.Empty(t, users.created)
}

func TestAddMemberHappyPath(t *testing.T) {
	fx := newBindingFixtures()
	users, accounts := fx.users, fx.accounts
	svc := NewUserService(passThroughTx{}, users, accounts, nil, nil, fakeQuota{}, fakeAudit{})

	member, err := svc.AddMember(tenantCtx(t, 1), &AddMemberRequest{AccountID: 30, Nickname: "新同学"})
	assert.NoError(t, err)
	assert.Equal(t, uint(30), member.AccountId)
	assert.Equal(t, uint(1), member.TenantID)
	assert.Equal(t, "新同学", member.Nickname)
	assert.Len(t, users.created, 1)

	// 同一账号加入其他租户不受影响（跨租户多成员关系合法）
	member2, err := svc.AddMember(tenantCtx(t, 2), &AddMemberRequest{AccountID: 30})
	assert.NoError(t, err)
	assert.Equal(t, uint(2), member2.TenantID)
}

func TestAddMemberRequiresTenantContext(t *testing.T) {
	fx := newBindingFixtures()
	users, accounts := fx.users, fx.accounts
	svc := NewUserService(passThroughTx{}, users, accounts, nil, nil, fakeQuota{}, fakeAudit{})

	// 无租户上下文（如平台域误用）：直接拒绝
	_, err := svc.AddMember(context.Background(), &AddMemberRequest{AccountID: 30})
	assert.Error(t, err)
}
