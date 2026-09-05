package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRuntimeModelsDoNotEnableSoftDelete(t *testing.T) {
	// Runtime 表是不可软删的历史事实。DryRun 编译查询即可防止未来误嵌入
	// 含 gorm.DeletedAt 的通用基类，从而查询数据库中不存在的 deleted_at 列。
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost"}), &gorm.Config{
		DisableAutomaticPing: true,
		DryRun:               true,
	})
	if !assert.NoError(t, err) {
		return
	}

	for name, destination := range map[string]any{
		"instance":   &[]WfInstance{},
		"execution":  &[]WfExecution{},
		"node":       &[]WfNodeInstance{},
		"task":       &[]WfTask{},
		"task actor": &[]WfTaskActor{},
	} {
		t.Run(name, func(t *testing.T) {
			statement := db.Find(destination).Statement
			assert.NotContains(t, statement.SQL.String(), "deleted_at")
		})
	}
}
