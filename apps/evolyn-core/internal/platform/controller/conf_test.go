package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"evolyn/internal/platform/auth/pki"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAppConfRoute 应用配置端点：匿名可访问，返回统一响应封装，
// 区号列表含简道云 conf 口径的三语文案与值，pki 段下发 RSA 公钥
func TestAppConfRoute(t *testing.T) {
	keypair, err := pki.Load("")
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	e := gin.New()
	api := e.Group("/api/v1")
	NewAppConfController(keypair).RegisterRoute(api)

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
	assert.Contains(t, body, `"pki":{"algorithm":"rsa","keys":{"public_key":"-----BEGIN PUBLIC KEY-----`)
}
