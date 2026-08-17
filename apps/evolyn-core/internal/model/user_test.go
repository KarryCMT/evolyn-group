package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserCacheKey(t *testing.T) {
	// Redis Key 规范：{resource}:{tenant}:{rest}
	u := &User{BaseModel: BaseModel{TenantID: 2}}
	assert.Equal(t, "users:2:id", u.CacheKey())

	// 请求构造对象未携带租户时兜底默认租户
	u0 := &User{}
	assert.Equal(t, "users:1:id", u0.CacheKey())
}
