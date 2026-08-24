// Package objectstore 封装 S3 兼容对象存储的数据面能力。业务域只依赖本包
// 的窄接口，RustFS/MinIO/AWS SDK 的具体差异不泄漏到 Controller 或 Service。
package objectstore

import (
	"context"
	"time"
)

// PresignedRequest 是浏览器直传/下载所需的短期授权信息。
type PresignedRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

// ObjectInfo 是完成确认所需的最小对象元数据。
type ObjectInfo struct {
	Size           int64
	ContentType    string
	ChecksumSHA256 string // S3 x-amz-checksum-sha256（Base64）
}

// Store 是文件域依赖的 S3 子集。对象键、bucket、签名有效期均由业务服务
// 决定；调用方不接触 AccessKey/SecretKey。
type Store interface {
	PresignPut(ctx context.Context, bucket, key string, expires time.Duration, headers map[string]string) (*PresignedRequest, error)
	PresignGet(ctx context.Context, bucket, key string, expires time.Duration) (*PresignedRequest, error)
	Stat(ctx context.Context, bucket, key string) (*ObjectInfo, error)
	Remove(ctx context.Context, bucket, key string) error
}
