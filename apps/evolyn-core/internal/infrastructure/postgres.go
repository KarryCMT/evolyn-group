package infrastructure

import (
	"fmt"
	"log"
	"os"
	"time"

	"evolyn/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewPostgres(conf *config.DBConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
		conf.Host, conf.User, conf.Password, conf.Name, conf.Port)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true, // 查重类 First() 0 行命中是预期，降噪（ADR-008 顺手项）
				Colorful:                  false,
			},
		),
	})
	if err != nil {
		return nil, err
	}

	// 租户过滤统一收口：GORM 侧全局 Callback（见 tenant.go）
	if err := RegisterTenantCallbacks(db); err != nil {
		return nil, err
	}

	return db, nil
}
