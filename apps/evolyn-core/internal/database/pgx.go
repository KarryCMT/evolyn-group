package database

import (
	"context"
	"fmt"
	"time"

	"evolyn/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPgxPool 创建 PostgreSQL pgx 连接池。
// pgx 主要用于低代码引擎中的动态 SQL、DDL、批量操作等场景。
func NewPgxPool(conf *config.DBConfig) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		conf.User,
		conf.Password,
		conf.Host,
		conf.Port,
		conf.Name,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pgx config failed: %w", err)
	}

	// 后续建议放到 app.yaml 配置。
	poolConfig.MaxConns = 30
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool failed: %w", err)
	}

	// 启动阶段立即验证数据库连接。
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres failed: %w", err)
	}

	return pool, nil
}
