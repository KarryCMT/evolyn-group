package model

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
	Plan   string `json:"plan" gorm:"size:32;not null;default:free"` // 套餐：配额挂 P1 落地
	Status string `json:"status" gorm:"size:16;not null;default:active"`

	BaseModel
}

func (*Tenant) TableName() string {
	return "tenants"
}
