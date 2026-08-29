// Phase 3 Resolver 单测：角色/表单用户字段/部门负责人解析与空集语义。
package assignment

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"evolyn/internal/engine/workflow/model"
)

// fakeOrg OrganizationProvider 桩。
type fakeOrg struct {
	roleMembers   []model.Actor
	deptManagers  []model.Actor
	starterMgrErr bool
}

func (f *fakeOrg) ResolveRoleMembers(ctx context.Context, tenantID uint, roleCode string) ([]model.Actor, error) {
	return f.roleMembers, nil
}

func (f *fakeOrg) ResolveDepartmentMembers(ctx context.Context, tenantID, deptID uint) ([]model.Actor, error) {
	return nil, nil
}

func (f *fakeOrg) ResolveDepartmentManager(ctx context.Context, tenantID, deptID uint) ([]model.Actor, error) {
	return f.deptManagers, nil
}

func (f *fakeOrg) ResolveStarterManager(ctx context.Context, tenantID, starterMemberID uint) ([]model.Actor, error) {
	if f.starterMgrErr {
		return nil, errors.New("not supported")
	}
	return nil, nil
}

func (f *fakeOrg) MemberDisplayName(ctx context.Context, tenantID, memberID uint) string {
	return "成员"
}

func resolveCtx() *model.WorkflowContext {
	return &model.WorkflowContext{TenantID: 1, Form: map[string]any{}}
}

func TestRoleResolver(t *testing.T) {
	resolver := &RoleResolver{org: &fakeOrg{roleMembers: []model.Actor{{MemberID: 7}}}}
	actors, err := resolver.Resolve(context.Background(), ResolveInput{
		Ctx:  resolveCtx(),
		Spec: model.AssigneeSpec{Type: model.AssigneeTypeRole, RoleCode: "finance"},
	})
	require.NoError(t, err)
	require.Len(t, actors, 1)
	assert.Equal(t, uint(7), actors[0].MemberID)

	// 角色下无有效成员 → 稳定错误（禁止静默跳过节点）
	_, err = (&RoleResolver{org: &fakeOrg{}}).Resolve(context.Background(), ResolveInput{
		Ctx:  resolveCtx(),
		Spec: model.AssigneeSpec{Type: model.AssigneeTypeRole, RoleCode: "empty"},
	})
	var notFound *ErrAssigneeNotFound
	require.ErrorAs(t, err, &notFound)
}

func TestFormFieldResolver(t *testing.T) {
	resolver := &FormFieldResolver{}

	cases := []struct {
		name    string
		value   any
		members []uint
		empty   bool
	}{
		{"数字", float64(3), []uint{3}, false},
		{"数字字符串", "5", []uint{5}, false},
		{"数组", []any{float64(3), "5"}, []uint{3, 5}, false},
		{"无值", nil, nil, true},
		{"非法字符串", "boss", nil, true},
		{"负数", float64(-1), nil, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := resolveCtx()
			ctx.Form["manager"] = c.value
			actors, err := resolver.Resolve(context.Background(), ResolveInput{
				Ctx:  ctx,
				Spec: model.AssigneeSpec{Type: model.AssigneeTypeFormField, FormField: "manager"},
			})
			if c.empty {
				var notFound *ErrAssigneeNotFound
				require.ErrorAs(t, err, &notFound)
				return
			}
			require.NoError(t, err)
			require.Len(t, actors, len(c.members))
			for i, member := range c.members {
				assert.Equal(t, member, actors[i].MemberID)
			}
		})
	}
}

func TestDepartmentManagerResolver(t *testing.T) {
	resolver := &DepartmentManagerResolver{org: &fakeOrg{deptManagers: []model.Actor{{MemberID: 9}}}}
	actors, err := resolver.Resolve(context.Background(), ResolveInput{
		Ctx:  resolveCtx(),
		Spec: model.AssigneeSpec{Type: model.AssigneeTypeDepartmentManager, DeptID: 2},
	})
	require.NoError(t, err)
	require.Len(t, actors, 1)
	assert.Equal(t, uint(9), actors[0].MemberID)

	// 部门未设置负责人 → 稳定错误
	_, err = (&DepartmentManagerResolver{org: &fakeOrg{}}).Resolve(context.Background(), ResolveInput{
		Ctx:  resolveCtx(),
		Spec: model.AssigneeSpec{Type: model.AssigneeTypeDepartmentManager, DeptID: 2},
	})
	var notFound *ErrAssigneeNotFound
	require.ErrorAs(t, err, &notFound)
}

func TestRegistryRegisteredTypes(t *testing.T) {
	reg := NewRegistry(nil, &fakeOrg{})
	for _, typ := range []model.AssigneeType{
		model.AssigneeTypeUser, model.AssigneeTypeRole,
		model.AssigneeTypeFormField, model.AssigneeTypeDepartmentManager,
	} {
		_, ok := reg.ResolverOf(typ)
		assert.True(t, ok, "type %s 应已注册", typ)
		assert.True(t, ResolverEnabled(typ), "type %s 应已启用", typ)
	}
	for _, typ := range []model.AssigneeType{model.AssigneeTypeDepartment, model.AssigneeTypeStarterManager} {
		_, ok := reg.ResolverOf(typ)
		assert.False(t, ok, "type %s 不应注册", typ)
		assert.False(t, ResolverEnabled(typ), "type %s 不应启用", typ)
	}
}
