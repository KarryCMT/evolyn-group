package service

import (
	"context"
)

// Entry 审计事件描述：Module/Action/Resource* 为必填语义，身份字段为 0 时
// 自动从请求 ctx 解析（Actor/RequestMeta/Tenant），平台运营域等无租户上下文
// 的调用方需显式填写 TenantID
type Entry struct {
	Module       string      // 业务域：tenant / iam / ...
	Action       string      // 动作：create / update / delete / bind / unbind / status / ...
	ResourceType string      // 资源类型：tenant / member / role / group / department / ...
	ResourceID   string      // 资源标识（ID 或编码）
	TenantID     uint        // 0 时从 ctx 解析
	AccountID    uint        // 0 时从 ctx 解析
	MemberID     uint        // 0 时从 ctx 解析
	Before       interface{} // 变更前快照，可空
	After        interface{} // 变更后快照，可空
}

// Recorder 业务审计记录器（FIX-013）：失败只记日志不阻断业务主流程
type Recorder interface {
	Record(ctx context.Context, e Entry)
}
