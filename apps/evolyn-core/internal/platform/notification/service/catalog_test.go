package service

import (
	"testing"

	"evolyn/internal/platform/notification/model"

	"github.com/stretchr/testify/assert"
)

// 事件目录注册表测试（文档 16.1）：未知事件、参数校验、模板纯文本输出、
// 受控动作构造
func TestCatalogLookupAndOrder(t *testing.T) {
	// 八个稳定分类全部登记且顺序稳定
	categories := Categories()
	assert.Len(t, categories, 8)
	assert.Equal(t, CategoryAppLog, categories[1].Code)

	// 事件按分类检索且顺序稳定（事件码只增不改）
	events := EventsOfCategory(CategoryAppLog)
	assert.Len(t, events, 5)
	assert.Equal(t, "application.asset.changed", events[0].Code)
	assert.Equal(t, "application.output_sync.failed", events[4].Code)

	// 未知分类/事件
	_, ok := LookupCategory("not-exists")
	assert.False(t, ok)
	_, ok = LookupEvent("not.existing.event")
	assert.False(t, ok)
}

func TestValidateParams(t *testing.T) {
	def, ok := LookupEvent("application.data_flow.run_failed")
	assert.True(t, ok)

	// 必填齐全通过
	assert.NoError(t, ValidateParams(def, map[string]string{"taskName": "客户汇总"}))
	// 必填缺失拒绝
	assert.Error(t, ValidateParams(def, map[string]string{}))
	// 未知键拒绝（Schema 是唯一入口，防未受控参数进入展示快照）
	assert.Error(t, ValidateParams(def, map[string]string{"taskName": "x", "evil": "<script>"}))
	// 超长拒绝
	assert.Error(t, ValidateParams(def, map[string]string{"taskName": string(make([]byte, 200))}))
}

func TestRenderContentPlain(t *testing.T) {
	def, _ := LookupEvent("application.asset.changed")

	// 占位符替换 + actorName 内置变量；输出为纯文本，不拼接 HTML
	content := RenderContent(def, map[string]string{"appName": "CRM", "verb": "创建了应用"}, "李同学")
	assert.Equal(t, "李同学创建了应用「CRM」", content)

	// actorName 空白回退「系统」
	content = RenderContent(def, map[string]string{"appName": "CRM", "verb": "删除了应用"}, "  ")
	assert.Equal(t, "系统删除了应用「CRM」", content)

	// 参数值不做任何标记解释（纯文本插值）
	content = RenderContent(def, map[string]string{"appName": "<b>x</b>", "verb": "v"}, "u")
	assert.Equal(t, "uv「<b>x</b>」", content)

	// 未知占位符原样保留
	content = RenderContent(EventDef{Template: "{a}+{missing}"}, map[string]string{"a": "1"}, "")
	assert.Equal(t, "1+{missing}", content)
}

func TestBuildAction(t *testing.T) {
	def, _ := LookupEvent("application.asset.changed")

	// 稳定动作码 + 参数键白名单取值
	action := BuildAction(def, map[string]string{"appCode": "app_abc", "verb": "x", "appName": "n"})
	assert.Equal(t, map[string]string{"type": "open_application", "appCode": "app_abc"}, action)

	// 无动作事件返回 nil
	noAction := EventDef{Code: "x"}
	assert.Nil(t, BuildAction(noAction, nil))
}

func TestChannelRules(t *testing.T) {
	def, _ := LookupEvent("application.asset.changed")
	assert.True(t, channelSupported(def, ChannelEmail))
	assert.False(t, channelSupported(def, "voice"))
	assert.True(t, channelLocked(def, ChannelSystem))
	assert.False(t, channelLocked(def, ChannelEmail))

	// P1/P2 渠道能力：email/sms 恒不可用（P3 Provider 接入后经配置注入）
	capabilities := channelCapabilities()
	assert.True(t, capabilities[ChannelSystem].Available)
	assert.False(t, capabilities[ChannelEmail].Available)
	assert.NotEmpty(t, capabilities[ChannelEmail].Reason)
	assert.False(t, capabilities[ChannelSMS].Available)
}

func TestRecipientLabels(t *testing.T) {
	assert.Equal(t, "创建者", RecipientLabel(model.RecipientEventActor))
	assert.Equal(t, "指定成员", RecipientLabel(model.RecipientEventAudience))
	assert.Equal(t, "系统管理员", RecipientLabel(model.RecipientTenantAdmin))
	// 未知类型原样返回（防目录演进后展示空白）
	assert.Equal(t, "someone_else", RecipientLabel("someone_else"))
}
