// Package model 应用管理域菜单模型（M2-菜单，docs/低代码平台/应用管理/
// 应用菜单接口功能设计方案.md）：应用内资产导航树，分组/表单/仪表盘/页面
// 统一为菜单节点；读取快照出网结构 rootEntryIds + entryMap 与方案 §6.2 对齐
package model

import (
	kernel "evolyn/internal/model"
)

// 菜单节点类型（方案 §5.1 CHECK 约束同口径）：group 分组无资产引用；
// 其余类型节点必须引用同类型资产（资产域落地后由服务层校验归属）
const (
	MenuEntryTypeGroup     = "group"
	MenuEntryTypeForm      = "form"
	MenuEntryTypeDashboard = "dashboard"
	MenuEntryTypePage      = "page"
)

// MenuEntry 应用菜单节点：租户/应用归属由服务层校验回填（查询受租户
// Callback 过滤）；code 为出网 entryId，parent_entry_id 组成树，
// 同父读取序 sort_order ASC, code ASC（tiebreak 与出网编码同源）
type MenuEntry struct {
	ID            uint    `json:"id" gorm:"autoIncrement;primaryKey"`
	TenantID      uint    `json:"tenantId"`                                       // 由租户 Callback 注入/过滤
	ApplicationID uint    `json:"applicationId" gorm:"not null"`                  // 归属应用（服务层校验同租户）
	Code          string  `json:"code" gorm:"size:64;not null"`                   // menu_ 前缀服务端生成编码，出网即 entryId
	ParentEntryID *uint   `json:"parentEntryId" gorm:"index"`                     // 根节点 NULL；父节点须同应用且为 group
	EntryType     string  `json:"entryType" gorm:"size:16;not null"`              // group / form / dashboard / page
	Name          string  `json:"name" gorm:"size:128;not null"`                  // 展示名
	Icon          string  `json:"icon" gorm:"size:32"`                            // 稳定图标键（空串出网投影为 null）
	Color         string  `json:"color" gorm:"size:32"`                           // 稳定颜色键（空串出网投影为 null）
	TargetType    *string `json:"targetType" gorm:"size:16"`                      // 资产引用类型：group 为 NULL（CHECK 约束），非分组等于 EntryType
	TargetID      *uint   `json:"targetId"`                                       // 资产域内部数字主键（出网投影为资产公开编码）
	SortOrder     int64   `json:"sortOrder" gorm:"not null;default:0"`            // 同父排序值，新增 1024 间隔
	Config        Config  `json:"config" gorm:"type:jsonb;not null;default:'{}'"` // 小型显示配置

	kernel.TenantBaseModel
}

func (*MenuEntry) TableName() string { return "application_menu_entries" }

// MenuEntryTarget 出网资产引用：id 为资产域稳定公开编码（由资产查询投影），
// 不是数据库自增主键；资产域未落地前菜单中不出现资产节点
type MenuEntryTarget struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// MenuEntryCapabilities 当前成员对节点的运行时能力（方案 §6.2）：读取时由
// 应用级权限与状态派生，不落库；资产级授权落地后再叠加资产访问结果
type MenuEntryCapabilities struct {
	View     bool `json:"view"`
	Manage   bool `json:"manage"`
	Move     bool `json:"move"`
	Delete   bool `json:"delete"`
	Favorite bool `json:"favorite"` // 个人收藏域未落地前恒 false
}

// MenuFeatures 已注册后端能力的投影（方案 §6.2）：只表达真实存在的
// 能力，流程引擎未接入时 workflow 恒 false，不按菜单形态猜测
type MenuFeatures struct {
	Workflow bool `json:"workflow"`
}

// MenuEntryDetail entryMap 节点出网视图：icon/color 空串投影为 null；
// 非分组节点携带 target；资产能力摘要（features）随资产域落地补充
type MenuEntryDetail struct {
	EntryID       string                `json:"entryId"`
	ParentEntryID *string               `json:"parentEntryId"`
	Type          string                `json:"type"`
	Name          string                `json:"name"`
	Icon          *string               `json:"icon"`
	Color         *string               `json:"color"`
	SortOrder     int64                 `json:"sortOrder"`
	Target        *MenuEntryTarget      `json:"target"`
	Capabilities  MenuEntryCapabilities `json:"capabilities"`
}

// MenuSnapshot 菜单读取接口出网结构（方案 §6.2）：menuRevision 为菜单
// 乐观并发口令（管理写接口回传 baseMenuRevision）；rootEntryIds 为根
// 节点有序编码，entryMap 仅含当前成员可见节点
type MenuSnapshot struct {
	ApplicationCode string                     `json:"applicationCode"`
	MenuRevision    int64                      `json:"menuRevision"`
	RootEntryIDs    []string                   `json:"rootEntryIds"`
	EntryMap        map[string]MenuEntryDetail `json:"entryMap"`
	Features        MenuFeatures               `json:"features"`
}
