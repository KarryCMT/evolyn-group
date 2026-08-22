// Package service 登录日志域服务：登录链路 best-effort 写入 + 账号自查查询。
package service

import (
	"context"

	"evolyn/internal/platform/auth/loginlog/model"
)

// Entry 登录事件描述：身份字段由登录链路显式传入（登录时 ctx 内尚无
// 操作者/租户上下文可解析），IP/UA/RequestID 从请求元数据自动补全。
// Method 取 loginlog/model 的 Method* 常量（OAuth 为前缀拼接 provider）
type Entry struct {
	AccountID uint
	TenantID  uint
	MemberID  uint
	Method    string
}

// Recorder 登录日志记录器：写入失败只告警不阻断登录主流程（与审计域同口径）
type Recorder interface {
	Record(ctx context.Context, e Entry)
}

// PageQuery 本人登录日志分页查询参数；日期为 yyyy-MM-dd 东八区自然日，
// StartDate/EndDate 构成闭区间，空串不过滤
type PageQuery struct {
	Page      int    // 页码，1 起
	PageSize  int    // 每页条数，上限 100
	StartDate string // yyyy-MM-dd，可空
	EndDate   string // yyyy-MM-dd，可空
}

// PageResult 登录日志分页结果
type PageResult struct {
	Items []model.LoginLog `json:"items"`
	Total int64            `json:"total"`
}

// QueryService 登录日志查询（账号自查）：accountID 由控制器从会话 claims
// 注入，服务层不提供任何跨账号查询面
type QueryService interface {
	ListByAccount(ctx context.Context, accountID uint, q PageQuery) (*PageResult, error)
}
