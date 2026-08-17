package common

import (
	"context"
	"net/http/httptest"
	"testing"

	"evolyn/pkg/utils/request"

	"evolyn/internal/model"

	"evolyn/pkg/utils/trace"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
)

func TestTraceContext(t *testing.T) {

	trace := trace.New("test", logr.Discard())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SetTrace(c, nil)
	assert.Nil(t, GetTrace(c))

	SetTrace(c, trace)

	TraceStep(c, "msg")

	assert.NotNil(t, GetTrace(c))
}

func TestUserContext(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SetUser(c, nil)
	assert.Nil(t, GetUser(c))

	user := &model.User{ID: 1, Name: "some"}
	SetUser(c, user)

	assert.Equal(t, user, GetUser(c))
}

func TestRequestInfoContext(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SetRequestInfo(c, nil)
	assert.Nil(t, GetRequestInfo(c))

	ri := &request.RequestInfo{Verb: "get", Resource: "apps"}
	SetRequestInfo(c, ri)

	assert.Equal(t, ri, GetRequestInfo(c))
}

func TestTenantContext(t *testing.T) {
	// context.Context 注入/读取：GORM Callback 与引擎数据路径的租户来源
	ctx := NewTenantContext(context.Background(), 3)
	tenantID, ok := TenantIDFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, uint(3), tenantID)

	_, ok = TenantIDFromContext(context.Background())
	assert.False(t, ok)

	// gin.Context 注入/读取
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SetTenant(c, 0) // 零值不写入
	assert.Equal(t, uint(0), GetTenant(c))

	SetTenant(c, 3)
	assert.Equal(t, uint(3), GetTenant(c))
}
