package service

import (
	"context"

	"evolyn/internal/platform/iam/model"
)

// ctx 约定：触及数据访问的方法统一以 ctx 为首参，由 controller 从
// c.Request.Context() 透传；租户 ID 藏在 ctx 中由仓储层 GORM Callback
// 自动注入，service 不感知租户细节。纯计算方法（Validate/Default/
// ListOperations）不落库，不携带 ctx。

// AccountService 账号服务（登录身份，平台级）
type AccountService interface {
	// Auth 账号密码校验并解析登录成员（TenantCode 可选指定目标租户）
	Auth(ctx context.Context, auser *model.AuthUser) (*model.Account, *model.User, error)
	// AuthByPhone 验证码登录：手机号定位账号并解析登录成员（验证码由调用方校验）
	AuthByPhone(ctx context.Context, phone, tenantCode string) (*model.Account, *model.User, error)
	// PhoneRegistered 手机号是否已注册（登录场景发码前校验：未注册不发短信，
	// 不做成员解析等副作用，纯存在性查询）
	PhoneRegistered(ctx context.Context, phone string) (bool, error)
	// Register 注册：创建账号 + 默认租户成员
	Register(ctx context.Context, account *model.Account) (*model.Account, *model.User, error)
	// RegisterByPhone 短信免密注册：已注册手机号等价短信登录（created=false），
	// 否则服务端生成随机登录名/密码建号（验证码由调用方经 sms 域校验）
	RegisterByPhone(ctx context.Context, phone string) (*model.Account, *model.User, bool, error)
	// CreateOAuthAccount OAuth 链路：复用或创建账号，返回默认成员
	CreateOAuthAccount(ctx context.Context, account *model.Account) (*model.Account, *model.User, error)
	// ListTenants 账号的成员关系列表（含 isOwner）
	ListTenants(ctx context.Context, accountID uint) ([]TenantMembership, error)
	// SwitchTenant 校验并返回切换租户后的账号+成员
	SwitchTenant(ctx context.Context, accountID, tenantID uint) (*model.Account, *model.User, error)
	// GetUserInfo 登录聚合信息（账号+成员+租户/套餐）
	GetUserInfo(ctx context.Context, accountID uint, member *model.User) (*UserInfoResult, error)
	// ResetPasswordByPhone 密码找回：凭 scene=reset 验证码重设（P1-3）
	ResetPasswordByPhone(ctx context.Context, phone, newPassword string) error
	// 账号自助（P3-2）
	GetProfile(ctx context.Context, accountID uint) (*model.Account, error)
	UpdateProfile(ctx context.Context, account *model.Account) (*model.Account, error)
	// BindEmail 仅由已完成双重验证码校验的控制器调用；email 已经认证域规范化。
	BindEmail(ctx context.Context, accountID uint, email string) (*model.Account, error)
	ChangePassword(ctx context.Context, accountID uint, oldPassword, newPassword string) error
	// EnsurePhoneAvailable 换绑手机号可用性预检（格式 + 未被占用）：
	// 供控制器在消费一次性短信验证码前调用，避免号码已占用时白白耗码
	EnsurePhoneAvailable(ctx context.Context, phone string) error
	// ChangePhone 换绑手机号落库（上线前整改 P2）：旧/新手机号验证码由调用方
	// 经 sms 域（scene=rebind）校验，本方法只负责查重（DB 唯一索引兜底）与写库
	ChangePhone(ctx context.Context, accountID uint, newPhone string) (*model.Account, error)
	Validate(*model.Account) error
	Default(*model.Account)
}

// DepartmentService 部门服务（租户内组织架构，P3-2）
type DepartmentService interface {
	List(ctx context.Context) ([]model.Department, error)
	Tree(ctx context.Context) ([]*DepartmentNode, error)
	Create(ctx context.Context, dept *model.Department) (*model.Department, error)
	Update(ctx context.Context, id string, dept *model.Department) (*model.Department, error)
	Delete(ctx context.Context, id string) error
	SetMemberDepartments(ctx context.Context, memberID string, departmentIDs []uint) error
}

// UserService 成员服务（租户内身份）
type UserService interface {
	List(ctx context.Context) (model.Users, error)
	ListPage(ctx context.Context, query model.MemberListQuery) (*model.MemberPage, error)
	Get(ctx context.Context, id string) (*model.User, error)
	Update(ctx context.Context, id string, member *model.User) (*model.User, error)
	UpdateStatus(ctx context.Context, id, status string) (*model.User, error)
	Delete(ctx context.Context, id string) error
	GetGroups(ctx context.Context, id string) ([]model.Group, error)
	AddRole(ctx context.Context, id, rid string) error
	DelRole(ctx context.Context, id, rid string) error
	// AddMember 「Account 成为租户成员」入口（FIX-010）：校验账号/租户/配额/
	// 重复成员后创建成员，并按需绑定部门与角色（同租户校验，FIX-006）
	AddMember(ctx context.Context, req *AddMemberRequest) (*model.User, error)
}

// MemberInvitationService 管理手动邀请、通讯录模板导入和公开邀请链接。
type MemberInvitationService interface {
	Create(ctx context.Context, inviterMemberID uint, req MemberInvitationRequest) (*model.MemberInvitation, error)
	Import(ctx context.Context, inviterMemberID uint, content []byte) (*model.MemberInvitationBatchResult, error)
	GetPublicLink(ctx context.Context) (*model.TenantPublicInvitationLink, error)
	UpdatePublicLink(ctx context.Context, inviterMemberID uint, enabled bool) (*model.TenantPublicInvitationLink, error)
	AcceptPublicLink(ctx context.Context, accountID uint, nickname, token string) (*model.User, error)
	// AcceptPersonalInvite 消费单人邀请 token（文档 5.3）：单事务完成校验 →
	// 创建正式成员 → 迁移邀请档案与部门归属 → 邀请状态置 accepted
	AcceptPersonalInvite(ctx context.Context, accountID uint, token string) (*model.User, error)
}

// MemberFieldService 成员字段配置（成员信息管理）：租户级显示策略读取与即时更新
type MemberFieldService interface {
	// GetSnapshot 完整配置快照（15 预置字段恒完整，读取侧幂等补齐默认行）
	GetSnapshot(ctx context.Context) (*model.MemberFieldConfigSnapshot, error)
	// UpdateField 单字段即时更新（乐观锁 + 锁定/联动校验），返回最新整页快照
	UpdateField(ctx context.Context, fieldKey string, req *MemberFieldSettingUpdateRequest) (*model.MemberFieldConfigSnapshot, error)
	// SeedDefaults 为指定租户预置默认配置（租户开通事务内调用，幂等）
	SeedDefaults(ctx context.Context, tenantID uint) error
}

// MemberFieldSettingUpdateRequest 单字段即时更新请求：指针为 nil 表示本次
// 不变更该开关；revision 为页面读取到的租户配置版本号（乐观锁）
type MemberFieldSettingUpdateRequest struct {
	PersonalVisible  *bool `json:"personalVisible"`
	PersonalEditable *bool `json:"personalEditable"`
	CardVisible      *bool `json:"cardVisible"`
	Revision         int64 `json:"revision"`
}

// MemberProfileService 正式成员扩展档案：本人视图（按字段配置裁剪）与管理员
// 视图（全量 + 卡片裁剪）的读取与更新
type MemberProfileService interface {
	// GetMyProfile 本人资料：Values 按 personalVisible 裁剪，EditableKeys
	// 为允许提交的扩展字段集合
	GetMyProfile(ctx context.Context, memberID uint) (*model.MemberProfileView, error)
	// UpdateMyProfile 本人更新：仅接受 personalEditable 的扩展字段
	UpdateMyProfile(ctx context.Context, memberID uint, values map[string]string) (*model.MemberProfileView, error)
	// GetMemberProfile 管理员读取：全量值 + cardVisible 裁剪的卡片视图 + 字段元数据
	GetMemberProfile(ctx context.Context, memberID uint) (*model.MemberProfileAdminView, error)
	// UpdateMemberProfile 管理员维护：扩展字段与企业内编号 identifier
	UpdateMemberProfile(ctx context.Context, memberID uint, req *MemberProfileUpdateRequest) (*model.MemberProfileAdminView, error)
}

// MemberProfileUpdateRequest 管理员维护成员档案请求：Values 为扩展字段
// （key → 值，整体合并覆盖提交项）；Identifier 指针非空才变更编号
type MemberProfileUpdateRequest struct {
	Identifier *string           `json:"identifier"`
	Values     map[string]string `json:"values"`
}

// AddMemberRequest 拉人入租户请求：AccountID 与 AccountName 二选一
type AddMemberRequest struct {
	AccountID     uint   `json:"accountId"`
	AccountName   string `json:"accountName"`
	Nickname      string `json:"nickname"`
	DepartmentIDs []uint `json:"departmentIds"`
	RoleIDs       []uint `json:"roleIds"`
}

// MemberInvitationRequest 对齐通讯录批量导入模板中的完整成员档案。
// 手机和邮箱至少填写一项；日期按 YYYY-MM-DD 字符串保存，避免时区改变原始档案日期。
type MemberInvitationRequest struct {
	Name            string   `json:"name"`
	Identifier      string   `json:"identifier"`
	Phone           string   `json:"phone"`
	Email           string   `json:"email"`
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

type GroupService interface {
	List(ctx context.Context) ([]model.Group, error)
	Create(ctx context.Context, user *model.User, group *model.Group) (*model.Group, error)
	Get(ctx context.Context, id string) (*model.Group, error)
	Update(ctx context.Context, id string, group *model.Group) (*model.Group, error)
	Delete(ctx context.Context, id string) error
	GetUsers(ctx context.Context, gid string) (model.Users, error)
	AddUser(ctx context.Context, user *model.User, gid string) error
	DelUser(ctx context.Context, user *model.User, gid string) error
	AddRole(ctx context.Context, id, rid string) error
	DelRole(ctx context.Context, id, rid string) error
}

type RBACService interface {
	List(ctx context.Context) ([]model.Role, error)
	Create(ctx context.Context, role *model.Role) (*model.Role, error)
	Get(ctx context.Context, id string) (*model.Role, error)
	Update(ctx context.Context, id string, role *model.Role) (*model.Role, error)
	Delete(ctx context.Context, id string) error
	ListResources(ctx context.Context) ([]model.Resource, error)
	ListOperations() ([]model.Operation, error)
}
