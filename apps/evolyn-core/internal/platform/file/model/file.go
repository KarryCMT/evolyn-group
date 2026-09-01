package model

import kernel "evolyn/internal/model"

const (
	FileStateUploading = "uploading"
	FileStateReady     = "ready"
)

// File 是租户文件元数据；object_key 仅供服务端访问，绝不作为 API 字段返回。
type File struct {
	ID           uint             `json:"-" gorm:"autoIncrement;primaryKey"`
	Code         string           `json:"id" gorm:"size:64;not null"`
	Bucket       string           `json:"-" gorm:"size:128;not null"`
	ObjectKey    string           `json:"-" gorm:"size:768;not null"`
	OriginalName string           `json:"name" gorm:"size:255;not null"`
	ContentType  string           `json:"contentType" gorm:"size:255;not null"`
	DeclaredSize int64            `json:"declaredSize" gorm:"not null"`
	ActualSize   int64            `json:"size" gorm:"not null;default:0"`
	SHA256       string           `json:"sha256,omitempty" gorm:"size:64"`
	State        string           `json:"state" gorm:"size:16;not null"`
	ExpiresAt    *kernel.JSONTime `json:"expiresAt,omitempty"`
	// CreatorMemberID 是文件归属成员，用于上传者访问边界；它不同于继承的
	// CreatorID（创建账号审计字段，pf_accounts.id）。
	CreatorMemberID uint `json:"-" gorm:"column:creator_member_id;not null"`

	kernel.TenantBaseModel
}

func (*File) TableName() string { return "tn_files" }
