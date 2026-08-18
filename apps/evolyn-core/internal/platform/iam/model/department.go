package model

import (
	kernel "evolyn/internal/model"
)

// 部门状态：disabled 表示停用（离职归档语义随 P2 生命周期细化）
const (
	DeptActive   = "active"
	DeptDisabled = "disabled"
)

// Department 部门（租户内组织架构，邻接表树：ParentId=0 为根）。
// 与 Group（权限分组）语义分离：部门承载组织结构与汇报关系，
// 角色授权仍走 Group/Role（架构文档 26.1）
type Department struct {
	ID       uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	ParentId uint   `json:"parentId" gorm:"index;not null;default:0"` // 0 = 根节点
	Name     string `json:"name" gorm:"size:100;not null"`
	Order    int    `json:"order" gorm:"not null;default:0"` // 同级排序，小在前
	Status   string `json:"status" gorm:"size:16;not null;default:active"`

	kernel.BaseModel
}

func (*Department) TableName() string {
	return "departments"
}
