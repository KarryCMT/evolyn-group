package objectstore

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"evolyn/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// RustFS 是 RustFS S3 兼容接口的实现。MinIO Go SDK 只作为严格 S3 客户端
// 使用，不调用 MinIO 管理 API，因此后续切换其他 S3 实现无需改业务层。
type RustFS struct {
	client     *minio.Client
	signClient *minio.Client
}

func NewRustFS(conf config.StorageConfig) (*RustFS, error) {
	endpoint, secure, err := parseEndpoint(conf.Endpoint, conf.UseSSL)
	if err != nil {
		return nil, fmt.Errorf("解析 storage.endpoint: %w", err)
	}
	options := &minio.Options{
		Creds:  credentials.NewStaticV4(conf.AccessKey, conf.SecretKey, ""),
		Secure: secure,
		Region: "us-east-1",
	}
	client, err := minio.New(endpoint, options)
	if err != nil {
		return nil, fmt.Errorf("创建 RustFS S3 客户端: %w", err)
	}

	// 内外网地址不同的时候，签名必须基于浏览器实际访问的 Host/Scheme。
	signClient := client
	if conf.ExternalEndpoint != "" {
		externalEndpoint, externalSecure, err := parseEndpoint(conf.ExternalEndpoint, conf.UseSSL)
		if err != nil {
			return nil, fmt.Errorf("解析 storage.externalEndpoint: %w", err)
		}
		signClient, err = minio.New(externalEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(conf.AccessKey, conf.SecretKey, ""),
			Secure: externalSecure,
			Region: "us-east-1",
		})
		if err != nil {
			return nil, fmt.Errorf("创建 RustFS 预签名客户端: %w", err)
		}
	}
	return &RustFS{client: client, signClient: signClient}, nil
}

func parseEndpoint(raw string, defaultSecure bool) (string, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false, fmt.Errorf("地址不能为空")
	}
	if !strings.Contains(raw, "://") {
		return strings.TrimRight(raw, "/"), defaultSecure, nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Path != "" && u.Path != "/" {
		return "", false, fmt.Errorf("必须是 host:port 或不带路径的 http(s) URL")
	}
	switch u.Scheme {
	case "http":
		return u.Host, false, nil
	case "https":
		return u.Host, true, nil
	default:
		return "", false, fmt.Errorf("仅支持 http 或 https")
	}
}

func (r *RustFS) PresignPut(ctx context.Context, bucket, key string, expires time.Duration, headers map[string]string) (*PresignedRequest, error) {
	extraHeaders := make(http.Header, len(headers))
	for key, value := range headers {
		extraHeaders.Set(key, value)
	}
	u, err := r.signClient.PresignHeader(ctx, http.MethodPut, bucket, key, expires, nil, extraHeaders)
	if err != nil {
		return nil, err
	}
	return &PresignedRequest{Method: "PUT", URL: u.String(), Headers: headers}, nil
}

func (r *RustFS) PresignGet(ctx context.Context, bucket, key string, expires time.Duration) (*PresignedRequest, error) {
	u, err := r.signClient.PresignedGetObject(ctx, bucket, key, expires, nil)
	if err != nil {
		return nil, err
	}
	return &PresignedRequest{Method: "GET", URL: u.String(), Headers: map[string]string{}}, nil
}

func (r *RustFS) Stat(ctx context.Context, bucket, key string) (*ObjectInfo, error) {
	info, err := r.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}
	return &ObjectInfo{Size: info.Size, ContentType: info.ContentType, ChecksumSHA256: info.ChecksumSHA256}, nil
}

func (r *RustFS) Remove(ctx context.Context, bucket, key string) error {
	return r.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}
