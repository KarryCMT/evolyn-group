package infrastructure

import (
	"context"
	"reflect"
	"testing"

	"evolyn/internal/contextx"
	"evolyn/internal/platform/iam/model"
	workflowmodel "evolyn/internal/platform/workflow/model"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestSetTenantID(t *testing.T) {
	// 结构体：零值填充
	u := model.User{}
	setTenantID(reflect.ValueOf(&u).Elem(), 5)
	assert.Equal(t, uint(5), u.TenantID)

	// 已有租户值不覆盖
	setTenantID(reflect.ValueOf(&u).Elem(), 9)
	assert.Equal(t, uint(5), u.TenantID)

	// 切片批量填充
	users := make([]model.User, 2)
	setTenantID(reflect.ValueOf(users), 7)
	assert.Equal(t, uint(7), users[0].TenantID)
	assert.Equal(t, uint(7), users[1].TenantID)
}

func TestTenantQueryCallbackQualifiesCurrentTableInJoin(t *testing.T) {
	// DryRun 只编译 SQL，不连接数据库；覆盖审批中心这类主表与关联表都含
	// tenant_id 的查询，确保租户 Callback 不会生成 PostgreSQL 歧义列。
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost"}), &gorm.Config{
		DisableAutomaticPing: true,
		DryRun:               true,
	})
	if !assert.NoError(t, err) {
		return
	}
	assert.NoError(t, RegisterTenantCallbacks(db))

	statement := db.WithContext(contextx.NewTenantContext(context.Background(), 7)).
		Model(&workflowmodel.WfTask{}).
		Joins("JOIN wf_task_actor ON wf_task_actor.task_id = wf_task.id").
		Where("wf_task.status IN ?", []string{"PENDING"}).
		Select("wf_task.*").
		Find(&[]workflowmodel.WfTask{}).Statement

	assert.Contains(t, statement.SQL.String(), `"wf_task"."tenant_id"`)
	assert.NotContains(t, statement.SQL.String(), "deleted_at")
}
