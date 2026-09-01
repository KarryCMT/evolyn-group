// Package service 产品日志域服务（000064）：查询规范化（分页/日期/成员/
// 应用/事件校验）、租户隔离复核、筛选项聚合、导出任务编排与受控出网投影。
// 不接管审计日志写入（audit 域），避免页面模块反向耦合所有业务写路径
package service

import (
	"context"
	"time"

	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/productlog/model"
	"evolyn/internal/platform/productlog/repository"
)

// 分页与导出限制（留存策略/套餐接入前的平台级固定值，接口层校验；
// 后续随设计 §9 留存策略批次改为策略下发）
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
	// MaxExportRows 单次导出行数上限：超过即要求缩小筛选范围（更大规模
	// 随异步导出任务承接）
	MaxExportRows = 50000
	// ExportFileTTL 导出文件有效期（过期后不可下载，需重新导出）
	ExportFileTTL = 24 * time.Hour
	// memberOptionLimit 操作人筛选项单次拉取上限（产品日志筛选用途足够；
	// 超出部分随成员选择器分页批演进）
	memberOptionLimit = 500
	// applicationOptionLimit 应用筛选项单次拉取上限（应用数受 apps 配额
	// 约束，上限仅作防御）
	applicationOptionLimit = 500
)

// MemberDirectory 成员目录窄端口：操作人筛选的归属校验与筛选项聚合。
// 由装配层以 iam 成员仓储适配（本域不反向依赖 iam）
type MemberDirectory interface {
	// ValidateMember 校验成员属于当前租户且有效；无效时返回
	// productlog.ErrMemberInvalid
	ValidateMember(ctx context.Context, tenantID, memberID uint) error
	// ListMembers 当前租户可选操作人清单（有效成员，按 ID 升序）
	ListMembers(ctx context.Context, tenantID uint) ([]model.MemberOption, error)
}

// ApplicationDirectory 应用目录窄端口：应用筛选的归属校验与筛选项聚合。
// 由装配层以 application 仓储适配（本域不反向依赖 application 域）
type ApplicationDirectory interface {
	// ValidateApplication 校验应用属于当前租户且未删除；无效时返回
	// productlog.ErrApplicationInvalid
	ValidateApplication(ctx context.Context, tenantID, applicationID uint) error
	// ListApplications 当前租户有效应用清单（筛选项；已删除应用不返回）
	ListApplications(ctx context.Context, tenantID uint) ([]model.ApplicationOption, error)
}

// ProductLogService 产品日志服务（租户侧只读 + 导出）
type ProductLogService interface {
	// List 产品日志分页查询（GET /product-logs）：只读产品分类白名单内的
	// 审计流水，应用维度过滤不可替代租户过滤
	List(ctx context.Context, tenantID uint, q model.ProductLogQuery) (*model.ProductLogPage, error)
	// Options 筛选项聚合（GET /product-logs/options）：产品分类及事件码、
	// 可选操作人、有效应用；均由服务端下发，前端不硬编码
	Options(ctx context.Context, tenantID uint) (*model.ProductLogOptions, error)
	// CreateExport 创建导出任务（POST /product-logs/exports）：固化筛选
	// 条件与申请人，一期同步生成（行数受 MaxExportRows 约束），提交后
	// best-effort 记导出行为审计（企业治理类）
	CreateExport(ctx context.Context, tenantID uint, req model.CreateExportRequest) (*model.ExportTaskView, error)
	// GetExport 任务状态查询（GET /product-logs/exports/:id）：租户归属
	// 复核，ready 且已过期投影为 expired
	GetExport(ctx context.Context, tenantID, taskID uint) (*model.ExportTaskView, error)
	// ExportFile 下载内容读取（下载端点复核导出权限后调用）：复核租户
	// 归属/状态/有效期，返回文件名与 CSV 字节
	ExportFile(ctx context.Context, tenantID, taskID uint) (*model.ExportFileContent, error)
}

// NewProductLogService 产品日志服务工厂：repo 只读仓储；members/apps 为
// 成员/应用目录适配器（可空——空时跳过对应归属校验与筛选项，供单测/降级
// 使用）；audit 为导出行为审计记录器（可空）
func NewProductLogService(
	repo repository.Repository,
	members MemberDirectory,
	apps ApplicationDirectory,
	audit auditservice.Recorder,
) ProductLogService {
	return &productLogService{repo: repo, members: members, apps: apps, audit: audit}
}
