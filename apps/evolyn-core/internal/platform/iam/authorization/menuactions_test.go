package authorization

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// adminPerms 租户管理员权限集：URL 门全量 + form-actions 全量。
func adminPerms() map[string]bool {
	return map[string]bool{
		"applications:create": true, "applications:get": true, "applications:list": true,
		"applications:update": true, "applications:patch": true, "applications:delete": true,
		"forms:create": true, "forms:get": true, "forms:list": true,
		"forms:update": true, "forms:patch": true, "forms:delete": true,
		"form-actions:switch-type": true, "form-actions:copy-in-app": true,
		"form-actions:copy-cross-app": true, "form-actions:hide": true,
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
	// 隐藏还需要菜单管理门（applications:patch）
	perms["forms:create"] = true
	delete(perms, "applications:patch")
	assert.False(t, MenuActionsOf(perms, "form")[MenuActionHide])
}

func TestMenuActionsOfInsufficientMember(t *testing.T) {
	// 普通成员（authenticated 基线）：无任何管理/动作授权，动作全 false
	perms := map[string]bool{"applications:get": true, "form-records:create": true}
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
		"applications:patch":       true,
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

func TestMenuActionsOfDashboardNotLanded(t *testing.T) {
	// 仪表盘资产域未落地：动作占位恒 false（与 MenuFeatures.workflow 同口径）
	actions := MenuActionsOf(adminPerms(), "dashboard")
	for code, granted := range actions {
		assert.False(t, granted, "dashboard action %s must stay unlanded", code)
	}
}

func TestMenuActionsOfUnknownAssetType(t *testing.T) {
	// 未登记节点类型（page 等）：空表，投影端收敛为全 false
	assert.Empty(t, MenuActionsOf(adminPerms(), "page"))
}
