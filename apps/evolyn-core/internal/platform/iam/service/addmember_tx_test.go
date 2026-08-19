package service

import (
	"context"
	"errors"
	"testing"

	kernel "evolyn/internal/model"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ---- FIX-021 事务语义桩 ----
//
// 单测层以「快照/恢复」模拟数据库事务：rollbackTx 在 fn 失败时恢复全部
// 桩仓储状态（等价回滚），成功则保留（等价提交）。真实 PostgreSQL 链路的
// 回滚由集成测试验证（见 sec_tenant_integration_test.go）。

// passThroughTx 不携带事务语义、直接执行 fn：适用于不关心回滚的既有用例
type passThroughTx struct{}

func (passThroughTx) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// txStore AddMember 全链路涉及的桩数据面（可整体快照/恢复）
type txStore struct {
	users        map[uint]*model.User
	roles        map[uint]*model.Role
	accounts     map[uint]*model.Account
	departments  map[uint]*model.Department
	userRoles    [][2]uint       // (memberID, roleID) 成功绑定记录
	deptBindings map[uint][]uint // memberID -> departmentIDs
	nextUserID   uint            // 成员 ID 分配器（模拟自增主键）
}

type txStoreSnapshot struct {
	users        map[uint]*model.User
	roles        map[uint]*model.Role
	accounts     map[uint]*model.Account
	departments  map[uint]*model.Department
	userRoles    [][2]uint
	deptBindings map[uint][]uint
	nextUserID   uint
}

func (s *txStore) snapshot() *txStoreSnapshot {
	snap := &txStoreSnapshot{
		users:        make(map[uint]*model.User, len(s.users)),
		roles:        make(map[uint]*model.Role, len(s.roles)),
		accounts:     make(map[uint]*model.Account, len(s.accounts)),
		departments:  make(map[uint]*model.Department, len(s.departments)),
		userRoles:    append([][2]uint(nil), s.userRoles...),
		deptBindings: make(map[uint][]uint, len(s.deptBindings)),
		nextUserID:   s.nextUserID,
	}
	for k, v := range s.users {
		snap.users[k] = v
	}
	for k, v := range s.roles {
		snap.roles[k] = v
	}
	for k, v := range s.accounts {
		snap.accounts[k] = v
	}
	for k, v := range s.departments {
		snap.departments[k] = v
	}
	for k, v := range s.deptBindings {
		snap.deptBindings[k] = append([]uint(nil), v...)
	}
	return snap
}

func (s *txStore) restore(snap *txStoreSnapshot) {
	s.users = snap.users
	s.roles = snap.roles
	s.accounts = snap.accounts
	s.departments = snap.departments
	s.userRoles = snap.userRoles
	s.deptBindings = snap.deptBindings
	s.nextUserID = snap.nextUserID
}

// rollbackTx 模拟数据库事务：fn 失败恢复快照（回滚），成功保留（提交）
type rollbackTx struct {
	store *txStore
}

func (f rollbackTx) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	snap := f.store.snapshot()
	if err := fn(ctx); err != nil {
		f.store.restore(snap)
		return err
	}
	return nil
}

// ---- 桩仓储：内嵌接口零实现 + 失败注入 ----

type txUserRepo struct {
	repository.UserRepository
	store          *txStore
	failCreate     error          // Create 失败注入
	failAddRoleFor map[uint]error // 指定 roleID 的 AddRole 失败注入
}

func (f *txUserRepo) GetByAccountAndTenant(ctx context.Context, accountID, tenantID uint) (*model.User, error) {
	for _, u := range f.usersLookup() {
		if u.AccountId == accountID && u.TenantID == tenantID {
			return u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *txUserRepo) usersLookup() map[uint]*model.User { return f.store.users }

func (f *txUserRepo) Create(ctx context.Context, member *model.User) (*model.User, error) {
	if f.failCreate != nil {
		return nil, f.failCreate
	}
	f.store.nextUserID++
	member.ID = f.store.nextUserID
	copied := *member
	f.store.users[member.ID] = &copied
	return member, nil
}

func (f *txUserRepo) AddRole(ctx context.Context, role *model.Role, user *model.User) error {
	if err, ok := f.failAddRoleFor[role.ID]; ok {
		return err
	}
	f.store.userRoles = append(f.store.userRoles, [2]uint{user.ID, role.ID})
	return nil
}

type txDeptRepo struct {
	repository.DepartmentRepository
	store *txStore
}

func (f *txDeptRepo) GetByID(ctx context.Context, id uint) (*model.Department, error) {
	if d, ok := f.store.departments[id]; ok {
		return d, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *txDeptRepo) SetMemberDepartments(ctx context.Context, member *model.User, departmentIDs []uint) error {
	f.store.deptBindings[member.ID] = append([]uint(nil), departmentIDs...)
	return nil
}

type txRBACRepo struct {
	repository.RBACRepository
	store *txStore
}

func (f *txRBACRepo) GetRoleByID(ctx context.Context, id int) (*model.Role, error) {
	if r, ok := f.store.roles[uint(id)]; ok {
		return r, nil
	}
	return nil, gorm.ErrRecordNotFound
}

type txAccountRepo struct {
	repository.AccountRepository
	store *txStore
}

func (f *txAccountRepo) GetByID(ctx context.Context, id uint) (*model.Account, error) {
	if a, ok := f.store.accounts[id]; ok {
		return a, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *txAccountRepo) GetByName(ctx context.Context, name string) (*model.Account, error) {
	for _, a := range f.store.accounts {
		if a.Name == name {
			return a, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

// newTxMemberFixtures 双租户（1/2）夹具：账号 30 未入租；租户 1 有部门 11、
// 角色 5；租户 2 有部门 12、角色 6（用于跨租户失败注入）
func newTxMemberFixtures() (*txStore, *txUserRepo, *txDeptRepo, *txRBACRepo, *txAccountRepo) {
	store := &txStore{
		users: map[uint]*model.User{},
		roles: map[uint]*model.Role{
			5: {ID: 5, Name: "role-t1", TenantBaseModel: kernel.TenantBaseModel{TenantID: 1}},
			6: {ID: 6, Name: "role-t2", TenantBaseModel: kernel.TenantBaseModel{TenantID: 2}},
		},
		accounts: map[uint]*model.Account{
			30: {ID: 30, Name: "acc-new", Nickname: "acc-new"},
		},
		departments: map[uint]*model.Department{
			11: {ID: 11, Name: "dept-t1", TenantBaseModel: kernel.TenantBaseModel{TenantID: 1}},
			12: {ID: 12, Name: "dept-t2", TenantBaseModel: kernel.TenantBaseModel{TenantID: 2}},
		},
		deptBindings: map[uint][]uint{},
	}
	return store,
		&txUserRepo{store: store},
		&txDeptRepo{store: store},
		&txRBACRepo{store: store},
		&txAccountRepo{store: store}
}

// ---- TX-MEMBER-001~005（FIX-021 验收用例）----

// TX-MEMBER-001：Member Create 失败，不产生任何关系数据
func TestTXMember001CreateFailNoResidue(t *testing.T) {
	store, users, depts, rbac, accounts := newTxMemberFixtures()
	users.failCreate = errors.New("db: insert users failed")
	svc := NewUserService(rollbackTx{store}, users, accounts, rbac, depts, nil, fakeAudit{})

	_, err := svc.AddMember(tenantCtx(t, 1), &AddMemberRequest{AccountID: 30, DepartmentIDs: []uint{11}, RoleIDs: []uint{5}})
	assert.Error(t, err)
	assert.Empty(t, store.users)
	assert.Empty(t, store.userRoles)
	assert.Empty(t, store.deptBindings)
}

// TX-MEMBER-002：Department Binding 失败（伪造他租部门），Member 回滚
func TestTXMember002DepartmentBindingFailRollsBackMember(t *testing.T) {
	store, users, depts, rbac, accounts := newTxMemberFixtures()
	svc := NewUserService(rollbackTx{store}, users, accounts, rbac, depts, nil, fakeAudit{})

	// 部门 12 属租户 2：同租户校验失败发生在 Member 创建之后
	_, err := svc.AddMember(tenantCtx(t, 1), &AddMemberRequest{AccountID: 30, DepartmentIDs: []uint{12}})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrCrossTenantBinding))
	assert.Empty(t, store.users, "成员必须随事务回滚")
	assert.Empty(t, store.deptBindings)
}

// TX-MEMBER-003：Role Binding 失败，Member + Department Binding 全部回滚
func TestTXMember003RoleBindingFailRollsBackAll(t *testing.T) {
	store, users, depts, rbac, accounts := newTxMemberFixtures()
	svc := NewUserService(rollbackTx{store}, users, accounts, rbac, depts, nil, fakeAudit{})

	// 部门 11 同租户绑定成功后，角色 6（租户 2）触发失败
	_, err := svc.AddMember(tenantCtx(t, 1), &AddMemberRequest{AccountID: 30, DepartmentIDs: []uint{11}, RoleIDs: []uint{6}})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrCrossTenantBinding))
	assert.Empty(t, store.users, "成员必须随事务回滚")
	assert.Empty(t, store.deptBindings, "已完成的部门绑定必须随事务回滚")
	assert.Empty(t, store.userRoles)
}

// TX-MEMBER-004：多角色绑定中后者失败，前面已绑定的 Role 同样回滚
func TestTXMember004PartialRoleBindingRollsBack(t *testing.T) {
	store, users, depts, rbac, accounts := newTxMemberFixtures()
	// 租户 1 的第二个角色，绑定它时注入失败
	store.roles[7] = &model.Role{ID: 7, Name: "role-t1b", TenantBaseModel: kernel.TenantBaseModel{TenantID: 1}}
	users.failAddRoleFor = map[uint]error{7: errors.New("db: insert user_roles failed")}
	svc := NewUserService(rollbackTx{store}, users, accounts, rbac, depts, nil, fakeAudit{})

	// 角色 5 绑定成功后角色 7 失败：Role1 不得保留（禁止部分成功状态）
	_, err := svc.AddMember(tenantCtx(t, 1), &AddMemberRequest{AccountID: 30, RoleIDs: []uint{5, 7}})
	assert.Error(t, err)
	assert.Empty(t, store.users)
	assert.Empty(t, store.userRoles, "已绑定的 Role1 必须随事务回滚")
}

// TX-MEMBER-005：正常成功路径下 Member/Department/Role 关系完整
func TestTXMember005HappyPath(t *testing.T) {
	store, users, depts, rbac, accounts := newTxMemberFixtures()
	svc := NewUserService(rollbackTx{store}, users, accounts, rbac, depts, nil, fakeAudit{})

	member, err := svc.AddMember(tenantCtx(t, 1), &AddMemberRequest{
		AccountID: 30, Nickname: "新同学", DepartmentIDs: []uint{11}, RoleIDs: []uint{5},
	})
	assert.NoError(t, err)
	assert.Equal(t, uint(1), member.TenantID)
	assert.Len(t, store.users, 1)
	assert.Equal(t, map[uint][]uint{member.ID: {11}}, store.deptBindings)
	assert.Equal(t, [][2]uint{{member.ID, 5}}, store.userRoles)
}
