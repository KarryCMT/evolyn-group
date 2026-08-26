package model

import (
	kernel "evolyn/internal/model"
)

// CreateBlankRequest 创建空白应用请求（POST /applications，§8.1）。
// icon/color 为稳定枚举键，可省略由服务端取默认；租户/owner/code 等
// 服务端字段一律不由客户端传入
type CreateBlankRequest struct {
	Name  string `json:"name" binding:"required" example:"测试应用"`
	Icon  string `json:"icon" example:"bookmark"`
	Color string `json:"color" example:"primary"`
}

// UpdateApplicationRequest 更新应用请求（PATCH /applications/:id，§8.3）：
// 白名单字段 name/icon/color/sortOrder/status，指针区分「未传」与「传空」。
// status 仅允许 active↔archived 互转（归档/恢复，§5.4 复用 patch 动词）
type UpdateApplicationRequest struct {
	Name      *string `json:"name"`
	Icon      *string `json:"icon"`
	Color     *string `json:"color"`
	SortOrder *int64  `json:"sortOrder"`
	Status    *string `json:"status"`
}

// ApplicationSource 来源摘要（出网子对象，§8.1）
type ApplicationSource struct {
	Type    string `json:"type"`    // blank / template
	Channel string `json:"channel"` // self / template_center / admin / api
}

// ApplicationCapabilities 当前请求成员的运行时能力（§9.2）：读取时由
// 角色规则 + 是否 owner + 应用状态派生，不落库；M2-A 仅 view/edit/delete
type ApplicationCapabilities struct {
	View   bool `json:"view"`
	Edit   bool `json:"edit"`
	Delete bool `json:"delete"`
}

// ApplicationDetail 应用出网视图（创建/详情/列表条目共用）
type ApplicationDetail struct {
	ID              uint                    `json:"id"`
	Code            string                  `json:"code"`
	Name            string                  `json:"name"`
	Icon            string                  `json:"icon"`
	Color           string                  `json:"color"`
	Source          ApplicationSource       `json:"source"`
	Status          string                  `json:"status"`
	ProvisionStatus string                  `json:"provisionStatus"`
	HomeMode        string                  `json:"homeMode"`
	OwnerMemberID   uint                    `json:"ownerMemberId"`
	CreatorMemberID uint                    `json:"creatorMemberId"`
	SortOrder       int64                   `json:"sortOrder"`
	Capabilities    ApplicationCapabilities `json:"capabilities"`
	CreatedAt       kernel.JSONTime         `json:"createdAt"`
	UpdatedAt       kernel.JSONTime         `json:"updatedAt"`
}

// ListApplicationsQuery 应用列表查询（§8.3）：keyword 按名称模糊、status
// 过滤（active/archived），cursor 为不透明 base64url 游标（内部编码
// sort_order+id），limit 默认 20、上限 100
type ListApplicationsQuery struct {
	Keyword string
	Status  string
	Limit   int
	Cursor  string
}

// ApplicationPage 游标分页结果：nextCursor 为空且 hasMore=false 表示到末页；
// 客户端只原样回传 nextCursor，不解析其内容
type ApplicationPage struct {
	Items      []ApplicationDetail `json:"items"`
	NextCursor string              `json:"nextCursor"`
	HasMore    bool                `json:"hasMore"`
}
