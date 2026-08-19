package infrastructure

import (
	"context"

	"gorm.io/gorm"
)

// txSessionKey 事务 session 在 ctx 中的携带键（FIX-020/021）：
// 键类型不导出，事务的开启/传播只允许经 TxManager 与 ResolveDB 收口
type txSessionKey struct{}

// TxManager 统一事务管理器（FIX-020/021 整改第一步）：Service 层通过
// WithinTransaction 声明原子边界，事务 session 经 ctx 向下传播，Repository
// 侧统一经 ResolveDB 取连接，自动加入同一事务——避免各仓储独立连接造成
// 「前序已提交、后续失败」的半写状态
type TxManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{db: db}
}

// WithinTransaction 在单个数据库事务内执行 fn：fn 返回 nil 提交，返回
// error 整体回滚。ctx 已携带事务 session 时直接复用外层事务（嵌套不新开），
// 保证「Service → Service」跨层调用共享同一原子边界
func (m *TxManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if InTransaction(ctx) {
		return fn(ctx)
	}
	// WithContext 先行：tx 内语句的 Statement.Context 继承调用方 ctx，
	// GORM 租户 Callback（从 Statement.Context 读租户）在事务内照常生效
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txSessionKey{}, tx))
	})
}

// InTransaction 当前 ctx 是否已携带事务 session
func InTransaction(ctx context.Context) bool {
	_, ok := ctx.Value(txSessionKey{}).(*gorm.DB)
	return ok
}

// ResolveDB 解析 ctx 携带的事务 session，是 Repository 取连接的唯一入口：
// 有事务则加入外层事务（随其提交/回滚），无事务退回常规请求会话。
// 事务 session 上再 WithContext 只是更新 Statement.Context（clone==0 原地
// 修改），不会新开连接或破坏事务，同时保证派生 ctx（如剥离租户的
// contextx.DetachTenant）的过滤语义在事务内仍然成立
func ResolveDB(ctx context.Context, base *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txSessionKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return base.WithContext(ctx)
}
