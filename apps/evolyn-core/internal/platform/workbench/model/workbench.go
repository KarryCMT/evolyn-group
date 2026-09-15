// Package model 自定义工作台域数据模型（000078 企业级配置定版）。表结构
// 唯一事实来源是 migrations/000078，本包只做 GORM 映射。每租户一行，
// content 承载与前端 @evolyn.do/dashboard 镜像的 DashboardSchema 单文档；
// 企业管理员配置、全员共用；行定位以 tenant_id 唯一定位
package model

import (
	"database/sql/driver"
	"encoding/json"

	kernel "evolyn/internal/model"
)

// TenantWorkbench 企业自定义工作台：租户开通事务内种子默认布局
// （WorkbenchSeeder，口径同通知设置/产品配置种子器），此后由企业管理员
// 经 /workbench 保存演进；revision 是保存乐观锁口令（种子 1，每次保存 +1）。
// 刻意不嵌 TenantBaseModel（无软删），仓储不依赖 GORM 租户 Callback 自动
// 过滤，一律显式 tenantID 条件定位
type TenantWorkbench struct {
	ID        uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	TenantID  uint            `json:"tenantId" gorm:"not null"`
	Content   Content         `json:"content" gorm:"type:jsonb;not null"`
	Revision  int64           `json:"revision" gorm:"not null;default:1"`
	CreatedAt kernel.JSONTime `json:"createdAt"`
	UpdatedAt kernel.JSONTime `json:"updatedAt"`
}

// TableName 显式映射租户工作台表（000063 命名空间前缀 tn_）
func (*TenantWorkbench) TableName() string { return "tn_workbenches" }

// DefaultWorkbenchDocument 租户开通种子与 000078 存量回填共用的默认工作台
// 文档，与前端 apps/web/src/dashboard/defaultWorkbench.ts 逐字镜像（卡片
// 类型/标题/坐标/约束/presetKey 一致）。修改默认布局时三处同步：本常量、
// 000078 迁移 SQL 的回填字面量、前端默认布局
const DefaultWorkbenchDocument = `{"version":1,"widgets":[
  {"id":"onboarding-0-0","type":"onboarding","title":"新手引导","x":0,"y":0,"w":12,"h":2,"noResize":true},
  {"id":"greeting-0-2","type":"greeting","title":"问候语","x":0,"y":2,"w":3,"h":1,"minW":3,"minH":1,"maxH":1,"presetKey":"greeting"},
  {"id":"favorites-3-2","type":"favorites","title":"最近使用","x":3,"y":2,"w":9,"h":2,"minW":4,"minH":2,"presetKey":"recent"},
  {"id":"shortcut-0-4","type":"shortcut","title":"未命名快捷入口","x":0,"y":4,"w":12,"h":2,"minH":2,"presetKey":"shortcut"},
  {"id":"todo-0-6","type":"todo","title":"流程中心","x":0,"y":6,"w":3,"h":4,"minW":3,"minH":3,"presetKey":"todo"},
  {"id":"favorites-3-6","type":"favorites","title":"我的收藏","x":3,"y":6,"w":9,"h":2,"minW":4,"minH":2,"presetKey":"favorites"},
  {"id":"apps-3-8","type":"apps","title":"我的应用","x":3,"y":8,"w":9,"h":3,"minW":4,"minH":3,"presetKey":"apps"},
  {"id":"charts-3-11","type":"charts","title":"我的图表","x":3,"y":11,"w":9,"h":2,"minW":4,"minH":2,"presetKey":"my-charts"}
]}`

// Content 工作台文档 JSONB 载体：保存路径经服务端校验器终审后存入规范化
// JSON；读取路径原样透传（结构合法性由前端 normalizeDashboardSchema 二次
// 归一化），本类型不作文档结构建模
type Content json.RawMessage

// Value 空值落合法 JSON null，其余原样以字符串写入 JSONB 列
func (c Content) Value() (driver.Value, error) {
	if len(c) == 0 {
		return "null", nil
	}
	return string(c), nil
}

// Scan 兼容 pgx 的 []byte 与 lib/pq 的 string 回读；复制落盘避免持有驱动缓冲
func (c *Content) Scan(value interface{}) error {
	switch data := value.(type) {
	case nil:
		*c = nil
	case []byte:
		*c = Content(append([]byte(nil), data...))
	case string:
		*c = Content(data)
	}
	return nil
}

// MarshalJSON 让视图直接内联文档（json.RawMessage 语义），避免二次转义
func (c Content) MarshalJSON() ([]byte, error) {
	if len(c) == 0 {
		return []byte("null"), nil
	}
	return c, nil
}

// UnmarshalJSON 按原文保存请求中的文档（复制避免引用请求体缓冲）
func (c *Content) UnmarshalJSON(data []byte) error {
	*c = Content(append([]byte(nil), data...))
	return nil
}

// WorkbenchView 工作台读取/保存响应：document 为前端 DashboardSchema 原文
type WorkbenchView struct {
	Revision int64           `json:"revision"`
	Document json.RawMessage `json:"document"`
}

// SaveWorkbenchRequest 保存请求：revision 为乐观锁口令（首次保存传 0，
// 服务端落 1；之后必须携带上次返回值）
type SaveWorkbenchRequest struct {
	Revision int64           `json:"revision"`
	Document json.RawMessage `json:"document"`
}
