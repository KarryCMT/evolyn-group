package service

import (
	"context"
	"testing"

	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"
	tenantmodel "evolyn/internal/platform/tenant/model"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ---- 管理组服务桩：内存数据面模拟仓储行为 ----

// adminGroupRepoStub 管理组仓储桩：实现服务路径用到的方法，其余经内嵌
// 接口零实现（调用即 panic，测试立即暴露未预期路径）
type adminGroupRepoStub struct {
	repository.AdminGroupRepository
	nextID      uint
	rows        map[uint]*model.AdminGroup
	members     map[uint][]uint // groupID -> memberIDs
	memberRoles map[uint][]uint // memberID -> roleIDs（内置组代理路径）
	users       map[uint]*model.User
	roleID      uint // tenant-admin 角色 ID
}

func newAdminGroupRepoStub() *adminGroupRepoStub {
	return &adminGroupRepoStub{
		nextID:      100,
		rows:        map[uint]*model.AdminGroup{},
		members:     map[uint][]uint{},
		memberRoles: map[uint][]uint{},
		users:       map[uint]*model.User{},
		roleID:      9,
	}
}

func (r *adminGroupRepoStub) GetByID(_ context.Context, id uint) (*model.AdminGroup, error) {
	if row, ok := r.rows[id]; ok {
		return row, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *adminGroupRepoStub) GetByName(_ context.Context, name string) (*model.AdminGroup, error) {
	for _, row := range r.rows {
		if row.Name == name {
			return row, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *adminGroupRepoStub) ListByTenant(context.Context) ([]model.AdminGroup, error) {
	out := make([]model.AdminGroup, 0, len(r.rows))
	for _, row := range r.rows {
		out = append(out, *row)
	}
	return out, nil
}

func (r *adminGroupRepoStub) Create(_ context.Context, group *model.AdminGroup) (*model.AdminGroup, error) {
	r.nextID++
	group.ID = r.nextID
	copied := *group
	r.rows[group.ID] = &copied
	return group, nil
}

func (r *adminGroupRepoStub) UpdateConfig(_ context.Context, id uint, config model.AdminGroupScopeConfig) error {
	if row, ok := r.rows[id]; ok {
		row.ScopeConfig = config
		return nil
	}
	return gorm.ErrRecordNotFound
}

func (r *adminGroupRepoStub) Rename(_ context.Context, id uint, name string) error {
	if row, ok := r.rows[id]; ok {
		row.Name = name
		return nil
	}
	return gorm.ErrRecordNotFound
}

func (r *adminGroupRepoStub) Delete(_ context.Context, id uint) error {
	delete(r.rows, id)
	return nil
}

func (r *adminGroupRepoStub) ListMemberIDs(_ context.Context, groupID uint) ([]uint, error) {
	return r.members[groupID], nil
}

func (r *adminGroupRepoStub) ReplaceMembers(_ context.Context, groupID, _ uint, memberIDs []uint) error {
	r.members[groupID] = append([]uint(nil), memberIDs...)
	return nil
}

func (r *adminGroupRepoStub) ListGroupIDsOfMember(_ context.Context, memberID uint) ([]uint, error) {
	ids := make([]uint, 0)
	for groupID, memberIDs := range r.members {
		for _, id := range memberIDs {
			if id == memberID {
				ids = append(ids, groupID)
			}
		}
	}
	return ids, nil
}

func (r *adminGroupRepoStub) ListByIDs(_ context.Context, ids []uint) ([]model.AdminGroup, error) {
	out := make([]model.AdminGroup, 0, len(ids))
	for _, id := range ids {
		if row, ok := r.rows[id]; ok {
			out = append(out, *row)
		}
	}
	return out, nil
}

func (r *adminGroupRepoStub) MemberCounts(context.Context) (map[uint]int, error) {
	counts := make(map[uint]int, len(r.members))
	for groupID, memberIDs := range r.members {
		counts[groupID] = len(memberIDs)
	}
	return counts, nil
}

func (r *adminGroupRepoStub) DeleteMembersOfGroup(_ context.Context, groupID uint) error {
	delete(r.members, groupID)
	return nil
}

func (r *adminGroupRepoStub) ResolveBuiltinRoleID(context.Context) (uint, error) {
	return r.roleID, nil
}

func (r *adminGroupRepoStub) ListBuiltinMembers(_ context.Context, roleID uint) ([]model.User, error) {
	users := make([]model.User, 0)
	for memberID, roleIDs := range r.memberRoles {
		for _, id := range roleIDs {
			if id == roleID {
				if user, ok := r.users[memberID]; ok {
					users = append(users, *user)
				}
			}
		}
	}
	return users, nil
}

func (r *adminGroupRepoStub) CountBuiltinMembers(ctx context.Context, roleID uint) (int64, error) {
	users, err := r.ListBuiltinMembers(ctx, roleID)
	if err != nil {
		return 0, err
	}
	return int64(len(users)), nil
}

// adminGroupUserRepoStub 成员仓储桩：GetMemberDetail 按 ID 返回预置成员
// （跨租户/不存在 → NotFound，模拟 Callback 过滤语义）；角色绑定写回
// adminGroupRepoStub.memberRoles，使内置组代理路径的差量增删可见
type adminGroupUserRepoStub struct {
	repository.UserRepository
	users  map[uint]*model.User
	groups *adminGroupRepoStub
}

func (r *adminGroupUserRepoStub) GetMemberDetail(_ context.Context, id uint) (*model.User, error) {
	if user, ok := r.users[id]; ok {
		return user, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *adminGroupUserRepoStub) GetUserByID(_ context.Context, id uint) (*model.User, error) {
	return r.GetMemberDetail(context.Background(), id)
}

func (r *adminGroupUserRepoStub) AddRole(_ context.Context, role *model.Role, user *model.User) error {
	for _, id := range r.groups.memberRoles[user.ID] {
		if id == role.ID {
			return nil
		}
	}
	r.groups.memberRoles[user.ID] = append(r.groups.memberRoles[user.ID], role.ID)
	return nil
}

func (r *adminGroupUserRepoStub) DelRole(_ context.Context, role *model.Role, user *model.User) error {
	next := make([]uint, 0, len(r.groups.memberRoles[user.ID]))
	for _, id := range r.groups.memberRoles[user.ID] {
		if id != role.ID {
			next = append(next, id)
		}
	}
	r.groups.memberRoles[user.ID] = next
	return nil
}

// adminGroupDeptRepoStub 部门仓储桩：List 返回预置部门
type adminGroupDeptRepoStub struct {
	repository.DepartmentRepository
	ids []uint
}

func (r *adminGroupDeptRepoStub) List(context.Context) ([]model.Department, error) {
	out := make([]model.Department, 0, len(r.ids))
	for _, id := range r.ids {
		out = append(out, model.Department{ID: id})
	}
	return out, nil
}

// adminGroupRbacRepoStub 角色仓储桩：List 返回预置角色
type adminGroupRbacRepoStub struct {
	repository.RBACRepository
	ids []uint
}

func (r *adminGroupRbacRepoStub) List(context.Context) ([]model.Role, error) {
	out := make([]model.Role, 0, len(r.ids))
	for _, id := range r.ids {
		out = append(out, model.Role{ID: id})
	}
	return out, nil
}

// adminGroupTenantReaderStub 仅暴露租户创建人，覆盖管理组成员写入的创建人保护规则。
type adminGroupTenantReaderStub struct {
	ownerAccountID *uint
}

func (r adminGroupTenantReaderStub) GetByID(_ context.Context, id uint) (*tenantmodel.Tenant, error) {
	return &tenantmodel.Tenant{ID: id, OwnerAccountId: r.ownerAccountID}, nil
}

func newAdminGroupServiceForTest(groups *adminGroupRepoStub, users map[uint]*model.User) AdminGroupService {
	return newAdminGroupServiceForTestWithTenant(groups, users, nil)
}

func newAdminGroupServiceForTestWithTenant(
	groups *adminGroupRepoStub,
	users map[uint]*model.User,
	tenants AdminGroupTenantReader,
) AdminGroupService {
	groups.users = users
	userStub := &adminGroupUserRepoStub{users: users, groups: groups}
	return NewAdminGroupService(
		passThroughTx{}, groups, userStub,
		&adminGroupDeptRepoStub{ids: []uint{1, 2}},
		&adminGroupRbacRepoStub{ids: []uint{11, 12}},
		nil, tenants, nil,
	)
}

// seedBuiltinStub 预置一个内置系统管理员组（含一名持有 tenant-admin 角色的成员）
func seedBuiltinStub(t *testing.T, groups *adminGroupRepoStub, ownerID uint) uint {
	t.Helper()
	group := &model.AdminGroup{Name: model.AdminGroupBuiltinName, Scope: model.AdminGroupScopeSystem, BuiltIn: true}
	created, err := groups.Create(context.Background(), group)
	assert.NoError(t, err)
	groups.memberRoles[ownerID] = []uint{groups.roleID}
	return created.ID
}

// ---- 创建 ----

func TestAdminGroupCreateValidatesNameAndScope(t *testing.T) {
	groups := newAdminGroupRepoStub()
	svc := newAdminGroupServiceForTest(groups, map[uint]*model.User{})

	// scope 非法
	_, err := svc.Create(tenantCtx(t, 7), &AdminGroupCreateRequest{Scope: "cluster", Name: "测试"})
	assert.ErrorIs(t, err, ErrAdminGroupConfigInvalid)

	// 名称为空 / 超长（>30 字符）
	_, err = svc.Create(tenantCtx(t, 7), &AdminGroupCreateRequest{Scope: model.AdminGroupScopeSystem, Name: "  "})
	assert.ErrorIs(t, err, ErrAdminGroupNameInvalid)
	_, err = svc.Create(tenantCtx(t, 7), &AdminGroupCreateRequest{Scope: model.AdminGroupScopeSystem, Name: string(make([]byte, 0)) + "一二三四五六七八九十一二三四五六七八九十一二三四五六七八九十一"})
	assert.ErrorIs(t, err, ErrAdminGroupNameInvalid)

	// 合法创建：默认配置对齐前端初始态
	detail, err := svc.Create(tenantCtx(t, 7), &AdminGroupCreateRequest{Scope: model.AdminGroupScopeSystem, Name: "测试"})
	assert.NoError(t, err)
	assert.False(t, detail.BuiltIn)
	assert.Equal(t, model.AdminScopePartial, detail.DepartmentMode)
	assert.False(t, detail.DepartmentEnabled)
	assert.NotNil(t, detail.Members)
}

func TestAdminGroupCreateRejectsDuplicateName(t *testing.T) {
	groups := newAdminGroupRepoStub()
	svc := newAdminGroupServiceForTest(groups, map[uint]*model.User{})

	_, err := svc.Create(tenantCtx(t, 7), &AdminGroupCreateRequest{Scope: model.AdminGroupScopeSystem, Name: "测试"})
	assert.NoError(t, err)
	_, err = svc.Create(tenantCtx(t, 7), &AdminGroupCreateRequest{Scope: model.AdminGroupScopeApplication, Name: "测试"})
	assert.ErrorIs(t, err, ErrAdminGroupDuplicateName)
}

// ---- 内置组守卫 ----

func TestAdminGroupBuiltinGuards(t *testing.T) {
	groups := newAdminGroupRepoStub()
	users := map[uint]*model.User{
		1: {ID: 1, Nickname: "张三", Status: model.MemberStatusActive},
		2: {ID: 2, Nickname: "李四", Status: model.MemberStatusActive},
	}
	svc := newAdminGroupServiceForTest(groups, users)
	ctx := tenantCtx(t, 7)
	builtinID := seedBuiltinStub(t, groups, 1)

	// 配置区块 / 改名 / 删除一律拒绝
	_, err := svc.Update(ctx, builtinID, &AdminGroupPatchRequest{
		DepartmentScope: &model.AdminDepartmentScope{Enabled: true, Mode: model.AdminScopeAll},
	})
	assert.ErrorIs(t, err, ErrAdminGroupBuiltinImmutable)

	name := "改名"
	_, err = svc.Update(ctx, builtinID, &AdminGroupPatchRequest{Name: &name})
	assert.ErrorIs(t, err, ErrAdminGroupBuiltinImmutable)

	assert.ErrorIs(t, svc.Delete(ctx, builtinID), ErrAdminGroupBuiltinImmutable)

	// 成员清空守卫：唯一系统管理员不可清空
	_, err = svc.Update(ctx, builtinID, &AdminGroupPatchRequest{Members: &[]uint{}})
	assert.ErrorIs(t, err, ErrAdminGroupLastAdmin)

	// 合法成员替换：详情返回角色绑定推导的成员
	detail, err := svc.Update(ctx, builtinID, &AdminGroupPatchRequest{Members: &[]uint{1, 2}})
	assert.NoError(t, err)
	assert.Len(t, detail.Members, 2)

	// 跨租户/不存在成员：拒绝
	_, err = svc.Update(ctx, builtinID, &AdminGroupPatchRequest{Members: &[]uint{1, 99}})
	assert.ErrorIs(t, err, ErrAdminGroupMemberInvalid)

	// 离职成员：拒绝
	users[3] = &model.User{ID: 3, Nickname: "王五", Status: model.MemberStatusResigned}
	_, err = svc.Update(ctx, builtinID, &AdminGroupPatchRequest{Members: &[]uint{1, 3}})
	assert.ErrorIs(t, err, ErrAdminGroupMemberInvalid)
}

func TestAdminGroupRejectsTenantCreatorInCustomGroup(t *testing.T) {
	groups := newAdminGroupRepoStub()
	ownerAccountID := uint(101)
	users := map[uint]*model.User{
		1: {ID: 1, AccountId: ownerAccountID, Nickname: "创建者", Status: model.MemberStatusActive},
		2: {ID: 2, AccountId: 102, Nickname: "成员", Status: model.MemberStatusActive},
	}
	svc := newAdminGroupServiceForTestWithTenant(
		groups,
		users,
		adminGroupTenantReaderStub{ownerAccountID: &ownerAccountID},
	)
	ctx := tenantCtx(t, 7)

	created, err := svc.Create(ctx, &AdminGroupCreateRequest{Scope: model.AdminGroupScopeSystem, Name: "通讯录组"})
	assert.NoError(t, err)

	// 创建人拥有租户固定所有者权限，不能经自定义管理组重复授予。
	_, err = svc.Update(ctx, created.ID, &AdminGroupPatchRequest{Members: &[]uint{1}})
	assert.ErrorIs(t, err, ErrAdminGroupTenantCreatorNotAllowed)

	// 非创建人仍可正常加入管理组。
	_, err = svc.Update(ctx, created.ID, &AdminGroupPatchRequest{Members: &[]uint{2}})
	assert.NoError(t, err)
}

// ---- 范围区块更新 ----

func TestAdminGroupUpdateScopeBlocks(t *testing.T) {
	groups := newAdminGroupRepoStub()
	svc := newAdminGroupServiceForTest(groups, map[uint]*model.User{})
	ctx := tenantCtx(t, 7)

	created, err := svc.Create(ctx, &AdminGroupCreateRequest{Scope: model.AdminGroupScopeSystem, Name: "通讯录组"})
	assert.NoError(t, err)
	id := created.ID

	// 部门 partial：ID 必须属于本租户（桩预置 1、2）
	_, err = svc.Update(ctx, id, &AdminGroupPatchRequest{
		DepartmentScope: &model.AdminDepartmentScope{Enabled: true, Mode: model.AdminScopePartial, DepartmentIDs: []uint{1, 88}},
	})
	assert.ErrorIs(t, err, ErrAdminGroupConfigInvalid)

	detail, err := svc.Update(ctx, id, &AdminGroupPatchRequest{
		DepartmentScope: &model.AdminDepartmentScope{Enabled: true, Mode: model.AdminScopePartial, DepartmentIDs: []uint{1, 2}},
	})
	assert.NoError(t, err)
	assert.True(t, detail.DepartmentEnabled)
	assert.Equal(t, []uint{1, 2}, detail.DepartmentIDs)

	// mode=all：清单归空（语义由模式表达）
	detail, err = svc.Update(ctx, id, &AdminGroupPatchRequest{
		DepartmentScope: &model.AdminDepartmentScope{Enabled: true, Mode: model.AdminScopeAll, DepartmentIDs: []uint{1}},
	})
	assert.NoError(t, err)
	assert.Empty(t, detail.DepartmentIDs)

	// 角色可管理未开可见：联动拒绝
	_, err = svc.Update(ctx, id, &AdminGroupPatchRequest{
		RoleScope: &model.AdminRoleScope{Visible: false, Manage: true, Mode: model.AdminScopeAll},
	})
	assert.ErrorIs(t, err, ErrAdminGroupConfigInvalid)

	// 角色 partial：ID 校验 + 展开返回
	detail, err = svc.Update(ctx, id, &AdminGroupPatchRequest{
		RoleScope: &model.AdminRoleScope{Visible: true, Manage: true, Mode: model.AdminScopePartial, RoleIDs: []uint{11, 12}},
	})
	assert.NoError(t, err)
	assert.True(t, detail.RoleVisible)
	assert.True(t, detail.RoleManage)
	assert.Equal(t, []uint{11, 12}, detail.RoleIDs)

	// system 组提交应用区块：scope 不符
	_, err = svc.Update(ctx, id, &AdminGroupPatchRequest{
		ApplicationScope: &model.AdminApplicationScope{Manage: true},
	})
	assert.ErrorIs(t, err, ErrAdminGroupScopeMismatch)

	// PATCH 携带多区块：整体替换语义下拒绝
	_, err = svc.Update(ctx, id, &AdminGroupPatchRequest{
		ExternalOrg: &model.AdminExternalOrgScope{Enabled: true},
		RoleScope:   &model.AdminRoleScope{Visible: true, Mode: model.AdminScopeAll},
	})
	assert.ErrorIs(t, err, ErrAdminGroupConfigInvalid)

	// 空区块：拒绝
	_, err = svc.Update(ctx, id, &AdminGroupPatchRequest{})
	assert.ErrorIs(t, err, ErrAdminGroupConfigInvalid)
}

func TestAdminGroupApplicationScopeUpdate(t *testing.T) {
	groups := newAdminGroupRepoStub()
	svc := newAdminGroupServiceForTest(groups, map[uint]*model.User{})
	ctx := tenantCtx(t, 7)

	created, err := svc.Create(ctx, &AdminGroupCreateRequest{Scope: model.AdminGroupScopeApplication, Name: "应用组"})
	assert.NoError(t, err)

	// 全量语义：allApplications=true 清单归空
	detail, err := svc.Update(ctx, created.ID, &AdminGroupPatchRequest{
		ApplicationScope: &model.AdminApplicationScope{AllApplications: true, ApplicationIDs: []uint{5}, Manage: true},
	})
	assert.NoError(t, err)
	assert.True(t, detail.AllApplications)
	assert.True(t, detail.ApplicationManage)
	assert.Empty(t, detail.ApplicationIDs)

	// 通讯录抽屉区块（application 组专属）
	detail, err = svc.Update(ctx, created.ID, &AdminGroupPatchRequest{
		AddressBook: &model.AdminAddressBookScope{DepartmentEnabled: true, RoleVisible: true},
	})
	assert.NoError(t, err)
	assert.NotNil(t, detail.AddressBook)
	assert.True(t, detail.AddressBook.DepartmentEnabled)
}

func TestAdminGroupDeleteCustom(t *testing.T) {
	groups := newAdminGroupRepoStub()
	svc := newAdminGroupServiceForTest(groups, map[uint]*model.User{})
	ctx := tenantCtx(t, 7)

	created, err := svc.Create(ctx, &AdminGroupCreateRequest{Scope: model.AdminGroupScopeSystem, Name: "待删组"})
	assert.NoError(t, err)
	groups.members[created.ID] = []uint{1}

	assert.NoError(t, svc.Delete(ctx, created.ID))
	_, err = svc.Get(ctx, created.ID)
	assert.ErrorIs(t, err, ErrAdminGroupNotFound)
	assert.Empty(t, groups.members[created.ID])
}

func TestAdminGroupListSeedsBuiltinLazily(t *testing.T) {
	groups := newAdminGroupRepoStub()
	groups.users = map[uint]*model.User{1: {ID: 1, Nickname: "张三", Status: model.MemberStatusActive}}
	groups.memberRoles[1] = []uint{groups.roleID}
	svc := newAdminGroupServiceForTest(groups, groups.users)

	// 空库读取：读取侧幂等补种内置组，成员数走角色绑定推导
	summaries, err := svc.List(tenantCtx(t, 7), model.AdminGroupScopeSystem)
	assert.NoError(t, err)
	assert.Len(t, summaries, 1)
	assert.True(t, summaries[0].BuiltIn)
	assert.Equal(t, model.AdminGroupBuiltinName, summaries[0].Name)
	assert.Equal(t, 1, summaries[0].MemberCount)
}

func TestAdminGroupScopesOfMember(t *testing.T) {
	groups := newAdminGroupRepoStub()
	users := map[uint]*model.User{
		1: {ID: 1, Nickname: "张三", Status: model.MemberStatusActive,
			Roles: []model.Role{{ID: 9, Name: model.TenantAdminRoleName}}},
		2: {ID: 2, Nickname: "李四", Status: model.MemberStatusActive},
	}
	svc := newAdminGroupServiceForTest(groups, users)
	ctx := tenantCtx(t, 7)

	created, err := svc.Create(ctx, &AdminGroupCreateRequest{Scope: model.AdminGroupScopeSystem, Name: "通讯录组"})
	assert.NoError(t, err)
	_, err = svc.Update(ctx, created.ID, &AdminGroupPatchRequest{Members: &[]uint{2}})
	assert.NoError(t, err)
	// 系统管理员：systemAdmin=true
	scopes, err := svc.ScopesOfMember(ctx, 1)
	assert.NoError(t, err)
	assert.True(t, scopes.SystemAdmin)
	assert.Empty(t, scopes.Groups)

	// 普通管理组成员：systemAdmin=false + 所属组
	scopes, err = svc.ScopesOfMember(ctx, 2)
	assert.NoError(t, err)
	assert.False(t, scopes.SystemAdmin)
	assert.Len(t, scopes.Groups, 1)
}
