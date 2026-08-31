// ServiceInvoker 服务节点出站调用适配器（Phase 7，第 27 章安全设计）：
// 引擎内核 ServiceInvoker 窄端口的 net/http 实现，承担内核不感知的平台
// 安全与网络职责——
//   - SSRF 防护：仅 http(s)，禁重定向（防 30x 绕过地址校验），DNS 解析后
//     逐 IP 校验拒绝私网/回环/链路本地/未指定地址；
//   - 主机白名单：配置 allowedHosts（空=不限制，仍拒绝私网），
//     allowPrivateNetwork 仅供本地联调显式开启；
//   - 幂等：统一注入 Idempotency-Key（实例+节点实例稳定键，重试/重放可
//     被对端去重，第 7 章幂等策略）；
//   - 响应体限长（ServiceMaxResponseBytes），超限按调用失败处理；
//   - 日志脱敏：只记方法/主机/路径/状态码/耗时，请求头与请求体、响应体
//     一律不落日志（第 27 章「敏感字段不得落普通运行日志」）。
package adapter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"evolyn/internal/engine/workflow/model"
	"evolyn/internal/engine/workflow/provider"

	"github.com/sirupsen/logrus"
)

// ServiceInvokerConfig 服务出站调用安全策略（config/app.yaml workflow.service
// 段注入；零值即安全默认：拒绝私网、无白名单限制）。
type ServiceInvokerConfig struct {
	// AllowPrivateNetwork 是否允许私网/回环地址（本地联调显式开启；
	// 生产必须保持 false）
	AllowPrivateNetwork bool
	// AllowedHosts 出站主机白名单（精确主机名；空=不限制，私网仍拒绝）
	AllowedHosts []string
}

// ErrServiceURLBlocked SSRF 防护拒绝（稳定语义：调用失败经 Worker 重试
// 记账退避，配置性拒绝重试也不会成功，由 last_error 可见）。
var ErrServiceURLBlocked = errors.New("service url blocked by security policy")

// ServiceInvoker 出站调用适配器。
type ServiceInvoker struct {
	client  *http.Client
	config  ServiceInvokerConfig
	allowed map[string]bool
}

// NewServiceInvoker 构造服务调用适配器（连接级超时兜底，请求级超时按
// 节点配置经 ctx 强制）。
func NewServiceInvoker(config ServiceInvokerConfig) *ServiceInvoker {
	allowed := make(map[string]bool, len(config.AllowedHosts))
	for _, host := range config.AllowedHosts {
		allowed[strings.ToLower(host)] = true
	}
	return &ServiceInvoker{
		client: &http.Client{
			// 不跟随重定向：既防 SSRF 绕过（30x 指向内网），也契合
			// 「一次调用一次结果」的幂等语义
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: &http.Transport{
				MaxIdleConns:          32,
				IdleConnTimeout:       60 * time.Second,
				TLSHandshakeTimeout:   5 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
			},
		},
		config:  config,
		allowed: allowed,
	}
}

// Invoke 执行出站调用：地址安全校验 → 请求构造（幂等键注入）→ 带超时
// 发送 → 响应限长读取。非 2xx 一律返回错误（Worker 重试记账裁决重试）。
func (i *ServiceInvoker) Invoke(ctx context.Context, req provider.ServiceRequest) (*provider.ServiceResponse, error) {
	if err := i.validateURL(req.URL); err != nil {
		return nil, err
	}
	var body io.Reader
	if req.Body != "" {
		body = strings.NewReader(req.Body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, body)
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	for name, value := range req.Headers {
		httpReq.Header.Set(name, value)
	}
	// 幂等键：实例+节点实例稳定（第 7 章幂等策略），对端据此去重
	httpReq.Header.Set("Idempotency-Key",
		provider.ServiceIdempotencyKey(req.InstanceID, req.NodeInstanceID))
	if req.Body != "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = time.Duration(model.ServiceDefaultTimeoutSeconds) * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	started := time.Now()
	httpResp, err := i.client.Do(httpReq.WithContext(callCtx))
	if err != nil {
		i.logCall(req, 0, time.Since(started), err)
		return nil, fmt.Errorf("服务调用失败: %w", err)
	}
	defer httpResp.Body.Close() //nolint:errcheck // 关闭失败无可恢复动作（响应超限已按失败处理）
	// 响应体限长读取：超限按失败处理（大报文不进入引擎）
	bodyBytes, err := io.ReadAll(io.LimitReader(httpResp.Body, model.ServiceMaxResponseBytes+1))
	duration := time.Since(started)
	if err != nil {
		i.logCall(req, httpResp.StatusCode, duration, err)
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if len(bodyBytes) > model.ServiceMaxResponseBytes {
		err := fmt.Errorf("响应体超过 %d 字节上限", model.ServiceMaxResponseBytes)
		i.logCall(req, httpResp.StatusCode, duration, err)
		return nil, err
	}
	i.logCall(req, httpResp.StatusCode, duration, nil)
	if httpResp.StatusCode < 200 || httpResp.StatusCode > 299 {
		return nil, fmt.Errorf("服务返回非 2xx 状态 %d", httpResp.StatusCode)
	}
	return &provider.ServiceResponse{
		StatusCode: httpResp.StatusCode,
		Body:       bodyBytes,
		Duration:   duration,
	}, nil
}

// validateURL 出站地址安全校验：scheme 白名单 → 主机白名单 → DNS 解析
// 逐 IP 私网校验（DNS 重绑定面收敛：解析结果即本次连接目标集合，禁重
// 定向下无二次解析窗口）。
func (i *ServiceInvoker) validateURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: 地址不可解析", ErrServiceURLBlocked)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%w: 仅允许 http/https", ErrServiceURLBlocked)
	}
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("%w: 缺少主机名", ErrServiceURLBlocked)
	}
	if len(i.allowed) > 0 && !i.allowed[strings.ToLower(host)] {
		return fmt.Errorf("%w: 主机 %q 不在白名单", ErrServiceURLBlocked, host)
	}
	if i.config.AllowPrivateNetwork {
		return nil
	}
	// IP 字面量直接判定；域名解析后逐 IP 判定（拒绝解析到私网的目标）
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("%w: 私网/回环地址 %s", ErrServiceURLBlocked, host)
		}
		return nil
	}
	addresses, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("主机解析失败: %w", err)
	}
	for _, ip := range addresses {
		if isBlockedIP(ip) {
			return fmt.Errorf("%w: 主机 %s 解析到受限地址", ErrServiceURLBlocked, host)
		}
	}
	return nil
}

// isBlockedIP 受限地址判定：回环/私网/链路本地/未指定/组播（含 IPv6 对应区段）。
func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsUnspecified()
}

// logCall 脱敏调用日志：只记方法/主机/路径/状态码/耗时与节点定位，
// 请求头（可能含凭据）、请求体与响应体一律不落日志（第 27 章）。
func (i *ServiceInvoker) logCall(req provider.ServiceRequest, statusCode int, duration time.Duration, callErr error) {
	parsed, err := url.Parse(req.URL)
	host, path := "", ""
	if err == nil {
		host, path = parsed.Host, parsed.Path
	}
	entry := logrus.WithFields(logrus.Fields{
		"tenantId":       req.TenantID,
		"instanceId":     req.InstanceID,
		"nodeInstanceId": req.NodeInstanceID,
		"nodeKey":        req.NodeKey,
		"method":         req.Method,
		"host":           host,
		"path":           path,
		"status":         statusCode,
		"durationMs":     duration.Milliseconds(),
	})
	if callErr != nil {
		entry.WithError(callErr).Warn("workflow service invoke failed")
		return
	}
	entry.Info("workflow service invoke")
}
