package model

import (
	// 共享内核（BaseModel）暂驻 internal/model
	kernel "evolyn/internal/model"
)

const (
	RootGroup            = "root"
	AuthenticatedGroup   = "system:authenticated"
	UnAuthenticatedGroup = "system:unauthenticated"
	SystemGroup          = "system"
	CustomGroup          = "custom"
)

type Group struct {
	ID       uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	Name     string `json:"name" gorm:"size:100;not null;index"` // 租户内唯一：服务层预检 + 部分唯一索引兜底（FIX-003）
	Kind     string `json:"kind" gorm:"size:100"`
	Describe string `json:"describe" gorm:"size:1024;"`
	Users    []User `json:"users" gorm:"many2many:tn_user_groups;"`
	Roles    []Role `json:"roles" gorm:"many2many:tn_group_roles;"`

	kernel.TenantBaseModel
}

// TableName 显式映射租户用户组表（000063 命名空间前缀 tn_），避免依赖
// GORM 默认推导
func (*Group) TableName() string { return "tn_groups" }

type CreatedGroup struct {
	Name     string `json:"name"`
	Describe string `json:"describe"`
}

func (g *CreatedGroup) GetGroup(_ uint) *Group {
	return &Group{
		Name:     g.Name,
		Describe: g.Describe,
	}
}

type UpdatedGroup struct {
	Name     string `json:"name"`
	Describe string `json:"describe"`
}

func (g *UpdatedGroup) GetGroup(_ uint) *Group {
	return &Group{
		Name:     g.Name,
		Describe: g.Describe,
	}
}
