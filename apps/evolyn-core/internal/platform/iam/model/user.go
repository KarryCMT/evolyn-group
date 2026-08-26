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

// 成员状态归属租户成员，而非平台账号：同一账号可在不同租户拥有不同的在职状态。
const (
	MemberStatusActive   = "active"
	MemberStatusDisabled = "disabled"
	MemberStatusResigned = "resigned"
	// MemberStatusAll 仅用于列表查询，数据库中不会保存该值；“全部成员”不含离职成员，
	// 离职成员经组织页独立入口查看。
	MemberStatusAll = "all"
)

// User 租户成员（users 表，ADR-006 账号×成员拆分）：
// 登录身份（name/password/phone 等）在 Account，本结构仅承载租户内身份——
// 归属账号、租户内昵称、部门/分组/角色。存量账号字段列由回填策略处理，此处不声明
type User struct {
	ID         uint             `json:"id" gorm:"autoIncrement;primaryKey"`
	AccountId  uint             `json:"accountId" gorm:"index;not null;default:0"` // 归属平台账号，存量回填前为 0
	Nickname   string           `json:"nickname" gorm:"size:100"`                  // 租户内展示名，空则前端回落账号昵称
	Status     string           `json:"status" gorm:"size:16;not null;default:active"`
	ResignedAt *kernel.JSONTime `json:"resignedAt"`
	// Account 仅供成员列表读模型聚合账号资料，避免把平台账号作为成员接口的持久化写入字段。
	Account     *Account     `json:"-" gorm:"foreignKey:AccountId;references:ID"`
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

// MemberListQuery 组织成员列表查询条件。DepartmentID 为零时不按部门筛选；
// Status 为 all/空时查询在职成员（启用与停用），离职成员需显式传 resigned。
type MemberListQuery struct {
	DepartmentID uint
	RoleID       uint
	Status       string
	Keyword      string
	Page         int
	PageSize     int
}

// MemberDepartment 成员列表所需的部门字段子集，避免将租户内部字段直接出网。
type MemberDepartment struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// MemberRole 成员列表所需的角色字段子集。
type MemberRole struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// MemberListItem 组织页的成员行读模型：成员身份来自 users，联系方式和头像来自 accounts。
type MemberListItem struct {
	ID          uint               `json:"id"`
	AccountID   uint               `json:"accountId"`
	Name        string             `json:"name"`
	Phone       string             `json:"phone"`
	Email       string             `json:"email"`
	Avatar      string             `json:"avatar"`
	Status      string             `json:"status"`
	ResignedAt  *kernel.JSONTime   `json:"resignedAt"`
	Departments []MemberDepartment `json:"departments"`
	Roles       []MemberRole       `json:"roles"`
}

// MemberPage 成员列表页；分页总数由服务端筛选后的成员数给出。
type MemberPage struct {
	Items []MemberListItem `json:"items"`
	Total int64            `json:"total"`
}
