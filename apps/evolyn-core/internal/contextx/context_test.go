package contextx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTenantContext(t *testing.T) {
	// context.Context 注入/读取：GORM Callback 与引擎数据路径的租户来源
	ctx := NewTenantContext(context.Background(), 3)
	tenantID, ok := TenantIDFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, uint(3), tenantID)

	_, ok = TenantIDFromContext(context.Background())
	assert.False(t, ok)
}
