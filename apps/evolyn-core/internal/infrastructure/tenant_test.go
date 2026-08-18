package infrastructure

import (
	"reflect"
	"testing"

	"evolyn/internal/platform/iam/model"

	"github.com/stretchr/testify/assert"
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
