// Package service 版本信息域服务（一期）：权益解析、订阅查询、人工授予、
// 到期投影与幂等降级。活动订阅及其套餐版本是权益事实源，tenants.plan/
// tenants.quotas 仅为 QuotaService 过渡期的兼容投影，订阅变更在同一事务内
// 同步两侧（设计 4.4.1）
package service

import (
	"context"

	"evolyn/internal/platform/edition/model"
)

// TxManager 事务边界抽象（FIX-021）：具体实现在 infrastructure（ctx 传播
// 事务 session），Service 只依赖最小接口，便于单测以直通实现模拟事务
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// EditionService 版本信息服务（租户侧读取 + 平台运营面 + 到期任务 + 集成钩子）
type EditionService interface {
	// GetCurrent 租户侧版本信息概览（设计 4.5.1）：订阅投影（到期未降级
	// 立即返回 expired）、配额（meteringStatus/limitSource）、功能权益
	GetCurrent(ctx context.Context, tenantID uint) (*model.CurrentEdition, error)
	// GetTenantEdition 平台侧租户版本详情：当前概览 + 历史订阅 + 覆盖记录
	GetTenantEdition(ctx context.Context, tenantID uint) (*model.TenantEditionDetail, error)
	// ListGrantableVersions 可授予的已发布基础套餐版本（运营选择器）
	ListGrantableVersions(ctx context.Context) ([]model.GrantableVersion, error)
	// Grant 平台侧订阅写入：grant 授予/替换（同事务关旧→建新→覆盖→投影），
	// cancel 取消并降级免费版；提交后写审计
	Grant(ctx context.Context, tenantID, operatorAccountID uint, req *model.GrantRequest) error
	// DowngradeExpiredOnce 到期降级单轮扫描（幂等；worker 周期调用），
	// 返回本轮成功降级的订阅数
	DowngradeExpiredOnce(ctx context.Context) (int, error)

	// GuardLimit 到期守卫（tenant 域 QuotaService 集成，设计 4.4.1）：
	// 活动订阅已到期时返回 decided=true 与「免费快照 + 仅有效 manual 覆盖」
	// 的生效上限，QuotaService 以其替代旧 tenants.plan/quotas 读取；
	// 未到期/无订阅/非存量键返回 decided=false 继续走旧路径
	GuardLimit(ctx context.Context, tenantID uint, resourceKey string) (limit int64, decided bool, err error)
	// SeedInitial 租户创建事务内补种初始订阅（tenant 域集成）：free/pro
	// 落 active 长期订阅；trial 无到期信息落 legacy_pending_review 待补录
	SeedInitial(ctx context.Context, tenantID uint, planCode string) error
}
