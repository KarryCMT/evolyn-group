// Package service 内 events.go 维护企业日志的事件注册表（000036）：
// module+resourceType+action → 稳定事件码/日志范围/中文操作名/摘要模板。
// 既有审计调用方不改代码即自动获得展示投影；新业务可用 Entry 的显式字段
// 覆盖推导结果。注册表是事件语义的唯一事实源，企业日志筛选项接口也读这里。
package service

import (
	"sort"
	"strings"
)

// 企业日志日志范围（category_code 稳定编码）：与前端筛选项文案一一对应，
// 只增不改（改码会导致历史行筛选失效）。应用内操作（应用/菜单/表单/流程/
// 数据/应用权限）不进企业日志分类——产品日志与 企业日志查询范围互斥，
// 见 product_events.go 产品分类
const (
	CategoryMemberManagement = "member_management" // 成员管理
	CategoryOrganization     = "organization"      // 组织架构
	CategoryRolePermission   = "role_permission"   // 角色权限
	CategoryTenantSettings   = "tenant_settings"   // 企业设置
	CategoryFileStorage      = "file_storage"      // 文件管理
	CategoryAccountSecurity  = "account_security"  // 账号安全
	CategoryLogExport        = "log_export"        // 日志导出
)

// 分类展示顺序（企业日志筛选项按此排序）与展示名
var categoryCatalog = []struct {
	Code string
	Name string
}{
	{CategoryMemberManagement, "成员管理"},
	{CategoryOrganization, "组织架构"},
	{CategoryRolePermission, "角色权限"},
	{CategoryTenantSettings, "企业设置"},
	{CategoryFileStorage, "文件管理"},
	{CategoryAccountSecurity, "账号安全"},
	{CategoryLogExport, "日志导出"},
}

// categoryNameByCode 分类码 → 展示名（企业 + 产品两目录合并：读取侧按行上
// 的 category_code 展示，无需关心目录归属）
var categoryNameByCode = func() map[string]string {
	m := make(map[string]string, len(categoryCatalog)+len(productCategoryCatalog))
	for _, c := range categoryCatalog {
		m[c.Code] = c.Name
	}
	for _, c := range productCategoryCatalog {
		m[c.Code] = c.Name
	}
	return m
}()

// CategoryName 分类码的中文展示名；未知编码原样返回（防注册表演进后历史
// 行展示空白）
func CategoryName(code string) string {
	if name, ok := categoryNameByCode[code]; ok {
		return name
	}
	return code
}

// EventView 事件出网视图（企业日志筛选项「操作类型」）
type EventView struct {
	Code string `json:"code"` // 稳定事件码，如 iam.member.update
	Name string `json:"name"` // 中文操作名，如「更新成员」
}

// CategoryView 分类出网视图（企业日志筛选项「日志范围」）
type CategoryView struct {
	Code   string      `json:"code"`
	Name   string      `json:"name"`
	Events []EventView `json:"events"`
}

// ResourceMeta 资源事件元数据：module/resourceType 组合 → 分类与资源标签
type ResourceMeta struct {
	Category string // 日志范围码
	Label    string // 资源中文标签，用于事件名与摘要
	// Verbs 资源级动作动词覆盖（key=action）：优先于全局 actionVerbs。
	// 事件名 = 动词 + Label，摘要 = 动词 + Label + 「目标名」；供动词与
	// 资源标签直拼不自然或需与企业日志区分口径的资源定制（如应用创建
	// 用「创建」而非全局「添加」）
	Verbs map[string]string
}

// resourceRegistry 既有审计调用方的资源登记表：键为 "module/resourceType"。
// 新增业务资源时在此登记即可获得展示投影；分类决定归属入口（企业日志/
// 产品日志），两目录查询范围互斥
var resourceRegistry = map[string]ResourceMeta{
	// 成员管理
	"iam/member":                        {CategoryMemberManagement, "成员", nil},
	"iam/member_status":                 {CategoryMemberManagement, "成员状态", nil},
	"iam/member_role":                   {CategoryMemberManagement, "成员角色", nil},
	"iam/member_department":             {CategoryMemberManagement, "成员部门", nil},
	"iam/member_invitation":             {CategoryMemberManagement, "成员邀请", nil},
	"iam/tenant_public_invitation_link": {CategoryMemberManagement, "公开邀请链接", nil},
	"iam/member_field_setting":          {CategoryMemberManagement, "成员信息设置", nil},
	"iam/member_profile":                {CategoryMemberManagement, "成员档案", nil},
	// 账号安全（账号自助：换绑手机号/绑定邮箱等）
	"iam/account": {CategoryAccountSecurity, "账号", nil},
	// 组织架构
	"iam/department":   {CategoryOrganization, "部门", nil},
	"iam/group":        {CategoryOrganization, "分组", nil},
	"iam/group_member": {CategoryOrganization, "分组成员", nil},
	"iam/group_role":   {CategoryOrganization, "分组角色", nil},
	// 角色权限
	"iam/role":        {CategoryRolePermission, "角色", nil},
	"iam/role_group":  {CategoryRolePermission, "角色分组", nil},
	"iam/role_member": {CategoryRolePermission, "角色成员", nil},
	"iam/admin_group": {CategoryRolePermission, "管理组", nil},
	// 企业设置
	"tenant/tenant":                {CategoryTenantSettings, "企业信息", nil},
	"tenantproduct/tenant_product": {CategoryTenantSettings, "产品设置", nil},
	"edition/tenant_subscription":  {CategoryTenantSettings, "版本订阅", nil},
	// 消息中心（通知偏好与自定义提醒对象属于租户级设置；用户查看/已读消息
	// 不记审计，避免高频噪声）
	"notification/notification_setting": {CategoryTenantSettings, "通知设置", nil},
	"notification/custom_recipient":     {CategoryTenantSettings, "提醒对象", nil},
	// 应用管理（产品日志）：应用本体操作，归属产品日志「应用管理」分类
	"application/application": {CategoryProductApplication, "应用", map[string]string{
		"create": "创建", "update": "更新", "delete": "删除",
	}},
	// 应用菜单配置（产品日志）
	"application/application_menu_entry": {CategoryProductApplicationMenu, "菜单", map[string]string{
		"create": "创建", "update": "更新",
	}},
	// 表单资产（产品日志）：表单及表单设计变更
	"form/form": {CategoryProductForm, "表单", map[string]string{
		"create": "创建", "update": "更新", "publish": "发布",
		"delete": "删除", "switch-type": "切换", "copy": "复制",
	}},
	// 流程定义（产品日志）
	"workflow/workflow": {CategoryProductWorkflow, "流程", map[string]string{
		"create": "创建", "update": "更新", "publish": "发布", "delete": "删除",
	}},
	// 表单记录提交（产品日志「应用数据」）：全体成员的日常数据操作
	"form/form_record": {CategoryProductData, "表单数据", map[string]string{
		"submit": "提交",
	}},
	// 表单权限组（表单权限 P1）：应用内权限范围变更，归属产品日志「应用权限」
	"form/form_permission_group": {CategoryProductAppPermission, "表单权限组", nil},
	// 文件管理
	"file/file": {CategoryFileStorage, "文件", nil},
	// 日志导出（日志域自身的导出行为审计，属企业治理）
	"enterpriselog/export": {CategoryLogExport, "日志导出", nil},
	"productlog/export":    {CategoryLogExport, "产品日志导出", nil},
}

// actionVerbs 动作词映射：action → 中文动词（事件名与摘要共用）；
// 资源级覆盖见 ResourceMeta.Verbs
var actionVerbs = map[string]string{
	"create":          "添加",
	"update":          "更新",
	"delete":          "删除",
	"bind":            "绑定",
	"unbind":          "解绑",
	"status":          "变更状态",
	"reorder":         "调整排序",
	"upload_init":     "发起上传",
	"upload_complete": "完成上传",
	"update_scope":    "调整可用范围",
	"update_name":     "修改名称",
	"update_enabled":  "更新启停状态",
	"transfer_owner":  "转移创建人",
	"downgrade":       "降级",
	"change_phone":    "换绑手机号",
	"bind_email":      "绑定邮箱",
	// 产品日志动作（默认动词；资源级覆盖优先）
	"publish":      "发布",
	"update-draft": "更新",
	"switch-type":  "切换",
	"copy":         "复制",
	"submit":       "提交",
	"archive":      "归档",
	"restore":      "恢复",
}

// resourceActions 每个资源实际出现的动作清单（与各域既有调用方对齐）；
// 未登记的 action 仍会落库（事件码机械拼接），只是不进筛选项。
// form/form 的 update-draft 刻意不登记：草稿保存与属性更新共用「更新表单」
// 口径展示，筛选项只保留一个「更新」入口
var resourceActions = map[string][]string{
	"iam/member":                         {"create", "update", "delete", "bind"},
	"iam/member_status":                  {"status"},
	"iam/member_role":                    {"bind", "unbind"},
	"iam/member_department":              {"bind", "unbind"},
	"iam/member_invitation":              {"create", "update"},
	"iam/tenant_public_invitation_link":  {"update"},
	"iam/member_field_setting":           {"update"},
	"iam/member_profile":                 {"update"},
	"iam/account":                        {"change_phone", "bind_email", "update"},
	"iam/department":                     {"create", "update", "delete"},
	"iam/group":                          {"create", "update", "delete"},
	"iam/group_member":                   {"bind", "unbind"},
	"iam/group_role":                     {"bind", "unbind"},
	"iam/role":                           {"create", "update", "delete", "reorder"},
	"iam/role_group":                     {"create", "update", "delete", "reorder"},
	"iam/role_member":                    {"bind", "unbind"},
	"iam/admin_group":                    {"create", "update", "delete"},
	"tenant/tenant":                      {"create", "update", "update_name", "transfer_owner", "status"},
	"tenantproduct/tenant_product":       {"update_enabled", "update_scope"},
	"edition/tenant_subscription":        {"update", "downgrade"},
	"notification/notification_setting":  {"update"},
	"notification/custom_recipient":      {"create", "delete"},
	"application/application":            {"create", "update", "delete"},
	"application/application_menu_entry": {"create", "update"},
	"form/form":                          {"create", "update", "publish", "delete", "switch-type", "copy"},
	"workflow/workflow":                  {"create", "update", "publish", "delete"},
	"form/form_record":                   {"submit"},
	"form/form_permission_group":         {"create", "update", "delete"},
	"file/file":                          {"upload_init", "upload_complete", "delete"},
	"enterpriselog/export":               {"create"},
	"productlog/export":                  {"create"},
}

// EventCodeOf 机械拼接稳定事件码：module.resourceType.action
func EventCodeOf(module, resourceType, action string) string {
	return module + "." + resourceType + "." + action
}

// verbOf 动作动词解析：资源级覆盖（ResourceMeta.Verbs）优先于全局表
func verbOf(meta ResourceMeta, action string) (string, bool) {
	if meta.Verbs != nil {
		if verb, ok := meta.Verbs[action]; ok {
			return verb, true
		}
	}
	verb, ok := actionVerbs[action]
	return verb, ok
}

// EventName 事件码的中文操作名（读取侧展示），如 iam.member.update → 更新成员；
// 未知事件码原样返回
func EventName(code string) string {
	_, _, meta, action, ok := splitEventCode(code)
	if !ok {
		return code
	}
	verb, known := verbOf(meta, action)
	if !known {
		return code
	}
	return verb + meta.Label
}

// ResolveResource 登记表查询：module/resourceType → 资源元数据
func ResolveResource(module, resourceType string) (ResourceMeta, bool) {
	meta, ok := resourceRegistry[module+"/"+resourceType]
	return meta, ok
}

// splitEventCode 拆解事件码为 module/resourceType/action 三段（资源须已登记）
func splitEventCode(code string) (module, resourceType string, meta ResourceMeta, action string, ok bool) {
	parts := strings.Split(code, ".")
	if len(parts) != 3 {
		return "", "", ResourceMeta{}, "", false
	}
	meta, ok = resourceRegistry[parts[0]+"/"+parts[1]]
	if !ok {
		return "", "", ResourceMeta{}, "", false
	}
	return parts[0], parts[1], meta, parts[2], true
}

// CatalogCategories 分类与事件目录（企业日志筛选项接口数据源）：按
// categoryCatalog 顺序输出；事件按资源键排序生成（map 遍历无序，排序保证
// 筛选项目录稳定）
func CatalogCategories() []CategoryView {
	keys := make([]string, 0, len(resourceActions))
	for key := range resourceActions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	categories := make([]CategoryView, 0, len(categoryCatalog))
	for _, c := range categoryCatalog {
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

// KnownEvent 事件码是否已登记（企业日志筛选参数校验用）：资源与动作
// （resourceActions 清单）均命中才算已知，与筛选项目录严格一致
func KnownEvent(code string) bool {
	module, resourceType, _, action, ok := splitEventCode(code)
	if !ok {
		return false
	}
	for _, known := range resourceActions[module+"/"+resourceType] {
		if known == action {
			return true
		}
	}
	return false
}

// KnownCategory 分类码是否已登记（企业日志筛选参数校验用）
func KnownCategory(code string) bool {
	_, ok := categoryNameByCode[code]
	return ok
}

// BuildSummary 生成脱敏操作详情：动词 + 资源标签（+ 目标展示名）。
// 只拼装受控的展示级字段，敏感值不经此函数入摘要
func BuildSummary(module, resourceType, action, targetName string) string {
	meta, ok := resourceRegistry[module+"/"+resourceType]
	if !ok {
		return ""
	}
	verb, ok := verbOf(meta, action)
	if !ok {
		return ""
	}
	if targetName == "" {
		return verb + meta.Label
	}
	return verb + meta.Label + "「" + targetName + "」"
}
