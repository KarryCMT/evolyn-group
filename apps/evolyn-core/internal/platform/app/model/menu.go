// Package model 应用管理域菜单模型（M2-菜单，docs/低代码平台/应用管理/
// 应用菜单接口功能设计方案.md）：应用内资产导航树，分组/表单/仪表盘/页面
// 统一为菜单节点；读取快照出网结构 rootMenuIds + nodeMap 与方案 §6.2 对齐
package model

import (
	kernel "evolyn/internal/model"
)

// 菜单节点类型（方案 §5.1 CHECK 约束同口径）：group 分组无资产引用；
// 其余类型节点必须引用同类型资产（资产域落地后由服务层校验归属）
const (
	MenuTypeGroup     = "group"
	MenuTypeForm      = "form"
	MenuTypeDashboard = "dashboard"
	MenuTypePage      = "page"
)

// MenuNode 应用菜单节点：租户/应用归属由服务层校验回填（查询受租户
// Callback 过滤）；code 为出网 menuId，parent_menu_id 组成树，
// 同父读取序 sort_order ASC, code ASC（tiebreak 与出网编码同源）
type MenuNode struct {
	ID           uint    `json:"id" gorm:"autoIncrement;primaryKey"`
	TenantID     uint    `json:"tenantId"`                                       // 由租户 Callback 注入/过滤
	AppID        uint    `json:"appId" gorm:"not null"`                          // 归属应用（服务层校验同租户）
	Code         string  `json:"code" gorm:"size:64;not null"`                   // menu_ 前缀服务端生成编码，出网即 menuId
	ParentMenuID *uint   `json:"parentMenuId" gorm:"index"`                      // 根节点 NULL；父节点须同应用且为 group
	MenuType     string  `json:"menuType" gorm:"size:16;not null"`               // group / form / dashboard / page
	Name         string  `json:"name" gorm:"size:128;not null"`                  // 展示名（资产节点以资产域为事实源，改名经资产接口同步）
	Icon         string  `json:"icon" gorm:"size:32"`                            // 稳定图标键（空串出网投影为 null）
	Color        string  `json:"color" gorm:"size:32"`                           // 稳定颜色键（空串出网投影为 null）
	TargetType   *string `json:"targetType" gorm:"size:16"`                      // 资产引用类型：group 为 NULL（CHECK 约束），非分组等于 MenuType
	TargetID     *uint   `json:"targetId"`                                       // 资产域内部数字主键（出网投影为资产公开编码）
	SortOrder    int64   `json:"sortOrder" gorm:"not null;default:0"`            // 同父排序值，新增 1024 间隔
	Config       Config  `json:"config" gorm:"type:jsonb;not null;default:'{}'"` // 小型显示配置
	Hidden       bool    `json:"hidden" gorm:"not null;default:false"`           // 对成员隐藏（导航隐藏）：普通成员读侧裁剪，菜单管理成员仍可见

	kernel.TenantBaseModel
}

func (*MenuNode) TableName() string { return "tn_app_menu_nodes" }

// MenuFavorite 成员对菜单节点的个人收藏（ADR-011）：个人状态而非授权
// 对象，不参与菜单共享结构与修订号；(member_id, menu_id) 唯一幂等；
// 节点软删时同事务硬删关联行（个人状态无保留价值，不做软删）
type MenuFavorite struct {
	ID        uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	TenantID  uint            `json:"tenantId"`                 // 由租户 Callback 注入/过滤
	MemberID  uint            `json:"memberId" gorm:"not null"` // 收藏成员（Repository 读写一律叠加本列双条件）
	AppID     uint            `json:"appId" gorm:"not null"`    // 节点所属应用
	MenuID    uint            `json:"menuId" gorm:"not null"`   // 收藏的菜单节点
	CreatedAt kernel.JSONTime `json:"createdAt"`
}

func (*MenuFavorite) TableName() string { return "tn_app_menu_favorites" }

// MenuNodeTarget 出网资产引用：code 为资产域稳定公开编码（由资产查询投影），
// 不是数据库自增主键；formType 仅在 form 目标上返回，其他资产类型省略。
type MenuNodeTarget struct {
	Type     string `json:"type"`
	Code     string `json:"code"`
	FormType string `json:"formType,omitempty"`
}

// MenuNodeActions 节点按钮能力（ADR-011）：读时由动作注册表
// （authorization.MenuActionsOf）× 权限集 × 应用状态派生，不落库；
// 未适配当前节点类型的动作恒 false（分组无表单专属动作等）。
// favorite 是个人状态动作不进本结构——凡可见即可收藏，收藏状态经
// MenuNodeDetail.Favorited 出网。
type MenuNodeActions struct {
	Edit          bool `json:"edit"`
	Rename        bool `json:"rename"`
	SwitchType    bool `json:"switchType"`
	ReferenceView bool `json:"referenceView"`
	CopyInApp     bool `json:"copyInApp"`
	CopyCrossApp  bool `json:"copyCrossApp"`
	Move          bool `json:"move"`
	Hide          bool `json:"hide"`
	Delete        bool `json:"delete"`
}

// MenuNodeCapabilities 当前成员对节点的运行时能力（方案 §6.2）：读取时
// 派生，不落库。按钮级动作以 Actions 为唯一事实源（ADR-011：动作注册表 ×
// 权限集 × 应用可编辑状态派生，未适配当前节点类型的动作恒 false）；
// favorite 是个人状态能力（凡可见即可收藏），View 为可见标记（树构建过滤）。
// 旧契约的 manage/move/delete 粗粒度字段已废弃——同一语义由 Actions 的
// rename/move/delete 承载，不再双写。
type MenuNodeCapabilities struct {
	View     bool            `json:"view"`
	Favorite bool            `json:"favorite"`
	Actions  MenuNodeActions `json:"actions"`
}

// MenuFeatures 已注册后端能力的投影（方案 §6.2）：只表达真实存在的
// 能力，流程引擎未接入时 workflow 恒 false，不按菜单形态猜测
type MenuFeatures struct {
	Workflow bool `json:"workflow"`
}

// MenuNodeDetail nodeMap 节点出网视图：icon/color 空串投影为 null；
// 非分组节点携带 target；Favorited 为当前成员的收藏状态（个人状态出网）
type MenuNodeDetail struct {
	MenuID       string               `json:"menuId"`
	ParentMenuID *string              `json:"parentMenuId"`
	Type         string               `json:"type"`
	Name         string               `json:"name"`
	Icon         *string              `json:"icon"`
	Color        *string              `json:"color"`
	SortOrder    int64                `json:"sortOrder"`
	Target       *MenuNodeTarget      `json:"target"`
	Capabilities MenuNodeCapabilities `json:"capabilities"`
	Favorited    bool                 `json:"favorited"`
}

// MenuSnapshot 菜单读取接口出网结构（方案 §6.2）：menuRevision 为菜单
// 乐观并发口令（管理写接口回传 baseMenuRevision）；rootMenuIds 为根
// 节点有序编码，nodeMap 仅含当前成员可见节点
type MenuSnapshot struct {
	AppCode      string                    `json:"appCode"`
	MenuRevision int64                     `json:"menuRevision"`
	RootMenuIDs  []string                  `json:"rootMenuIds"`
	NodeMap     map[string]MenuNodeDetail `json:"nodeMap"`
	Features     MenuFeatures              `json:"features"`
}

// CreateMenuGroupRequest 创建菜单分组请求。parentMenuId 为空时创建根分组；
// baseMenuRevision 是客户端最近一次读取到的菜单修订号，用于阻止陈旧写入。
type CreateMenuGroupRequest struct {
	Name             string  `json:"name" binding:"required" example:"订单管理"`
	ParentMenuID     *string `json:"parentMenuId" example:"menu_0123456789abcdef"`
	BaseMenuRevision int64   `json:"baseMenuRevision" binding:"required,min=1" example:"1"`
}

// MenuGroupMutation 创建分组后的最小增量结果；客户端以 menuRevision 更新
// 乐观锁口令，并重新读取菜单快照获得服务端排序后的完整树。
type MenuGroupMutation struct {
	MenuID       string  `json:"menuId"`
	ParentMenuID *string `json:"parentMenuId"`
	Name         string  `json:"name"`
	MenuRevision int64   `json:"menuRevision"`
}

// UpdateMenuNodeRequest 菜单节点管理更新（PATCH
// /apps/code/:code/menu/nodes/:menuCode，ADR-011）。指针字段
// 区分「未提交」与「提交零值」：name 仅分组可改（资产节点名称以资产域为
// 事实源，经资产接口修改）；hidden 仅资产节点可设（分组可见性由后代派生）；
// parentMenuCode 非空指针即移动节点（null 指针移动到根级），移动节点追加
// 到目标父节点末位（服务端排序，不信任客户端排序值）。baseMenuRevision 为
// 菜单乐观并发口令。
type UpdateMenuNodeRequest struct {
	Name             *string `json:"name"`
	Hidden           *bool   `json:"hidden"`
	ParentMenuCode   *string `json:"parentMenuCode"`
	BaseMenuRevision int64   `json:"baseMenuRevision" binding:"required,min=1" example:"1"`
}

// MenuNodeMutation 节点管理更新后的最小增量结果（口径同 MenuGroupMutation）。
type MenuNodeMutation struct {
	MenuID       string `json:"menuId"`
	MenuRevision int64  `json:"menuRevision"`
}

// CreateMenuFavoriteRequest 收藏菜单节点（POST /menu-favorites）。
type CreateMenuFavoriteRequest struct {
	AppCode  string `json:"appCode" binding:"required" example:"app_6f391a21e65b4d8c"`
	MenuCode string `json:"menuCode" binding:"required" example:"menu_0123456789abcdef"`
}

// MenuFavoriteMutation 收藏/取消收藏结果：Favorited 为操作后的状态
// （重复收藏/取消幂等，返回当前状态而非报错）。
type MenuFavoriteMutation struct {
	MenuID    string `json:"menuId"`
	Favorited bool   `json:"favorited"`
}
