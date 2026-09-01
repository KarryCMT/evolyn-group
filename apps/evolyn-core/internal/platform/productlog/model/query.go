// Package model 产品日志域模型（000064）：只读查询条件、受控出网投影与
// 导出任务。产品日志的写入分属 audit 域（业务事务提交后 best-effort 落
// tn_audit_logs），本域只读，不反向耦合应用/表单/流程等写路径。
package model

import (
	kernel "evolyn/internal/model"
)

// ProductLogQuery 产品日志查询参数（GET /product-logs）：日期为 yyyy-MM-dd
// 东八区自然日，StartDate/EndDate 构成闭区间（服务端换算半开区间下推
// 数据库），空串不过滤；Keyword 匹配应用名/操作对象/操作详情（受控展示
// 字段，不查原始快照）
type ProductLogQuery struct {
	CategoryCode  string // 可选：产品日志范围码（见 audit/service 产品分类）
	EventCode     string // 可选：操作类型事件码
	MemberID      uint   // 可选：当前租户成员 ID（服务端校验租户归属）
	ApplicationID uint   // 可选：当前租户应用 ID（服务端校验归属；不可替代租户过滤）
	Keyword       string // 可选：应用/对象/摘要关键词
	StartDate     string
	EndDate       string
	Page          int
	PageSize      int
}

// ProductLogItem 产品日志出网行（受控投影）：不含请求 ID/用户代理/原始
// 快照/内部资源 ID；所属应用为写时快照（应用删除后仍可展示）
type ProductLogItem struct {
	ActorName       string          `json:"actorName"`       // 操作人（快照优先，存量行回查当前昵称兜底）
	OperatedAt      kernel.JSONTime `json:"operatedAt"`      // 操作时间（秒级东八区）
	CategoryCode    string          `json:"categoryCode"`    // 日志范围码（前端筛选项联动用）
	CategoryName    string          `json:"categoryName"`    // 日志范围展示名；存量历史行降级「历史操作记录」
	EventName       string          `json:"eventName"`       // 操作类型展示名；存量历史行降级「历史操作记录」
	ApplicationName string          `json:"applicationName"` // 所属应用名称快照；非应用内操作为空串
	TargetName      string          `json:"targetName"`      // 操作对象（目标资源名称快照）
	Summary         string          `json:"summary"`         // 服务端脱敏操作详情
	IP              string          `json:"ip"`
}

// ProductLogPage 产品日志分页结果
type ProductLogPage struct {
	Items []ProductLogItem `json:"items"`
	Total int64            `json:"total"`
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

// MemberOption 操作人筛选项（当前租户有效成员）
type MemberOption struct {
	MemberID uint   `json:"memberId"`
	Name     string `json:"name"` // 成员显示名（昵称，空则前端回落账号昵称口径由服务端兜底为昵称/登录名）
}

// ApplicationOption 应用筛选项（当前租户有效应用；已删除应用只在列表结果
// 中按快照展示，不作为可选筛选项）
type ApplicationOption struct {
	ApplicationID uint   `json:"applicationId"`
	Code          string `json:"code"`
	Name          string `json:"name"`
}

// ProductLogOptions 筛选项聚合出网（GET /product-logs/options）
type ProductLogOptions struct {
	Categories   []CategoryOption    `json:"categories"`
	Members      []MemberOption      `json:"members"`
	Applications []ApplicationOption `json:"applications"`
}
