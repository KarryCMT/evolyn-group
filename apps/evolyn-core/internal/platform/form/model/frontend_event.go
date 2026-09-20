package model

// FrontendEventRequestEntry 是前端事件 Header/Body 的稳定键值协议。
type FrontendEventRequestEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// FrontendEventRequest 描述受控 HTTP 请求；Format 是响应解析格式。
type FrontendEventRequest struct {
	Method string                      `json:"method"`
	URL    string                      `json:"url"`
	Header []FrontendEventRequestEntry `json:"header"`
	Body   []FrontendEventRequestEntry `json:"body"`
	Format string                      `json:"format"`
}

// FrontendEventAction 将响应中的受限路径映射回表单字段。
type FrontendEventAction struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

// FrontendEvent 是协议 v10 的前端事件定义。依赖索引只用于客户端快速判断，
// 服务端校验与执行始终从模板重新提取引用，绝不信任索引数组。
type FrontendEvent struct {
	ID              string                `json:"id"`
	Enabled         bool                  `json:"enabled"`
	Name            string                `json:"name"`
	Description     string                `json:"description"`
	Trigger         string                `json:"trigger"`
	TriggerType     string                `json:"trigger_type"`
	RequestType     int                   `json:"request_type"`
	Request         FrontendEventRequest  `json:"request"`
	RequestRely     []string              `json:"request_rely"`
	Action          []FrontendEventAction `json:"action"`
	ActionRely      []string              `json:"action_rely"`
	SubformFillRule string                `json:"subform_fill_rule"`
}

// FrontendEventExecuteRequest 是调试/运行统一输入。Sequence 由浏览器递增，
// 服务端原样回传，客户端仅接受最新序号的结果。
type FrontendEventExecuteRequest struct {
	Values   map[string]any `json:"values"`
	Sequence int64          `json:"sequence"`
}

type FrontendEventRequestSummary struct {
	Method      string   `json:"method"`
	URL         string   `json:"url"`
	HeaderNames []string `json:"headerNames"`
	Body        string   `json:"body,omitempty"`
}

type FrontendEventResponseSummary struct {
	StatusCode int    `json:"statusCode"`
	DurationMS int64  `json:"durationMs"`
	Format     string `json:"format"`
	Body       string `json:"body"`
}

// FrontendEventExecuteResult 不回显请求 Header 值，避免凭据进入浏览器和日志。
type FrontendEventExecuteResult struct {
	Sequence        int64                        `json:"sequence"`
	Writes          map[string]any               `json:"writes"`
	RequestSummary  FrontendEventRequestSummary  `json:"requestSummary"`
	ResponseSummary FrontendEventResponseSummary `json:"responseSummary"`
	ErrorCode       string                       `json:"errorCode,omitempty"`
}
