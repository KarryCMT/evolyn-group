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
	Delete(ctx context.Context, member *iammodel.User, code string) error
	CleanupExpired(ctx context.Context) error
}
