package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"evolyn/internal/config"
	"evolyn/internal/contextx"
	"evolyn/internal/infrastructure"
	"evolyn/internal/infrastructure/objectstore"
	kernel "evolyn/internal/model"
	"evolyn/internal/platform/audit/service"
	filedomain "evolyn/internal/platform/file"
	filemodel "evolyn/internal/platform/file/model"
	"evolyn/internal/platform/file/repository"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	tenantservice "evolyn/internal/platform/tenant/service"

	"gorm.io/gorm"
)

type fileService struct {
	tx      *infrastructure.TxManager
	repo    repository.FileRepository
	quota   tenantservice.StorageQuotaService
	audit   service.Recorder
	store   objectstore.Store
	storage config.StorageConfig
}

func (s *fileService) Get(ctx context.Context, member *iammodel.User, code string) (*filemodel.File, error) {
	if _, err := validateMember(ctx, member); err != nil {
		return nil, err
	}
	return s.fileForMember(ctx, member, code)
}

func NewFileService(tx *infrastructure.TxManager, repo repository.FileRepository, quota tenantservice.StorageQuotaService, audit service.Recorder, store objectstore.Store, storage config.StorageConfig) FileService {
	return &fileService{tx: tx, repo: repo, quota: quota, audit: audit, store: store, storage: storage}
}

func (s *fileService) CreateUpload(ctx context.Context, member *iammodel.User, req *filemodel.CreateUploadRequest) (*filemodel.UploadDetail, error) {
	if !s.storage.Enabled || s.store == nil {
		return nil, filedomain.ErrStorageDisabled
	}
	if err := validateCreate(req, s.storage.MaxUploadBytes); err != nil {
		return nil, err
	}
	tenantID, err := validateMember(ctx, member)
	if err != nil {
		return nil, err
	}
	code, err := newCode()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	expiresAt := now.Add(s.storage.PresignTTL())
	jsonExpiresAt := kernelTime(expiresAt)
	objectKey := strings.Trim(s.storage.Prefix+"/tenant/"+fmt.Sprint(tenantID)+"/file/"+code, "/")
	file := &filemodel.File{
		Code:         code,
		Bucket:       s.storage.Bucket,
		ObjectKey:    objectKey,
		OriginalName: strings.TrimSpace(req.Filename),
		ContentType:  strings.TrimSpace(req.ContentType),
		DeclaredSize: req.Size,
		SHA256:       strings.ToLower(strings.TrimSpace(req.SHA256)),
		State:        filemodel.FileStateUploading,
		ExpiresAt:    &jsonExpiresAt,
		CreatorID:    member.ID,
	}
	if err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		return s.quota.CheckAndReserveStorage(txCtx, tenantID, req.Size, func(ctx context.Context) error {
			return s.repo.Create(ctx, file)
		})
	}); err != nil {
		return nil, err
	}

	// 预签名只在本地计算，不会把文件流量转发经过 Gin；若签名失败，过期
	// 上传会话不会形成对象，后续清理任务可释放预留。
	headers := map[string]string{"Content-Type": file.ContentType}
	if file.SHA256 != "" {
		headers["x-amz-checksum-sha256"], err = hexSHA256ToBase64(file.SHA256)
		if err != nil {
			return nil, filedomain.ErrRequestInvalid
		}
	}
	presigned, err := s.store.PresignPut(ctx, file.Bucket, file.ObjectKey, s.storage.PresignTTL(), headers)
	if err != nil {
		return nil, httpx.Wrap(filedomain.ErrStorageDisabled, err)
	}
	s.audit.Record(ctx, service.Entry{Module: "file", Action: "upload_init", ResourceType: "file", ResourceID: code, After: map[string]interface{}{"size": req.Size, "contentType": file.ContentType}})
	return &filemodel.UploadDetail{FileID: code, State: file.State, ExpiresAt: jsonExpiresAt, Upload: &filemodel.UploadRequest{Method: presigned.Method, URL: presigned.URL, Headers: presigned.Headers}}, nil
}

func (s *fileService) Complete(ctx context.Context, member *iammodel.User, code string) (*filemodel.File, error) {
	if !s.storage.Enabled || s.store == nil {
		return nil, filedomain.ErrStorageDisabled
	}
	if _, err := validateMember(ctx, member); err != nil {
		return nil, err
	}
	file, err := s.fileForMember(ctx, member, code)
	if err != nil {
		return nil, err
	}
	if file.State == filemodel.FileStateReady {
		return file, nil // 幂等确认，网络重试不会重复记账或审计
	}
	if file.State != filemodel.FileStateUploading {
		return nil, filedomain.ErrStateInvalid
	}
	info, err := s.store.Stat(ctx, file.Bucket, file.ObjectKey)
	if err != nil {
		return nil, httpx.Wrap(filedomain.ErrObjectInvalid, err)
	}
	if info.Size != file.DeclaredSize {
		return nil, httpx.Wrap(filedomain.ErrObjectInvalid, fmt.Errorf("file %s size declared=%d actual=%d", file.Code, file.DeclaredSize, info.Size))
	}
	if file.SHA256 != "" && !matchesSHA256(file.SHA256, info.ChecksumSHA256) {
		return nil, httpx.Wrap(filedomain.ErrObjectInvalid, fmt.Errorf("file %s checksum mismatch", file.Code))
	}
	contentType := strings.TrimSpace(info.ContentType)
	if contentType == "" {
		contentType = file.ContentType
	}
	updated, err := s.repo.MarkReady(ctx, file.Code, info.Size, contentType)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, filedomain.ErrStateInvalid
	}
	file.State, file.ActualSize, file.ContentType, file.ExpiresAt = filemodel.FileStateReady, info.Size, contentType, nil
	s.audit.Record(ctx, service.Entry{Module: "file", Action: "upload_complete", ResourceType: "file", ResourceID: file.Code, After: map[string]interface{}{"size": file.ActualSize, "contentType": file.ContentType}})
	return file, nil
}

func (s *fileService) DownloadURL(ctx context.Context, member *iammodel.User, code string) (*filemodel.DownloadURLResponse, error) {
	if !s.storage.Enabled || s.store == nil {
		return nil, filedomain.ErrStorageDisabled
	}
	if _, err := validateMember(ctx, member); err != nil {
		return nil, err
	}
	file, err := s.fileForMember(ctx, member, code)
	if err != nil {
		return nil, err
	}
	if file.State != filemodel.FileStateReady {
		return nil, filedomain.ErrStateInvalid
	}
	presigned, err := s.store.PresignGet(ctx, file.Bucket, file.ObjectKey, s.storage.PresignTTL())
	if err != nil {
		return nil, httpx.Wrap(filedomain.ErrStorageDisabled, err)
	}
	expiresAt := kernelTime(time.Now().Add(s.storage.PresignTTL()))
	return &filemodel.DownloadURLResponse{Method: presigned.Method, URL: presigned.URL, Headers: presigned.Headers, ExpiresAt: expiresAt}, nil
}

// PublicDownloadURL 为头像等公开展示资源签发短期对象地址，普通文件不会链接到公开入口。
func (s *fileService) PublicDownloadURL(ctx context.Context, code string) (*filemodel.DownloadURLResponse, error) {
	if !s.storage.Enabled || s.store == nil {
		return nil, filedomain.ErrStorageDisabled
	}
	if strings.TrimSpace(code) == "" {
		return nil, filedomain.ErrNotFound
	}
	file, err := s.repo.GetByCode(ctx, code)
	if err == gorm.ErrRecordNotFound {
		return nil, filedomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if file.State != filemodel.FileStateReady {
		return nil, filedomain.ErrStateInvalid
	}
	presigned, err := s.store.PresignGet(ctx, file.Bucket, file.ObjectKey, s.storage.PresignTTL())
	if err != nil {
		return nil, httpx.Wrap(filedomain.ErrStorageDisabled, err)
	}
	expiresAt := kernelTime(time.Now().Add(s.storage.PresignTTL()))
	return &filemodel.DownloadURLResponse{Method: presigned.Method, URL: presigned.URL, Headers: presigned.Headers, ExpiresAt: expiresAt}, nil
}

func (s *fileService) Delete(ctx context.Context, member *iammodel.User, code string) error {
	if !s.storage.Enabled || s.store == nil {
		return filedomain.ErrStorageDisabled
	}
	if _, err := validateMember(ctx, member); err != nil {
		return err
	}
	file, err := s.fileForMember(ctx, member, code)
	if err != nil {
		return err
	}
	// 先删对象，再软删元数据：存储删除可重试且幂等；不把远程 I/O 放进 DB
	// 事务，避免长事务占用租户行锁。
	if err := s.store.Remove(ctx, file.Bucket, file.ObjectKey); err != nil {
		return httpx.Wrap(filedomain.ErrObjectInvalid, err)
	}
	if err := s.repo.SoftDelete(ctx, file); err != nil {
		return err
	}
	s.audit.Record(ctx, service.Entry{Module: "file", Action: "delete", ResourceType: "file", ResourceID: file.Code})
	return nil
}

// CleanupExpired 清理未确认的上传会话。对象存储删除成功后才软删元数据，
// 这样配额只会在对象确实不可再访问时释放；S3 Delete 是幂等操作，可安全重试。
func (s *fileService) CleanupExpired(ctx context.Context) error {
	if !s.storage.Enabled || s.store == nil {
		return nil
	}
	files, err := s.repo.ListExpiredUploads(ctx, time.Now())
	if err != nil {
		return err
	}
	for i := range files {
		file := &files[i]
		if err := s.store.Remove(ctx, file.Bucket, file.ObjectKey); err != nil {
			return err
		}
		if err := s.repo.SoftDelete(ctx, file); err != nil {
			return err
		}
	}
	return nil
}

func (s *fileService) fileForMember(ctx context.Context, member *iammodel.User, code string) (*filemodel.File, error) {
	if strings.TrimSpace(code) == "" {
		return nil, filedomain.ErrNotFound
	}
	file, err := s.repo.GetByCode(ctx, code)
	if err == gorm.ErrRecordNotFound {
		return nil, filedomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if file.CreatorID != member.ID {
		// 文件引用与业务资源权限尚未落地前，上传者是最小访问边界；后续由
		// file_references 的所属业务资源授权替代这条临时约束。
		return nil, filedomain.ErrNotFound
	}
	return file, nil
}

func validateMember(ctx context.Context, member *iammodel.User) (uint, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok || member == nil || member.ID == 0 || member.TenantID != tenantID {
		return 0, filedomain.ErrNotFound
	}
	return tenantID, nil
}

func validateCreate(req *filemodel.CreateUploadRequest, maxSize int64) error {
	if req == nil || strings.TrimSpace(req.Filename) == "" || len([]rune(strings.TrimSpace(req.Filename))) > 255 || strings.TrimSpace(req.ContentType) == "" || len(req.ContentType) > 255 || req.Size <= 0 {
		return filedomain.ErrRequestInvalid
	}
	if req.Size > maxSize {
		return filedomain.ErrTooLarge
	}
	if hash := strings.TrimSpace(req.SHA256); hash != "" && (len(hash) != 64 || !isHex(hash)) {
		return filedomain.ErrRequestInvalid
	}
	return nil
}

func newCode() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "fil_" + hex.EncodeToString(bytes), nil
}

func isHex(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}

func hexSHA256ToBase64(value string) (string, error) {
	bytes, err := hex.DecodeString(value)
	if err != nil || len(bytes) != 32 {
		return "", fmt.Errorf("invalid sha256")
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

func matchesSHA256(hexValue, base64Value string) bool {
	if base64Value == "" {
		return false
	}
	bytes, err := base64.StdEncoding.DecodeString(base64Value)
	return err == nil && strings.EqualFold(hexValue, hex.EncodeToString(bytes))
}

func kernelTime(t time.Time) kernel.JSONTime { return kernel.JSONTime(t) }
