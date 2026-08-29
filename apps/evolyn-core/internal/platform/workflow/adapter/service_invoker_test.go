// Phase 7 适配器测试：出站调用安全策略——SSRF 私网拒绝/白名单/方案白名单、
// 幂等键注入、非 2xx 失败语义、响应体限长。
package adapter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"evolyn/internal/engine/workflow/provider"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateURLBlocksPrivateTargets(t *testing.T) {
	invoker := NewServiceInvoker(ServiceInvokerConfig{})
	for _, raw := range []string{
		"http://127.0.0.1:8080/x",
		"http://10.0.0.1/x",
		"http://192.168.1.1/x",
		"http://[::1]/x",
		"http://localhost/x",
	} {
		err := invoker.validateURL(raw)
		require.Error(t, err, raw)
		assert.True(t, errors.Is(err, ErrServiceURLBlocked), raw)
	}
	// 方案白名单与可解析性
	assert.True(t, errors.Is(invoker.validateURL("ftp://example.com/x"), ErrServiceURLBlocked))
	assert.Error(t, invoker.validateURL("http://non-existent-host.invalid/x"))
}

func TestValidateURLAllowedHostsAndPrivateBypass(t *testing.T) {
	// 白名单：未登记主机拒绝；登记主机（公网 IP 字面量，免 DNS）放行
	invoker := NewServiceInvoker(ServiceInvokerConfig{AllowedHosts: []string{"93.184.216.34"}})
	assert.True(t, errors.Is(invoker.validateURL("https://other.example.com/x"), ErrServiceURLBlocked))
	assert.NoError(t, invoker.validateURL("https://93.184.216.34/x"))

	// 本地联调开关：显式允许私网（生产保持默认拒绝）
	local := NewServiceInvoker(ServiceInvokerConfig{AllowPrivateNetwork: true})
	assert.NoError(t, local.validateURL("http://127.0.0.1:8080/x"))
}

func TestInvokeEndToEnd(t *testing.T) {
	var gotKey, gotContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("Idempotency-Key")
		gotContentType = r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"o-1"}}`))
	}))
	defer server.Close()

	invoker := NewServiceInvoker(ServiceInvokerConfig{AllowPrivateNetwork: true})
	resp, err := invoker.Invoke(context.Background(), provider.ServiceRequest{
		TenantID: 1, InstanceID: 7, NodeInstanceID: 3,
		Method: "POST", URL: server.URL,
		Headers: map[string]string{"X-Trace": "t-1"}, Body: `{"a":1}`,
		TimeoutSeconds: 5,
	})
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, `{"data":{"id":"o-1"}}`, string(resp.Body))
	assert.Equal(t, provider.ServiceIdempotencyKey(7, 3), gotKey)
	assert.Equal(t, "application/json", gotContentType)
}

func TestInvokeNon2xxIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	invoker := NewServiceInvoker(ServiceInvokerConfig{AllowPrivateNetwork: true})
	_, err := invoker.Invoke(context.Background(), provider.ServiceRequest{
		Method: "GET", URL: server.URL, TimeoutSeconds: 5,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}
