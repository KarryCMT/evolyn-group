package model

import (
	"encoding/json"
	"fmt"

	// BaseModel 在共享内核 internal/model；默认租户常量在租户域模型（跨域常量引用，单向无环）
	kernel "evolyn/internal/model"
	tenantmodel "evolyn/internal/platform/tenant/model"
)

const (
	UserAssociation         = "Users"
	GroupAssociation        = "Groups"
	DepartmentAssociation   = "Departments"
	UserAuthInfoAssociation = "AuthInfos" // 兼容常量：AuthInfo 已迁移挂账号（ADR-006），列名残留期间引用收敛于此
)

// User 租户成员（users 表，ADR-006 账号×成员拆分）：
// 登录身份（name/password/phone 等）在 Account，本结构仅承载租户内身份——
// 归属账号、租户内昵称、部门/分组/角色。存量账号字段列由回填策略处理，此处不声明
type User struct {
	ID          uint         `json:"id" gorm:"autoIncrement;primaryKey"`
	AccountId   uint         `json:"accountId" gorm:"index;not null;default:0"` // 归属平台账号，存量回填前为 0
	Nickname    string       `json:"nickname" gorm:"size:100"`                  // 租户内展示名，空则前端回落账号昵称
	Departments []Department `json:"departments" gorm:"many2many:department_users;"`
	Groups      []Group      `json:"groups" gorm:"many2many:user_groups;"`
	Roles       []Role       `json:"roles" gorm:"many2many:user_roles;"`

	kernel.TenantBaseModel
}

func (*User) TableName() string {
	return "users"
}

// CacheKey Redis Key 规范：{resource}:{tenant}:{rest}（架构文档 18/26.4 章）。
// 请求构造对象未携带租户时兜底默认租户，由中间件保证常规路径不触发
func (u *User) CacheKey() string {
	tenantID := u.TenantID
	if tenantID == 0 {
		tenantID = tenantmodel.DefaultTenantID
	}
	return fmt.Sprintf("%s:%d:id", u.TableName(), tenantID)
}

func (u *User) MarshalBinary() ([]byte, error) {
	return json.Marshal(u)
}

func (u *User) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, u)
}

// UpdatedMember 成员资料更新（租户内语义，仅昵称；账号资料走 accounts 域）
type UpdatedMember struct {
	Nickname string `json:"nickname"`
}

func (u *UpdatedMember) GetMember() *User {
	return &User{
		Nickname: u.Nickname,
	}
}

// AuthUser 登录请求：name 或 phone + password（账号级校验）；
// 或 phone + smsCode（验证码登录，控制器先经 sms 域校验）；
// OAuth 走 AuthType/AuthCode；TenantCode 可选——指定登录目标租户，
// 不填则取该账号的第一个成员关系（默认租户体验）
type AuthUser struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Password   string `json:"password"`
	SmsCode    string `json:"smsCode"`
	TenantCode string `json:"tenantCode"`
	SetCookie  bool   `json:"setCookie"`
	AuthType   string `json:"authType"`
	AuthCode   string `json:"authCode"`
}

type Users []User
