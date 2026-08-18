package ginctx

import (
	"net/http/httptest"
	"testing"

	"evolyn/internal/platform/iam/model"
	"evolyn/internal/utils/request"
	"evolyn/internal/utils/trace"

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

func TestGinTenantContext(t *testing.T) {
	// gin.Context 注入/读取
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SetTenant(c, 0) // 零值不写入
	assert.Equal(t, uint(0), GetTenant(c))

	SetTenant(c, 3)
	assert.Equal(t, uint(3), GetTenant(c))
}
