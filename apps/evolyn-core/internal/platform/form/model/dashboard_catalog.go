package model

import queryengine "evolyn/internal/engine/query"

// DashboardDataSource 是表单域向仪表盘域暴露的最小只读数据源视图。
// AppID 只用于服务端同应用校验，不作为前端跨应用查询入口。
type DashboardDataSource struct {
	AppID            uint   `json:"-"`
	Code             string `json:"code"`
	Name             string `json:"name"`
	PublishedVersion int    `json:"publishedVersion"`
	SchemaRevision   string `json:"schemaRevision"`
}

// DashboardFieldCapability 将发布快照字段身份与 Query DSL 能力一起冻结出网。
// FieldID 是协议 v8 起的不可变身份，FieldCode 仅用于后端编译兼容既有记录字段。
type DashboardFieldCapability struct {
	FieldID     string                          `json:"fieldId"`
	FieldCode   string                          `json:"fieldCode"`
	Label       string                          `json:"label"`
	Type        queryengine.FieldType           `json:"type"`
	Filterable  bool                            `json:"filterable"`
	Sortable    bool                            `json:"sortable"`
	Projectable bool                            `json:"projectable"`
	Groupable   bool                            `json:"groupable"`
	Aggregates  []queryengine.AggregateOperator `json:"aggregates"`
}

// DashboardFieldCatalog 是单个已发布表单经权限裁剪后的字段目录。
type DashboardFieldCatalog struct {
	AppID            uint                       `json:"-"`
	FormCode         string                     `json:"formCode"`
	FormName         string                     `json:"formName"`
	PublishedVersion int                        `json:"publishedVersion"`
	SchemaRevision   string                     `json:"schemaRevision"`
	Fields           []DashboardFieldCapability `json:"fields"`
}
