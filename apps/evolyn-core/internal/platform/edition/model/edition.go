// Package model 版本信息域数据模型：套餐目录、不可变套餐版本快照、
// 租户订阅与特批权益覆盖。表结构唯一事实来源是 migrations/000030（FIX-009），
// 本包只做 GORM 映射与 JSONB 载体
package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"

	kernel "evolyn/internal/model"
)

// 订阅状态（设计 4.4.3）
const (
	SubscriptionActive              = "active"                // 活动基础订阅
	SubscriptionExpired             = "expired"               // 到期已降级（任务落库后的终态）
	SubscriptionReplaced            = "replaced"              // 被新订阅替换
	SubscriptionCancelled           = "cancelled"             // 人工取消
	SubscriptionLegacyPendingReview = "legacy_pending_review" // 存量试用待补录
)

// 授予方式（设计 2.2：trial 是授予方式而非长期套餐商品）
const (
	GrantSystem      = "system"       // 系统初始/自动降级
	GrantManual      = "manual"       // 平台运营人工授予
	GrantSelfService = "self_service" // 用户自助购买（二期）
	GrantTrial       = "trial"        // 试用授予
)

// 覆盖来源（设计 4.4.4）
const (
	OverrideSourceLegacy = "legacy" // 旧 pf_tenants.quotas 迁移，只读
	OverrideSourceManual = "manual" // 平台运营特批
	OverrideSourceTrial  = "trial"  // 试用临时特批，与订阅同日到期
)

// 响应 limitSource 取值：表达当前上限的解析来源，不等同于覆盖记录 source
const (
	LimitSourcePlanVersion    = "plan_version"    // 当前有效套餐版本快照
	LimitSourceTenantOverride = "tenant_override" // 有效 manual/trial 覆盖
	LimitSourceLegacyQuota    = "legacy_quota"    // 尚未迁移完毕的旧覆盖
	LimitSourceExpiryFallback = "expiry_fallback" // 订阅已到期、等待物化免费版
)

// 资源分类（设计 2.2 / 4.3.2）
const (
	CategoryStock    = "stock"    // 存量配额：时点上不得超过
	CategoryPeriodic = "periodic" // 周期额度：周期内累计
)

// 权益资源键（新键空间；存储统一为 storage_bytes 字节，旧 storage_gb 仅
// 是 QuotaService 兼容键）
const (
	ResourceApps       = "apps"
	ResourceMembers    = "members"
	ResourceForms      = "forms"
	ResourceStorage    = "storage_bytes"
	ResourceWorkflowMo = "workflow_runs_month"
)

// GiB 一期存储上限的换算基准（与 QuotaService.storageLimitAndUsage 一致）
const GiB int64 = 1024 * 1024 * 1024

// EditionPlan 套餐目录：稳定编码与展示名，不含价格
type EditionPlan struct {
	ID        uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	Code      string          `json:"code" gorm:"size:32;not null;uniqueIndex"`
	Name      string          `json:"name" gorm:"size:64;not null"`
	Status    string          `json:"status" gorm:"size:16;not null;default:active"`
	Kind      string          `json:"kind" gorm:"size:16;not null;default:base"`
	CreatedAt kernel.JSONTime `json:"createdAt"`
	UpdatedAt kernel.JSONTime `json:"updatedAt"`
}

func (*EditionPlan) TableName() string { return "pf_edition_plans" }

// EditionPlanVersion 套餐版本：不可变权益快照。已发布只能新增不能修改
type EditionPlanVersion struct {
	ID                    uint             `json:"id" gorm:"autoIncrement;primaryKey"`
	PlanID                uint             `json:"planId" gorm:"not null;uniqueIndex:uk_plan_version,priority:1"`
	Version               int              `json:"version" gorm:"not null;uniqueIndex:uk_plan_version,priority:2"`
	DisplayName           string           `json:"displayName" gorm:"size:64;not null"`
	BillingCycle          string           `json:"billingCycle" gorm:"size:16;not null;default:none"`
	CompatibilityPlanCode string           `json:"compatibilityPlanCode" gorm:"column:compatibility_plan_code;size:32;not null"`
	Entitlements          EntitlementSet   `json:"entitlements" gorm:"type:jsonb;not null"`
	PublishedAt           kernel.JSONTime  `json:"publishedAt"`
	RetiredAt             *kernel.JSONTime `json:"retiredAt,omitempty"`
	CreatedAt             kernel.JSONTime  `json:"createdAt"`
	UpdatedAt             kernel.JSONTime  `json:"updatedAt"`
}

func (*EditionPlanVersion) TableName() string { return "pf_edition_plan_versions" }

// TenantSubscription 租户订阅：权益事实源。tn_subscriptions 由平台侧
// 与 worker 经显式 tenant_id 条件读写（仓储统一剥离租户上下文），不依赖
// GORM 租户 Callback 过滤
type TenantSubscription struct {
	ID                uint             `json:"id" gorm:"autoIncrement;primaryKey"`
	TenantID          uint             `json:"tenantId" gorm:"not null;index"`
	PlanVersionID     uint             `json:"planVersionId" gorm:"not null"`
	Status            string           `json:"status" gorm:"size:32;not null"`
	GrantType         string           `json:"grantType" gorm:"size:16;not null"`
	StartsAt          kernel.JSONTime  `json:"startsAt"`
	EndsAt            *kernel.JSONTime `json:"endsAt,omitempty"`
	OperatorAccountID *uint            `json:"operatorAccountId,omitempty"`
	Remark            string           `json:"remark" gorm:"size:512;not null;default:''"`
	CreatedAt         kernel.JSONTime  `json:"createdAt"`
	UpdatedAt         kernel.JSONTime  `json:"updatedAt"`

	// 关联只读快照（查询时按需预加载，出网组装用；落库时忽略）
	PlanCode     string          `json:"planCode,omitempty" gorm:"-"`
	PlanName     string          `json:"planName,omitempty" gorm:"-"`
	DisplayName  string          `json:"displayName,omitempty" gorm:"-"`
	Entitlements *EntitlementSet `json:"-" gorm:"-"`
}

func (*TenantSubscription) TableName() string { return "tn_subscriptions" }

// TenantEntitlementOverride 特批权益覆盖：manual/trial/legacy 三来源，
// 无软删（替换即物理删除，审计经 tn_audit_logs 留痕）
type TenantEntitlementOverride struct {
	ID                uint             `json:"id" gorm:"autoIncrement;primaryKey"`
	TenantID          uint             `json:"tenantId" gorm:"not null;index"`
	EntitlementKey    string           `json:"entitlementKey" gorm:"size:64;not null"`
	Value             int64            `json:"value" gorm:"not null"`
	Reason            string           `json:"reason" gorm:"size:255;not null;default:''"`
	Source            string           `json:"source" gorm:"size:16;not null"`
	StartsAt          kernel.JSONTime  `json:"startsAt"`
	EndsAt            *kernel.JSONTime `json:"endsAt,omitempty"`
	OperatorAccountID *uint            `json:"operatorAccountId,omitempty"`
	CreatedAt         kernel.JSONTime  `json:"createdAt"`
	UpdatedAt         kernel.JSONTime  `json:"updatedAt"`
}

func (*TenantEntitlementOverride) TableName() string { return "tn_entitlement_overrides" }

// ---- 权益快照 JSONB 载体 ----

// ResourceRule 资源规则：Limit 语义 -1 不限量 / 0 不可用 / 正数上限
type ResourceRule struct {
	Key        string `json:"key"`
	Category   string `json:"category"`
	Limit      int64  `json:"limit"`
	Unit       string `json:"unit"`
	ResetCycle string `json:"resetCycle,omitempty"` // 仅周期额度（monthly/yearly）
}

// FeatureRule 功能权益规则：一期仅表达「已落地能力的可用性与参数上限」
type FeatureRule struct {
	Key         string         `json:"key"`
	Group       string         `json:"group"`
	Name        string         `json:"name"`
	Available   bool           `json:"available"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	Description string         `json:"description,omitempty"`
}

// EntitlementSet 套餐版本权益快照
type EntitlementSet struct {
	Resources []ResourceRule `json:"resources"`
	Features  []FeatureRule  `json:"features"`
}

func (e EntitlementSet) Value() (driver.Value, error) { return json.Marshal(e) }

func (e *EntitlementSet) Scan(v interface{}) error {
	if v == nil {
		*e = EntitlementSet{}
		return nil
	}
	switch data := v.(type) {
	case []byte:
		return json.Unmarshal(data, e)
	case string:
		return json.Unmarshal([]byte(data), e)
	default:
		return fmt.Errorf("cannot scan %T into EntitlementSet", v)
	}
}

// ---- 存储双键换算（设计 4.3.2/4.4.1：一期只允许 -1/0/整 GiB，禁止取整）----

// ValidStorageLimit 校验存储上限是否满足一期约束：-1、0 或 GiB 整数倍
func ValidStorageLimit(v int64) bool {
	if v == -1 || v == 0 {
		return true
	}
	return v > 0 && v%GiB == 0
}

// StorageBytesToGB 字节→旧 storage_gb 键值：仅对已通过 ValidStorageLimit
// 的值调用（正数保证整除，无舍入）；-1 不限量原样透传，避免整数除法把 -1
// 折成 0（0 是「禁用」语义，不可混淆）
func StorageBytesToGB(b int64) int64 {
	if b < 0 {
		return b
	}
	return b / GiB
}

// GBToStorageBytes 旧 storage_gb 键值→字节：乘法前做 int64 溢出校验，
// 与 QuotaService.storageLimitAndUsage 的防护口径一致
func GBToStorageBytes(gb int64) (int64, error) {
	if gb < 0 {
		return gb, nil // -1 不限量原样透传
	}
	if gb > math.MaxInt64/GiB {
		return 0, fmt.Errorf("storage quota overflow: %d GB", gb)
	}
	return gb * GiB, nil
}
