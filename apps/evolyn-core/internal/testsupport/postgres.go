// Package testsupport 集成测试基础设施（FIX-022/023）：为需要真实
// PostgreSQL 的测试提供按需建库、迁移与清理，测试间完全隔离。
// DSN 取自环境变量 TEST_PG_DSN（指向任意有 createdb 权限的库，如
// postgres 维护库）；未设置时测试自动 Skip，保持离线单测套件全绿
package testsupport

import (
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"testing"
	"time"

	"evolyn/internal/config"
	"evolyn/internal/infrastructure"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PostgresDsn 返回集成测试 DSN；未配置时第二个返回值为 false
func PostgresDsn() (string, bool) {
	dsn := os.Getenv("TEST_PG_DSN")
	return dsn, dsn != ""
}

// NewPostgres 为单个测试创建独立数据库（名称随机，测试结束自动 DROP），
// 应用 migrations/ 全部迁移并注册 GORM 租户 Callback——等价于生产启动路径
func NewPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	db := NewPostgresRaw(t)

	// 生产同款链路：版本化迁移 + 租户 Callback（FIX-009/022）
	if err := infrastructure.NewMigrator(db).Up(); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return db
}

// NewPostgresRaw 仅建库并注册 GORM 租户 Callback，不应用迁移：
// 供 Migration 集成测试从空库开始自行驱动迁移（MIGRATE-INT-*）
func NewPostgresRaw(t *testing.T) *gorm.DB {
	t.Helper()

	dsn, ok := PostgresDsn()
	if !ok {
		t.Skip("TEST_PG_DSN not set; skipping PostgreSQL integration test")
	}

	admin := openDB(t, dsn)
	dbName := fmt.Sprintf("evolyn_it_%d_%d", time.Now().UnixNano(), rand.Intn(100000))

	if err := admin.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName)).Error; err != nil {
		t.Fatalf("create test database %s: %v", dbName, err)
	}
	t.Cleanup(func() {
		// WITH (FORCE) 断开残留连接（PG >= 13），保证 DROP 总能成功
		if err := admin.Exec(fmt.Sprintf("DROP DATABASE %s WITH (FORCE)", dbName)).Error; err != nil {
			t.Errorf("drop test database %s: %v", dbName, err)
		}
	})

	testDsn := replaceDatabase(t, dsn, dbName)
	db := openDB(t, testDsn)

	if err := infrastructure.RegisterTenantCallbacks(db); err != nil {
		t.Fatalf("register tenant callbacks: %v", err)
	}
	return db
}

// DisabledRedis 返回禁用模式的 Redis：HGet 报 RedisDisableError、写操作
// 静默跳过，集成测试无需真实 Redis
func DisabledRedis() *infrastructure.RedisDB {
	rdb, err := infrastructure.NewRedisClient(&config.RedisConfig{Enable: false})
	if err != nil {
		panic(fmt.Sprintf("testsupport: disabled redis should never fail: %v", err))
	}
	return rdb
}

func openDB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open postgres (%s): %v", redactDsn(dsn), err)
	}
	return db
}

// replaceDatabase 把 DSN 的库名替换为测试库（兼容 URL 与 key=value 两种形式）
func replaceDatabase(t *testing.T, dsn, dbName string) string {
	t.Helper()
	if u, err := url.Parse(dsn); err == nil && u.Scheme != "" {
		u.Path = "/" + dbName
		return u.String()
	}
	// key=value 形式：无 database 键时追加
	if len(dsn) > 0 && dsn[len(dsn)-1] != ' ' {
		dsn += " "
	}
	return dsn + "database=" + dbName
}

// redactDsn 日志脱敏：去掉查询串中可能携带的密码
func redactDsn(dsn string) string {
	if u, err := url.Parse(dsn); err == nil && u.Scheme != "" {
		if u.User != nil {
			u.User = url.User(u.User.Username())
		}
		return u.String()
	}
	return "<key-value dsn>"
}
