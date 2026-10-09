package model

import kernel "evolyn/internal/model"

type PrecreateRequest struct {
	RequestID      string `json:"requestId" binding:"required" example:"e2ac40f5-e645-4ac8-a1f2-c6610547b333"`
	Name           string `json:"name" binding:"required" example:"未命名仪表盘"`
	ParentMenuCode string `json:"parentMenuCode" example:"menu_ab12cd34ef56ab12"`
}

type UpdateRequest struct {
	Name           *string `json:"name"`
	Icon           *string `json:"icon"`
	Color          *string `json:"color"`
	ParentMenuCode *string `json:"parentMenuCode"`
}

type SaveDraftRequest struct {
	ExpectedRevision int64       `json:"expectedRevision" binding:"required"`
	ProtocolVersion  int         `json:"protocolVersion" binding:"required"`
	Document         JSONContent `json:"document" binding:"required"`
}

type SaveDraftResult struct {
	DraftRevision int64       `json:"draftRevision"`
	Document      JSONContent `json:"document"`
}

type PreviewQueryRequest struct {
	DraftRevision int64          `json:"draftRevision" binding:"required"`
	Page          int            `json:"page"`
	FilterValues  map[string]any `json:"filterValues"`
}

// RuntimeBootstrap 是成员运行页所需的最小投影。仪表盘发布能力开放前，
// Version 对应当前已保存草稿 revision；运行页查询必须回传同一口令。
type RuntimeBootstrap struct {
	Code            string      `json:"code"`
	Name            string      `json:"name"`
	ProtocolVersion int         `json:"protocolVersion"`
	Version         int64       `json:"version"`
	Document        JSONContent `json:"document"`
}

type RuntimeQueryRequest struct {
	Version      int64          `json:"version" binding:"required"`
	Page         int            `json:"page"`
	FilterValues map[string]any `json:"filterValues"`
}

type QueryResultColumn struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

type PreviewQueryResult struct {
	DatasetID string              `json:"datasetId"`
	Columns   []QueryResultColumn `json:"columns"`
	Rows      []map[string]any    `json:"rows"`
	Total     int64               `json:"total"`
	Page      int                 `json:"page"`
	PageSize  int                 `json:"pageSize"`
}

type PublishedSummary struct {
	Version     int              `json:"version"`
	PublishedAt *kernel.JSONTime `json:"publishedAt"`
}

type Detail struct {
	AppID            uint             `json:"appId"`
	AppCode          string           `json:"appCode"`
	Code             string           `json:"code"`
	Name             string           `json:"name"`
	Icon             string           `json:"icon"`
	Color            string           `json:"color"`
	ProtocolVersion  int              `json:"protocolVersion"`
	DraftRevision    int64            `json:"draftRevision"`
	Draft            JSONContent      `json:"draft"`
	PublishedVersion int              `json:"publishedVersion"`
	Published        PublishedSummary `json:"published"`
	CreatedAt        kernel.JSONTime  `json:"createdAt"`
	UpdatedAt        kernel.JSONTime  `json:"updatedAt"`
}
