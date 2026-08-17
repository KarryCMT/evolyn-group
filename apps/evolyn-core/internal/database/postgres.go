package database

import (
	"fmt"

	"evolyn/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgres(conf *config.DBConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
		conf.Host, conf.User, conf.Password, conf.Name, conf.Port)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 租户过滤统一收口：GORM 侧全局 Callback（见 tenant.go）
	if err := RegisterTenantCallbacks(db); err != nil {
		return nil, err
	}

	return db, nil
}
