// Package repository 版本信息域数据访问（ADR-007 域内小三层）：仅做持久化，
// 一律经 infrastructure.ResolveDB 取连接加入 ctx 传播事务。本域表均带
// tenant_id 但由平台侧与 worker 读写，仓储统一剥离请求租户上下文、以显式
// tenant_id 条件定位，不依赖 GORM 租户 Callback
package repository

import (
	"context"
	"time"

	"evolyn/internal/platform/edition/model"
)

// EditionRepository 版本信息域仓储
type EditionRepository interface {
	// GetCurrentSubscription 取租户当前订阅：优先 active，其次
	// legacy_pending_review（存量试用待补录）；无则返回 ErrRecordNotFound
	GetCurrentSubscription(ctx context.Context, tenantID uint) (*model.TenantSubscription, error)
	// ListSubscriptions 租户订阅历史（含运营备注，仅平台面调用）
	ListSubscriptions(ctx context.Context, tenantID uint) ([]model.TenantSubscription, error)
	// ListExpiredActive 活动且已到期的订阅（到期降级任务扫描）
	ListExpiredActive(ctx context.Context, now time.Time) ([]model.TenantSubscription, error)
	// LockSubscription 事务内 SELECT ... FOR UPDATE 重检订阅行（任务幂等）
	LockSubscription(ctx context.Context, id uint) (*model.TenantSubscription, error)
	// CloseSubscription 条件关闭订阅（fromStatus 不匹配时影响 0 行），
	// 与部分唯一索引共同保证「重复执行不产生第二条活动订阅」
	CloseSubscription(ctx context.Context, id uint, fromStatus, toStatus string) error
	// CreateSubscription 创建订阅（TenantID 由调用方显式赋值）
	CreateSubscription(ctx context.Context, sub *model.TenantSubscription) error
	// GetPlanVersionWithPlan 加载套餐版本及其所属套餐目录
	GetPlanVersionWithPlan(ctx context.Context, id uint) (*model.EditionPlanVersion, *model.EditionPlan, error)
	// GetLatestPublishedByCompat 按兼容套餐代码取最新已发布版本
	// （免费版降级目标、无订阅兜底视图用）
	GetLatestPublishedByCompat(ctx context.Context, compatCode string) (*model.EditionPlanVersion, error)
	// ListPublishedBaseVersions 可授予的已发布基础套餐版本（运营选择器）
	ListPublishedBaseVersions(ctx context.Context) ([]model.EditionPlanVersion, []model.EditionPlan, error)
	// ListValidOverrides 生效中的覆盖（starts_at<=now 且未到期）
	ListValidOverrides(ctx context.Context, tenantID uint, now time.Time) ([]model.TenantEntitlementOverride, error)
	// ListAllOverrides 租户全部覆盖记录（含失效，仅平台面）
	ListAllOverrides(ctx context.Context, tenantID uint) ([]model.TenantEntitlementOverride, error)
	// ReplaceActiveOverrides 全量替换 manual+trial 覆盖（同事务先删后插；
	// legacy 行不受影响）；items 为空即清空两类来源
	ReplaceActiveOverrides(ctx context.Context, tenantID uint, items []model.TenantEntitlementOverride) error
	// DeleteStaleOverrides 删除降级时应清理的覆盖：trial 来源或已到期
	DeleteStaleOverrides(ctx context.Context, tenantID uint, now time.Time) error
	// Migrate 开发/测试 AutoMigrate 路径（FIX-009：生产只走 SQL 迁移）
	Migrate() error
}
