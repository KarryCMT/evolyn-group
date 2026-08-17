package model

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	UserAssociation         = "Users"
	UserAuthInfoAssociation = "AuthInfos"
	GroupAssociation        = "Groups"
)

type User struct {
	ID        uint       `json:"id" gorm:"autoIncrement;primaryKey"`
	Name      string     `json:"name" gorm:"size:100;not null;unique"`
	Password  string     `json:"-" gorm:"size:256;"`
	Email     string     `json:"email" gorm:"size:256;"`
	Avatar    string     `json:"avatar" gorm:"size:256;"`
	AuthInfos []AuthInfo `json:"authInfos" gorm:"foreignKey:UserId;references:ID"`
	Groups    []Group    `json:"groups" gorm:"many2many:user_groups;"`
	Roles     []Role     `json:"roles" gorm:"many2many:user_roles;"`

	BaseModel
}

func (*User) TableName() string {
	return "users"
}

// CacheKey Redis Key 规范：{resource}:{tenant}:{rest}（架构文档 18/26.4 章）。
// Update 等路径传入的 user 可能未携带 TenantID（请求构造对象），此时兜底默认租户；
// M1 context 线程化后由中间件保证
func (u *User) CacheKey() string {
	tenantID := u.TenantID
	if tenantID == 0 {
		tenantID = DefaultTenantID
	}
	return fmt.Sprintf("%s:%d:id", u.TableName(), tenantID)
}

func (u *User) MarshalBinary() ([]byte, error) {
	return json.Marshal(u)
}

func (u *User) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, u)
}

type AuthInfo struct {
	ID           uint      `json:"id" gorm:"autoIncrement;primaryKey"`
	UserId       uint      `json:"userId" gorm:"size:256"`
	Url          string    `json:"url" gorm:"size:256"`
	AuthType     string    `json:"authType" gorm:"size:256"`
	AuthId       string    `json:"authId" gorm:"size:256"`
	AccessToken  string    `json:"-" gorm:"size:256"`
	RefreshToken string    `json:"-" gorm:"size:256"`
	Expiry       time.Time `json:"-"`

	BaseModel
}

func (*AuthInfo) TableName() string {
	return "auth_infos"
}

type CreatedUser struct {
	Name      string     `json:"name"`
	Password  string     `json:"password"`
	Email     string     `json:"email"`
	Avatar    string     `json:"avatar"`
	AuthInfos []AuthInfo `json:"authInfos"`
}

func (u *CreatedUser) GetUser() *User {
	return &User{
		Name:      u.Name,
		Password:  u.Password,
		Email:     u.Email,
		Avatar:    u.Avatar,
		AuthInfos: u.AuthInfos,
	}
}

type UpdatedUser struct {
	Name      string     `json:"name"`
	Password  string     `json:"password"`
	Email     string     `json:"email"`
	AuthInfos []AuthInfo `json:"authInfos"`
}

func (u *UpdatedUser) GetUser() *User {
	return &User{
		Name:      u.Name,
		Password:  u.Password,
		Email:     u.Email,
		AuthInfos: u.AuthInfos,
	}
}

type AuthUser struct {
	Name      string `json:"name"`
	Password  string `json:"password"`
	SetCookie bool   `json:"setCookie"`
	AuthType  string `json:"authType"`
	AuthCode  string `json:"authCode"`
}

type UserRole struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

func (u *UserRole) GetUser() *User {
	return &User{
		ID:   u.ID,
		Name: u.Name,
	}
}

type UserInfo struct {
	User
	InRoot bool   `json:"inRoot"`
	Role   string `json:"role"`
}

type Users []User
