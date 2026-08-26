package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	kernel "evolyn/internal/model"
)

const (
	MemberInvitationSourceManual = "manual"
	MemberInvitationSourceBatch  = "batch"
	MemberInvitationPending      = "pending"
	MemberInvitationAccepted     = "accepted"
)

// MemberInviteProfile 承载通讯录模板中的成员档案字段。邀请尚未接受时没有
// users 记录，档案先与邀请一同保存，避免创建无法登录的占位账号。
type MemberInviteProfile struct {
	DepartmentIDs   []uint   `json:"departmentIds"`
	DepartmentNames []string `json:"departmentNames"`
	Alias           string   `json:"alias"`
	EmployeeNo      string   `json:"employeeNo"`
	Gender          string   `json:"gender"`
	Title           string   `json:"title"`
	EmploymentType  string   `json:"employmentType"`
	HiredAt         string   `json:"hiredAt"`
	WorkLocation    string   `json:"workLocation"`
	Birthday        string   `json:"birthday"`
	Education       string   `json:"education"`
}

func (p MemberInviteProfile) Value() (driver.Value, error) { return json.Marshal(p) }

func (p *MemberInviteProfile) Scan(value interface{}) error {
	if value == nil {
		*p = MemberInviteProfile{}
		return nil
	}
	switch data := value.(type) {
	case []byte:
		return json.Unmarshal(data, p)
	case string:
		return json.Unmarshal([]byte(data), p)
	default:
		return fmt.Errorf("cannot scan %T into MemberInviteProfile", value)
	}
}

// MemberInvitation 是待接受的成员邀请。单人邀请通过 inviteToken 与后续注册/
// 加入流程关联；当前组织页只创建、批量导入和展示邀请入口，不把待邀请人误计入成员数。
type MemberInvitation struct {
	ID              uint                `json:"id" gorm:"autoIncrement;primaryKey"`
	InviterMemberID uint                `json:"inviterMemberId" gorm:"not null;default:0"`
	Name            string              `json:"name" gorm:"size:80;not null"`
	Identifier      string              `json:"identifier" gorm:"size:50"`
	Phone           string              `json:"phone" gorm:"size:32"`
	Email           string              `json:"email" gorm:"size:256"`
	Profile         MemberInviteProfile `json:"profile" gorm:"type:jsonb;not null;default:'{}'"`
	InviteToken     string              `json:"inviteToken" gorm:"size:64;not null"`
	Source          string              `json:"source" gorm:"size:16;not null;default:manual"`
	Status          string              `json:"status" gorm:"size:16;not null;default:pending"`

	kernel.TenantBaseModel
}

func (*MemberInvitation) TableName() string { return "member_invitations" }

// TenantPublicInvitationLink 是租户公开邀请的开关和不可预测 token。
type TenantPublicInvitationLink struct {
	ID              uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	Token           string `json:"token" gorm:"size:64;not null"`
	Enabled         bool   `json:"enabled" gorm:"not null;default:false"`
	CreatorMemberID uint   `json:"creatorMemberId" gorm:"not null;default:0"`

	kernel.TenantBaseModel
}

func (*TenantPublicInvitationLink) TableName() string { return "tenant_public_invitation_links" }

// MemberInvitationBatchResult 为批量导入提供逐行可见的成功/失败结果。
type MemberInvitationBatchResult struct {
	SuccessCount int      `json:"successCount"`
	FailedRows   []string `json:"failedRows"`
}
