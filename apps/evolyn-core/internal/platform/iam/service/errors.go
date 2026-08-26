package service

import (
	"net/http"

	"evolyn/internal/platform/httpx"
)

// 稳定业务错误码（整改 FIX-002/003/004/006；ADR-008 起承载于 BizError）：
// 错误文本即对外错误码的历史口径由 Code 字段承接，调用方 errors.Is 判定不变；
// 细节通过 httpx.Wrap 附加（只入日志，不出网）
var (
	// ErrCrossTenantBinding 跨租户关系绑定被拒绝（FIX-006）：
	// Member/Group/Department/Role 两端必须属于同一租户
	ErrCrossTenantBinding = httpx.NewBiz("CROSS_TENANT_BINDING_REJECTED", "跨租户的关系绑定被拒绝", http.StatusForbidden)

	// ErrDuplicateName 租户内重名（FIX-002/003）：Role/Group 名称租户内唯一
	ErrDuplicateName = httpx.NewBiz("DUPLICATE_NAME", "名称已存在", http.StatusConflict)

	// ErrDuplicateMember 同租户重复成员（FIX-004）：一个账号在一个租户仅一个有效成员
	ErrDuplicateMember = httpx.NewBiz("DUPLICATE_MEMBER", "该账号已是本租户成员", http.StatusConflict)

	// ErrCredentialsInvalid 密码登录凭证错误（ADR-008）：账号不存在与密码错误
	// 统一文案，不泄露账号存在性；401
	ErrCredentialsInvalid = httpx.NewBiz("AUTH_CREDENTIALS_INVALID", "账号或密码错误", http.StatusUnauthorized)

	// ErrAccountNotFound 短信登录手机号未注册（ADR-008）：验证码已通过，
	// 区别于凭证错误，可引导走注册；401
	ErrAccountNotFound = httpx.NewBiz("AUTH_ACCOUNT_NOT_FOUND", "该手机号未注册", http.StatusUnauthorized)

	// ErrNotMember 账号与目标租户无成员关系（ADR-008）：指定租户登录/切换租户；
	// 账号名/租户编码等细节经 Wrap 只入日志；403
	ErrNotMember = httpx.NewBiz("AUTH_NOT_MEMBER", "该账号不属于此租户", http.StatusForbidden)

	// ErrDuplicatePhone 手机号已被其他账号绑定（上线前整改 P2，换绑流程）：
	// 手机号全局唯一（部分唯一索引 uk_accounts_phone 兜底）；409
	ErrDuplicatePhone = httpx.NewBiz("DUPLICATE_PHONE", "该手机号已被其他账号绑定", http.StatusConflict)

	// ErrPhoneInvalid 手机号格式非法（与 sms 域 AUTH_PHONE_INVALID 同码同文案：
	// iam 不反向依赖认证域，就地复制口径）
	ErrPhoneInvalid = httpx.NewBiz("AUTH_PHONE_INVALID", "手机号格式不正确", http.StatusBadRequest)

	// ErrEmailBindRequired 邮箱属于账号安全凭据，必须走手机号身份验证和邮箱
	// 验证码双重校验链路，禁止经普通资料更新接口绕过。
	ErrEmailBindRequired = httpx.NewBiz("AUTH_EMAIL_BIND_REQUIRED", "邮箱请通过绑定流程修改", http.StatusBadRequest)

	// ErrPhoneNotBound 无当前手机号时无法完成邮箱绑定所需的身份持有证明。
	ErrPhoneNotBound = httpx.NewBiz("AUTH_PHONE_NOT_BOUND", "当前账号未绑定手机号", http.StatusBadRequest)
	// ErrMemberStatusInvalid 成员状态仅允许 active、disabled、resigned。
	ErrMemberStatusInvalid = httpx.NewBiz("MEMBER_STATUS_INVALID", "成员状态不合法", http.StatusBadRequest)
	// ErrOrganizationRoleGroupNameInvalid 角色组名称为空或仅含空白字符。
	ErrOrganizationRoleGroupNameInvalid = httpx.NewBiz("ROLE_GROUP_NAME_INVALID", "角色组名称不合法", http.StatusBadRequest)
	// ErrOrganizationRoleRequestInvalid 角色分组、角色成员操作的路径或请求参数无效。
	ErrOrganizationRoleRequestInvalid = httpx.NewBiz("ORGANIZATION_ROLE_REQUEST_INVALID", "角色操作参数不合法", http.StatusBadRequest)
	// ErrMemberInvitationInvalid 邀请成员档案字段不符合通讯录模板约束。
	ErrMemberInvitationInvalid = httpx.NewBiz("MEMBER_INVITATION_INVALID", "成员邀请信息不合法", http.StatusBadRequest)
	// ErrMemberInvitationContactRequired 邀请必须至少提供一个实际可联系的方式。
	ErrMemberInvitationContactRequired = httpx.NewBiz("MEMBER_INVITATION_CONTACT_REQUIRED", "手机号和邮箱至少填写一项", http.StatusBadRequest)
	// ErrMemberInvitationImportFile 上传文件不是有效的通讯录 Excel 模板或超过大小限制。
	ErrMemberInvitationImportFile = httpx.NewBiz("MEMBER_INVITATION_IMPORT_FILE_INVALID", "请上传不超过 5MB 的有效通讯录模板", http.StatusBadRequest)
	// ErrMemberFieldNotFound 预置字段 key 不在服务端注册表中（成员信息管理一期）。
	ErrMemberFieldNotFound = httpx.NewBiz("MEMBER_FIELD_NOT_FOUND", "成员字段不存在", http.StatusNotFound)
	// ErrMemberFieldLocked 试图修改注册表锁定的配置项（可见/可编辑/卡片固定）。
	ErrMemberFieldLocked = httpx.NewBiz("MEMBER_FIELD_LOCKED", "该字段配置不允许修改", http.StatusForbidden)
	// ErrMemberFieldConfigInvalid 可编辑与可见联动规则冲突或提交值非法。
	ErrMemberFieldConfigInvalid = httpx.NewBiz("MEMBER_FIELD_CONFIG_INVALID", "字段配置不合法", http.StatusBadRequest)
	// ErrMemberFieldConfigConflict revision 过期：配置已被其他管理员修改，前端应以最新快照重试。
	ErrMemberFieldConfigConflict = httpx.NewBiz("MEMBER_FIELD_CONFIG_CONFLICT", "配置已更新，请刷新后重试", http.StatusConflict)
	// ErrMemberProfileInvalid 扩展资料字段、日期格式或长度不合法；也用于拒绝
	// 经通用资料接口提交的非扩展字段（手机/邮箱/部门/角色走专用接口）。
	ErrMemberProfileInvalid = httpx.NewBiz("MEMBER_PROFILE_INVALID", "成员资料不合法", http.StatusBadRequest)
	// ErrMemberInvitationAcceptInvalid 单人邀请 token、状态或受邀身份校验失败。
	ErrMemberInvitationAcceptInvalid = httpx.NewBiz("MEMBER_INVITATION_ACCEPT_INVALID", "邀请不存在、已失效或与当前账号不匹配", http.StatusBadRequest)
)
