// Package service 内 product_events.go 维护产品日志的事件目录（000064，
// docs/低代码平台/产品日志/）：产品日志与企业日志共用 tn_audit_logs 事实表
// 与事件注册表（events.go），但查询范围按 category_code 互斥——产品日志
// 只覆盖「当前租户内各应用及其应用内资源操作」，登录与企业治理行为归企业
// 日志。本文件是产品分类的唯一事实源，产品日志筛选项接口与参数校验读这里。
package service

import (
	"sort"
	"strings"
)

// 产品日志日志范围（category_code 稳定编码）：与产品日志页筛选项文案一一
// 对应，只增不改（改码会导致历史行筛选失效）。application 与企业日志历史
// 分类同名——应用本体操作自 000064 起归产品日志，企业日志目录不再含该分类
const (
	CategoryProductApplication     = "application"      // 应用管理
	CategoryProductApplicationMenu = "application_menu" // 菜单配置
	CategoryProductForm            = "form"             // 表单管理
	CategoryProductWorkflow        = "workflow"         // 流程管理
	CategoryProductData            = "data"             // 应用数据
	CategoryProductAppPermission   = "app_permission"   // 应用权限
)

// 产品分类展示顺序（产品日志筛选项按此排序）与展示名
var productCategoryCatalog = []struct {
	Code string
	Name string
}{
	{CategoryProductApplication, "应用管理"},
	{CategoryProductApplicationMenu, "菜单配置"},
	{CategoryProductForm, "表单管理"},
	{CategoryProductWorkflow, "流程管理"},
	{CategoryProductData, "应用数据"},
	{CategoryProductAppPermission, "应用权限"},
}

// productCategoryCodes 产品分类码全集（快照缓存）：产品日志查询白名单与
// 企业日志查询排除名单共用
var productCategoryCodes = func() map[string]struct{} {
	codes := make(map[string]struct{}, len(productCategoryCatalog))
	for _, c := range productCategoryCatalog {
		codes[c.Code] = struct{}{}
	}
	return codes
}()

// IsProductCategory 分类码是否归属产品日志目录（企业日志排除名单判据）
func IsProductCategory(code string) bool {
	_, ok := productCategoryCodes[code]
	return ok
}

// ProductCategoryCodes 产品分类码列表（企业日志查询的排除名单）
func ProductCategoryCodes() []string {
	codes := make([]string, 0, len(productCategoryCatalog))
	for _, c := range productCategoryCatalog {
		codes = append(codes, c.Code)
	}
	return codes
}

// KnownProductCategory 分类码是否已登记产品日志目录（产品日志筛选参数校验用）
func KnownProductCategory(code string) bool {
	return IsProductCategory(code)
}

// KnownProductEvent 事件码是否已登记产品日志目录（产品日志筛选参数校验用）：
// 资源归属产品分类且动作在 resourceActions 清单内，与筛选项目录严格一致
func KnownProductEvent(code string) bool {
	module, resourceType, meta, action, ok := splitEventCode(code)
	if !ok || !IsProductCategory(meta.Category) {
		return false
	}
	for _, known := range resourceActions[module+"/"+resourceType] {
		if known == action {
			return true
		}
	}
	return false
}

// CatalogProductCategories 产品日志分类与事件目录（筛选项接口数据源）：
// 按 productCategoryCatalog 顺序输出；事件按资源键排序生成（目录稳定）
func CatalogProductCategories() []CategoryView {
	keys := make([]string, 0, len(resourceActions))
	for key := range resourceActions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	categories := make([]CategoryView, 0, len(productCategoryCatalog))
	for _, c := range productCategoryCatalog {
		view := CategoryView{Code: c.Code, Name: c.Name, Events: []EventView{}}
		for _, key := range keys {
			meta, ok := resourceRegistry[key]
			if !ok || meta.Category != c.Code {
				continue
			}
			parts := strings.SplitN(key, "/", 2)
			for _, action := range resourceActions[key] {
				code := EventCodeOf(parts[0], parts[1], action)
				view.Events = append(view.Events, EventView{Code: code, Name: EventName(code)})
			}
		}
		categories = append(categories, view)
	}
	return categories
}
