package repository

import (
	"context"
	"testing"
	"time"

	kernel "evolyn/internal/model"
	"evolyn/internal/platform/auth/loginlog/model"
	"evolyn/internal/testsupport"

	"github.com/stretchr/testify/assert"
)

// 登录日志仓储集成测试（真实 PostgreSQL，TEST_PG_DSN 未设置时自动跳过）：
// 覆盖账号隔离、倒序分页与东八区自然日闭区间过滤
func TestLoginLogListByAccountIntegration(t *testing.T) {
	db := testsupport.NewPostgres(t)
	repo := NewRepository(db)
	ctx := context.Background()
	cst := kernel.CSTLocation()

	// 账号 1 三条（跨三天）+ 账号 2 一条：验证账号维度隔离。
	// created_at 由用例显式指定（GORM 对非零值不自动填充）
	rows := []model.LoginLog{
		{AccountID: 1, TenantID: 10, MemberID: 100, Method: model.MethodPassword, Client: model.ClientWeb, IP: "210.21.226.222",
			CreatedAt: kernel.JSONTime(time.Date(2026, 8, 20, 9, 0, 0, 0, cst))},
		{AccountID: 1, TenantID: 10, MemberID: 100, Method: model.MethodSMS, Client: model.ClientWap, IP: "27.46.93.3",
			CreatedAt: kernel.JSONTime(time.Date(2026, 8, 21, 18, 30, 0, 0, cst))},
		{AccountID: 1, TenantID: 10, MemberID: 100, Method: model.MethodOAuth + "github", Client: model.ClientWeb, IP: "183.238.228.138",
			CreatedAt: kernel.JSONTime(time.Date(2026, 8, 22, 8, 15, 0, 0, cst))},
		{AccountID: 2, TenantID: 20, MemberID: 200, Method: model.MethodPassword, Client: model.ClientWeb, IP: "1.2.4.8",
			CreatedAt: kernel.JSONTime(time.Date(2026, 8, 22, 10, 0, 0, 0, cst))},
	}
	for i := range rows {
		assert.NoError(t, repo.Create(ctx, &rows[i]))
	}

	t.Run("账号隔离与倒序", func(t *testing.T) {
		logs, total, err := repo.ListByAccount(ctx, 1, time.Time{}, time.Time{}, 0, 10)
		assert.NoError(t, err)
		assert.EqualValues(t, 3, total)
		assert.Len(t, logs, 3)
		// 最新在前（id 倒序）
		assert.Equal(t, model.MethodOAuth+"github", logs[0].Method)
		assert.Equal(t, model.MethodSMS, logs[1].Method)
		assert.Equal(t, model.MethodPassword, logs[2].Method)
	})

	t.Run("分页", func(t *testing.T) {
		logs, total, err := repo.ListByAccount(ctx, 1, time.Time{}, time.Time{}, 1, 1)
		assert.NoError(t, err)
		assert.EqualValues(t, 3, total)
		assert.Len(t, logs, 1)
		assert.Equal(t, model.MethodSMS, logs[0].Method, "第二页第一条应为中位记录")
	})

	t.Run("自然日闭区间", func(t *testing.T) {
		// 只看 8-21：start=8-21 零点、endExcl=8-22 零点，恰好命中一条
		start := time.Date(2026, 8, 21, 0, 0, 0, 0, cst)
		endExcl := time.Date(2026, 8, 22, 0, 0, 0, 0, cst)
		logs, total, err := repo.ListByAccount(ctx, 1, start, endExcl, 0, 10)
		assert.NoError(t, err)
		assert.EqualValues(t, 1, total)
		assert.Len(t, logs, 1)
		assert.Equal(t, model.MethodSMS, logs[0].Method)
	})

	t.Run("开区间上界不含当日零点后", func(t *testing.T) {
		// 8-20 ~ 8-21：endExcl=8-21 零点，8-21 当天的记录不应命中
		start := time.Date(2026, 8, 20, 0, 0, 0, 0, cst)
		endExcl := time.Date(2026, 8, 21, 0, 0, 0, 0, cst)
		_, total, err := repo.ListByAccount(ctx, 1, start, endExcl, 0, 10)
		assert.NoError(t, err)
		assert.EqualValues(t, 1, total)
	})

	t.Run("无匹配", func(t *testing.T) {
		_, total, err := repo.ListByAccount(ctx, 999, time.Time{}, time.Time{}, 0, 10)
		assert.NoError(t, err)
		assert.EqualValues(t, 0, total)
	})
}
