// Package service 自定义工作台域服务（000078 企业级配置定版）：企业工作台
// 文档的校验（前端镜像，schema.go）、存取、保存乐观锁与租户开通种子。
// Service 不依赖 Gin/HTTP 细节；数据范围恒为「当前租户」（controller 只传
// JWT/租户上下文解析出的租户标识），保存权限（workbench:update 仅企业
// 管理员）由 RBAC 中间件裁决
package service

import (
	"context"

	"evolyn/internal/platform/workbench/model"
)

// TxManager 事务边界抽象（FIX-020）：具体实现在 infrastructure（ctx 传播
// 事务 session），Service 只依赖最小接口，便于单测以直通实现模拟事务
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// Repository 本域仓储（repository/interface.go 的最小依赖面）
type Repository interface {
	FindByTenant(ctx context.Context, tenantID uint) (*model.TenantWorkbench, error)
	Create(ctx context.Context, record *model.TenantWorkbench) error
	UpdateContentWithRevision(ctx context.Context, id uint, fromRevision int64, content model.Content) (bool, error)
}

// WorkbenchService 自定义工作台服务
type WorkbenchService interface {
	// Get 读取企业工作台（GET /workbench，全员可读）：正常路径下租户开通
	// 即种子默认布局，行恒存在；行缺失（异常路径）返回 nil，前端回退
	// 本地默认布局。content 原样透传，结构归一化由前端执行（与服务端
	// 保存终审职责分离）
	Get(ctx context.Context, tenantID uint) (*model.WorkbenchView, error)
	// Save 保存企业工作台（PUT /workbench，仅企业管理员）：文档经服务端
	// 校验器终审，首次保存（revision=0）插入 revision=1，此后按 revision
	// 乐观锁更新；版本过期返回 WORKBENCH_REVISION_CONFLICT
	Save(ctx context.Context, tenantID uint, req *model.SaveWorkbenchRequest) (*model.WorkbenchView, error)
	// SeedDefaults 租户开通事务内初始化默认工作台（口径同
	// NotificationSettingSeeder/ProductConfigSeeder）：写入
	// DefaultWorkbenchDocument、revision=1；幂等，已有行（含并发种子竞态）
	// 无副作用
	SeedDefaults(ctx context.Context, tenantID uint) error
}
