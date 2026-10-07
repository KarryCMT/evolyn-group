package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	kernel "evolyn/internal/model"
)

type JSONContent json.RawMessage

func (j JSONContent) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	return string(j), nil
}

func (j *JSONContent) Scan(value any) error {
	switch data := value.(type) {
	case []byte:
		*j = append((*j)[:0], data...)
	case string:
		*j = append((*j)[:0], data...)
	case nil:
		*j = nil
	default:
		return fmt.Errorf("dashboard: cannot scan JSONContent from %T", value)
	}
	return nil
}

func (j JSONContent) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

func (j *JSONContent) UnmarshalJSON(data []byte) error {
	*j = append((*j)[:0], data...)
	return nil
}

// Dashboard 是应用内业务仪表盘资产；公开接口只暴露 Code。
type Dashboard struct {
	ID               uint        `json:"id" gorm:"autoIncrement;primaryKey"`
	AppID            uint        `json:"appId" gorm:"not null"`
	Code             string      `json:"code" gorm:"size:64;not null"`
	Name             string      `json:"name" gorm:"size:128;not null"`
	Icon             string      `json:"icon" gorm:"size:32;not null;default:''"`
	Color            string      `json:"color" gorm:"size:32;not null;default:''"`
	ProtocolVersion  int         `json:"protocolVersion" gorm:"not null;default:1"`
	DraftContent     JSONContent `json:"draft" gorm:"type:jsonb;not null"`
	DraftRevision    int64       `json:"draftRevision" gorm:"not null;default:1"`
	LatestVersionID  *uint       `json:"latestVersionId"`
	PublishedVersion int         `json:"publishedVersion" gorm:"not null;default:0"`
	CreatorMemberID  uint        `json:"creatorMemberId" gorm:"not null"`

	kernel.TenantBaseModel
}

func (*Dashboard) TableName() string { return "tn_dashboards" }

// DashboardVersion 是只追加、只读取的发布快照模型。
type DashboardVersion struct {
	ID                  uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	TenantID            uint            `json:"tenantId" gorm:"not null;index"`
	DashboardID         uint            `json:"dashboardId" gorm:"not null"`
	VersionNo           int             `json:"versionNo" gorm:"not null"`
	SourceDraftRevision int64           `json:"sourceDraftRevision" gorm:"not null"`
	ProtocolVersion     int             `json:"protocolVersion" gorm:"not null"`
	Content             JSONContent     `json:"content" gorm:"type:jsonb;not null"`
	ContentChecksum     string          `json:"contentChecksum" gorm:"size:64;not null"`
	PublishedByMemberID uint            `json:"publishedByMemberId" gorm:"not null"`
	PublishedAt         kernel.JSONTime `json:"publishedAt"`
	CreatedAt           kernel.JSONTime `json:"createdAt"`
}

func (*DashboardVersion) TableName() string { return "tn_dashboard_versions" }

type DashboardVersionSubject struct {
	ID                 uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	TenantID           uint            `json:"tenantId" gorm:"not null;index"`
	DashboardVersionID uint            `json:"dashboardVersionId" gorm:"not null"`
	SubjectType        string          `json:"subjectType" gorm:"size:20;not null"`
	SubjectID          *uint           `json:"subjectId"`
	CreatedAt          kernel.JSONTime `json:"createdAt"`
}

func (*DashboardVersionSubject) TableName() string { return "tn_dashboard_version_subjects" }

// CreateBinding 在事务开始阶段占用请求标识，完成创建后回填 DashboardID。
type CreateBinding struct {
	ID          uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	TenantID    uint            `json:"tenantId" gorm:"not null"`
	MemberID    uint            `json:"memberId" gorm:"not null"`
	RequestID   string          `json:"requestId" gorm:"size:64;not null"`
	RequestHash string          `json:"requestHash" gorm:"size:64;not null"`
	DashboardID *uint           `json:"dashboardId"`
	CreatedAt   kernel.JSONTime `json:"createdAt"`
	UpdatedAt   kernel.JSONTime `json:"updatedAt"`
}

func (*CreateBinding) TableName() string { return "tn_dashboard_create_bindings" }
