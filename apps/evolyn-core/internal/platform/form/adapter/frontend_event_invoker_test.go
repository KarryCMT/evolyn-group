package adapter

import (
	"errors"
	"net"
	"testing"

	"evolyn/internal/platform/form/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrontendEventInvokerURLPolicy(t *testing.T) {
	invoker := NewFrontendEventInvoker(FrontendEventInvokerConfig{
		AllowedHosts: []string{"api.lingyanyun.com"},
	})
	_, err := invoker.validateURL("https://127.0.0.1/internal")
	require.Error(t, err)
	assert.ErrorIs(t, err, service.ErrFrontendEventURLBlocked)

	_, err = invoker.validateURL("https://example.com/member")
	require.Error(t, err)
	assert.ErrorIs(t, err, service.ErrFrontendEventURLBlocked)

	parsed, err := invoker.validateURL("https://api.lingyanyun.com/member")
	require.NoError(t, err)
	assert.Equal(t, "api.lingyanyun.com", parsed.Hostname())
}

func TestFrontendEventBlockedIP(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "172.16.0.1", "192.168.1.1", "169.254.1.1", "::1"} {
		assert.True(t, frontendEventBlockedIP(net.ParseIP(raw)), raw)
	}
	assert.False(t, frontendEventBlockedIP(net.ParseIP("8.8.8.8")))
	assert.True(t, errors.Is(service.ErrFrontendEventURLBlocked, service.ErrFrontendEventURLBlocked))
}

func TestFrontendEventRestrictedHeader(t *testing.T) {
	assert.False(t, frontendEventRestrictedHeader("Authorization"))
	assert.True(t, frontendEventRestrictedHeader("Cookie"))
}
