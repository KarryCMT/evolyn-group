package model

import (
	"time"

	"gorm.io/gorm"
)

type BaseModel struct {
	// TenantID 数据归属租户，not null + default:1 使 AutoMigrate 加列时自动把存量行回填默认租户
	TenantID  uint           `json:"tenantId" gorm:"index;not null;default:1"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-"` // soft delete
}
