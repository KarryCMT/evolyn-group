package model

import queryengine "evolyn/internal/engine/query"

// FormDataSource 是设计器可选择的同应用表单数据源。
type FormDataSource struct {
	Code             string `json:"code"`
	Name             string `json:"name"`
	PublishedVersion int    `json:"publishedVersion"`
	SchemaRevision   string `json:"schemaRevision"`
}

// DataSourceField 是权限裁剪后的稳定字段能力描述。客户端保存 FieldID，
// FieldCode 仅随请求回传供服务端根据冻结快照编译，不能作为持久身份。
type DataSourceField struct {
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

type FormFieldCatalog struct {
	FormCode         string            `json:"formCode"`
	FormName         string            `json:"formName"`
	PublishedVersion int               `json:"publishedVersion"`
	SchemaRevision   string            `json:"schemaRevision"`
	Fields           []DataSourceField `json:"fields"`
}
