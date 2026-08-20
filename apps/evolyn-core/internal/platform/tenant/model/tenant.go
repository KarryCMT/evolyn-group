package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	kernel "evolyn/internal/model"
)

// 租户状态（生命周期见架构文档第 26.2 节）
const (
	TenantActive  = "active"
	TenantFrozen  = "frozen"
	TenantDeleted = "deleted"
)

const (
	// DefaultTenantCode 默认租户编码：单租户/私有化场景下所有存量与新建数据的归属
	DefaultTenantCode = "default"
	// DefaultTenantID 默认租户 ID（首条自增记录），CacheKey 等无租户上下文的兜底值
	DefaultTenantID uint = 1
)

// Tenant 租户（企业/组织），平台一级资源。
// 角色模型：平台运营方 -> 租户 -> 租户管理员 -> 部门/成员/角色（租户内封闭）。
// 说明：平台固定侧主键沿用 uint 自增与既有模型保持一致；动态侧（JSONB/物理表）按文档用 UUID。
type Tenant struct {
	ID     uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	Code   string `json:"code" gorm:"size:64;not null;uniqueIndex"` // 租户编码，登录识别用
	Name   string `json:"name" gorm:"size:128;not null"`
	Plan   string `json:"plan" gorm:"size:32;not null;default:free"`     // 套餐，取值见 plan.go
	Status string `json:"status" gorm:"size:16;not null;default:active"` // active/frozen/deleted

	// OwnerAccountId 开通者账号（is_owner 语义锚点，ADR-006）。
	// FIX-016：NULL = 暂未设置 Owner（原 0 哨兵值废除），非空时数据库层
	// 外键引用 accounts(id)，不再可能出现指向不存在账号的 Owner
	OwnerAccountId *uint        `json:"ownerAccountId" gorm:"index"`
	Config         TenantConfig `json:"config" gorm:"type:jsonb"` // 品牌/水印/时区/语言（26.5）
	Quotas         Quotas       `json:"quotas" gorm:"type:jsonb"` // 套餐配额覆盖，空则用套餐默认值

	// 注销生命周期（FIX-012）：deleted 状态记录申请与保留截止，到期由 Purge Worker 清理
	DeleteRequestedAt *kernel.JSONTime `json:"deleteRequestedAt"` // 注销申请时间
	RetentionUntil    *kernel.JSONTime `json:"retentionUntil"`    // 数据保留截止时间
	PurgedAt          *kernel.JSONTime `json:"purgedAt"`          // 最终清理完成时间（墓碑标记）

	kernel.PlatformBaseModel // 平台一级资源，无 tenant_id（FIX-014）
}

func (*Tenant) TableName() string {
	return "tenants"
}

// TenantConfig 租户级配置（架构文档 26.5 config JSONB，对标简道云 tenant 域）
type TenantConfig struct {
	Watermark  WatermarkConfig  `json:"watermark"`  // 水印
	Theme      ThemeConfig      `json:"theme"`      // 品牌主题
	Timezone   string           `json:"timezone"`   // IANA 时区，空串按服务端默认
	Locale     string           `json:"locale"`     // 语言（zh_cn/en_us/...），空串按服务端默认
	Onboarding OnboardingConfig `json:"onboarding"` // 注册向导采集的企业画像（个性化模板/运营统计）
}

type WatermarkConfig struct {
	Enabled bool   `json:"enabled"`
	Color   string `json:"color"`   // light / dark
	Density string `json:"density"` // normal / compact
}

type ThemeConfig struct {
	AppNaviColor string `json:"appNaviColor"` // 应用导航栏配色：dark / light
}

// OnboardingConfig 注册向导第 2 步「创建团队」采集项（对齐截图口径）：
// 仅作画像与模板推荐，不参与权限/配额判定
type OnboardingConfig struct {
	Demand          string   `json:"demand"`          // 你的需求（单选，选填）
	Industry        string   `json:"industry"`        // 所属行业（单选）
	ManagementNeeds []string `json:"managementNeeds"` // 企业内部管理需求（多选）
}

// 零值配置的合理默认：水印关、导航深色
func DefaultTenantConfig() TenantConfig {
	return TenantConfig{
		Watermark: WatermarkConfig{Enabled: false, Color: "light", Density: "normal"},
		Theme:     ThemeConfig{AppNaviColor: "dark"},
	}
}

func (c TenantConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *TenantConfig) Scan(v interface{}) error {
	if v == nil {
		*c = DefaultTenantConfig()
		return nil
	}
	switch data := v.(type) {
	case []byte:
		return json.Unmarshal(data, c)
	case string:
		return json.Unmarshal([]byte(data), c)
	default:
		return fmt.Errorf("cannot scan %T into TenantConfig", v)
	}
}
