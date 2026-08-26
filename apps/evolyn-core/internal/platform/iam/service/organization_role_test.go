package service

import (
	"context"
	"testing"

	kernel "evolyn/internal/model"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type organizationRoleGroupsRepo struct {
	repository.RoleGroupRepository
	groups map[uint]*model.RoleGroup
}

func (r *organizationRoleGroupsRepo) List(context.Context) ([]model.RoleGroup, error) {
	items := make([]model.RoleGroup, 0, len(r.groups))
	for _, group := range r.groups {
		items = append(items, *group)
	}
	return items, nil
}

func (r *organizationRoleGroupsRepo) GetByID(_ context.Context, id uint) (*model.RoleGroup, error) {
	if group, ok := r.groups[id]; ok {
		return group, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *organizationRoleGroupsRepo) GetByName(_ context.Context, name string) (*model.RoleGroup, error) {
	for _, group := range r.groups {
		if group.Name == name {
			return group, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *organizationRoleGroupsRepo) Create(_ context.Context, group *model.RoleGroup) (*model.RoleGroup, error) {
	group.ID = uint(len(r.groups) + 1)
	r.groups[group.ID] = group
	return group, nil
}

type organizationRolesRepo struct {
	repository.RBACRepository
	roles map[uint]*model.Role
}

func (r *organizationRolesRepo) List(context.Context) ([]model.Role, error) {
	items := make([]model.Role, 0, len(r.roles))
	for _, role := range r.roles {
		items = append(items, *role)
	}
	return items, nil
}

func (r *organizationRolesRepo) GetRoleByID(_ context.Context, id int) (*model.Role, error) {
	if role, ok := r.roles[uint(id)]; ok {
		return role, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *organizationRolesRepo) GetRoleByName(_ context.Context, name string) (*model.Role, error) {
	for _, role := range r.roles {
		if role.Name == name {
			return role, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *organizationRolesRepo) Create(_ context.Context, role *model.Role) (*model.Role, error) {
	role.ID = uint(len(r.roles) + 1)
	r.roles[role.ID] = role
	return role, nil
}

func (r *organizationRolesRepo) Update(_ context.Context, role *model.Role) (*model.Role, error) {
	r.roles[role.ID] = role
	return role, nil
}

type organizationListService struct {
	UserService
	query model.MemberListQuery
}

func (s *organizationListService) ListPage(_ context.Context, query model.MemberListQuery) (*model.MemberPage, error) {
	s.query = query
	return &model.MemberPage{}, nil
}

func TestOrganizationRoleTreeCreatesDefaultGroupAndMapsUngroupedRoles(t *testing.T) {
	roleGroups := &organizationRoleGroupsRepo{groups: map[uint]*model.RoleGroup{}}
	roles := &organizationRolesRepo{roles: map[uint]*model.Role{
		9: {ID: 9, Name: "销售", TenantBaseModel: kernel.TenantBaseModel{TenantID: 1}},
	}}
	svc := NewOrganizationRoleService(passThroughTx{}, roles, roleGroups, nil, &organizationListService{}, nil)

	tree, err := svc.Tree(context.Background())

	if assert.NoError(t, err) && assert.Len(t, tree.Groups, 1) && assert.Len(t, tree.Groups[0].Roles, 1) {
		assert.Equal(t, model.DefaultRoleGroupName, tree.Groups[0].Name)
		assert.EqualValues(t, tree.Groups[0].ID, tree.Groups[0].Roles[0].GroupID)
	}
}

func TestOrganizationRoleListMembersFiltersByRole(t *testing.T) {
	roles := &organizationRolesRepo{roles: map[uint]*model.Role{9: {ID: 9, Name: "销售"}}}
	memberService := &organizationListService{}
	svc := NewOrganizationRoleService(passThroughTx{}, roles, &organizationRoleGroupsRepo{groups: map[uint]*model.RoleGroup{}}, nil, memberService, nil)

	_, err := svc.ListMembers(context.Background(), "9", model.MemberListQuery{Keyword: "张三"})

	assert.NoError(t, err)
	assert.EqualValues(t, 9, memberService.query.RoleID)
	assert.Equal(t, "张三", memberService.query.Keyword)
}
