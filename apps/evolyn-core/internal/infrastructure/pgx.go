package infrastructure

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"evolyn/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPgxPool(conf *config.DBConfig) (*pgxpool.Pool, error) {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(conf.User, conf.Password),
		Host:   fmt.Sprintf("%s:%d", conf.Host, conf.Port),
		Path:   conf.Name,
	}

	query := u.Query()
	query.Set("sslmode", "disable")
	u.RawQuery = query.Encode()

	dsn := u.String()

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pgx config failed: %w", err)
	}

	poolConfig.MaxConns = 30
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = time.Minute

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool failed: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres failed: %w", err)
	}

	return pool, nil
}
