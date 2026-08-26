package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	kernel "evolyn/internal/model"
)

// MemberProfileAttributes 正式成员的租户内扩展档案（member_profiles.attributes）。
// 只允许字段注册表中定义的扩展 key（alias/employeeId/gender/position/employment/
// hireDate/workplace/birthday/education），日期统一 YYYY-MM-DD、文本最长 50 字符，
// 由服务层校验（文档 4.2）。不重复存储手机号、邮箱、部门和角色。
type MemberProfileAttributes map[string]string

func (a MemberProfileAttributes) Value() (driver.Value, error) {
	if a == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(a)
}

func (a *MemberProfileAttributes) Scan(value interface{}) error {
	if value == nil {
		*a = MemberProfileAttributes{}
		return nil
	}
	switch data := value.(type) {
	case []byte:
		return json.Unmarshal(data, a)
	case string:
		return json.Unmarshal([]byte(data), a)
	default:
		return fmt.Errorf("cannot scan %T into MemberProfileAttributes", value)
	}
}

// MemberProfile 正式成员扩展档案：工号等企业资料在邀请接受时从邀请草稿迁入，
// 之后由成员本人（按字段配置）或管理员维护。identifier 为企业内编号，
// 租户内有效记录唯一
type MemberProfile struct {
	ID         uint                    `json:"id" gorm:"autoIncrement;primaryKey"`
	MemberID   uint                    `json:"memberId" gorm:"not null;index"`
	Identifier string                  `json:"identifier" gorm:"size:50"`
	Attributes MemberProfileAttributes `json:"attributes" gorm:"type:jsonb;not null;default:'{}'"`

	kernel.TenantBaseModel
}

func (*MemberProfile) TableName() string { return "member_profiles" }

// MemberProfileView 成员资料读取视图：Values 为「字段 key → 展示值」映射
// （部门/角色为逗号分隔名称），读取侧已按字段配置裁剪；
// EditableKeys 为当前身份可经通用资料接口提交的扩展字段 key 集合
type MemberProfileView struct {
	Values       map[string]string `json:"values"`
	EditableKeys []string          `json:"editableKeys"`
}

// MemberProfileAdminView 管理员视角的成员资料：全量字段值（不受 personalVisible
// 裁剪）+ 按 cardVisible 裁剪的卡片视图（前端成员卡片必须消费服务端裁剪结果，
// 不得自行读取隐藏字段再隐藏 DOM，文档 5.2）+ 字段元数据（渲染表单用）
type MemberProfileAdminView struct {
	Values      map[string]string        `json:"values"`
	CardValues  map[string]string        `json:"cardValues"`
	FieldConfig []MemberFieldSettingView `json:"fieldConfig"`
}
