package model

import kernel "evolyn/internal/model"

const QRTargetFormRecord = "form_record"

// QRToken 只承载定位关系，不承载业务数据或权限结论。权限在每次扫码时
// 根据当前成员和当前记录状态重新计算。
type QRToken struct {
	ID              uint             `json:"-" gorm:"autoIncrement;primaryKey"`
	TenantID        uint             `json:"-" gorm:"index;not null;uniqueIndex:uk_tn_label_qr_tokens_target,priority:1"`
	Token           string           `json:"-" gorm:"size:64;not null;uniqueIndex:uk_tn_label_qr_tokens_token"`
	AppID           uint             `json:"-" gorm:"not null"`
	FormID          uint             `json:"-" gorm:"not null;uniqueIndex:uk_tn_label_qr_tokens_target,priority:2"`
	RecordID        uint             `json:"-" gorm:"not null;uniqueIndex:uk_tn_label_qr_tokens_target,priority:3"`
	TemplateID      uint             `json:"-" gorm:"not null;uniqueIndex:uk_tn_label_qr_tokens_target,priority:4"`
	TargetType      string           `json:"-" gorm:"size:32;not null;uniqueIndex:uk_tn_label_qr_tokens_target,priority:5"`
	Enabled         bool             `json:"-" gorm:"not null"`
	ExpireAt        *kernel.JSONTime `json:"-"`
	CreatorMemberID uint             `json:"-" gorm:"not null"`
	LastScanAt      *kernel.JSONTime `json:"lastScanAt,omitempty"`
	ScanCount       int64            `json:"scanCount" gorm:"not null"`
	CreatedAt       kernel.JSONTime  `json:"-"`
	UpdatedAt       kernel.JSONTime  `json:"-"`
}

func (*QRToken) TableName() string { return "tn_label_qr_tokens" }
