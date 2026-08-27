// Package model 企业日志域模型（000036）：只读查询条件、受控出网投影与
// 导出任务。登录/操作日志的写入分属 auth/loginlog 与 audit 域，本域只读。
package model

import (
	kernel "evolyn/internal/model"
)

// 登录平台常量（client 列取值，与 loginlog/model 对齐；出网透传，前端映射文案）
const (
	ClientWeb     = "web"
	ClientWap     = "wap"
	ClientUnknown = "unknown"
)

// LoginLogQuery 登录日志查询参数（GET /enterprise-logs/login）：日期为
// yyyy-MM-dd 东八区自然日，StartDate/EndDate 构成闭区间（服务端换算半开
// 区间下推数据库），空串不过滤
type LoginLogQuery struct {
	MemberID  uint   // 可选：当前租户成员 ID（服务端校验租户归属）
	StartDate string // yyyy-MM-dd，可空
	EndDate   string // yyyy-MM-dd，可空
	Page      int    // 页码，1 起
	PageSize  int    // 每页条数，上限 100
}

// OperationLogQuery 操作日志查询参数（GET /enterprise-logs/operations）：
// 在登录日志公共参数外支持日志范围与操作类型筛选
type OperationLogQuery struct {
	MemberID     uint   // 可选：当前租户成员 ID
	CategoryCode string // 可选：日志范围码（见 audit/service 分类常量）
	EventCode    string // 可选：操作类型事件码（见 audit/service 事件注册表）
	StartDate    string
	EndDate      string
	Page         int
	PageSize     int
}

// LoginLogItem 登录日志出网行（受控投影）：仅展示字段，不含 UA/RequestID/
// Method（登录方式保留作详情或后续筛选，不混入列表）
type LoginLogItem struct {
	ActorName string          `json:"actorName"` // 登录人（展示快照优先，存量行回查当前昵称兜底）
	LoggedAt  kernel.JSONTime `json:"loggedAt"`  // 登录时间（秒级东八区）
	Location  string          `json:"location"`  // 登录地
	Client    string          `json:"client"`    // 登录平台（web/wap/unknown，前端映射文案）
	IP        string          `json:"ip"`
}

// LoginLogPage 登录日志分页结果
type LoginLogPage struct {
	Items []LoginLogItem `json:"items"`
	Total int64          `json:"total"`
}

// OperationLogItem 操作日志出网行（受控投影）：不含请求 ID/用户代理/原始
// 快照/内部资源 ID，操作详情为服务端脱敏摘要
type OperationLogItem struct {
	ActorName    string          `json:"actorName"`    // 操作人
	OperatedAt   kernel.JSONTime `json:"operatedAt"`   // 发生时间（秒级东八区）
	CategoryName string          `json:"categoryName"` // 日志范围展示名；存量历史行降级「历史操作记录」
	EventName    string          `json:"eventName"`    // 操作类型展示名；存量历史行降级「历史操作记录」
	Summary      string          `json:"summary"`      // 脱敏后的操作详情
	IP           string          `json:"ip"`
}

// OperationLogPage 操作日志分页结果
type OperationLogPage struct {
	Items []OperationLogItem `json:"items"`
	Total int64              `json:"total"`
}

// EventOption 操作类型筛选项
type EventOption struct {
	Code string `json:"code"` // 稳定事件码
	Name string `json:"name"` // 中文操作名
}

// CategoryOption 日志范围筛选项（含该范围下可选的操作类型清单）
type CategoryOption struct {
	Code   string        `json:"code"`
	Name   string        `json:"name"`
	Events []EventOption `json:"events"`
}
