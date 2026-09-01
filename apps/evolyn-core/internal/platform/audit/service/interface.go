package service

import (
	"context"
)

// Entry 审计事件描述：Module/Action/Resource* 为必填语义，身份字段为 0 时
// 自动从请求 ctx 解析（Actor/RequestMeta/Tenant），平台运营域等无租户上下文
// 的调用方需显式填写 TenantID。
//
// 企业日志展示投影（000036，全部可选）：EventCode/CategoryCode 缺省时按
// module+resourceType+action 经事件注册表推导（见 events.go）；ActorName
// 空时经 ActorNamer 按成员 ID 解析；TargetName 空时从 Before/After 的
// name/nickname 展示级键提取；Summary 空时由注册表模板生成（只拼装展示级
// 字段，禁止把密码/验证码/令牌/完整手机号邮箱等敏感值写入任何投影字段）
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

	EventCode    string // 显式事件码（覆盖注册表推导，新业务专用）
	CategoryCode string // 显式日志范围码（EventCode 显式且本字段为空时按注册表补全）
	ActorName    string // 操作人显示名快照（空时经 ActorNamer 按成员 ID 解析）
	TargetName   string // 目标资源展示名快照（空时从 Before/After 提取）
	Summary      string // 直接指定操作详情（覆盖模板生成，须自行保证脱敏）

	// 产品日志应用维度快照（000064，全部可选）：应用内操作填写，写时固化
	// （应用删除/改名后历史展示不失真）；ApplicationID 为 0 时三字段均不落
	ApplicationID   uint   // 应用 ID（查询与租户归属校验维度）
	ApplicationCode string // 应用稳定编码快照
	ApplicationName string // 应用名称快照
}

// ActorNamer 操作者显示名解析窄端口：装配层以 iam 成员仓储适配（audit 域
// 不反向依赖 iam）；仅用于写时固化快照，解析失败返回空串即可
type ActorNamer interface {
	MemberDisplayName(ctx context.Context, memberID uint) string
}

// Recorder 业务审计记录器（FIX-013）：失败只记日志不阻断业务主流程
type Recorder interface {
	Record(ctx context.Context, e Entry)
}
