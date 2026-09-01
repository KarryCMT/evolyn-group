package model

import (
	"database/sql/driver"
	"fmt"

	kernel "evolyn/internal/model"
)

// 导出日志类型（kind 列取值）
const (
	ExportKindLogin     = "login"     // 登录日志
	ExportKindOperation = "operation" // 操作日志
)

// ExportKindLabel 导出类型的中文展示名（摘要与文件名用）
func ExportKindLabel(kind string) string {
	switch kind {
	case ExportKindLogin:
		return "登录日志"
	case ExportKindOperation:
		return "操作日志"
	default:
		return kind
	}
}

// 导出任务状态（status 列取值 + 读时投影）
const (
	ExportStatusPending = "pending" // 生成中（异步任务接入后使用）
	ExportStatusReady   = "ready"   // 就绪可下载
	ExportStatusFailed  = "failed"  // 生成失败
	// ExportStatusExpired 读时投影：任务 ready 但已过 expires_at，不可下载
	ExportStatusExpired = "expired"
)

// ExportFilters 提交时固化的筛选条件快照（与列表查询参数同构，不含分页：
// 导出覆盖当前筛选条件下的全部授权数据）。JSON 序列化存 JSONB
type ExportFilters struct {
	MemberID     uint   `json:"memberId,omitempty"`
	CategoryCode string `json:"categoryCode,omitempty"`
	EventCode    string `json:"eventCode,omitempty"`
	StartDate    string `json:"startAt,omitempty"`
	EndDate      string `json:"endAt,omitempty"`
}

// FiltersJSON 筛选快照 JSONB 载体：空串落 NULL（与 audit.JSONText 同口径）
type FiltersJSON string

func (f FiltersJSON) Value() (driver.Value, error) {
	if len(f) == 0 {
		return nil, nil
	}
	return []byte(f), nil
}

func (f *FiltersJSON) Scan(v interface{}) error {
	switch data := v.(type) {
	case nil:
		*f = ""
	case []byte:
		*f = FiltersJSON(data)
	case string:
		*f = FiltersJSON(data)
	default:
		return fmt.Errorf("cannot scan %T into enterpriselog.FiltersJSON", v)
	}
	return nil
}

// ExportTask 企业日志导出任务（tn_enterprise_log_exports，000036）：追加型任务
// 记录，显式 tenant 过滤（平台级表，不走租户 Callback）；一期同步生成、
// 内容内联存储（file_data），异步导出与对象存储文件引用随留存策略批次接入
type ExportTask struct {
	ID        uint        `json:"id" gorm:"autoIncrement;primaryKey"`
	TenantID  uint        `json:"tenantId" gorm:"not null"`            // 任务归属租户：读取/下载均复核
	AccountID uint        `json:"accountId" gorm:"not null;default:0"` // 申请人平台账号
	MemberID  uint        `json:"memberId" gorm:"not null;default:0"`  // 申请人租户成员
	Kind      string      `json:"kind" gorm:"size:16;not null"`        // login / operation
	Filters   FiltersJSON `json:"-" gorm:"type:jsonb"`                 // 固化筛选快照
	Total     int64       `json:"total" gorm:"not null;default:0"`     // 导出数据量（行数）
	Status    string      `json:"status" gorm:"size:16;not null;default:pending"`
	FileName  string      `json:"fileName" gorm:"size:128;not null;default:''"` // 下载文件名
	FileData  string      `json:"-" gorm:"type:text;not null;default:''"`       // 一期内联 CSV 内容
	// ExpiresAt 导出文件过期时间（NULL=永不过期；创建时按平台策略写入）
	ExpiresAt *kernel.JSONTime `json:"expiresAt"`
	CreatedAt kernel.JSONTime  `json:"createdAt"`
}

func (*ExportTask) TableName() string { return "tn_enterprise_log_exports" }

// ExportTaskView 导出任务出网视图（任务状态查询与创建响应共用）：
// 不回传文件内容，下载走专用端点（复核导出权限与租户归属）
type ExportTaskView struct {
	ID        uint            `json:"id"`
	Kind      string          `json:"kind"`
	KindLabel string          `json:"kindLabel"`
	Filters   ExportFilters   `json:"filters"`
	Total     int64           `json:"total"`
	Status    string          `json:"status"` // pending/ready/failed/expired（读时投影）
	FileName  string          `json:"fileName"`
	ExpiresAt kernel.JSONTime `json:"expiresAt"` // 零值出空串
	CreatedAt kernel.JSONTime `json:"createdAt"`
}

// CreateExportRequest 创建导出任务请求（POST /enterprise-logs/exports）：
// kind + 与列表完全相同的筛选条件；分页参数不参与（导出为全量授权数据）
type CreateExportRequest struct {
	Kind         string `json:"kind"`         // login / operation
	MemberID     uint   `json:"memberId"`     // 可选
	CategoryCode string `json:"categoryCode"` // operation 可选
	EventCode    string `json:"eventCode"`    // operation 可选
	StartDate    string `json:"startAt"`      // 可选，yyyy-MM-dd
	EndDate      string `json:"endAt"`        // 可选，yyyy-MM-dd
}

// ExportFileContent 下载内容（专用端点复核权限后返回）
type ExportFileContent struct {
	FileName    string
	ContentType string
	Data        []byte
}
