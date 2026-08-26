package service

import (
	"context"
	"errors"
	"testing"

	auditservice "evolyn/internal/platform/audit/service"
	iammodel "evolyn/internal/platform/iam/model"
	iamrepository "evolyn/internal/platform/iam/repository"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantrepository "evolyn/internal/platform/tenant/repository"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---- FIX-020 事务语义桩 ----
//
// 单测层以「快照/恢复」模拟数据库事务：openRollbackTx 在 fn 失败时恢复
// 全部桩仓储状态（等价回滚），成功保留（等价提交）。真实 PostgreSQL 链路
// 的回滚由集成测试验证（见 sec_tenant_integration_test.go）。

// openStore Open 全链路涉及的桩数据面（可整体快照/恢复）
type openStore struct {
	tenants     map[uint]*tenantmodel.Tenant
	accounts    map[uint]*iammodel.Account
	users       map[uint]*iammodel.User
	departments map[uint]*iammodel.Department
	roles       map[uint]*iammodel.Role
	groups      map[uint]*iammodel.Group
	memberDepts [][2]uint // (memberID, departmentID)
	userRoles   [][2]uint // (memberID, roleID)
	groupRoles  [][2]uint // (groupID, roleID)
	seq         uint      // 各表共用的自增主键分配器
}

type openStoreSnapshot struct {
	tenants     map[uint]*tenantmodel.Tenant
	accounts    map[uint]*iammodel.Account
	users       map[uint]*iammodel.User
	departments map[uint]*iammodel.Department
	roles       map[uint]*iammodel.Role
	groups      map[uint]*iammodel.Group
	memberDepts [][2]uint
	userRoles   [][2]uint
	groupRoles  [][2]uint
	seq         uint
}

func (s *openStore) snapshot() *openStoreSnapshot {
	snap := &openStoreSnapshot{
		tenants:     make(map[uint]*tenantmodel.Tenant, len(s.tenants)),
		accounts:    make(map[uint]*iammodel.Account, len(s.accounts)),
		users:       make(map[uint]*iammodel.User, len(s.users)),
		departments: make(map[uint]*iammodel.Department, len(s.departments)),
		roles:       make(map[uint]*iammodel.Role, len(s.roles)),
		groups:      make(map[uint]*iammodel.Group, len(s.groups)),
		memberDepts: append([][2]uint(nil), s.memberDepts...),
		userRoles:   append([][2]uint(nil), s.userRoles...),
		groupRoles:  append([][2]uint(nil), s.groupRoles...),
		seq:         s.seq,
	}
	for k, v := range s.tenants {
		snap.tenants[k] = v
	}
	for k, v := range s.accounts {
		snap.accounts[k] = v
	}
	for k, v := range s.users {
		snap.users[k] = v
	}
	for k, v := range s.departments {
		snap.departments[k] = v
	}
	for k, v := range s.roles {
		snap.roles[k] = v
	}
	for k, v := range s.groups {
		snap.groups[k] = v
	}
	return snap
}

func (s *openStore) restore(snap *openStoreSnapshot) {
	s.tenants = snap.tenants
	s.accounts = snap.accounts
	s.users = snap.users
	s.departments = snap.departments
	s.roles = snap.roles
	s.groups = snap.groups
	s.memberDepts = snap.memberDepts
	s.userRoles = snap.userRoles
	s.groupRoles = snap.groupRoles
	s.seq = snap.seq
}

func (s *openStore) nextID() uint {
	s.seq++
	return s.seq
}

// openRollbackTx 模拟数据库事务：fn 失败恢复快照（回滚），成功保留（提交）
type openRollbackTx struct {
	store *openStore
}

func (f openRollbackTx) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	snap := f.store.snapshot()
	if err := fn(ctx); err != nil {
		f.store.restore(snap)
		return err
	}
	return nil
}

// ---- 桩仓储：内嵌接口零实现 + 失败注入 ----

type openTenantRepo struct {
	tenantrepository.TenantRepository
	store      *openStore
	failCreate error
}

func (f *openTenantRepo) GetByCode(ctx context.Context, code string) (*tenantmodel.Tenant, error) {
	for _, t := range f.store.tenants {
		if t.Code == code {
			return t, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *openTenantRepo) GetByID(ctx context.Context, id uint) (*tenantmodel.Tenant, error) {
	if tenant, ok := f.store.tenants[id]; ok {
		return tenant, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *openTenantRepo) LockByID(ctx context.Context, id uint) error { return nil }

func (f *openTenantRepo) UpdateOwner(ctx context.Context, tenantID, ownerAccountID uint) error {
	tenant, err := f.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}
	tenant.OwnerAccountId = &ownerAccountID
	return nil
}

func (f *openTenantRepo) Create(ctx context.Context, tenant *tenantmodel.Tenant) (*tenantmodel.Tenant, error) {
	if f.failCreate != nil {
		return nil, f.failCreate
	}
	tenant.ID = f.store.nextID()
	copied := *tenant
	f.store.tenants[tenant.ID] = &copied
	return tenant, nil
}

type openAccountRepo struct {
	iamrepository.AccountRepository
	store      *openStore
	failCreate error
}

func (f *openAccountRepo) GetByID(ctx context.Context, id uint) (*iammodel.Account, error) {
	if a, ok := f.store.accounts[id]; ok {
		return a, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *openAccountRepo) Create(ctx context.Context, account *iammodel.Account) (*iammodel.Account, error) {
	if f.failCreate != nil {
		return nil, f.failCreate
	}
	account.ID = f.store.nextID()
	copied := *account
	f.store.accounts[account.ID] = &copied
	return account, nil
}

type openUserRepo struct {
	iamrepository.UserRepository
	store      *openStore
	failCreate error
}

type openDepartmentRepo struct {
	iamrepository.DepartmentRepository
	store      *openStore
	failCreate error
}

func (f *openDepartmentRepo) Create(ctx context.Context, department *iammodel.Department) (*iammodel.Department, error) {
	if f.failCreate != nil {
		return nil, f.failCreate
	}
	department.ID = f.store.nextID()
	copied := *department
	f.store.departments[department.ID] = &copied
	return department, nil
}

func (f *openDepartmentRepo) SetMemberDepartments(ctx context.Context, member *iammodel.User, departmentIDs []uint) error {
	for _, departmentID := range departmentIDs {
		f.store.memberDepts = append(f.store.memberDepts, [2]uint{member.ID, departmentID})
	}
	return nil
}

func (f *openUserRepo) Create(ctx context.Context, member *iammodel.User) (*iammodel.User, error) {
	if f.failCreate != nil {
		return nil, f.failCreate
	}
	member.ID = f.store.nextID()
	copied := *member
	f.store.users[member.ID] = &copied
	return member, nil
}

func (f *openUserRepo) AddRole(ctx context.Context, role *iammodel.Role, user *iammodel.User) error {
	f.store.userRoles = append(f.store.userRoles, [2]uint{user.ID, role.ID})
	return nil
}

func (f *openUserRepo) GetByAccountAndTenant(ctx context.Context, accountID, tenantID uint) (*iammodel.User, error) {
	for _, member := range f.store.users {
		if member.AccountId == accountID && member.TenantID == tenantID {
			return member, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

type openRBACRepo struct {
	iamrepository.RBACRepository
	store         *openStore
	failOnNthRole int // 第 N 次 Create 失败（1 起算；0 表示不失败）
	roleCreates   int
}

func (f *openRBACRepo) Create(ctx context.Context, role *iammodel.Role) (*iammodel.Role, error) {
	f.roleCreates++
	if f.failOnNthRole > 0 && f.roleCreates == f.failOnNthRole {
		return nil, errors.New("db: insert roles failed")
	}
	role.ID = f.store.nextID()
	copied := *role
	f.store.roles[role.ID] = &copied
	return role, nil
}

func (f *openRBACRepo) GetRoleByName(ctx context.Context, name string) (*iammodel.Role, error) {
	for _, role := range f.store.roles {
		if role.Name == name {
			return role, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

type openGroupRepo struct {
	iamrepository.GroupRepository
	store            *openStore
	failCreateGroups error
	failAddRole      error
}

func (f *openGroupRepo) CreateGroups(ctx context.Context, groups []iammodel.Group, conds ...clause.Expression) error {
	if f.failCreateGroups != nil {
		return f.failCreateGroups
	}
	for i := range groups {
		groups[i].ID = f.store.nextID()
		copied := groups[i]
		f.store.groups[copied.ID] = &copied
	}
	return nil
}

func (f *openGroupRepo) AddRole(ctx context.Context, role *iammodel.Role, group *iammodel.Group) error {
	if f.failAddRole != nil {
		return f.failAddRole
	}
	f.store.groupRoles = append(f.store.groupRoles, [2]uint{group.ID, role.ID})
	return nil
}

// openIAM 聚合桩：满足 tenant/service.IAMRepositories 接口
type openIAM struct {
	account    *openAccountRepo
	user       *openUserRepo
	rbac       *openRBACRepo
	group      *openGroupRepo
	department *openDepartmentRepo
}

func (o *openIAM) Account() iamrepository.AccountRepository       { return o.account }
func (o *openIAM) User() iamrepository.UserRepository             { return o.user }
func (o *openIAM) RBAC() iamrepository.RBACRepository             { return o.rbac }
func (o *openIAM) Group() iamrepository.GroupRepository           { return o.group }
func (o *openIAM) Department() iamrepository.DepartmentRepository { return o.department }

// openAudit 计数桩：验证审计只在事务提交成功后记录
type openAudit struct{ entries int }

func (a *openAudit) Record(context.Context, auditservice.Entry) { a.entries++ }

// openQuota 可配置失败的配额桩
type openQuota struct{ err error }

func (q openQuota) Check(ctx context.Context, tenantID uint, key string) error { return q.err }
func (q openQuota) Usage(ctx context.Context, tenantID uint, key string) (int64, error) {
	return 0, nil
}

// CheckAndReserve 桩：透传错误/直接透传 fn（本测试不覆盖并发语义）
func (q openQuota) CheckAndReserve(ctx context.Context, tenantID uint, key string, fn func(ctx context.Context) error) error {
	if q.err != nil {
		return q.err
	}
	return fn(ctx)
}

// newOpenFixtures 开通租户测试夹具：全空库起步，各桩带失败注入位
func newOpenFixtures() (*openStore, *openTenantRepo, *openIAM, *openAudit) {
	store := &openStore{
		tenants:     map[uint]*tenantmodel.Tenant{},
		accounts:    map[uint]*iammodel.Account{},
		users:       map[uint]*iammodel.User{},
		departments: map[uint]*iammodel.Department{},
		roles:       map[uint]*iammodel.Role{},
		groups:      map[uint]*iammodel.Group{},
	}
	iam := &openIAM{
		account:    &openAccountRepo{store: store},
		user:       &openUserRepo{store: store},
		rbac:       &openRBACRepo{store: store},
		group:      &openGroupRepo{store: store},
		department: &openDepartmentRepo{store: store},
	}
	return store, &openTenantRepo{store: store}, iam, &openAudit{}
}

func openRequest() *OpenTenantRequest {
	return &OpenTenantRequest{
		Code: "acme", Name: "ACME", Plan: tenantmodel.PlanFree,
		OwnerName: "owner-acme", OwnerPassword: "secret123",
	}
}

func newOpenService(store *openStore, tx TxManager, tenantRepo *openTenantRepo, iam *openIAM, audit *openAudit) TenantService {
	return NewTenantService(tx, tenantRepo, iam, openQuota{}, audit, 0)
}

// ---- TX-TENANT-001~005（FIX-020 验收用例）----

// TX-TENANT-001：创建 Tenant 失败，Account/Tenant/Member/Role/Group 不产生脏数据
func TestTXTenant001TenantCreateFailNoResidue(t *testing.T) {
	store, tenantRepo, iam, audit := newOpenFixtures()
	tenantRepo.failCreate = errors.New("db: insert tenants failed")
	svc := newOpenService(store, openRollbackTx{store}, tenantRepo, iam, audit)

	_, err := svc.Open(context.Background(), openRequest())
	assert.Error(t, err)
	assert.Empty(t, store.tenants)
	assert.Empty(t, store.users)
	assert.Empty(t, store.departments)
	assert.Empty(t, store.roles)
	assert.Empty(t, store.groups)
	assert.Empty(t, store.accounts, "新建的 owner 账号必须随事务回滚")
	assert.Zero(t, audit.entries, "失败路径不落审计")
}

// TX-TENANT-002：创建 Owner Member 失败，Tenant 等前序写入全部回滚
func TestTXTenant002MemberCreateFailRollsBackTenant(t *testing.T) {
	store, tenantRepo, iam, audit := newOpenFixtures()
	iam.user.failCreate = errors.New("db: insert users failed")
	svc := newOpenService(store, openRollbackTx{store}, tenantRepo, iam, audit)

	_, err := svc.Open(context.Background(), openRequest())
	assert.Error(t, err)
	assert.Empty(t, store.tenants, "已创建的租户必须回滚")
	assert.Empty(t, store.accounts, "新建的 owner 账号必须回滚")
	assert.Empty(t, store.users)
	assert.Empty(t, store.departments)
	assert.Empty(t, store.roles)
}

// TX-TENANT-003：seedTenantBaseline 创建 Role 失败，前序数据全部回滚
func TestTXTenant003SeedRoleFailRollsBackAll(t *testing.T) {
	store, tenantRepo, iam, audit := newOpenFixtures()
	iam.rbac.failOnNthRole = 2 // 第二个基线角色创建失败
	svc := newOpenService(store, openRollbackTx{store}, tenantRepo, iam, audit)

	_, err := svc.Open(context.Background(), openRequest())
	assert.Error(t, err)
	assert.Empty(t, store.roles)
	assert.Empty(t, store.groups)
	assert.Empty(t, store.users)
	assert.Empty(t, store.departments)
	assert.Empty(t, store.tenants)
	assert.Empty(t, store.accounts)
}

// TX-TENANT-004：Group-Role Binding 失败，所有 Provisioning 数据全部回滚
func TestTXTenant004GroupRoleBindingFailRollsBackAll(t *testing.T) {
	store, tenantRepo, iam, audit := newOpenFixtures()
	iam.group.failAddRole = errors.New("db: insert group_roles failed")
	svc := newOpenService(store, openRollbackTx{store}, tenantRepo, iam, audit)

	_, err := svc.Open(context.Background(), openRequest())
	assert.Error(t, err)
	// 种子已推进到组-角色绑定阶段：此时租户/账号/成员/角色/分组均已写入，
	// 事务回滚后必须全部消失
	assert.Empty(t, store.tenants)
	assert.Empty(t, store.accounts)
	assert.Empty(t, store.users)
	assert.Empty(t, store.departments)
	assert.Empty(t, store.roles)
	assert.Empty(t, store.groups)
	assert.Empty(t, store.groupRoles)
}

// TX-TENANT-005：正常流程成功后，所有基线数据一次性可见且关系完整
func TestTXTenant005HappyPathBaselineComplete(t *testing.T) {
	store, tenantRepo, iam, audit := newOpenFixtures()
	svc := newOpenService(store, openRollbackTx{store}, tenantRepo, iam, audit)

	tenant, err := svc.Open(context.Background(), openRequest())
	assert.NoError(t, err)

	assert.Len(t, store.tenants, 1)
	assert.Len(t, store.accounts, 1, "owner 账号已创建")
	assert.Len(t, store.users, 1, "owner 成员已创建")
	assert.Len(t, store.departments, 1, "租户顶级部门已创建")
	assert.Len(t, store.memberDepts, 1, "owner 已归属顶级部门")
	assert.Len(t, store.roles, 3, "基线角色×3")
	assert.Len(t, store.groups, 3, "系统分组×3")
	assert.Len(t, store.groupRoles, 3, "组-角色绑定×3")
	assert.Len(t, store.userRoles, 1, "owner 绑定 tenant-admin")
	assert.Equal(t, 1, audit.entries, "成功路径落一条审计")

	// 关系完整性：owner 绑定的必须是本租户的 tenant-admin 角色。
	// 回归防护：旧实现按名回查（无租户过滤）会命中其他租户的同名角色
	var memberID uint
	for id := range store.users {
		memberID = id
	}
	bound := store.userRoles[0]
	boundRole := store.roles[bound[1]]
	assert.Equal(t, memberID, bound[0])
	assert.Equal(t, TenantAdminRole, boundRole.Name)
	assert.Equal(t, tenant.ID, boundRole.TenantID, "owner 不得绑定其他租户的角色")
	assert.Contains(t, boundRole.Rules, iammodel.Rule{Resource: iammodel.MemberResource, Operation: iammodel.AllOperation}, "创建者必须拥有成员管理权限")

	for _, role := range store.roles {
		assert.Equal(t, tenant.ID, role.TenantID)
	}
	for _, group := range store.groups {
		assert.Equal(t, tenant.ID, group.TenantID)
	}
	for departmentID, department := range store.departments {
		assert.Equal(t, tenant.ID, department.TenantID)
		assert.Equal(t, tenant.Name, department.Name)
		assert.Nil(t, department.ParentId, "顶级部门不应有父部门")
		assert.Equal(t, [2]uint{memberID, departmentID}, store.memberDepts[0])
	}
}

func TestTransferOwnerAddsMembershipAndTenantAdmin(t *testing.T) {
	store, tenantRepo, iam, audit := newOpenFixtures()
	svc := newOpenService(store, openRollbackTx{store}, tenantRepo, iam, audit)
	tenant, err := svc.Open(context.Background(), openRequest())
	assert.NoError(t, err)

	// 目标账号尚未加入租户，转移流程应在同一事务内补齐成员与管理员角色。
	target := &iammodel.Account{ID: 100, Name: "target", Nickname: "新创建人"}
	store.accounts[target.ID] = target
	assert.NoError(t, svc.TransferOwner(context.Background(), tenant.ID, target.ID))

	assert.Equal(t, target.ID, *store.tenants[tenant.ID].OwnerAccountId)
	member, err := iam.user.GetByAccountAndTenant(context.Background(), target.ID, tenant.ID)
	assert.NoError(t, err)
	assert.Equal(t, "新创建人", member.Nickname)

	var adminRoleID uint
	for id, role := range store.roles {
		if role.Name == TenantAdminRole {
			adminRoleID = id
			break
		}
	}
	assert.Contains(t, store.userRoles, [2]uint{member.ID, adminRoleID})
	assert.Equal(t, 2, audit.entries, "开通与转移均应记录审计")
}
