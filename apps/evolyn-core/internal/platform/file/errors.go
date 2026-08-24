// Package file 是租户文件元数据与对象上传编排域。对象本身留在 RustFS，
// PostgreSQL 仅保存可鉴权、可审计、可计量的元数据。
package file

import (
	"net/http"

	"evolyn/internal/platform/httpx"
)

var (
	ErrStorageDisabled = httpx.NewBiz("FILE_STORAGE_DISABLED", "文件存储尚未启用", http.StatusServiceUnavailable)
	ErrRequestInvalid  = httpx.NewBiz("FILE_REQUEST_INVALID", "文件上传参数无效", http.StatusBadRequest)
	ErrTooLarge        = httpx.NewBiz("FILE_TOO_LARGE", "文件大小超过允许范围", http.StatusBadRequest)
	ErrNotFound        = httpx.NewBiz("FILE_NOT_FOUND", "文件不存在或无权访问", http.StatusNotFound)
	ErrStateInvalid    = httpx.NewBiz("FILE_STATE_INVALID", "文件当前状态不支持此操作", http.StatusConflict)
	ErrObjectInvalid   = httpx.NewBiz("FILE_OBJECT_INVALID", "上传对象校验失败，请重新上传", http.StatusBadRequest)
)
