// Package authorization 菜单按钮动作注册表（ADR-011）：应用菜单右键按钮
// 的授权词汇与派生规则。动作不是 URL 资源——form-actions 资源不对应任何
// 路由首段，中间件 URL 门永不命中，动作裁决由各域 Service 按权限集复核，
// 菜单读取时经 MenuActionsOf 投影按钮能力（读时派生，不落库）。
package authorization

// MenuAction 菜单节点动作码（稳定公开词汇，前端按钮与后端裁决共用）。
// 同一动作在不同节点类型上的授权键可能不同（如 rename：分组挂
// applications:patch，表单挂 forms:patch），以 MenuActionsForAsset 为准。
type MenuAction string

const (
	// MenuActionEdit 编辑（表单打开设计器/仪表盘打开搭建器）
	MenuActionEdit MenuAction = "edit"
	// MenuActionRename 修改名称（表单节点含图标/颜色，经资产接口修改）
	MenuActionRename MenuAction = "rename"
	// MenuActionSwitchType 切换表单类型（standard↔workflow，流程数据保留）
	MenuActionSwitchType MenuAction = "switch-type"
	// MenuActionReferenceView 查看引用视图（表单被哪些应用菜单引用）
	MenuActionReferenceView MenuAction = "reference-view"
	// MenuActionCopyInApp 复制到当前应用（仪表盘的「复制」同挂本动作码）
	MenuActionCopyInApp MenuAction = "copy-in-app"
	// MenuActionCopyCrossApp 复制到其他应用（需目标应用可创建表单）
	MenuActionCopyCrossApp MenuAction = "copy-cross-app"
	// MenuActionMove 移动节点（换父分组/根级）
	MenuActionMove MenuAction = "move"
	// MenuActionHide 对成员隐藏（导航隐藏：普通成员读侧裁剪，不做访问控制）
	MenuActionHide MenuAction = "hide"
	// MenuActionDelete 删除
	MenuActionDelete MenuAction = "delete"
)

// MenuActionSpec 单个节点类型上单个动作的授权元数据。
type MenuActionSpec struct {
	Code MenuAction
	// Grants 授权键列表（AND 语义：全部命中才视为持有）。除动作授权键
	// （form-actions:*）外一并声明对应的 URL 门权限键，保证「按钮不撒谎」
	// ——前端按投影渲染的按钮不会被中间件 403。
	Grants []string
	// Landed 后端端点是否已落地；未落地动作按 false 投影（与 MenuFeatures
	// 「只表达真实存在的能力」同口径，不以按钮集合猜测未实现接口）
	Landed bool
}

// 节点类型字面量与 application_menu_entries.entry_type 对齐（form 节点的
// target.formType 只影响「切换类型」的展示文案，不影响授权键，注册表按
// entry_type 建档即可）。仪表盘资产域未落地，动作先以 Landed=false 占位，
// 避免仪表盘节点出现后投影出死按钮。
const (
	menuAssetTypeGroup     = "group"
	menuAssetTypeForm      = "form"
	menuAssetTypeDashboard = "dashboard"
)

// menuActionRegistry 动作注册表（唯一事实源）：节点类型 → 动作授权表。
// 收藏是个人状态动作不进注册表——凡节点可见即可收藏（menu-favorites 资源
// 授全体成员），favorite 能力恒随 view 出网。
var menuActionRegistry = map[string][]MenuActionSpec{
	menuAssetTypeGroup: {
		{Code: MenuActionRename, Grants: []string{"applications:patch"}, Landed: true},
		{Code: MenuActionMove, Grants: []string{"applications:patch"}, Landed: true},
		{Code: MenuActionDelete, Grants: []string{"applications:delete"}, Landed: true},
	},
	menuAssetTypeForm: {
		{Code: MenuActionEdit, Grants: []string{"forms:update"}, Landed: true},
		{Code: MenuActionRename, Grants: []string{"forms:patch"}, Landed: true},
		// 切换类型路由 POST /forms/:code/switch-type，URL 门复用 create 动词
		//（与发布同口径），动作授权键独立控制
		{Code: MenuActionSwitchType, Grants: []string{"form-actions:switch-type", "forms:create"}, Landed: true},
		{Code: MenuActionReferenceView, Grants: []string{"forms:get"}, Landed: true},
		// 复制路由 POST /forms/:code/copy，URL 门同 create；目标应用在
		// Service 侧另核应用状态与配额
		{Code: MenuActionCopyInApp, Grants: []string{"form-actions:copy-in-app", "forms:create"}, Landed: true},
		{Code: MenuActionCopyCrossApp, Grants: []string{"form-actions:copy-cross-app", "forms:create"}, Landed: true},
		{Code: MenuActionMove, Grants: []string{"applications:patch"}, Landed: true},
		// 隐藏开关走菜单节点 PATCH（applications:patch 门），动作授权键独立控制
		{Code: MenuActionHide, Grants: []string{"form-actions:hide", "applications:patch"}, Landed: true},
		{Code: MenuActionDelete, Grants: []string{"forms:delete"}, Landed: true},
	},
	menuAssetTypeDashboard: {
		{Code: MenuActionEdit, Grants: nil, Landed: false},
		{Code: MenuActionRename, Grants: nil, Landed: false},
		{Code: MenuActionCopyInApp, Grants: nil, Landed: false},
		{Code: MenuActionMove, Grants: nil, Landed: false},
		{Code: MenuActionDelete, Grants: nil, Landed: false},
	},
}

// MenuActionsForAsset 返回指定节点类型的动作授权表；未登记类型返回空表
// （page 等后续资产类型接入时在此扩展，投影端天然收敛为全 false）。
func MenuActionsForAsset(assetType string) []MenuActionSpec {
	return menuActionRegistry[assetType]
}

// MenuActionsOf 按「注册表 × 权限集」求值：动作码 → 是否授权。未落地动作
// 恒 false；Grants 任一缺失即 false（AND 语义）。应用可编辑状态等与授权
// 无关的运行时因子由调用方（菜单投影/各域 Service）另行叠加。
func MenuActionsOf(perms map[string]bool, assetType string) map[MenuAction]bool {
	result := make(map[MenuAction]bool)
	for _, spec := range menuActionRegistry[assetType] {
		granted := spec.Landed && len(spec.Grants) > 0
		for _, key := range spec.Grants {
			if !perms[key] {
				granted = false
				break
			}
		}
		result[spec.Code] = granted
	}
	return result
}
