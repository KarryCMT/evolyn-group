// Package service 消息中心域服务（ADR-007 域内小三层）：事件目录注册表、
// 成员收件箱、租户通知设置、事务 Outbox 发布端口与扇出 Dispatcher。
// 跨域依赖一律走窄端口（MemberDirectory/AdminRecipientResolver），由装配层
// 适配，域内不得反向导入具体 IAM Service。
package service

import (
	"context"
	"time"

	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/notification/model"
	"evolyn/internal/platform/notification/repository"

	"github.com/sirupsen/logrus"
)

// TxManager 事务边界抽象（FIX-021）：与 application 域同形，具体实现在
// infrastructure，经 server.go 装配注入。
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// MemberDirectory 成员目录窄端口（iam 装配层适配）：校验成员租户归属/有效
// 状态并解析显示名；Dispatcher 扇出前的成员复核与模板 actorName 固化使用。
type MemberDirectory interface {
	ValidateMember(ctx context.Context, tenantID, memberID uint) error
	MemberDisplayName(ctx context.Context, tenantID, memberID uint) string
}

// AdminRecipientResolver 系统管理员解析窄端口：按内置系统管理员组的
// tenant-admin 角色绑定实时推导成员（单一事实源），不依赖可改名的角色名。
type AdminRecipientResolver interface {
	ResolveAdminMemberIDs(ctx context.Context, tenantID uint) ([]uint, error)
}

// InboxService 成员收件箱服务：只能读写「当前租户 × 当前成员」的收件箱，
// 已读状态不记业务审计（高频交互噪声）。
type InboxService interface {
	// UnreadSummary 顶栏未读摘要（只含未读数大于 0 的分类，排除过期消息）
	UnreadSummary(ctx context.Context, tenantID, memberID uint) (*model.UnreadSummaryView, error)
	// ListInbox 分类游标分页（eventCode 必须属于 categoryId；游标固化 occurred_at+id）
	ListInbox(ctx context.Context, tenantID, memberID uint, q model.InboxQuery) (*model.InboxPageView, error)
	// MarkRead 幂等标记单条已读（不改写首次 read_at），响应携带新摘要避免前端二次请求
	MarkRead(ctx context.Context, tenantID, memberID, inboxID uint) (*model.UnreadSummaryView, error)
	// MarkAllRead 标记当前分类（可选事件）through 之前的未读消息为已读
	MarkAllRead(ctx context.Context, tenantID, memberID uint, req model.ReadAllRequest) (*model.UnreadSummaryView, error)
}

// SettingService 租户通知设置服务：事件目录投影、偏好 PATCH（revision 乐观锁）
// 与自定义提醒对象管理；写操作提交后 best-effort 记业务审计（联系方式脱敏）。
type SettingService interface {
	// GetAggregate 设置聚合：分类/事件目录 + 租户有效偏好 + 渠道能力 + revision
	GetAggregate(ctx context.Context, tenantID uint) (*model.SettingAggregateView, error)
	// PatchPreference 事件偏好更新：channels 部分更新、recipients 出现即全量
	// 替换；旧 revision 返回 409（NOTIFICATION_SETTINGS_CONFLICT）
	PatchPreference(
		ctx context.Context, tenantID uint, eventCode string, req model.PatchPreferenceRequest,
	) (*model.PatchPreferenceResponse, error)
	// ListRecipients 自定义提醒对象列表（完整联系方式仅设置管理员可达）
	ListRecipients(ctx context.Context, tenantID uint) ([]model.CustomRecipientView, error)
	// CreateRecipient 新增提醒对象（手机/邮箱至少一项；上限受服务端配置约束）
	CreateRecipient(
		ctx context.Context, tenantID uint, req model.CreateCustomRecipientRequest,
	) (*model.CustomRecipientView, error)
	// DeleteRecipient 删除未被偏好引用的提醒对象；在用时返回 409 + usedByEventCodes
	DeleteRecipient(ctx context.Context, tenantID, id uint, revision int64) error
	// SeedDefaults 租户开通事务预置设置聚合根（NotificationSettingSeeder 端口）
	SeedDefaults(ctx context.Context, tenantID uint) error
}

// EventPublisher 业务域窄发布端口：业务 Service 在自身事务内调用（经
// infrastructure.ResolveDB 加入调用方事务），业务提交后由 Worker 物化；
// 不允许业务主记录已提交却因进程退出而永久丢失消息事件。
type EventPublisher interface {
	PublishInTx(ctx context.Context, event EventInput) error
}

// EventInput 结构化事件输入（发布方视角）：EventID 为全局幂等键（建议
// 「域:实体:操作:随机数」形态），Parameters 受事件注册表 Schema 限制；
// 跳转动作不随事件传输，由目录按事件码构造（稳定动作码+参数键白名单）。
type EventInput struct {
	EventID           string
	EventCode         string
	ActorMemberID     uint   // 事件发起成员；0=系统
	AudienceMemberIDs []uint // 事件显式成员受众（Worker 再复核同租户）
	Parameters        map[string]string
	OccurredAt        time.Time // 零值取当前时间
}

// NewInboxService 收件箱服务工厂
func NewInboxService(
	tx TxManager, messages repository.MessageRepository, retentionMonths int,
) InboxService {
	return newInboxService(tx, messages, retentionMonths)
}

// NewSettingService 通知设置服务工厂
func NewSettingService(
	tx TxManager,
	settings repository.SettingRepository,
	audit auditservice.Recorder,
	recipientLimit int,
) SettingService {
	return newSettingService(tx, settings, audit, recipientLimit)
}

// NewEventPublisher 事件发布端口工厂（业务事务内写 Outbox）
func NewEventPublisher(outbox repository.OutboxRepository) EventPublisher {
	return &eventPublisher{outbox: outbox}
}

// NewDispatcher Outbox 消费 Dispatcher 工厂（扇出参数见 Default* 常量）
func NewDispatcher(
	tx TxManager,
	outbox repository.OutboxRepository,
	messages repository.MessageRepository,
	settings repository.SettingRepository,
	members MemberDirectory,
	admins AdminRecipientResolver,
	retentionMonths int,
	logger *logrus.Logger,
) *Dispatcher {
	return newDispatcher(tx, outbox, messages, settings, members, admins, retentionMonths, logger)
}
