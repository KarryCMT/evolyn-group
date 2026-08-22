// Package repository 登录日志数据访问：追加写 + 账号维度只读查询。
package repository

import (
	"context"
	"time"

	"evolyn/internal/platform/auth/loginlog/model"
)

// LoginLogRepository 登录日志仓储契约
type LoginLogRepository interface {
	Create(ctx context.Context, log *model.LoginLog) error
	// ListByAccount 按账号倒序分页并返回总数；start 为闭区间下界、endExcl 为
	// 次日零点开区间上界（东八区自然日闭区间语义由服务层换算），零值不过滤
	ListByAccount(ctx context.Context, accountID uint, start, endExcl time.Time, offset, limit int) ([]model.LoginLog, int64, error)
	Migrate() error
}
