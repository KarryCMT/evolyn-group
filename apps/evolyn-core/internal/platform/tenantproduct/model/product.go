// Package model 产品中心域数据模型：平台产品目录、租户产品配置与
// 部门/成员范围关联。表结构唯一事实来源是 migrations/000033（FIX-009），
// 本包只做 GORM 映射。产品目录是平台级资源（不受租户 Callback 过滤）；
// 租户侧三表带 tenant_id，由本域仓储以显式条件定位（口径同 edition 域）
package model

import (
	kernel "evolyn/internal/model"
)

// 产品目录状态（文档 5.2）：平台停用后所有租户不可访问
const (
	CatalogStatusActive   = "active"
	CatalogStatusInactive = "inactive"
)

// 可用范围模式（文档 5.3）：all 不物化关联行；partial 至少一个部门或成员
const (
	ScopeModeAll     = "all"
	ScopeModePartial = "partial"
)

// ProductCatalog 平台内置产品目录：稳定机器码 + 展示信息 + 站内入口。
// 不是租户自建的 applications 应用，两者不共用表与权限资源
type ProductCatalog struct {
	ID        uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	Code      string          `json:"code" gorm:"size:64;not null"`
	Name      string          `json:"name" gorm:"size:64;not null"`
	Icon      string          `json:"icon" gorm:"size:64;not null"`
	EntryPath string          `json:"entryPath" gorm:"size:255;not null"`
	Status    string          `json:"status" gorm:"size:16;not null;default:active"`
	SortOrder int64           `json:"sortOrder" gorm:"not null;default:0"`
	CreatedAt kernel.JSONTime `json:"createdAt"`
	UpdatedAt kernel.JSONTime `json:"updatedAt"`
}

func (*ProductCatalog) TableName() string { return "pf_product_catalogs" }

// TenantProductConfig 租户产品主配置：每租户每产品一条有效记录（部分唯一
// 索引保证）。Revision 是配置乐观锁版本，启停与范围替换成功后递增；
// enabled=false 时保留范围关联，重新启用后恢复原范围（文档 5.5）
type TenantProductConfig struct {
	ID        uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	ProductID uint            `json:"productId" gorm:"not null"`
	Enabled   bool            `json:"enabled" gorm:"not null;default:true"`
	ScopeMode string          `json:"scopeMode" gorm:"size:16;not null;default:all"`
	Revision  int64           `json:"revision" gorm:"not null;default:1"`
	CreatedAt kernel.JSONTime `json:"createdAt"`
	UpdatedAt kernel.JSONTime `json:"updatedAt"`
	kernel.TenantBaseModel
}

func (*TenantProductConfig) TableName() string { return "tn_product_configs" }

// TenantProductDepartment 部门范围关联：仅 partial 时有记录，全量替换
// （先删后插）无软删。子部门经读时递归展开命中，不在此复制子部门 ID
type TenantProductDepartment struct {
	ID                    uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	TenantProductConfigID uint            `json:"tenantProductConfigId" gorm:"not null"`
	DepartmentID          uint            `json:"departmentId" gorm:"not null"`
	TenantID              uint            `json:"tenantId" gorm:"index;not null;default:1"`
	CreatedAt             kernel.JSONTime `json:"createdAt"`
}

func (*TenantProductDepartment) TableName() string { return "tn_product_departments" }

// TenantProductMember 成员范围关联：仅 partial 时有记录，全量替换无软删。
// 成员离职/禁用后历史关联保留供审计，但读取与访问判定均忽略（文档 5.5）
type TenantProductMember struct {
	ID                    uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	TenantProductConfigID uint            `json:"tenantProductConfigId" gorm:"not null"`
	MemberID              uint            `json:"memberId" gorm:"not null"`
	TenantID              uint            `json:"tenantId" gorm:"index;not null;default:1"`
	CreatedAt             kernel.JSONTime `json:"createdAt"`
}

func (*TenantProductMember) TableName() string { return "tn_product_members" }
