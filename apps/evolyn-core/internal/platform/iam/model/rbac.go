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
)

type Resource struct {
	ID    uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	Name  string `json:"name" gorm:"size:256;not null;unique"`
	Scope Scope  `json:"scope"`
	Kind  string `json:"kind"`
}
