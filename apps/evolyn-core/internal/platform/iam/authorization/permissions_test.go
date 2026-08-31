package authorization

import (
	"testing"

	"evolyn/internal/platform/iam/model"

	"github.com/stretchr/testify/assert"
)

func TestPermissionsOf(t *testing.T) {
	// 空成员：空集
	assert.Empty(t, PermissionsOf(nil))

	user := &model.User{
		Roles: []model.Role{
			{
				Name: "已认证用户",
				Rules: model.Rules{
					{Resource: "users", Operation: model.EditOperation},
					{Resource: "auth", Operation: model.AllOperation},
				},
			},
		},
		Groups: []model.Group{
			{
				Roles: []model.Role{
					{
						Name:  "viewer",
						Rules: model.Rules{{Resource: "forms", Operation: model.ViewOperation}},
					},
				},
			},
		},
	}

	permissions := PermissionsOf(user)

	// edit 展开为具体 CRUD 动词
	assert.True(t, permissions["users:create"])
	assert.True(t, permissions["users:delete"])
	assert.True(t, permissions["users:edit"]) // 聚合键保留
	// view 只含读动词
	assert.True(t, permissions["forms:get"])
	assert.True(t, permissions["forms:list"])
	assert.False(t, permissions["forms:create"])
	// 通配资源
	assert.True(t, permissions["auth:*"])
	assert.True(t, permissions["auth:create"])
	// 未授权资源
	assert.False(t, permissions["tenants:delete"])
}

func TestIsClusterAdminUsesLocalizedRoleName(t *testing.T) {
	user := &model.User{
		ID:    1,
		Roles: []model.Role{{Name: model.ClusterAdminRole}},
	}

	assert.True(t, IsClusterAdmin(user))
}

func TestPermissionsOfMenuActionCodes(t *testing.T) {
	// form-actions 通配规则（租户管理员基线/000047 补授同形）必须展开为
	// 具体按钮动作键：动作码不属于 CRUD 动词集，Contain 展开覆盖不到，
	// 缺失即菜单按钮图全 false（ADR-011 回归）
	user := &model.User{
		ID: 1,
		Roles: []model.Role{
			{Rules: model.Rules{{Resource: model.FormMenuActionResource, Operation: model.AllOperation}}},
		},
	}
	permissions := PermissionsOf(user)
	assert.True(t, permissions["form-actions:*"]) // 通配键本身保留
	assert.True(t, permissions["form-actions:switch-type"])
	assert.True(t, permissions["form-actions:copy-in-app"])
	assert.True(t, permissions["form-actions:copy-cross-app"])
	assert.True(t, permissions["form-actions:hide"])

	// 逐动作授权（自定义角色场景）：透明键即动作键，不放大其他动作
	user = &model.User{
		ID: 2,
		Roles: []model.Role{
			{Rules: model.Rules{{Resource: model.FormMenuActionResource, Operation: "hide"}}},
		},
	}
	permissions = PermissionsOf(user)
	assert.True(t, permissions["form-actions:hide"])
	assert.False(t, permissions["form-actions:switch-type"])
	assert.False(t, permissions["form-actions:copy-in-app"])

	// 全资源通配（*）：按 Authorize 的全资源语义放行全部动作键
	user = &model.User{
		ID: 3,
		Roles: []model.Role{
			{Rules: model.Rules{{Resource: model.All, Operation: model.AllOperation}}},
		},
	}
	permissions = PermissionsOf(user)
	assert.True(t, permissions["form-actions:switch-type"])
	assert.True(t, permissions["form-actions:hide"])

	// 非 form-actions 资源的通配不产出动作键（如 applications:*）
	user = &model.User{
		ID: 4,
		Roles: []model.Role{
			{Rules: model.Rules{{Resource: "applications", Operation: model.AllOperation}}},
		},
	}
	permissions = PermissionsOf(user)
	assert.False(t, permissions["form-actions:switch-type"])
}

// TestPermissionsOfFormDataAdmin 表单权限 P1（设计 §7.1 通配展开定版）：
// 精确授权 form-data:admin、资源通配 form-data:*（AllOperation）、全局通配
// *:*（All + AllOperation）三者等价产出 form-data:admin；form-permissions:*
// 不触发数据面旁路（S3 配置面/数据面分离）。
func TestPermissionsOfFormDataAdmin(t *testing.T) {
	// 精确动作授权
	precise := PermissionsOf(&model.User{ID: 1, Roles: []model.Role{
		{Rules: model.Rules{{Resource: model.FormDataResource, Operation: "admin"}}},
	}})
	assert.True(t, precise["form-data:admin"])

	// 资源通配
	resourceWildcard := PermissionsOf(&model.User{ID: 2, Roles: []model.Role{
		{Rules: model.Rules{{Resource: model.FormDataResource, Operation: model.AllOperation}}},
	}})
	assert.True(t, resourceWildcard["form-data:admin"])
	assert.True(t, resourceWildcard["form-data:*"])

	// 全局通配
	globalWildcard := PermissionsOf(&model.User{ID: 3, Roles: []model.Role{
		{Rules: model.Rules{{Resource: model.All, Operation: model.AllOperation}}},
	}})
	assert.True(t, globalWildcard["form-data:admin"])
	assert.True(t, globalWildcard["form-actions:switch-type"], "全局通配同时展开既有 form-actions 动作键")

	// 配置面全量不产出数据面旁路（S3）
	configOnly := PermissionsOf(&model.User{ID: 4, Roles: []model.Role{
		{Rules: model.Rules{{Resource: model.FormPermissionResource, Operation: model.AllOperation}}},
	}})
	assert.True(t, configOnly["form-permissions:list"])
	assert.True(t, configOnly["form-permissions:delete"])
	assert.False(t, configOnly["form-data:admin"])
}
