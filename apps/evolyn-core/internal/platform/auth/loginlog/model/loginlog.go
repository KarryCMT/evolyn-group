// Package model 登录日志域模型（000013）：会话建立事件流水。
package model

import kernel "evolyn/internal/model"

// 登录方式常量（method 列取值；语义值由服务层契约约定，模型层只落库）
const (
	MethodPassword = "password" // 账号密码登录
	MethodSMS      = "sms"      // 短信验证码登录
	MethodRegister = "register" // 注册即登录（注册向导最终提交，含已注册幂等重放）
	MethodOAuth    = "oauth_"   // 第三方登录前缀，拼接 provider：oauth_github / oauth_wechat
)

// 客户端形态常量（client 列取值，由 User-Agent 粗解析）
const (
	ClientWeb     = "web"     // 电脑网页
	ClientWap     = "wap"     // 手机网页（含平板）
	ClientUnknown = "unknown" // UA 缺失或不可识别
)

// LoginLog 登录日志：账号维度追加写流水，无更新/软删语义。
// 平台级资源（登录发生在租户上下文建立之前，ADR-006）——tenant_id/member_id
// 为本次会话绑定的快照，刻意不挂 TenantBaseModel：避免 GORM 租户 Callback 把
// 「按账号查」污染成「按租户查」（与 audit_logs 同口径）
type LoginLog struct {
	ID        uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	AccountID uint   `json:"accountId" gorm:"index;not null"`                // 登录账号（主查询维度）
	TenantID  uint   `json:"tenantId" gorm:"not null;default:0"`             // 本次登录进入的租户；0=无租户/平台场景
	MemberID  uint   `json:"memberId" gorm:"not null;default:0"`             // 本次登录绑定的成员；0=未解析到
	Method    string `json:"method" gorm:"size:32;not null"`                 // 登录方式，见 Method* 常量
	Client    string `json:"client" gorm:"size:32;not null;default:unknown"` // 客户端形态，见 Client* 常量
	IP        string `json:"ip" gorm:"size:64"`                              // 来源 IP
	Location  string `json:"location" gorm:"size:128"`                       // IP 归属地（写时离线解析）
	UserAgent string `json:"userAgent" gorm:"size:256"`
	RequestID string `json:"requestId" gorm:"size:64"`
	// 登录人显示名快照（000036 企业日志）：写时固化，成员改名/离职/删除后
	// 历史展示一致；存量历史行为空串，企业日志读取侧回查当前成员昵称兜底
	ActorNameSnapshot string          `json:"actorNameSnapshot" gorm:"size:128;not null;default:''"`
	CreatedAt         kernel.JSONTime `json:"createdAt"` // 登录时间（JSONTime 秒级东八区出网）
}

func (*LoginLog) TableName() string {
	return "login_logs"
}
