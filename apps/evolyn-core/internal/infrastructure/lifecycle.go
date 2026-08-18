package infrastructure

import (
	"context"

	"gorm.io/gorm"
)

// Ping 存活探测：postgres 与 redis 双通道（原全局 Repository.Ping 迁移而来）
func Ping(ctx context.Context, db *gorm.DB, rdb *RedisDB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if err = sqlDB.PingContext(ctx); err != nil {
		return err
	}

	if rdb == nil {
		return nil
	}
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return err
	}

	return nil
}

// Close 释放 postgres/redis 连接（原全局 Repository.Close 迁移而来）
func Close(db *gorm.DB, rdb *RedisDB) error {
	if sqlDB, err := db.DB(); err == nil && sqlDB != nil {
		if err := sqlDB.Close(); err != nil {
			return err
		}
	}

	if rdb != nil {
		if err := rdb.Close(); err != nil {
			return err
		}
	}

	return nil
}
