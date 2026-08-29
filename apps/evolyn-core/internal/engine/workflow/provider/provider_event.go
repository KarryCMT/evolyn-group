package provider

import (
	"context"
	"time"
)

// EventPublisher 领域事件发布窄端口（第 18 章）：内核声明的最小契约，
// 平台适配层桥接既有 notification 域 EventPublisher.PublishInTx /
// notification_outbox_events，不新建 domain_outbox。
//
// 事务原则（18.3）：人工审批状态更新 + wf_operation + outbox 发布记录
// 必须同一数据库事务（实现侧经 ResolveDB 加入调用方事务）；通知/Webhook
// 等外部消费失败只影响消费侧重试，不回滚审批事务。
type EventPublisher interface {
	PublishInTx(ctx context.Context, event Event) error
}

// Event 领域事件信封：EventName 取自 event 包冻结目录；
// Parameters 键白名单由平台层事件目录约束（稳定动作码 + 参数键白名单）。
type Event struct {
	// EventName 冻结事件名（workflow.*）
	EventName string
	// TenantID / InstanceID / TaskID 事件归属（TaskID 0=实例级事件）
	TenantID   uint
	InstanceID uint
	TaskID     uint
	// NodeInstanceID 节点实例维度归属（0=不区分）：同一实例内可重复发生的
	// 事件（如退回发起人）以它参与构造平台侧幂等键（Phase 6）
	NodeInstanceID uint
	// ActorMemberID 事件发起成员（0=系统，如超时自动动作）
	ActorMemberID uint
	// Parameters 受控参数（string 扁平键值，禁止嵌套结构）
	Parameters map[string]string
	// OccurredAt 发生时间（零值由适配层取当前时间）
	OccurredAt time.Time
}

// Clock 时钟窄端口：内核不直接取系统时间，Worker/事务时间戳统一经此
// 注入，保证测试可注入假时钟（第 19 章 execute_at 计算依赖）。
type Clock interface {
	Now() time.Time
}
