// Package repository 自定义工作台域数据访问（ADR-007 域内小三层）：仅做
// 持久化，一律经 infrastructure.ResolveDB 取连接加入 ctx 传播事务
// （FIX-020/021）。行定位由显式 tenantID 条件表达（租户开通种子的 ctx
// 租户与目标租户一致，显式条件消除歧义），不依赖 GORM 租户 Callback
package repository

import (
	"context"

	"evolyn/internal/platform/workbench/model"
)

// Repository 自定义工作台域仓储
type Repository interface {
	// FindByTenant 取租户工作台行；不存在返回 gorm.ErrRecordNotFound
	FindByTenant(ctx context.Context, tenantID uint) (*model.TenantWorkbench, error)
	// Create 插入工作台行（租户开通种子/首存）；(tenant_id) 唯一约束冲突
	// 返回错误，由服务层判定幂等或映射冲突
	Create(ctx context.Context, record *model.TenantWorkbench) error
	// UpdateContentWithRevision 乐观更新文档：revision 匹配才写入并同句递增
	// （revision+1 由 SQL 表达式完成，避免读改写竞态），0 行影响即版本过期
	UpdateContentWithRevision(ctx context.Context, id uint, fromRevision int64, content model.Content) (bool, error)

	// Migrate 开发/测试 AutoMigrate 路径（FIX-009：生产只走 SQL 迁移）
	Migrate() error
}
