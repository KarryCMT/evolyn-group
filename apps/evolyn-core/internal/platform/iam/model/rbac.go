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
)

type Scope string

const (
	ClusterScope Scope = "cluster"
)

// Role 租户内角色（Resource 是平台级目录，不挂租户）。
// FIX-001：内嵌 TenantBaseModel，补齐 created_at/updated_at/deleted_at
// 与 db.sql 对齐，删除统一为软删除（物理删除需显式 Unscoped）
type Role struct {
	ID        uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	Name      string `json:"name" gorm:"size:100;not null;index"` // 租户内唯一：服务层预检 + 部分唯一索引兜底（FIX-002）
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
	UserResource  = "users"
	GroupResource = "groups"
	RoleResource  = "roles"
	AuthResource  = "auth"
)

type Resource struct {
	ID    uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	Name  string `json:"name" gorm:"size:256;not null;unique"`
	Scope Scope  `json:"scope"`
	Kind  string `json:"kind"`
}
