// Package service 企业日志域服务（000036）：查询规范化（分页/日期/成员/
// 事件校验）、租户隔离复核、导出任务编排与受控出网投影。不接管登录日志与
// 审计日志的写入（分属 auth/loginlog 与 audit 域），避免页面模块反向耦合
// 所有业务写路径
package service

import (
	"context"
	"time"

	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/enterpriselog/model"
	"evolyn/internal/platform/enterpriselog/repository"
)

// 分页与导出限制（留存策略/套餐接入前的平台级固定值，接口层校验；
// 后续随第 8 节留存策略批次改为策略下发）
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
	// MaxExportRows 单次导出行数上限：超过即要求缩小筛选范围（更大规模
	// 随异步导出任务承接）
	MaxExportRows = 50000
	// ExportFileTTL 导出文件有效期（过期后不可下载，需重新导出）
	ExportFileTTL = 24 * time.Hour
)

// MemberDirectory 成员目录窄端口：登录人/操作人筛选的成员归属校验。
// 由装配层以 iam 成员仓储适配（本域不反向依赖 iam）
type MemberDirectory interface {
	// ValidateMember 校验成员属于当前租户且有效；无效时返回
	// enterpriselog.ErrMemberInvalid
	ValidateMember(ctx context.Context, tenantID, memberID uint) error
}

// EnterpriseLogService 企业日志服务（租户侧只读 + 导出）
type EnterpriseLogService interface {
	// ListLoginLogs 登录日志分页查询（GET /enterprise-logs/login）
	ListLoginLogs(ctx context.Context, tenantID uint, q model.LoginLogQuery) (*model.LoginLogPage, error)
	// ListOperationLogs 操作日志分页查询（GET /enterprise-logs/operations）
	ListOperationLogs(ctx context.Context, tenantID uint, q model.OperationLogQuery) (*model.OperationLogPage, error)
	// ListCategories 日志范围与操作类型筛选项（GET /enterprise-logs/operation-categories）
	ListCategories() []model.CategoryOption
	// CreateExport 创建导出任务（POST /enterprise-logs/exports）：固化筛选
	// 条件与申请人，一期同步生成（行数受 MaxExportRows 约束），提交后
	// best-effort 记导出行为审计
	CreateExport(ctx context.Context, tenantID uint, req model.CreateExportRequest) (*model.ExportTaskView, error)
	// GetExport 任务状态查询（GET /enterprise-logs/exports/:id）：租户归属
	// 复核，ready 且已过期投影为 expired
	GetExport(ctx context.Context, tenantID, taskID uint) (*model.ExportTaskView, error)
	// ExportFile 下载内容读取（下载端点复核导出权限后调用）：复核租户
	// 归属/状态/有效期，返回文件名与 CSV 字节
	ExportFile(ctx context.Context, tenantID, taskID uint) (*model.ExportFileContent, error)
}

// NewEnterpriseLogService 企业日志服务工厂：repo 只读仓储；members 为成员
// 目录适配器（可空——空时跳过成员归属校验，供单测/降级使用）；audit 为
// 导出行为审计记录器（可空）
func NewEnterpriseLogService(repo repository.Repository, members MemberDirectory, audit auditservice.Recorder) EnterpriseLogService {
	return &enterpriseLogService{repo: repo, members: members, audit: audit}
}
