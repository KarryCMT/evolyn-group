package authorization

import (
	"context"
	"testing"

	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"
	"evolyn/internal/utils/request"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ---- 管理组范围裁决（保守门）测试桩 ----

// authzUserRepo 成员仓储桩：按 ID 返回预置成员（含角色），NotFound 模拟
// Callback 租户过滤
type authzUserRepo struct {
	repository.UserRepository
	users map[uint]*model.User
}

func (r *authzUserRepo) GetUserByID(_ context.Context, id uint) (*model.User, error) {
	if user, ok := r.users[id]; ok {
		return user, nil
	}
	return nil, gorm.ErrRecordNotFound
}

// authzGroupRepo 系统组桩：authenticated/unauthenticated 均返回无角色空组，
// 使 RBAC 主判定不产生任何放行，隔离出纯管理组门的测试面
type authzGroupRepo struct {
	repository.GroupRepository
}

func (r *authzGroupRepo) GetGroupByName(_ context.Context, _ string) (*model.Group, error) {
	return &model.Group{Name: model.AuthenticatedGroup}, nil
}

// authzAdminGroupRepo 管理组仓储桩：members 记录 memberID → 组 ID 清单，
// groups 为组配置
type authzAdminGroupRepo struct {
	repository.AdminGroupRepository
	members map[uint][]uint
	groups  map[uint]*model.AdminGroup
}

func (r *authzAdminGroupRepo) ListGroupIDsOfMember(_ context.Context, memberID uint) ([]uint, error) {
	return r.members[memberID], nil
}

func (r *authzAdminGroupRepo) ListByIDs(_ context.Context, ids []uint) ([]model.AdminGroup, error) {
	out := make([]model.AdminGroup, 0, len(ids))
	for _, id := range ids {
		if group, ok := r.groups[id]; ok {
			out = append(out, *group)
		}
	}
	return out, nil
}

func newGateAuthorizer(adminGroups map[uint]*model.AdminGroup, memberships map[uint][]uint, member *model.User) *Authorizer {
	users := map[uint]*model.User{}
	if member != nil {
		users[member.ID] = member
	}
	return NewAuthorizer(
		&authzUserRepo{users: users},
		&authzGroupRepo{},
		&authzAdminGroupRepo{members: memberships, groups: adminGroups},
	)
}

func gateRI(resource, verb string) *request.RequestInfo {
	return &request.RequestInfo{Resource: resource, Verb: verb, IsResourceRequest: true}
}

func TestAuthorizeAdminGroupGateSystemScopes(t *testing.T) {
	ctx := context.Background()
	member := &model.User{ID: 42}
	adminGroups := map[uint]*model.AdminGroup{
		1: {ID: 1, Scope: model.AdminGroupScopeSystem, ScopeConfig: model.AdminGroupScopeConfig{
			Department: &model.AdminDepartmentScope{Enabled: true, Mode: model.AdminScopeAll},
			Role:       &model.AdminRoleScope{Visible: true, Manage: false, Mode: model.AdminScopeAll},
		}},
	}
	authorizer := newGateAuthorizer(adminGroups, map[uint][]uint{42: {1}}, member)

	// 部门全量（可见/可管理同一开关）：成员/部门管理放行
	ok, err := authorizer.Authorize(ctx, member, gateRI(model.MemberResource, request.ListOperation))
	assert.NoError(t, err)
	assert.True(t, ok)
	ok, err = authorizer.Authorize(ctx, member, gateRI("departments", request.DeleteOperation))
	assert.NoError(t, err)
	assert.True(t, ok)

	// 角色可见（读）放行、可管理（写）拒绝
	ok, err = authorizer.Authorize(ctx, member, gateRI(model.RoleResource, request.ListOperation))
	assert.NoError(t, err)
	assert.True(t, ok)
	ok, err = authorizer.Authorize(ctx, member, gateRI(model.RoleResource, request.DeleteOperation))
	assert.NoError(t, err)
	assert.False(t, ok)

	// 管理组资源永不经管理组授予（防自我扩权）
	ok, err = authorizer.Authorize(ctx, member, gateRI(model.AdminGroupResource, request.UpdateOperation))
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestAuthorizeAdminGroupGatePartialDenied(t *testing.T) {
	ctx := context.Background()
	member := &model.User{ID: 42}
	// partial 清单范围：保守门一律拒绝（数据过滤批落地后按资源放开）
	adminGroups := map[uint]*model.AdminGroup{
		1: {ID: 1, Scope: model.AdminGroupScopeSystem, ScopeConfig: model.AdminGroupScopeConfig{
			Department: &model.AdminDepartmentScope{Enabled: true, Mode: model.AdminScopePartial, DepartmentIDs: []uint{1}},
			Role:       &model.AdminRoleScope{Visible: true, Manage: true, Mode: model.AdminScopePartial, RoleIDs: []uint{2}},
		}},
	}
	authorizer := newGateAuthorizer(adminGroups, map[uint][]uint{42: {1}}, member)

	for _, ri := range []*request.RequestInfo{
		gateRI(model.MemberResource, request.ListOperation),
		gateRI("departments", request.UpdateOperation),
		gateRI(model.RoleResource, request.ListOperation),
	} {
		ok, err := authorizer.Authorize(ctx, member, ri)
		assert.NoError(t, err)
		assert.False(t, ok)
	}
}

func TestAuthorizeAdminGroupGateApplicationScopes(t *testing.T) {
	ctx := context.Background()
	member := &model.User{ID: 42}

	t.Run("可添加删除应用但仅部分应用可编辑", func(t *testing.T) {
		adminGroups := map[uint]*model.AdminGroup{
			1: {ID: 1, Scope: model.AdminGroupScopeApplication, ScopeConfig: model.AdminGroupScopeConfig{
				Application: &model.AdminApplicationScope{Manage: true, ApplicationIDs: []uint{5}},
				// application 组的部门/角色是分发范围，不授予部门/角色管理
				Department: &model.AdminDepartmentScope{Enabled: true, Mode: model.AdminScopeAll},
				Role:       &model.AdminRoleScope{Visible: true, Manage: true, Mode: model.AdminScopeAll},
			}},
		}
		authorizer := newGateAuthorizer(adminGroups, map[uint][]uint{42: {1}}, member)

		ok, err := authorizer.Authorize(ctx, member, gateRI("applications", request.CreateOperation))
		assert.NoError(t, err)
		assert.True(t, ok)
		ok, err = authorizer.Authorize(ctx, member, gateRI("applications", request.DeleteOperation))
		assert.NoError(t, err)
		assert.True(t, ok)
		// 非全量应用：编辑动词不放行（具体应用的编辑判定在应用域 evaluator）
		ok, err = authorizer.Authorize(ctx, member, gateRI("applications", request.UpdateOperation))
		assert.NoError(t, err)
		assert.False(t, ok)
		// 分发范围不等于通讯录管理权
		ok, err = authorizer.Authorize(ctx, member, gateRI("departments", request.ListOperation))
		assert.NoError(t, err)
		assert.False(t, ok)
		ok, err = authorizer.Authorize(ctx, member, gateRI(model.RoleResource, request.ListOperation))
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("全部应用可编辑", func(t *testing.T) {
		adminGroups := map[uint]*model.AdminGroup{
			1: {ID: 1, Scope: model.AdminGroupScopeApplication, ScopeConfig: model.AdminGroupScopeConfig{
				Application: &model.AdminApplicationScope{AllApplications: true},
			}},
		}
		authorizer := newGateAuthorizer(adminGroups, map[uint][]uint{42: {1}}, member)

		ok, err := authorizer.Authorize(ctx, member, gateRI("applications", request.UpdateOperation))
		assert.NoError(t, err)
		assert.True(t, ok)
		// Manage 未开：增删拒绝
		ok, err = authorizer.Authorize(ctx, member, gateRI("applications", request.CreateOperation))
		assert.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestAuthorizeAdminGroupGateUnion(t *testing.T) {
	ctx := context.Background()
	member := &model.User{ID: 42}
	// 多组并集：A 组部门全量、B 组角色可管理全量 → 成员同时具备两类能力
	adminGroups := map[uint]*model.AdminGroup{
		1: {ID: 1, Scope: model.AdminGroupScopeSystem, ScopeConfig: model.AdminGroupScopeConfig{
			Department: &model.AdminDepartmentScope{Enabled: true, Mode: model.AdminScopeAll},
		}},
		2: {ID: 2, Scope: model.AdminGroupScopeSystem, ScopeConfig: model.AdminGroupScopeConfig{
			Role: &model.AdminRoleScope{Visible: true, Manage: true, Mode: model.AdminScopeAll},
		}},
	}
	authorizer := newGateAuthorizer(adminGroups, map[uint][]uint{42: {1, 2}}, member)

	for _, ri := range []*request.RequestInfo{
		gateRI(model.MemberResource, request.ListOperation),
		gateRI(model.RoleResource, request.DeleteOperation),
	} {
		ok, err := authorizer.Authorize(ctx, member, ri)
		assert.NoError(t, err)
		assert.True(t, ok)
	}
}

func TestAuthorizeAdminGroupGateSkipsBuiltin(t *testing.T) {
	ctx := context.Background()
	member := &model.User{ID: 42}
	// 内置组不参与门判定：其能力应由 tenant-admin 角色 RBAC 覆盖，
	// 空配置内置组在门层不产生任何放行
	adminGroups := map[uint]*model.AdminGroup{
		1: {ID: 1, BuiltIn: true, Scope: model.AdminGroupScopeSystem},
	}
	authorizer := newGateAuthorizer(adminGroups, map[uint][]uint{42: {1}}, member)

	ok, err := authorizer.Authorize(ctx, member, gateRI(model.MemberResource, request.ListOperation))
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestIsTenantAdmin(t *testing.T) {
	assert.False(t, IsTenantAdmin(nil))
	assert.False(t, IsTenantAdmin(&model.User{ID: 1}))
	assert.True(t, IsTenantAdmin(&model.User{
		ID:    1,
		Roles: []model.Role{{Name: model.TenantAdminRoleName}},
	}))
	assert.True(t, IsTenantAdmin(&model.User{
		ID:     1,
		Groups: []model.Group{{Roles: []model.Role{{Name: model.TenantAdminRoleName}}}},
	}))
}
