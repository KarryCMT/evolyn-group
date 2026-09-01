package model

import (
	"gorm.io/gorm"
)

// PlatformBaseModel 平台一级资源公共字段（FIX-014 模型分层）：
// 适用于不归属任何租户的表（tenants/accounts/auth_infos 等）。CreatorID 与
// UpdaterID 始终引用 accounts.id：租户内记录也沿用账号身份，借 tenant_id +
// account_id 可还原当时租户成员，避免同名字段在不同表指向不同实体。
//
// 未认证注册、迁移和定时任务没有账号操作者时保留 NULL，绝不以 0 伪造账号。
// 刻意不含 TenantID，避免「租户属于哪个租户」的语义问题。
type PlatformBaseModel struct {
	CreatorID *uint          `json:"creatorId,omitempty" gorm:"column:creator_id"`
	UpdaterID *uint          `json:"updaterId,omitempty" gorm:"column:updater_id"`
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
