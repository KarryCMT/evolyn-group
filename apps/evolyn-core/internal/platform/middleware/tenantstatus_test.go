package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/tenant/repository"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// fakeTenantRepo 状态拦截测试桩：仅实现 GetStatus，其余方法零实现
type fakeTenantRepo struct {
	repository.TenantRepository
	status map[uint]string
}

func (f fakeTenantRepo) GetStatus(ctx context.Context, id uint) (string, error) {
	if s, ok := f.status[id]; ok {
		return s, nil
	}
	return "active", nil
}

func runStatusRequest(t *testing.T, tenantID uint, status string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	// 模拟 TenantMiddleware 已注入租户上下文（tenantID=0 表示未注入）；
	// TenantStatusMiddleware 从 gin 上下文取租户
	router.Use(func(c *gin.Context) {
		if tenantID != 0 {
			ginctx.SetTenant(c, tenantID)
		}
		c.Next()
	})
	router.Use(TenantStatusMiddleware(fakeTenantRepo{status: map[uint]string{tenantID: status}}))
	router.GET("/api/v1/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	router.ServeHTTP(w, req)
	return w
}

func TestTenantStatusMiddleware(t *testing.T) {
	cases := []struct {
		name       string
		tenantID   uint
		status     string
		wantCode   int
		wantInBody string
	}{
		{"active passes", 1, "active", http.StatusOK, ""},
		{"frozen rejected", 1, "frozen", http.StatusForbidden, "TENANT_FROZEN"},
		{"deleted rejected", 1, "deleted", http.StatusForbidden, "TENANT_DISABLED"},
		{"unknown status rejected", 1, "weird", http.StatusForbidden, "TENANT_DISABLED"},
		{"no tenant context passes", 0, "", http.StatusOK, ""}, // 登录/平台域链路
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := runStatusRequest(t, tc.tenantID, tc.status)
			assert.Equal(t, tc.wantCode, w.Code)
			if tc.wantInBody != "" {
				assert.Contains(t, w.Body.String(), tc.wantInBody)
			}
		})
	}
}
