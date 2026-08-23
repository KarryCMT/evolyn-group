// Package service 应用管理域业务服务（M2-A）：统一实例化编排、配额占位、
// capabilities 派生与审计时机控制（docs/低代码平台/应用管理/开发文档.md）
package service

import (
	"context"

	"evolyn/internal/platform/application/model"
	iammodel "evolyn/internal/platform/iam/model"
)

// ApplicationService 应用服务。member 为当前请求成员（认证中间件已加载
// 角色/分组），用于 owner/creator 绑定校验与 capabilities 派生；
// 租户归属一律从 ctx 取（§9.3），不接受客户端传入
type ApplicationService interface {
	// CreateBlank 创建空白应用（§5.1 统一实例化命令的 blank 分支）：
	// 单事务完成配额占位（CheckAndReserve）→ 应用记录 → 安装记录，
	// 提交后 best-effort 审计；M2-A 同步路径直接 ready
	CreateBlank(ctx context.Context, member *iammodel.User, req *model.CreateBlankRequest) (*model.ApplicationDetail, error)
	// List 当前租户应用列表（游标分页，keyword/status 过滤）
	List(ctx context.Context, member *iammodel.User, query model.ListApplicationsQuery) (*model.ApplicationPage, error)
	// Get 应用详情与运行时 capabilities
	Get(ctx context.Context, member *iammodel.User, id uint) (*model.ApplicationDetail, error)
	// GetByCode 按应用编码查详情（code 租户内唯一）：加载与权限复核
	// 口径同 Get，供工作区等以 code 定位应用的入口使用
	GetByCode(ctx context.Context, member *iammodel.User, code string) (*model.ApplicationDetail, error)
	// Update 白名单字段更新（name/icon/color/sortOrder/status）；
	// status 仅允许 active↔archived 互转承载归档/恢复（§5.4）
	Update(ctx context.Context, member *iammodel.User, id uint, req *model.UpdateApplicationRequest) (*model.ApplicationDetail, error)
	// Delete 软删（仅写 deleted_at；配额随计数口径自然释放）
	Delete(ctx context.Context, member *iammodel.User, id uint) error
}
