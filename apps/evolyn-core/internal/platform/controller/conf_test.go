package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestAppConfRoute 应用配置端点：匿名可访问，返回统一响应封装，
// 区号列表含简道云 conf 口径的三语文案与值
func TestAppConfRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	api := e.Group("/api/v1")
	NewAppConfController().RegisterRoute(api)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/app/conf", nil)
	e.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"code":200`)
	assert.Contains(t, body, `"calling_code_list"`)
	assert.Contains(t, body, `"value":"+86"`)
	assert.Contains(t, body, `"value":"+886"`)
	assert.Contains(t, body, `"value":"+852"`)
	assert.Contains(t, body, `"value":"+853"`)
	assert.Contains(t, body, `"zh_cn":"中国台湾 +886"`)
	assert.Contains(t, body, `"tenant_register":true`)
	assert.Contains(t, body, `"version"`)
}
