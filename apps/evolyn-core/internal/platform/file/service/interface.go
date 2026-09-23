package service

import (
	"context"

	filemodel "evolyn/internal/platform/file/model"
	iammodel "evolyn/internal/platform/iam/model"
)

type FileService interface {
	CreateUpload(ctx context.Context, member *iammodel.User, req *filemodel.CreateUploadRequest) (*filemodel.UploadDetail, error)
	Get(ctx context.Context, member *iammodel.User, code string) (*filemodel.File, error)
	Complete(ctx context.Context, member *iammodel.User, code string) (*filemodel.File, error)
	DownloadURL(ctx context.Context, member *iammodel.User, code string) (*filemodel.DownloadURLResponse, error)
	PublicDownloadURL(ctx context.Context, code string) (*filemodel.DownloadURLResponse, error)
	Delete(ctx context.Context, member *iammodel.User, code string) error
	CleanupExpired(ctx context.Context) error
	StoreGenerated(ctx context.Context, member *iammodel.User, input GeneratedFileInput) (*filemodel.File, error)
}

// GeneratedFileInput 是服务端 Worker 写入 RustFS 的受控文件，不允许调用方
// 指定 bucket 或完整对象键。
type GeneratedFileInput struct {
	Filename     string
	ContentType  string
	Content      []byte
	RelativePath string
}
