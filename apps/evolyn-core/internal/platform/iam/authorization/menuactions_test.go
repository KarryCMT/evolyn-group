package authorization

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// adminPerms 租户管理员权限集：URL 门全量 + 菜单动作全量。
func adminPerms() map[string]bool {
	return map[string]bool{
		"apps:create": true, "apps:get": true, "apps:list": true,
		"apps:update": true, "apps:patch": true, "apps:delete": true,
		"forms:create": true, "forms:get": true, "forms:list": true,
		"forms:update": true, "forms:patch": true, "forms:delete": true,
		"form-actions:switch-type": true, "form-actions:copy-in-app": true,
		"form-actions:copy-cross-app": true, "form-actions:hide": true,
		"dashboards:create": true, "dashboards:get": true, "dashboards:list": true,
		"dashboards:update": true, "dashboards:patch": true, "dashboards:delete": true,
		"dashboard-actions:design": true, "dashboard-actions:copy": true,
	}
}

func TestMenuActionsOfAdminFormNode(t *testing.T) {
	actions := MenuActionsOf(adminPerms(), "form")
	assert.True(t, actions[MenuActionEdit])
	assert.True(t, actions[MenuActionRename])
	assert.True(t, actions[MenuActionSwitchType])
	assert.True(t, actions[MenuActionReferenceView])
	assert.True(t, actions[MenuActionCopyInApp])
	assert.True(t, actions[MenuActionCopyCrossApp])
	assert.True(t, actions[MenuActionMove])
	assert.True(t, actions[MenuActionHide])
	assert.True(t, actions[MenuActionDelete])
}

func TestMenuActionsOfAdminGroupNode(t *testing.T) {
	actions := MenuActionsOf(adminPerms(), "group")
	assert.True(t, actions[MenuActionRename])
	assert.True(t, actions[MenuActionMove])
	assert.True(t, actions[MenuActionDelete])
	// 表单专属动作对分组不投影
	assert.False(t, actions[MenuActionEdit])
	assert.False(t, actions[MenuActionSwitchType])
	assert.False(t, actions[MenuActionHide])
}

func TestMenuActionsOfGrantsAreAND(t *testing.T) {
	// 只授动作键缺 URL 门键（forms:create）时，切换类型/复制不得投影，
	// 保证按钮不会被中间件 403（「按钮不撒谎」）
	perms := adminPerms()
	delete(perms, "forms:create")
	actions := MenuActionsOf(perms, "form")
	assert.False(t, actions[MenuActionSwitchType])
	assert.False(t, actions[MenuActionCopyInApp])
	assert.False(t, actions[MenuActionCopyCrossApp])
	// 隐藏还需要菜单管理门（apps:patch）
	perms["forms:create"] = true
	delete(perms, "apps:patch")
	assert.False(t, MenuActionsOf(perms, "form")[MenuActionHide])
}

func TestMenuActionsOfInsufficientMember(t *testing.T) {
	// 普通成员（authenticated 基线）：无任何管理/动作授权，动作全 false
	perms := map[string]bool{"apps:get": true, "form-records:create": true}
	for _, assetType := range []string{"group", "form"} {
		for code, granted := range MenuActionsOf(perms, assetType) {
			assert.False(t, granted, "action %s on %s should be denied", code, assetType)
		}
	}
}

func TestMenuActionsOfPartialGrants(t *testing.T) {
	// 只授予切换类型与隐藏动作（自定义角色场景）：其余动作保持拒绝
	perms := map[string]bool{
		"forms:create":             true,
		"apps:patch":               true,
		"form-actions:switch-type": true,
		"form-actions:hide":        true,
	}
	actions := MenuActionsOf(perms, "form")
	assert.True(t, actions[MenuActionSwitchType])
	assert.True(t, actions[MenuActionHide])
	assert.False(t, actions[MenuActionEdit])
	assert.False(t, actions[MenuActionCopyInApp])
	assert.False(t, actions[MenuActionCopyCrossApp])
	assert.False(t, actions[MenuActionDelete])
}

func TestMenuActionsOfDashboard(t *testing.T) {
	// 仪表盘菜单动作均已接通真实端点，按钮投影不得继续隐藏复制。
	actions := MenuActionsOf(adminPerms(), "dashboard")
	assert.True(t, actions[MenuActionEdit])
	assert.True(t, actions[MenuActionRename])
	assert.True(t, actions[MenuActionMove])
	assert.True(t, actions[MenuActionDelete])
	assert.True(t, actions[MenuActionCopyInApp])
}

func TestMenuActionsOfUnknownAssetType(t *testing.T) {
	// 未登记节点类型（page 等）：空表，投影端收敛为全 false
	assert.Empty(t, MenuActionsOf(adminPerms(), "page"))
}
