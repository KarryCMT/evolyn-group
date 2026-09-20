// Package adapter contains platform implementations of form-domain narrow ports.
package adapter

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"evolyn/internal/platform/form/service"

	"github.com/sirupsen/logrus"
)

const frontendEventMaxResponseBytes = 1 << 20

// FrontendEventInvokerConfig reuses the workflow outbound allow-list policy.
type FrontendEventInvokerConfig struct {
	AllowPrivateNetwork bool
	AllowedHosts        []string
}

// FrontendEventInvoker is the SSRF-safe HTTP implementation for form events.
type FrontendEventInvoker struct {
	client  *http.Client
	config  FrontendEventInvokerConfig
	allowed map[string]bool
}

func NewFrontendEventInvoker(config FrontendEventInvokerConfig) *FrontendEventInvoker {
	allowed := make(map[string]bool, len(config.AllowedHosts))
	for _, host := range config.AllowedHosts {
		if normalized := strings.ToLower(strings.TrimSpace(host)); normalized != "" {
			allowed[normalized] = true
		}
	}
	invoker := &FrontendEventInvoker{config: config, allowed: allowed}
	transport := &http.Transport{
		MaxIdleConns:          32,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}
	transport.DialContext = invoker.safeDialContext
	invoker.client = &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return invoker
}

func (i *FrontendEventInvoker) Invoke(ctx context.Context, request service.FrontendEventInvokeRequest) (*service.FrontendEventInvokeResponse, error) {
	parsed, err := i.validateURL(request.URL)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(callCtx, request.Method, request.URL, strings.NewReader(request.Body))
	if err != nil {
		return nil, fmt.Errorf("build frontend event request: %w", err)
	}
	httpRequest.Header.Set("Accept", "application/json, application/xml, text/xml;q=0.9")
	for name, value := range request.Headers {
		if frontendEventRestrictedHeader(name) {
			return nil, fmt.Errorf("%w: restricted header %q is forbidden", service.ErrFrontendEventURLBlocked, name)
		}
		httpRequest.Header.Set(name, value)
	}
	started := time.Now()
	httpResponse, err := i.client.Do(httpRequest)
	if err != nil {
		i.logCall(request, parsed, 0, time.Since(started), err)
		return nil, fmt.Errorf("invoke frontend event: %w", err)
	}
	defer httpResponse.Body.Close() //nolint:errcheck // response body has already been consumed below
	body, err := io.ReadAll(io.LimitReader(httpResponse.Body, frontendEventMaxResponseBytes+1))
	duration := time.Since(started)
	if err != nil {
		i.logCall(request, parsed, httpResponse.StatusCode, duration, err)
		return nil, fmt.Errorf("read frontend event response: %w", err)
	}
	if len(body) > frontendEventMaxResponseBytes {
		err = fmt.Errorf("frontend event response exceeds %d bytes", frontendEventMaxResponseBytes)
		i.logCall(request, parsed, httpResponse.StatusCode, duration, err)
		return nil, err
	}
	i.logCall(request, parsed, httpResponse.StatusCode, duration, nil)
	return &service.FrontendEventInvokeResponse{
		StatusCode: httpResponse.StatusCode,
		Body:       body,
		DurationMS: duration.Milliseconds(),
	}, nil
}

func (i *FrontendEventInvoker) validateURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		return nil, fmt.Errorf("%w: only absolute HTTPS URLs without user info are allowed", service.ErrFrontendEventURLBlocked)
	}
	host := strings.ToLower(parsed.Hostname())
	if len(i.allowed) > 0 && !i.allowed[host] {
		return nil, fmt.Errorf("%w: host is not in outbound allow-list", service.ErrFrontendEventURLBlocked)
	}
	if ip := net.ParseIP(host); ip != nil && !i.config.AllowPrivateNetwork && frontendEventBlockedIP(ip) {
		return nil, fmt.Errorf("%w: private or local address", service.ErrFrontendEventURLBlocked)
	}
	return parsed, nil
}

// safeDialContext validates every DNS result and dials the validated address directly,
// closing the DNS-rebinding gap between URL validation and the actual TCP connection.
func (i *FrontendEventInvoker) safeDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid outbound address", service.ErrFrontendEventURLBlocked)
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve frontend event host: %w", err)
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("resolve frontend event host: no addresses")
	}
	for _, address := range addresses {
		if !i.config.AllowPrivateNetwork && frontendEventBlockedIP(address.IP) {
			return nil, fmt.Errorf("%w: host resolves to private or local address", service.ErrFrontendEventURLBlocked)
		}
	}
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].IP.String(), port))
}

func frontendEventBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}

// frontendEventRestrictedHeader 与保存期校验保持一致。Authorization 允许透传，
// 其余会携带浏览器会话或伪造响应语义的 Header 仍由代理拒绝。
func frontendEventRestrictedHeader(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "cookie", "proxy-authorization", "set-cookie":
		return true
	default:
		return false
	}
}

func (i *FrontendEventInvoker) logCall(request service.FrontendEventInvokeRequest, parsed *url.URL, status int, duration time.Duration, callErr error) {
	entry := logrus.WithFields(logrus.Fields{
		"tenantId":   request.TenantID,
		"eventId":    request.EventID,
		"method":     request.Method,
		"host":       parsed.Host,
		"path":       parsed.Path,
		"status":     status,
		"durationMs": duration.Milliseconds(),
	})
	if callErr != nil {
		entry.WithError(callErr).Warn("form frontend event invoke failed")
		return
	}
	entry.Info("form frontend event invoke")
}
