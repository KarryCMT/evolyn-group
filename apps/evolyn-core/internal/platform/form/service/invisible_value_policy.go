// 不可见字段赋值策略（v6 纯领域服务，不依赖 Gin/GORM）。
//
// docs/低代码平台/表单设计器/不可见字段赋值前后端设计方案.md：
// content.submitRule 是表单默认策略，widget_submit_rules 是以顶层 widgetName
// 为键的字段级例外映射（优先于默认）；recompute 只允许已接入服务端确定性
// 派生计算执行器的字段，P3 交付执行器前对任何生效的 3 拒绝保存。
// 与前端 packages/form/src/schema/invisible-value-policy.ts 按同一语义镜像。
package service

// 三种赋值策略（协议整数载荷；禁止字符串数字）。
const (
	submitRulePreserve  = 1 // 保持原值：仅编辑/流程后续提交有基线时生效，新建为类型空值
	submitRuleClear     = 2 // 空值：覆盖为类型空值（新建表单默认）
	submitRuleRecompute = 3 // 始终重新计算：由服务端派生执行器计算后写入
)

// defaultSubmitRule 新建表单的默认策略（v6 设计方案 §3.1）。
const defaultSubmitRule = submitRuleClear

// submitRuleMaxSpecialRules widget_submit_rules 键数上限（v6 设计方案 §3.1）。
const submitRuleMaxSpecialRules = 500

// submitRuleEligibleTypes 可配置特殊赋值规则的顶层控件集（§3.2）：必须具备
// 普通用户提交值语义——布局控件（separator/button）、无用户值的展示/系统控件
// （richtext/sn/linkquery）与尚未开放运行能力的控件（subform 整体等）不可配置。
// 与 TS SUBMIT_RULE_ELIGIBLE_WIDGET_TYPES 逐条一致。
var submitRuleEligibleTypes = map[string]bool{
	"text": true, "textarea": true, "number": true, "datetime": true,
	"radiogroup": true, "checkboxgroup": true, "combo": true, "combocheck": true,
	"user": true, "usergroup": true,
}

// multiValueWidgetTypes 空值为空数组的控件集（与 TS codec 的
// MULTI_VALUE_WIDGET_TYPES 一致）：clear 策略写入 [] 而非 null。
var multiValueWidgetTypes = map[string]bool{
	"checkboxgroup": true, "combocheck": true, "usergroup": true,
	"deptgroup": true, "linkquery": true,
}

// derivedFieldExecutorTypes 已接入服务端确定性派生计算执行器的控件集：
// P3 交付前为空（通过 RegisterDerivedFieldExecutor 注册），因此任何生效的
// recompute 配置在草稿保存即被 validateSubmitRules 拒绝。
var derivedFieldExecutorTypes = map[string]bool{}

// RegisterDerivedFieldExecutor 登记派生字段执行端口（设计方案 §6.1 P3 扩展点）：
// 公式、联动、查询等实现经此窄端口注册，注册后方可配置 recompute。
func RegisterDerivedFieldExecutor(widgetType string) {
	derivedFieldExecutorTypes[widgetType] = true
}

// recomputeSupported 派生执行器是否已注册（未注册的类型不可配置 3）。
func recomputeSupported(widgetType string) bool {
	return derivedFieldExecutorTypes[widgetType]
}

// invisibleValuePolicy 已解析的赋值策略视图（防御式：非法片段回退默认值）。
type invisibleValuePolicy struct {
	defaultRule int
	special     map[string]int
}

// parseInvisibleValuePolicy 从发布快照文档（{content:{...}} 根）解析策略：
// 合法快照由 validateSubmitRules 保证形状；历史快照或外部数据缺键时回退
// 默认「空值」，保证读取侧永不因策略缺键失败。
func parseInvisibleValuePolicy(root map[string]any) invisibleValuePolicy {
	policy := invisibleValuePolicy{defaultRule: defaultSubmitRule, special: map[string]int{}}
	content, _ := root["content"].(map[string]any)
	if content == nil {
		return policy
	}
	if rule, ok := asInteger(content["submitRule"]); ok && rule >= 1 && rule <= 3 {
		policy.defaultRule = rule
	}
	rawRules, _ := content["widget_submit_rules"].(map[string]any)
	for key, raw := range rawRules {
		if rule, ok := asInteger(raw); ok && rule >= 1 && rule <= 3 {
			policy.special[key] = rule
		}
	}
	return policy
}

// strategyOf 策略解析（§3.3）：特殊规则优先于默认策略。
func (p invisibleValuePolicy) strategyOf(widgetName string) int {
	if rule, ok := p.special[widgetName]; ok {
		return rule
	}
	return p.defaultRule
}

// emptyValueForType 类型化空值（§4.1）：按字段值协议归一化而非删除 map 键——
// 多选为空数组，其余为 nil；镜像 TS codec 的 emptyWidgetValue。
func emptyValueForType(widgetType string) any {
	if multiValueWidgetTypes[widgetType] {
		return []any{}
	}
	return nil
}
