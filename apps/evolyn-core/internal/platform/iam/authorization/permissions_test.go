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
