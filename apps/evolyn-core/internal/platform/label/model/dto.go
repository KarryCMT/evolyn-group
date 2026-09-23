package model

import (
	"encoding/json"

	kernel "evolyn/internal/model"
)

type CreateTemplateRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	FormCode    string  `json:"formCode"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
	Unit        string  `json:"unit"`
	DPI         int     `json:"dpi"`
}

type SaveDraftRequest struct {
	DraftRevision int64           `json:"draftRevision"`
	Schema        json.RawMessage `json:"schema"`
}

type SaveDraftResult struct {
	DraftRevision int64 `json:"draftRevision"`
}

type PublishRequest struct {
	DraftRevision int64 `json:"draftRevision"`
}

type PublishResult struct {
	VersionNo int `json:"versionNo"`
}

type PreviewRequest struct {
	RecordID string `json:"recordId"`
	Format   string `json:"format"`
}

type RenderRequest struct {
	TemplateCode string `json:"templateCode"`
	RecordID     string `json:"recordId"`
	Format       string `json:"format"`
}

type TemplateSummary struct {
	Code                   string          `json:"code"`
	Name                   string          `json:"name"`
	Description            string          `json:"description"`
	AppID                  uint            `json:"appId"`
	FormCode               string          `json:"formCode"`
	Status                 string          `json:"status"`
	PublishedVersion       int             `json:"publishedVersion"`
	DraftRevision          int64           `json:"draftRevision"`
	PreviewedDraftRevision int64           `json:"previewedDraftRevision"`
	PublishedDraftRevision int64           `json:"publishedDraftRevision"`
	Width                  float64         `json:"width"`
	Height                 float64         `json:"height"`
	Unit                   string          `json:"unit"`
	DPI                    int             `json:"dpi"`
	CreatorMemberID        uint            `json:"creatorMemberId"`
	CreatedAt              kernel.JSONTime `json:"createdAt"`
	UpdatedAt              kernel.JSONTime `json:"updatedAt"`
}

type TemplateDetail struct {
	TemplateSummary
	Draft json.RawMessage `json:"draft"`
}

type TemplatePage struct {
	Items      []TemplateSummary `json:"items"`
	NextCursor string            `json:"nextCursor"`
}

type ListTemplatesQuery struct {
	Limit    int
	Cursor   string
	Keyword  string
	Status   string
	FormCode string
}
