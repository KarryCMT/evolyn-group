package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	kernel "evolyn/internal/model"
)

// 管理组 scope（与前端 AdministratorScope 对齐）：
// system=系统管理员页（内置系统管理组 + 通讯录管理组），
// application=灵衍云管理员页（普通管理组，管应用 + 分发范围）
const (
	AdminGroupScopeSystem      = "system"
	AdminGroupScopeApplication = "application"
)

// 范围模式：all=全部（无需 ID 清单），partial=部分（ID 清单）
const (
	AdminScopeAll     = "all"
	AdminScopePartial = "partial"
)

// AdminGroupResource 管理组资源（/admin-groups 路由前缀）：管理组自身的读写
// 仅授予租户管理员；该资源永不经管理组授予，防止通讯录管理组自我扩权
const AdminGroupResource = "admin-groups"

// TenantAdminRoleName 租户管理员角色名（内置系统管理员组的成员事实源）。
// 常量原驻 tenant/service，iam 不能反向依赖租户域服务，收敛到模型层共用
const TenantAdminRoleName = "租户管理员"

// AdminGroupBuiltinName 内置系统管理员组展示名（租户开通与存量回填共用）
const AdminGroupBuiltinName = "系统管理员"

// AdminGroup 管理组（权限中心-管理员模块）：一组成员 + 对某类管理对象
// （部门/角色/应用/互联组织）的带范围委托管理权。
// 内置组（built_in=true，scope=system）具备全量管理权，成员由 tenant-admin
// 角色绑定实时推导、不落 tn_admin_group_members 表；租户创建人在开通时绑定
// tenant-admin，是内置组不可移除的固定成员；scope_config 仅自定义组生效
type AdminGroup struct {
	ID   uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	Name string `json:"name" gorm:"size:30;not null"` // 租户内唯一：服务层预检 + 部分唯一索引兜底
	// Scope 限定 system/application，决定 scope_config 各区块的语义
	Scope   string `json:"scope" gorm:"size:16;not null"`
	BuiltIn bool   `json:"builtIn" gorm:"not null"` // bool 不带 gorm default tag：零值 false 必须显式写入（同 MemberFieldSetting 注释的坑）
	// ScopeConfig 出网不直接透出：详情接口经 AdminGroupDetailView 展开
	ScopeConfig AdminGroupScopeConfig `json:"-" gorm:"type:jsonb"`

	kernel.TenantBaseModel
}

func (*AdminGroup) TableName() string { return "tn_admin_groups" }

// AdminGroupMember 管理组成员绑定：移除即删行（不做软删），变更流水走审计域。
// 内置系统管理员组不使用本表
type AdminGroupMember struct {
	ID           uint `json:"id" gorm:"autoIncrement;primaryKey"`
	AdminGroupID uint `json:"adminGroupId" gorm:"not null;uniqueIndex:uk_tn_admin_group_member,priority:1"`
	MemberID     uint `json:"memberId" gorm:"not null;uniqueIndex:uk_tn_admin_group_member,priority:2"`
	// TenantID 显式声明：seed/回填路径可能无租户上下文，与 Callback 回填口径一致
	TenantID  uint      `json:"tenantId" gorm:"not null;index"`
	CreatedAt time.Time `json:"createdAt"`
}

func (*AdminGroupMember) TableName() string { return "tn_admin_group_members" }

// AdminDepartmentScope 部门范围。system 组为通讯录管理范围（Enabled 开关 +
// 全部/部分部门）；application 组为使用权分发范围（主行无开关，Enabled 恒 true）
type AdminDepartmentScope struct {
	Enabled       bool   `json:"enabled"`
	Mode          string `json:"mode"` // all | partial
	DepartmentIDs []uint `json:"departmentIds"`
}

// AdminRoleScope 角色范围：可见/可管理为两项独立授权（对齐前端两个独立勾选）
type AdminRoleScope struct {
	Visible bool   `json:"visible"`
	Manage  bool   `json:"manage"`
	Mode    string `json:"mode"` // all | partial
	RoleIDs []uint `json:"roleIds"`
}

// AdminExternalOrgScope 互联组织范围：互联组织域未落地，配置先存不参与执行
type AdminExternalOrgScope struct {
	Enabled bool `json:"enabled"`
}

// AdminApplicationScope 应用范围（仅 application 组）：可编辑应用集合 +
// 可添加/删除应用。AllApplications=true 为语义全量（新建应用自动纳入），
// 避免存全量 ID 清单在应用增删后漂移
type AdminApplicationScope struct {
	AllApplications bool   `json:"allApplications"`
	ApplicationIDs  []uint `json:"applicationIds"`
	Manage          bool   `json:"manage"`
}

// AdminAddressBookScope 通讯录管理子配置（仅 application 组的设置抽屉）：
// 应用管理员附带的一小块通讯录委托，与主行的分发范围（Department/Role）解耦
type AdminAddressBookScope struct {
	DepartmentEnabled bool `json:"departmentEnabled"`
	RoleVisible       bool `json:"roleVisible"`
	RoleManage        bool `json:"roleManage"`
	ExternalEnabled   bool `json:"externalEnabled"`
}

// AdminGroupScopeConfig 管理组范围配置（scope_config JSONB 单列，先例 roles.rules）：
// 区块指针 nil 表示不适用/未配置。system 组使用 Department/Role/ExternalOrg；
// application 组另加 Application/AddressBook。ID 清单的悬挂引用（部门/角色被删）
// 由读取侧解析时静默丢弃，不做删除钩子反向清理
type AdminGroupScopeConfig struct {
	Department  *AdminDepartmentScope  `json:"department,omitempty"`
	Role        *AdminRoleScope        `json:"role,omitempty"`
	ExternalOrg *AdminExternalOrgScope `json:"externalOrg,omitempty"`
	Application *AdminApplicationScope `json:"application,omitempty"`
	AddressBook *AdminAddressBookScope `json:"addressBook,omitempty"`
}

// Scan/Value 对齐 Rules 的 JSONB 口径（postgres 驱动以 []byte 回传）
func (c *AdminGroupScopeConfig) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}
	if len(bytes) == 0 {
		*c = AdminGroupScopeConfig{}
		return nil
	}
	return json.Unmarshal(bytes, c)
}

func (c AdminGroupScopeConfig) Value() (driver.Value, error) {
	b, err := json.Marshal(c)
	return string(b), err
}

// AdminGroupMemberView 管理组成员展示项：Name 为租户内展示名（昵称回落账号
// 昵称/登录名），Department 为首个部门名（多部门取其一，仅供标签展示）
type AdminGroupMemberView struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Department string `json:"department"`
}

// AdminGroupDetailView 管理组详情（管理员页权限面板读模型）：字段名与前端
// AdministratorGroup 一一对齐；各类 ID 清单返回原始 ID，展示名由前端结合
// 部门树/角色树/应用列表映射（后端不做名称反查，悬挂 ID 前端自然不渲染）
type AdminGroupDetailView struct {
	ID      uint                   `json:"id"`
	Name    string                 `json:"name"`
	Scope   string                 `json:"scope"`
	BuiltIn bool                   `json:"builtIn"`
	Members []AdminGroupMemberView `json:"members"`

	DepartmentEnabled bool   `json:"departmentEnabled"`
	DepartmentMode    string `json:"departmentMode"`
	DepartmentIDs     []uint `json:"departmentIds"`
	RoleVisible       bool   `json:"roleVisible"`
	RoleManage        bool   `json:"roleManage"`
	RoleMode          string `json:"roleMode"`
	RoleIDs           []uint `json:"roleIds"`
	ExternalEnabled   bool   `json:"externalEnabled"`

	// 以下仅 application 组返回
	ApplicationIDs    []uint                 `json:"applicationIds,omitempty"`
	AllApplications   bool                   `json:"allApplications"`
	ApplicationManage bool                   `json:"applicationManage"`
	AddressBook       *AdminAddressBookScope `json:"addressBook,omitempty"`
}

// AdminGroupSummary 管理组列表概要：内置组排最前，MemberCount 内置组为
// tenant-admin 角色绑定数量（包含租户创建人，实时推导），自定义组为成员表计数
type AdminGroupSummary struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Scope       string `json:"scope"`
	BuiltIn     bool   `json:"builtIn"`
	MemberCount int    `json:"memberCount"`
}

// MemberAdminScopes 当前成员的管理组身份聚合（/auth/admin-scopes）：
// SystemAdmin 即内置系统管理员组身份（tenant-admin 角色绑定），供前端
// 菜单/页面入口判定；Groups 仅含自定义管理组
type MemberAdminScopes struct {
	SystemAdmin bool                `json:"systemAdmin"`
	Groups      []AdminGroupSummary `json:"groups"`
}
