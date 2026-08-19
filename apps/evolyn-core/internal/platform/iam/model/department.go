package model

import (
	kernel "evolyn/internal/model"
)

// 部门状态：disabled 表示停用（离职归档语义随 P2 生命周期细化）
const (
	DeptActive   = "active"
	DeptDisabled = "disabled"
)

// Department 部门（租户内组织架构，邻接表树）。
// 与 Group（权限分组）语义分离：部门承载组织结构与汇报关系，
// 角色授权仍走 Group/Role（架构文档 26.1）
type Department struct {
	ID uint `json:"id" gorm:"autoIncrement;primaryKey"`
	// ParentId 父部门；NULL = 根节点（FIX-015：由 0=root 迁移为 NULL=root，
	// 非空时数据库层有自引用 FK，跨租户/不存在父由 FK 与服务层校验共同拦截）
	ParentId *uint  `json:"parentId" gorm:"index"`
	Name     string `json:"name" gorm:"size:100;not null"`
	Order    int    `json:"order" gorm:"not null;default:0"` // 同级排序，小在前
	Status   string `json:"status" gorm:"size:16;not null;default:active"`

	kernel.TenantBaseModel
}

func (*Department) TableName() string {
	return "departments"
}
