package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	kernel "evolyn/internal/model"
)

// Account 平台账号（登录身份）：登录名/手机号全局唯一，密码与第三方凭证挂账号。
// 租户内身份见 User（成员）；一个账号可对应多个租户的成员关系
type Account struct {
	ID       uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	Name     string `json:"name" gorm:"size:100;not null;uniqueIndex"` // 登录名，全局唯一
	Nickname string `json:"nickname" gorm:"size:100"`                  // 展示昵称（平台级）
	Phone    string `json:"phone" gorm:"size:32;uniqueIndex"`          // 手机号，全局唯一；空值不参与唯一约束（PG 多 NULL 允许）
	Email    string `json:"email" gorm:"size:256"`
	Password string `json:"-" gorm:"size:256"`
	Avatar   string `json:"avatar" gorm:"size:256"`
	// 账号注册引导画像（注册向导第 3 步「完善信息」）：角色/了解渠道是
	// 「人」的属性挂账号；租户级画像见 tenants.config 的 onboarding 段
	Onboarding AccountOnboarding `json:"onboarding" gorm:"type:jsonb;not null;default:'{}'"`
	AuthInfos  []AuthInfo        `json:"authInfos" gorm:"foreignKey:AccountId;references:ID"`

	kernel.PlatformBaseModel // 平台一级资源，无租户归属（FIX-014）
}

func (*Account) TableName() string {
	return "accounts"
}

// AccountOnboarding 账号注册引导画像（JSONB，迁移 000010）：
// 注册向导第 3 步采集的角色与了解渠道，运营侧用于新人画像分析，
// 值为空串表示未采集
type AccountOnboarding struct {
	Role    string `json:"role"`    // 角色：ceo / manager / it / member / teacher / student
	Channel string `json:"channel"` // 了解渠道：xiaohongshu / zhihu / referral / ...
}

func (o AccountOnboarding) Value() (driver.Value, error) {
	return json.Marshal(o)
}

func (o *AccountOnboarding) Scan(v interface{}) error {
	if v == nil {
		*o = AccountOnboarding{}
		return nil
	}
	switch data := v.(type) {
	case []byte:
		return json.Unmarshal(data, o)
	case string:
		return json.Unmarshal([]byte(data), o)
	default:
		return fmt.Errorf("cannot scan %T into AccountOnboarding", v)
	}
}

// AuthInfo 第三方登录凭证，归属账号（原挂 user，ADR-006 迁移；
// 存量 user_id 列由启动回填策略对齐 account_id，代码不再声明旧列）。
// (auth_type, auth_id) 租户无关唯一：部分唯一索引兜底（FIX-017）
type AuthInfo struct {
	ID           uint      `json:"id" gorm:"autoIncrement;primaryKey"`
	AccountId    uint      `json:"accountId" gorm:"index;not null;default:0"` // 存量回填前为 0（来源 user_id）
	Url          string    `json:"url" gorm:"size:256"`
	AuthType     string    `json:"authType" gorm:"size:256"`
	AuthId       string    `json:"authId" gorm:"size:256"`
	AccessToken  string    `json:"-" gorm:"size:256"`
	RefreshToken string    `json:"-" gorm:"size:256"`
	Expiry       time.Time `json:"-"`

	kernel.PlatformBaseModel // 平台级凭证，无租户归属
}

func (*AuthInfo) TableName() string {
	return "auth_infos"
}

// CreatedAccount 注册请求：创建账号并同时在默认租户建立成员关系（保持单租户默认体验）
type CreatedAccount struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Avatar   string `json:"avatar"`
}

func (c *CreatedAccount) GetAccount() *Account {
	return &Account{
		Name:     c.Name,
		Phone:    c.Phone,
		Email:    c.Email,
		Password: c.Password,
		Avatar:   c.Avatar,
	}
}
