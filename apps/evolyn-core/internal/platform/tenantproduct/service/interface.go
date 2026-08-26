// Package service 产品中心域服务（一期）：产品卡片组装、启停与可用范围
// 替换（乐观锁 + 统一事务）、租户开通种子与产品访问判定。Service 不依赖
// Gin/HTTP 细节；版本信息经窄端口只读消费 edition 域（文档 4）
package service

import (
	"context"

	editionmodel "evolyn/internal/platform/edition/model"
	"evolyn/internal/platform/tenantproduct/model"
)

// TxManager 事务边界抽象（FIX-020）：具体实现在 infrastructure（ctx 传播
// 事务 session），Service 只依赖最小接口，便于单测以直通实现模拟事务
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// EditionReader 版本信息只读窄端口（由 edition 服务结构性满足）：卡片
// 「当前版本」来自 edition 域事实源，不在产品配置中冗余存储（文档 2 结论 3）
type EditionReader interface {
	GetCurrent(ctx context.Context, tenantID uint) (*editionmodel.CurrentEdition, error)
}

// TenantProductService 产品中心服务（租户侧管理面 + 租户开通集成钩子）
type TenantProductService interface {
	// List 产品中心卡片列表（GET /tenant-products）：目录全量 + 当前租户
	// 配置/范围投影 + edition 版本投影 + 真实有效成员计数
	List(ctx context.Context, tenantID uint) (*model.ProductCenterView, error)
	// SetEnabled 启用/停用产品（PUT /tenant-products/:code/enabled）：
	// 锁定配置行校验 revision 后更新并 +1，提交后 best-effort 审计，
	// 返回最新卡片
	SetEnabled(ctx context.Context, tenantID uint, productCode string, req *model.UpdateEnabledRequest) (*model.ProductCard, error)
	// UpdateAccessScope 全量替换可用范围（PUT /tenant-products/:code/access-scope）：
	// 同一事务内校验全部 ID 的当前租户归属与有效性、更新主配置并替换
	// 范围关联，提交后 best-effort 审计，返回最新卡片
	UpdateAccessScope(ctx context.Context, tenantID uint, productCode string, req *model.UpdateAccessScopeRequest) (*model.ProductCard, error)
	// SeedDefaults 租户开通事务内初始化产品配置（文档 8.2）：为全部当前
	// active 目录创建 enabled=true、scope=all 的配置行，不建范围关联；
	// 幂等，已有配置的租户重复调用无副作用
	SeedDefaults(ctx context.Context, tenantID uint) error
}

// TenantProductAccessEvaluator 产品访问判定器（文档 6.4）：产品真实入口、
// 工作台路由的后端数据接口或后续跨服务网关必须调用本判定，前端隐藏入口
// 只是体验优化，不能作为授权边界。判定顺序：平台目录 active → 租户产品
// enabled → 成员为当前租户有效成员 → all 或命中直接成员/选中部门（含
// 子部门）；任何一步不满足均拒绝（false, nil），error 仅表示基础设施故障
type TenantProductAccessEvaluator interface {
	CanAccess(ctx context.Context, productCode string, memberID uint) (bool, error)
}
