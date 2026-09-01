package model

import kernel "evolyn/internal/model"

// RoleGroup 是内部组织页的角色展示分组。它只负责归类角色，不具有 Group 的
// 成员与权限继承语义；一个角色在页面上只能归属一个角色组。
type RoleGroup struct {
	ID   uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	Name string `json:"name" gorm:"size:100;not null;index"`
	// CreatorMemberID 是角色组的租户成员归属信息；公共 CreatorID 记录创建
	// 账号，二者不能复用同一列。
	CreatorMemberID uint `json:"creatorMemberId" gorm:"column:creator_member_id"`
	// Sort 是角色组在内部组织左树中的稳定展示顺序，数值越小越靠前。
	Sort int `json:"sort" gorm:"not null;default:0"`

	kernel.TenantBaseModel
}

const DefaultRoleGroupName = "默认"
