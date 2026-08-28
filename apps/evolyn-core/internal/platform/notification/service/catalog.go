package service

import (
	"fmt"
	"strings"

	"evolyn/internal/platform/notification/model"
)

// 渠道与分类稳定编码（历史数据与前端路由的契约，只增不改）
const (
	ChannelSystem = "system"
	ChannelEmail  = "email"
	ChannelSMS    = "sms"

	CategoryGroupProduct    = "product"    // 灵衍云分组
	CategoryGroupEnterprise = "enterprise" // 企业消息分组
)

// 分类码（八个稳定分类，只增不改；展示名与分组经设置接口下发）
const (
	CategoryDataReminder       = "data-reminder"
	CategoryAppLog             = "app-log"
	CategoryDocumentActivity   = "document-activity"
	CategoryUsageReminder      = "usage-reminder"
	CategoryContactsManagement = "contacts-management"
	CategoryOpenPlatform       = "open-platform"
	CategorySystemManagement   = "system-management"
	CategoryOperationNotice    = "operation-notice"
)

// CategoryDef 分类目录项：Configurable=false 表示尚无事件注册（如数据提醒/
// 文档动态依赖后续低代码引擎/文档协作域），仍出现在收件箱分类目录中。
type CategoryDef struct {
	Code         string
	Label        string
	Group        string
	Configurable bool
}

// categoryCatalog 分类目录（顺序即侧栏展示顺序，与前端分组一致）
var categoryCatalog = []CategoryDef{
	{CategoryDataReminder, "数据提醒", CategoryGroupProduct, false},
	{CategoryAppLog, "应用日志", CategoryGroupProduct, true},
	{CategoryDocumentActivity, "文档动态", CategoryGroupProduct, false},
	{CategoryUsageReminder, "用量提醒", CategoryGroupEnterprise, false},
	{CategoryContactsManagement, "通讯录管理", CategoryGroupEnterprise, false},
	{CategoryOpenPlatform, "开放平台", CategoryGroupEnterprise, false},
	{CategorySystemManagement, "系统管理", CategoryGroupEnterprise, false},
	{CategoryOperationNotice, "运营通知", CategoryGroupEnterprise, false},
}

// ParamDef 模板参数 Schema：值为纯字符串（受 MaxLen 约束），未知键拒绝，
// 防止未受控参数进入展示快照
type ParamDef struct {
	Name     string
	Required bool
	MaxLen   int
}

// EventDef 事件目录项：稳定事件码 + 中文展示名 + 纯文本模板 + 渠道与接收
// 规则默认值。模板修改不回写历史消息（快照在物化时固化）。跳转动作由目录
// 固定为「稳定动作码 + 参数键白名单」，生产者不能自由下发 action。
type EventDef struct {
	Code              string
	Category          string
	Label             string
	Severity          string
	Template          string // 纯文本模板，{key} 占位；{actorName} 为渲染器内置变量
	Params            []ParamDef
	ActionType        string   // 受控跳转动作码（空=无动作）；前端 action registry 映射路由
	ActionParamKeys   []string // 动作参数键白名单（取值来自模板参数）
	SupportedChannels []string
	LockedChannels    []string // 必选渠道（不可关闭），当前全部为 system
	DefaultChannels   map[string]bool
	DefaultRecipients []string // 默认接收规则（target_kind 序列）
}

// eventRegistry 事件注册表（唯一事实源，参照 audit/service/events.go 做法）：
// 事件码只增不改；首批为应用日志分类的 4 个事件 + 真实生产者应用资产变更。
var eventRegistry = map[string]EventDef{
	// 应用资产变更：application 域创建/删除应用事务内发布（P1 端到端生产者）
	"application.asset.changed": {
		Code:     "application.asset.changed",
		Category: CategoryAppLog,
		Label:    "应用资产变更",
		Severity: "info",
		Template: "{actorName}{verb}「{appName}」",
		Params: []ParamDef{
			{Name: "appName", Required: true, MaxLen: 128},
			{Name: "verb", Required: true, MaxLen: 16},
			{Name: "appCode", Required: true, MaxLen: 64},
		},
		ActionType:        "open_application",
		ActionParamKeys:   []string{"appCode"},
		SupportedChannels: []string{ChannelSystem, ChannelEmail, ChannelSMS},
		LockedChannels:    []string{ChannelSystem},
		DefaultChannels:   map[string]bool{ChannelSystem: true, ChannelEmail: false, ChannelSMS: false},
		DefaultRecipients: []string{model.RecipientEventActor, model.RecipientTenantAdmin},
	},
	// 数据推送提醒：数据推送任务结果（数据源随低代码引擎落地接入）
	"application.data_push.notice": {
		Code:     "application.data_push.notice",
		Category: CategoryAppLog,
		Label:    "数据推送提醒",
		Severity: "info",
		Template: "数据推送任务「{taskName}」{result}",
		Params: []ParamDef{
			{Name: "taskName", Required: true, MaxLen: 128},
			{Name: "result", Required: true, MaxLen: 64},
		},
		SupportedChannels: []string{ChannelSystem, ChannelEmail, ChannelSMS},
		LockedChannels:    []string{ChannelSystem},
		DefaultChannels:   map[string]bool{ChannelSystem: true, ChannelEmail: false, ChannelSMS: false},
		DefaultRecipients: []string{model.RecipientEventActor, model.RecipientTenantAdmin},
	},
	// 智能助手执行失败
	"application.assistant.run_failed": {
		Code:     "application.assistant.run_failed",
		Category: CategoryAppLog,
		Label:    "智能助手执行失败",
		Severity: "error",
		Template: "智能助手「{taskName}」执行失败，请检查配置后重试",
		Params: []ParamDef{
			{Name: "taskName", Required: true, MaxLen: 128},
		},
		SupportedChannels: []string{ChannelSystem, ChannelEmail, ChannelSMS},
		LockedChannels:    []string{ChannelSystem},
		DefaultChannels:   map[string]bool{ChannelSystem: true, ChannelEmail: false, ChannelSMS: false},
		DefaultRecipients: []string{model.RecipientEventActor, model.RecipientTenantAdmin},
	},
	// 数据流执行失败
	"application.data_flow.run_failed": {
		Code:     "application.data_flow.run_failed",
		Category: CategoryAppLog,
		Label:    "数据流执行失败",
		Severity: "error",
		Template: "数据流「{taskName}」执行失败，请检查节点配置",
		Params: []ParamDef{
			{Name: "taskName", Required: true, MaxLen: 128},
		},
		SupportedChannels: []string{ChannelSystem, ChannelEmail, ChannelSMS},
		LockedChannels:    []string{ChannelSystem},
		DefaultChannels:   map[string]bool{ChannelSystem: true, ChannelEmail: false, ChannelSMS: false},
		DefaultRecipients: []string{model.RecipientEventActor, model.RecipientTenantAdmin},
	},
	// 输出表同步失败
	"application.output_sync.failed": {
		Code:     "application.output_sync.failed",
		Category: CategoryAppLog,
		Label:    "输出表同步失败",
		Severity: "error",
		Template: "输出表「{taskName}」同步失败，请检查数据源与字段映射",
		Params: []ParamDef{
			{Name: "taskName", Required: true, MaxLen: 128},
		},
		SupportedChannels: []string{ChannelSystem, ChannelEmail, ChannelSMS},
		LockedChannels:    []string{ChannelSystem},
		DefaultChannels:   map[string]bool{ChannelSystem: true, ChannelEmail: false, ChannelSMS: false},
		DefaultRecipients: []string{model.RecipientEventActor, model.RecipientTenantAdmin},
	},
}

// LookupEvent 事件码 → 目录定义（未登记返回 false）
func LookupEvent(code string) (EventDef, bool) {
	def, ok := eventRegistry[code]
	return def, ok
}

// LookupCategory 分类码 → 目录定义（未登记返回 false）
func LookupCategory(code string) (CategoryDef, bool) {
	for _, category := range categoryCatalog {
		if category.Code == code {
			return category, true
		}
	}
	return CategoryDef{}, false
}

// Categories 分类目录快照（设置聚合与前端兜底数据源）
func Categories() []CategoryDef {
	out := make([]CategoryDef, len(categoryCatalog))
	copy(out, categoryCatalog)
	return out
}

// EventsOfCategory 分类下的事件目录（按注册表写入顺序稳定输出）
func EventsOfCategory(categoryCode string) []EventDef {
	out := make([]EventDef, 0, 4)
	for _, def := range eventOrder() {
		if def.Category == categoryCode {
			out = append(out, def)
		}
	}
	return out
}

// eventRegistry 迭代顺序不稳定（map），事件展示顺序按注册序列显式固定
func eventOrder() []EventDef {
	return []EventDef{
		eventRegistry["application.asset.changed"],
		eventRegistry["application.data_push.notice"],
		eventRegistry["application.assistant.run_failed"],
		eventRegistry["application.data_flow.run_failed"],
		eventRegistry["application.output_sync.failed"],
	}
}

// EventLabel 事件码中文展示名（未登记原样返回，防目录演进后历史行空白）
func EventLabel(code string) string {
	if def, ok := eventRegistry[code]; ok {
		return def.Label
	}
	return code
}

// recipientLabels 动态接收规则的中文标签（与前端展示一致）
var recipientLabels = map[string]string{
	model.RecipientEventActor:    "创建者",
	model.RecipientEventAudience: "指定成员",
	model.RecipientTenantAdmin:   "系统管理员",
}

// RecipientLabel 接收对象类型 → 中文标签；自定义联系人以姓名为标签（调用方
// 传入），未知类型原样返回
func RecipientLabel(kind string) string {
	if label, ok := recipientLabels[kind]; ok {
		return label
	}
	return kind
}

// channelCapabilities 渠道能力目录：P1/P2 通用邮件/短信 Provider 与计费
// 事实源未接入，email/sms 恒不可用（前端禁用勾选、后端拒绝新开启）；
// P3 落地后经配置端口注入真实能力
func channelCapabilities() map[string]model.ChannelCapabilityView {
	return map[string]model.ChannelCapabilityView{
		ChannelSystem: {Available: true, Reason: ""},
		ChannelEmail:  {Available: false, Reason: "通用邮件通道尚未配置"},
		ChannelSMS:    {Available: false, Reason: "短信计费通道尚未配置"},
	}
}

// channelSupported 渠道是否被事件支持
func channelSupported(def EventDef, channel string) bool {
	for _, supported := range def.SupportedChannels {
		if supported == channel {
			return true
		}
	}
	return false
}

// channelLocked 渠道是否为事件必选渠道
func channelLocked(def EventDef, channel string) bool {
	for _, locked := range def.LockedChannels {
		if locked == channel {
			return true
		}
	}
	return false
}

// ValidateParams 模板参数校验：必填齐全、未知键拒绝、长度受 MaxLen 约束
// （模板参数不得携带敏感数据，Schema 是唯一入口）
func ValidateParams(def EventDef, params map[string]string) error {
	known := make(map[string]ParamDef, len(def.Params))
	for _, p := range def.Params {
		known[p.Name] = p
	}
	for key, value := range params {
		def, ok := known[key]
		if !ok {
			return fmt.Errorf("未知模板参数 %q", key)
		}
		if len([]rune(value)) > def.MaxLen {
			return fmt.Errorf("模板参数 %q 超长", key)
		}
	}
	for key, p := range known {
		if p.Required {
			if value, ok := params[key]; !ok || strings.TrimSpace(value) == "" {
				return fmt.Errorf("缺少必填模板参数 %q", key)
			}
		}
	}
	return nil
}

// BuildAction 构造受控跳转动作（物化时固化）：仅目录登记的动作码与参数键
// 白名单内的取值；外部 URL 不允许，确需时必须服务端域名白名单+前端二次校验
func BuildAction(def EventDef, params map[string]string) map[string]string {
	if def.ActionType == "" {
		return nil
	}
	action := map[string]string{"type": def.ActionType}
	for _, key := range def.ActionParamKeys {
		if value, ok := params[key]; ok {
			action[key] = value
		}
	}
	return action
}

// RenderContent 渲染纯文本展示快照：{key} 占位替换，actorName 为渲染器内置
// 变量（由 Dispatcher 从事件发起成员解析固化）；未知占位符原样保留，不拼接
// HTML、不解释参数内容
func RenderContent(def EventDef, params map[string]string, actorName string) string {
	values := make(map[string]string, len(params)+1)
	for key, value := range params {
		values[key] = value
	}
	if strings.TrimSpace(actorName) == "" {
		actorName = "系统"
	}
	values["actorName"] = actorName

	var builder strings.Builder
	rest := def.Template
	for {
		open := strings.Index(rest, "{")
		if open < 0 {
			builder.WriteString(rest)
			return builder.String()
		}
		close := strings.Index(rest[open:], "}")
		if close < 0 {
			builder.WriteString(rest)
			return builder.String()
		}
		close += open
		builder.WriteString(rest[:open])
		key := rest[open+1 : close]
		if value, ok := values[key]; ok {
			builder.WriteString(value)
		} else {
			// 未知占位符原样保留（模板演进兼容，不吞不炸）
			builder.WriteString(rest[open : close+1])
		}
		rest = rest[close+1:]
	}
}
