// 动作资源注册表（表单权限 P1，设计 §7.1 通配展开定版）：资源 → 动作码集
// 的唯一事实源。动作资源不对应任何 URL 首段（中间件 URL 门永不命中，动作
// 授权因此不可能越权放大 URL 访问），动作键由各域 Service 按权限集复核；
// 通配规则展开迭代本注册表，保证 {resource, *} / {*:*} 与逐动作授权在权限
// 集里同形（Authorize 侧动作动词经 Operation.Contain 通用命中，无需特判）。
package authorization

import "evolyn/internal/platform/iam/model"

// formDataActionCodes form-data 资源的动作码：admin 是数据面旁路键（S3，
// 服务层判定专用，不挂 URL 门）——持 form-permissions:* 不获得数据旁路。
var formDataActionCodes = []string{"admin"}

// actionResourceRegistry 动作资源注册表：form-actions 为既有注册项（ADR-011
// 菜单按钮动作），form-data 为表单权限组数据面旁路追加注册项（P1）。
var actionResourceRegistry = map[string][]string{
	model.FormMenuActionResource: menuActionCodes,
	model.FormDataResource:       formDataActionCodes,
}

// expandActionKeys 判断 (resource, AllOperation) 通配规则应展开出的动作键：
// 资源为全局通配（*）时按 Authorize 的全资源语义放行注册表内全部资源的
// 全部动作键；资源精确命中注册表时展开该资源的动作码集。
func expandActionKeys(resource string) map[string][]string {
	expanded := make(map[string][]string)
	if resource == model.All {
		for name, codes := range actionResourceRegistry {
			expanded[name] = codes
		}
		return expanded
	}
	if codes, ok := actionResourceRegistry[resource]; ok {
		expanded[resource] = codes
	}
	return expanded
}
