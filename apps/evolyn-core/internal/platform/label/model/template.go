// Package model 承载标签模板草稿与不可变发布快照的持久化模型。
package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	kernel "evolyn/internal/model"
)

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusDisabled  = "disabled"
)

// Template 每个表单绑定一个有效模板；草稿全文以 LabelSchema JSONB 保存。
type Template struct {
	ID                     uint          `json:"-" gorm:"autoIncrement;primaryKey"`
	Code                   string        `json:"code" gorm:"size:64;not null"`
	Name                   string        `json:"name" gorm:"size:128;not null"`
	Description            string        `json:"description" gorm:"size:500;not null;default:''"`
	AppID                  uint          `json:"appId" gorm:"not null"`
	FormID                 uint          `json:"-" gorm:"not null"`
	FormCode               string        `json:"formCode" gorm:"size:64;not null"`
	Status                 string        `json:"status" gorm:"size:20;not null;default:draft"`
	DraftSchema            SchemaContent `json:"draft" gorm:"type:jsonb;not null"`
	DraftRevision          int64         `json:"draftRevision" gorm:"not null;default:1"`
	PreviewedDraftRevision int64         `json:"previewedDraftRevision" gorm:"not null;default:0"`
	LatestVersionID        *uint         `json:"-"`
	PublishedVersion       int           `json:"publishedVersion" gorm:"not null;default:0"`
	PublishedDraftRevision int64         `json:"publishedDraftRevision" gorm:"not null;default:0"`
	Width                  float64       `json:"width" gorm:"type:numeric(12,4);not null"`
	Height                 float64       `json:"height" gorm:"type:numeric(12,4);not null"`
	Unit                   string        `json:"unit" gorm:"size:10;not null;default:mm"`
	DPI                    int           `json:"dpi" gorm:"not null;default:300"`
	CreatorMemberID        uint          `json:"creatorMemberId" gorm:"not null"`

	kernel.TenantBaseModel
}

func (*Template) TableName() string { return "tn_label_templates" }

// TemplateVersion 是追加写、不可修改的发布快照。
type TemplateVersion struct {
	ID                  uint            `json:"-" gorm:"autoIncrement;primaryKey"`
	TemplateID          uint            `json:"-" gorm:"not null"`
	VersionNo           int             `json:"versionNo" gorm:"not null"`
	SchemaVersion       string          `json:"schemaVersion" gorm:"size:20;not null;default:1.0"`
	SchemaSnapshot      SchemaContent   `json:"schema" gorm:"type:jsonb;not null"`
	PublishedByMemberID uint            `json:"publishedByMemberId" gorm:"not null"`
	PublishedAt         kernel.JSONTime `json:"publishedAt"`
	TenantID            uint            `json:"-" gorm:"index;not null"`
	CreatedAt           kernel.JSONTime `json:"createdAt"`
}

func (*TemplateVersion) TableName() string { return "tn_label_template_versions" }

// SchemaContent 原样保存 LabelSchema，避免 map 往返改变未知可选字段。
type SchemaContent json.RawMessage

func (j SchemaContent) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	return string(j), nil
}

func (j *SchemaContent) Scan(value any) error {
	switch data := value.(type) {
	case nil:
		*j = SchemaContent("null")
	case []byte:
		*j = append((*j)[:0], data...)
	case string:
		*j = SchemaContent(data)
	default:
		return fmt.Errorf("label: cannot scan SchemaContent from %T", value)
	}
	return nil
}

func (j SchemaContent) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

func (j *SchemaContent) UnmarshalJSON(data []byte) error {
	*j = append((*j)[:0], data...)
	return nil
}
