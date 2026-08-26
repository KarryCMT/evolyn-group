package model

import (
	kernel "evolyn/internal/model"
)

// ---- 租户侧读取（GET /editions/current，设计 4.5.1）----

// CurrentEdition 版本信息概览：订阅 + 容量配额 + 功能权益 + 响应级统计时间
type CurrentEdition struct {
	Subscription SubscriptionView `json:"subscription"`
	Quotas       []QuotaView      `json:"quotas"`
	Features     []FeatureView    `json:"features"`
	AsOf         kernel.JSONTime  `json:"asOf"`
}

// SubscriptionView 当前订阅投影。Status 为读时投影：订阅到期而降级任务
// 未落库时立即返回 expired（设计 4.3.1）；legacy_pending_review 表示存量
// 试用待补录，前端展示「有效期待确认」且无倒计时
type SubscriptionView struct {
	PlanCode      string           `json:"planCode"`
	PlanName      string           `json:"planName"`
	Status        string           `json:"status"`
	GrantType     string           `json:"grantType"`
	StartsAt      kernel.JSONTime  `json:"startsAt"`
	EndsAt        *kernel.JSONTime `json:"endsAt,omitempty"`
	ExpiresAction string           `json:"expiresAction"` // none / downgrade_to_free
}

// QuotaView 资源容量视图。MeteringStatus=pending 时省略 Usage/UsagePercent/
// AsOf（不返回伪零值）；Limit 恒返回（-1 不限量 / 0 不可用 / 正数上限）；
// ResetCycle 仅周期额度返回
type QuotaView struct {
	Key            string           `json:"key"`
	Category       string           `json:"category"`
	Name           string           `json:"name"`
	Unit           string           `json:"unit"`
	Limit          int64            `json:"limit"`
	Usage          *int64           `json:"usage,omitempty"`
	UsagePercent   *float64         `json:"usagePercent,omitempty"`
	MeteringStatus string           `json:"meteringStatus"` // ready / pending
	LimitSource    string           `json:"limitSource"`
	AsOf           *kernel.JSONTime `json:"asOf,omitempty"`
	ResetCycle     string           `json:"resetCycle,omitempty"`
}

// FeatureView 功能权益视图：仅展示与前端入口裁剪，不替代 RBAC
type FeatureView struct {
	Group       string         `json:"group"`
	Key         string         `json:"key"`
	Name        string         `json:"name"`
	Available   bool           `json:"available"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	Description string         `json:"description,omitempty"`
}

// ---- 平台运营面（设计 4.5.2）----

// GrantAction 写操作类型：grant 授予/替换（默认），cancel 取消并降级免费版
const (
	GrantActionGrant  = "grant"
	GrantActionCancel = "cancel"
)

// OverrideInput 特批覆盖输入项
type OverrideInput struct {
	Key    string `json:"key" binding:"required"`
	Value  int64  `json:"value" binding:"required"`
	Reason string `json:"reason"`
}

// GrantRequest 人工授予请求。Overrides 三态：nil 不变 / 空数组清空既有
// manual+trial 覆盖 / 非空全量替换；legacy 覆盖不受影响
type GrantRequest struct {
	Action        string           `json:"action"`
	PlanVersionID uint             `json:"planVersionId"`
	GrantType     string           `json:"grantType"`
	StartsAt      *kernel.JSONTime `json:"startsAt"`
	EndsAt        *kernel.JSONTime `json:"endsAt"`
	Remark        string           `json:"remark"`
	Overrides     *[]OverrideInput `json:"overrides"`
}

// GrantableVersion 可授予的已发布基础套餐版本（运营界面选择器数据源）
type GrantableVersion struct {
	ID                    uint           `json:"id"`
	PlanCode              string         `json:"planCode"`
	PlanName              string         `json:"planName"`
	DisplayName           string         `json:"displayName"`
	Version               int            `json:"version"`
	BillingCycle          string         `json:"billingCycle"`
	CompatibilityPlanCode string         `json:"compatibilityPlanCode"`
	Entitlements          EntitlementSet `json:"entitlements"`
	GrantTypes            []string       `json:"grantTypes"` // 适用的授予方式
}

// TenantEditionDetail 平台侧租户版本详情：当前概览 + 历史订阅 + 覆盖记录
type TenantEditionDetail struct {
	TenantID  uint              `json:"tenantId"`
	Current   *CurrentEdition   `json:"current"`
	History   []SubscriptionRec `json:"history"`
	Overrides []OverrideRec     `json:"overrides"`
}

// SubscriptionRec 订阅历史记录（含运营备注，仅平台面可见）
type SubscriptionRec struct {
	ID                uint             `json:"id"`
	PlanCode          string           `json:"planCode"`
	PlanName          string           `json:"planName"`
	Status            string           `json:"status"`
	GrantType         string           `json:"grantType"`
	StartsAt          kernel.JSONTime  `json:"startsAt"`
	EndsAt            *kernel.JSONTime `json:"endsAt,omitempty"`
	OperatorAccountID *uint            `json:"operatorAccountId,omitempty"`
	Remark            string           `json:"remark"`
	CreatedAt         kernel.JSONTime  `json:"createdAt"`
}

// OverrideRec 覆盖记录（含来源与有效期，仅平台面可见）
type OverrideRec struct {
	ID                uint             `json:"id"`
	EntitlementKey    string           `json:"entitlementKey"`
	Value             int64            `json:"value"`
	Reason            string           `json:"reason"`
	Source            string           `json:"source"`
	StartsAt          kernel.JSONTime  `json:"startsAt"`
	EndsAt            *kernel.JSONTime `json:"endsAt,omitempty"`
	OperatorAccountID *uint            `json:"operatorAccountId,omitempty"`
}
