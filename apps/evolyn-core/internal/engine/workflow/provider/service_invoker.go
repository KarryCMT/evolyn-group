package provider

import (
	"context"
	"fmt"
	"time"
)

// ServiceInvoker 服务节点出站调用窄端口（Phase 7，第 12.1/27 章）：内核
// 声明的最小契约，平台适配层承担 HTTP 协议、SSRF 防护、主机白名单、超时
// 与日志脱敏——引擎内核禁依赖 net/http（第 5.3 章依赖规则）。
//
// 事务边界：调用发生在 Job Worker 的 service.invoke 独立事务内，业务
// 事务（发起/审批推进）不承载外部请求；调用失败经 wf_job 重试记账退避
// 重试，不回滚已提交的审批状态。
type ServiceInvoker interface {
	// Invoke 执行一次出站调用。非 2xx 响应与传输失败一律返回 error
	//（由 Worker 重试记账裁决是否重试）；2xx 响应返回状态码与响应体
	//（响应体仅供建内提取，禁止落日志）。
	Invoke(ctx context.Context, req ServiceRequest) (*ServiceResponse, error)
}

// ServiceRequest 服务节点调用请求：URL/Header/Body 已由内核完成模板插值
// （发布期预编译产物求值），适配层不再解释表达式。
type ServiceRequest struct {
	// TenantID / InstanceID / NodeInstanceID 调用归属（审计与幂等键构造）
	TenantID       uint
	InstanceID     uint
	NodeInstanceID uint
	// NodeKey 设计态节点 key（诊断定位）
	NodeKey string
	// Method HTTP 方法（校验器冻结值域）
	Method string
	// URL 插值后的最终请求地址（SSRF 防护在适配层强制）
	URL string
	// Headers 请求头（插值后；不含幂等键，由适配层统一注入）
	Headers map[string]string
	// Body 请求体（JSON 模板插值结果；空=无请求体）
	Body string
	// TimeoutSeconds 单次请求超时（校验器封顶 120s）
	TimeoutSeconds int
}

// ServiceResponse 服务节点调用响应。
type ServiceResponse struct {
	// StatusCode HTTP 状态码（2xx=成功语义，由内核判定后映射变量）
	StatusCode int
	// Body 响应体（适配层限制最大字节数；仅供 JSON 提取，禁止出网/落日志）
	Body []byte
	// Duration 本次调用耗时（操作流水/诊断用）
	Duration time.Duration
}

// ServiceIdempotencyKey 服务节点幂等键（第 7 章幂等策略）：按实例+节点实例
// 维度稳定——同一节点实例的重试/重放对对端呈现同一键，可据此去重。
func ServiceIdempotencyKey(instanceID, nodeInstanceID uint) string {
	return fmt.Sprintf("wf-%d-%d", instanceID, nodeInstanceID)
}
