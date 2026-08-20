package model

import (
	"gorm.io/gorm"
)

// PlatformBaseModel 平台一级资源公共字段（FIX-014 模型分层）：
// 适用于不归属任何租户的表（tenants/accounts/auth_infos 等），
// 刻意不含 TenantID，避免「租户属于哪个租户」的语义问题
type PlatformBaseModel struct {
	CreatedAt JSONTime       `json:"createdAt"`
	UpdatedAt JSONTime       `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-"` // soft delete
}

// TenantBaseModel 租户内资源公共字段（FIX-014 模型分层）：
// TenantID 为数据归属租户，not null + default:1 使加列时自动回填默认租户；
// GORM 租户 Callback 依据本字段自动注入过滤/回填
type TenantBaseModel struct {
	TenantID uint `json:"tenantId" gorm:"index;not null;default:1"`
	PlatformBaseModel
}
