// Package model 消息中心域数据模型：租户通知设置聚合与出网 DTO。
package model

import (
	kernel "evolyn/internal/model"
)

// Setting 租户通知设置聚合根：每租户一行有效记录；revision 覆盖偏好/接收
// 规则/自定义联系人整个聚合的乐观锁。租户开通事务预置，读取侧幂等兜底。
type Setting struct {
	ID       uint  `json:"id" gorm:"autoIncrement;primaryKey"`
	Revision int64 `json:"revision" gorm:"not null;default:1"`

	kernel.TenantBaseModel
}

func (*Setting) TableName() string { return "tenant_notification_settings" }

// Preference 事件偏好覆盖：只保存对事件注册表默认值的覆盖，无覆盖行时
// 投影注册表默认值（recipients_overridden 区分默认对象与显式配置为空）。
type Preference struct {
	ID                   uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	SettingID            uint   `json:"settingId" gorm:"not null"`
	EventCode            string `json:"eventCode" gorm:"size:128;not null"`
	SystemEnabled        bool   `json:"systemEnabled" gorm:"not null;default:true"`
	EmailEnabled         bool   `json:"emailEnabled" gorm:"not null;default:false"`
	SMSEnabled           bool   `json:"smsEnabled" gorm:"not null;default:false"`
	RecipientsOverridden bool   `json:"recipientsOverridden" gorm:"not null;default:false"`

	TenantID  uint            `json:"tenantId" gorm:"not null;default:1"`
	CreatedAt kernel.JSONTime `json:"createdAt"`
	UpdatedAt kernel.JSONTime `json:"updatedAt"`
}

func (*Preference) TableName() string { return "tenant_notification_preferences" }

// 接收对象类型（target_kind 稳定枚举）
const (
	RecipientEventActor      = "event_actor"      // 事件发起人（「创建者」）
	RecipientEventAudience   = "event_audience"   // 事件显式指定受众
	RecipientTenantAdmin     = "tenant_admin"     // 系统管理员（内置管理员组实时推导）
	RecipientCustomRecipient = "custom_recipient" // 自定义外部联系人（仅邮件/短信）
)

// PreferenceRecipient 事件偏好的接收规则关联：动态规则同一偏好内至多一条，
// 自定义联系人不能重复关联（服务层校验）；CHECK 强制与联系人 ID 组合合法。
type PreferenceRecipient struct {
	ID                uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	PreferenceID      uint   `json:"preferenceId" gorm:"not null"`
	TargetKind        string `json:"targetKind" gorm:"size:32;not null"`
	CustomRecipientID *uint  `json:"customRecipientId"` // 仅 custom_recipient 时非空

	TenantID  uint            `json:"tenantId" gorm:"not null;default:1"`
	CreatedAt kernel.JSONTime `json:"createdAt"`
}

func (*PreferenceRecipient) TableName() string {
	return "tenant_notification_preference_recipients"
}

// CustomRecipient 租户自定义外部提醒对象：软删除保留关联历史，删除前校验
// 未被偏好引用；手机/邮箱至少一项由服务层校验。
type CustomRecipient struct {
	ID       uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	Name     string `json:"name" gorm:"size:80;not null"`
	Mobile   string `json:"mobile" gorm:"size:32;not null;default:''"`
	Email    string `json:"email" gorm:"size:254;not null;default:''"`
	Revision int64  `json:"revision" gorm:"not null;default:1"`

	kernel.TenantBaseModel
}

func (*CustomRecipient) TableName() string { return "tenant_notification_custom_recipients" }

// ---- 通知设置聚合出网 DTO ----

// ChannelCapabilityView 渠道能力：available=false 时前端禁用勾选并展示 reason
type ChannelCapabilityView struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

// RecipientView 接收对象出网投影：动态规则按 kind 出中文标签；自定义联系人
// 追加姓名标签（联系方式只经联系人池接口暴露给管理员）
type RecipientView struct {
	Kind        string `json:"kind"`
	Label       string `json:"label"`
	RecipientID uint   `json:"recipientId,omitempty"` // 仅 custom_recipient
}

// RecipientInput 接收规则写入输入：custom_recipient 必须携带同租户联系人 ID
type RecipientInput struct {
	Kind        string `json:"kind" binding:"required"`
	RecipientID uint   `json:"recipientId"`
}

// SettingEventView 事件有效偏好（覆盖行 ?? 注册表默认投影）
type SettingEventView struct {
	Code              string          `json:"code"`
	Label             string          `json:"label"`
	Severity          string          `json:"severity"`
	SupportedChannels []string        `json:"supportedChannels"`
	LockedChannels    []string        `json:"lockedChannels"`
	Channels          map[string]bool `json:"channels"`   // system/email/sms 有效开关
	Recipients        []RecipientView `json:"recipients"` // 有效接收规则
}

// SettingCategoryView 设置页分类目录（含无事件分类，configurable=false）
type SettingCategoryView struct {
	ID           string             `json:"id"`
	Label        string             `json:"label"`
	Group        string             `json:"group"` // product/enterprise
	Configurable bool               `json:"configurable"`
	Events       []SettingEventView `json:"events"`
}

// SettingAggregateView GET /notification-settings 响应：目录 + 有效偏好 + 渠道能力 + 聚合 revision
type SettingAggregateView struct {
	Revision            int64                            `json:"revision"`
	Categories          []SettingCategoryView            `json:"categories"`
	ChannelCapabilities map[string]ChannelCapabilityView `json:"channelCapabilities"`
	// SmsBudget 云币/短信额度摘要：计费事实源未接入时为 null，前端隐藏数值，
	// 不使用样例数值兜底（P3 对接后填充）
	SmsBudget *SmsBudgetView `json:"smsBudget"`
}

// SmsBudgetView 短信费用摘要（P3 计费事实源接入后启用）
type SmsBudgetView struct {
	CoinBalance    int64 `json:"coinBalance"`
	SmsUnitCost    int64 `json:"smsUnitCost"`
	RemainingCount int64 `json:"remainingCount"`
}

// ChannelPatch 渠道部分更新：nil 键保持不变（指针三态）
type ChannelPatch struct {
	System *bool `json:"system"`
	Email  *bool `json:"email"`
	SMS    *bool `json:"sms"`
}

// PatchPreferenceRequest 事件偏好更新：channels 部分更新；recipients 一旦
// 出现即全量替换，缺省表示不修改；revision 为聚合乐观锁口令
type PatchPreferenceRequest struct {
	Revision   int64             `json:"revision"`
	Channels   *ChannelPatch     `json:"channels"`
	Recipients *[]RecipientInput `json:"recipients"`
}

// PatchPreferenceResponse 偏好更新成功响应：该事件的新有效偏好 + 新聚合 revision
type PatchPreferenceResponse struct {
	Revision int64            `json:"revision"`
	Event    SettingEventView `json:"event"`
}

// CustomRecipientView 自定义提醒对象出网（完整联系方式仅对设置管理员开放）
type CustomRecipientView struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Mobile   string `json:"mobile"`
	Email    string `json:"email"`
	Revision int64  `json:"revision"`
}

// CreateCustomRecipientRequest 新增提醒对象：携带聚合 revision 做乐观锁
type CreateCustomRecipientRequest struct {
	Revision int64  `json:"revision" binding:"min=1"`
	Name     string `json:"name" binding:"required"`
	Mobile   string `json:"mobile"`
	Email    string `json:"email"`
}
