package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	kernel "evolyn/internal/model"

	"evolyn/internal/utils/set"

	"evolyn/internal/utils/request"
)

const (
	All = "*"
	// ClusterAdminRole 是平台管理角色的中文展示名；平台鉴权使用该常量识别角色。
	ClusterAdminRole = "平台管理员"
)

type Scope string

const (
	ClusterScope Scope = "cluster"
)

// Role 租户内角色（Resource 是平台级目录，不挂租户）。
// FIX-001：内嵌 TenantBaseModel，补齐 created_at/updated_at/deleted_at
// 与 db.sql 对齐，删除统一为软删除（物理删除需显式 Unscoped）
type Role struct {
	ID          uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	Name        string `json:"name" gorm:"size:100;not null;index"` // 租户内唯一：服务层预检 + 部分唯一索引兜底（FIX-002）
	RoleGroupID *uint  `json:"roleGroupId" gorm:"index"`            // 组织页展示分组；不参与权限继承，避免与 Group 混用。
	// Sort 仅定义角色在所属展示分组中的顺序，数值越小越靠前。
	Sort      int    `json:"sort" gorm:"not null;default:0"`
	Scope     Scope  `json:"scope" gorm:"size:100"`
	Namespace string `json:"namespace"  gorm:"size:100"`
	Rules     Rules  `json:"rules" gorm:"type:json"`

	kernel.TenantBaseModel
}

const (
	AllOperation  Operation = "*"
	EditOperation Operation = "edit"
	ViewOperation Operation = "view"
)

type Operation string

var (
	EditOperationSet = set.NewString(request.CreateOperation, request.DeleteOperation, request.UpdateOperation, request.PatchOperation, request.GetOperation, request.ListOperation)
	ViewOperationSet = set.NewString(request.GetOperation, request.ListOperation)
)

func (op Operation) Contain(verb string) bool {
	switch op {
	case AllOperation:
		return true
	case EditOperation:
		return EditOperationSet.Has(verb)
	case ViewOperation:
		return ViewOperationSet.Has(verb)
	default:
		return string(op) == verb
	}
}

type Rule struct {
	Resource  string    `json:"resource"`
	Operation Operation `json:"operation"`
}

type Rules []Rule

func (r *Rules) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}

	result := Rules{}
	err := json.Unmarshal(bytes, &result)
	*r = result
	return err
}

func (r Rules) Value() (driver.Value, error) {
	b, err := json.Marshal(r)
	return string(b), err
}

const (
	ResourceKind = "resource"
	MenuKind     = "menu"
)

const (
	UserResource = "users"
	// MemberResource 与 /members 路由保持一致，代表租户成员的管理权限。
	// 账号自身资料仍通过 accounts 资源控制，避免把平台账号与租户成员混为一谈。
	MemberResource = "members"
	GroupResource  = "groups"
	RoleResource   = "roles"
	AuthResource   = "auth"
	TenantResource = "tenant"
	// MemberFieldSettingResource 与 /member-field-settings 路由保持一致，
	// 代表成员信息管理（字段设置/卡片展示）的租户级配置权限。
	MemberFieldSettingResource = "member-field-settings"
	// TenantProductResource 与 /tenant-products 路由保持一致，代表产品中心
	// （平台内置产品的启停与可用范围配置）的租户级管理权限。运行时产品
	// 访问不等同于本资源：命中范围的普通成员无需本权限即可使用产品。
	TenantProductResource = "tenant-products"
	// EnterpriseLogResource 与 /enterprise-logs 路由保持一致，代表企业日志
	//（登录日志/操作日志的只读查询与导出）权限：view 覆盖查询、create 覆盖
	// 导出任务创建（enterprise-logs:export 语义）。该资源不接受管理组间接
	// 放行——如需委派应增加细粒度数据范围设计，而非复用管理组全量范围。
	EnterpriseLogResource = "enterprise-logs"
	// FormResource 与 /forms 路由保持一致，代表表单资产的设计与管理权限
	//（创建/列表/详情/改名/草稿/发布/删除，ADR-010）。发布复用 create 动词。
	FormResource = "forms"
	// FormRecordResource 与 /form-records 路由保持一致，代表表单记录提交
	//（create 动词授予全体成员）：填写提交与表单设计权限彻底分离。
	FormRecordResource = "form-records"
	// NotificationResource 与 /notifications 路由保持一致，代表成员收件箱
	//（view 覆盖摘要/列表、update 覆盖已读，授予全体成员）；数据范围只能
	// 是「当前租户 × 当前成员」，由 Repository 双条件兜底。
	NotificationResource = "notifications"
	// NotificationSettingResource 与 /notification-settings 路由保持一致，
	// 代表租户级通知偏好与自定义提醒对象管理（仅授予租户管理员；包含外部
	// 联系人隐私数据，不经管理组范围回落放行）。
	NotificationSettingResource = "notification-settings"
	// FormMenuActionResource 表单菜单按钮动作授权资源（ADR-011）。刻意不对应
	// 任何 URL 首段：中间件 URL 门永不命中，动作键（switch-type/copy-in-app/
	// copy-cross-app/hide）只在各域 Service 内按权限集复核，并由菜单读取经
	// authorization.MenuActionsOf 投影按钮能力——动作授权因此不可能越权放大
	// URL 访问。
	FormMenuActionResource = "form-actions"
	// MenuFavoriteResource 与 /menu-favorites 路由保持一致，代表成员对应用
	// 菜单节点的个人收藏（个人状态而非授权对象：凡节点可见即可收藏）。
	// create 授予全体成员（收藏），delete 覆盖取消收藏，数据范围由
	// Repository 强制 member_id 双条件兜底。
	MenuFavoriteResource = "menu-favorites"
)

type Resource struct {
	ID    uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	Name  string `json:"name" gorm:"size:256;not null;unique"`
	Scope Scope  `json:"scope"`
	Kind  string `json:"kind"`
}
