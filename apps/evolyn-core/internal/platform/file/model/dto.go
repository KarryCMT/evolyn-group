package model

import kernel "evolyn/internal/model"

type CreateUploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	SHA256      string `json:"sha256"`
}

type UploadDetail struct {
	FileID    string          `json:"fileId"`
	State     string          `json:"state"`
	ExpiresAt kernel.JSONTime `json:"expiresAt"`
	Upload    *UploadRequest  `json:"upload,omitempty"`
}

type UploadRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type DownloadURLResponse struct {
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt kernel.JSONTime   `json:"expiresAt"`
}
