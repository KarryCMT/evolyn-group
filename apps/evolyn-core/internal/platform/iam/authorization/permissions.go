package authorization

import (
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/utils/request"
)

// permissionVerbs 权限布尔集展开的具体动词集（与 Operation.Contain 的判定口径一致）
var permissionVerbs = []string{
	request.CreateOperation,
	request.GetOperation,
	request.ListOperation,
	request.UpdateOperation,
	request.PatchOperation,
	request.DeleteOperation,
}

// menuActionCodes form-actions 资源的按钮动作码（ADR-011）：该资源的操作
// 粒度是动作码而非 CRUD 动词，通配规则展开时随动词一并展开成具体动作键，
// 保证 {form-actions, *} 与逐动作授权在权限集里同形（菜单按钮投影与各域
// Service 的动作复核读的都是具体动作键）
var menuActionCodes = []string{
	string(MenuActionSwitchType),
	string(MenuActionCopyInApp),
	string(MenuActionCopyCrossApp),
	string(MenuActionHide),
}

// PermissionsOf 由成员已有角色（含分组角色）推导权限布尔集，键为 "resource:verb"。
// edit/view 等聚合操作按 Contain 语义展开为具体动词；通配规则产出 "*:verb"。
// 纯函数不触碰仓储，供 /auth/permissions 与前端按钮级控制使用（M1 P2-6）
func PermissionsOf(user *model.User) map[string]bool {
	permissions := make(map[string]bool)
	if user == nil {
		return permissions
	}

	roles := make([]model.Role, 0)
	roles = append(roles, user.Roles...)
	for _, g := range user.Groups {
		roles = append(roles, g.Roles...)
	}

	for _, role := range roles {
		for _, rule := range role.Rules {
			resource := string(rule.Resource)
			// 聚合操作本身作为键保留（前端可直接判断 edit/view 语义）
			permissions[resource+":"+string(rule.Operation)] = true
			for _, verb := range permissionVerbs {
				if rule.Operation.Contain(verb) {
					permissions[resource+":"+verb] = true
				}
			}
			// form-actions 通配规则额外展开为按钮动作键（动作码不属于 CRUD
			// 动词集，上面的 Contain 展开覆盖不到）；资源通配（*）按
			// Authorize 的全资源语义同样放行全部动作键
			if rule.Operation == model.AllOperation &&
				(rule.Resource == model.All || rule.Resource == model.FormMenuActionResource) {
				for _, code := range menuActionCodes {
					permissions[model.FormMenuActionResource+":"+code] = true
				}
			}
		}
	}

	return permissions
}
