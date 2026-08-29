// Package adapter Workflow 引擎平台能力适配器（ADR-012）：实现引擎内核
// provider/ 窄端口，桥接平台设施。域内不反向依赖内核以外的任何具体实现。
package adapter

import (
	"context"

	"evolyn/internal/engine/workflow/provider"

	"github.com/sirupsen/logrus"
)

// EventPublisher 引擎事件发布适配器（Phase 2）。
//
// Phase 2 以结构化日志落事件流水，不接入持久化 Outbox：workflow.* 事件
// 目录已在内核冻结，但通知域事件目录注册、模板渲染与消费侧属于 Phase 6
// 「Existing Outbox Integration」范围；届时本适配器替换为桥接既有
// notification 域 EventPublisher.PublishInTx（同一事务写 outbox，
// Dispatcher 异步扇出），审批事务永不因通知失败回滚（第 18.3 章）。
type EventPublisher struct{}

// NewEventPublisher 构造事件发布适配器。
func NewEventPublisher() *EventPublisher { return &EventPublisher{} }

func (p *EventPublisher) PublishInTx(ctx context.Context, event provider.Event) error {
	logrus.WithFields(logrus.Fields{
		"eventName":  event.EventName,
		"tenantId":   event.TenantID,
		"instanceId": event.InstanceID,
		"taskId":     event.TaskID,
		"actor":      event.ActorMemberID,
	}).Info("workflow domain event (phase2 log sink)")
	return nil
}

// FormDirectory 表单目录只读窄端口（装配层由 form 域仓储适配）：流程发起时
// 把 (formCode, formVersionNo) 解析为内部 form_id / form_version_id 落库；
// 引擎运行期按实例冻结的 FormVersionID 取数（Phase 3 接 BusinessDataProvider）。
type FormDirectory interface {
	ResolveFormVersion(ctx context.Context, formCode string, versionNo int) (formID, formVersionID uint, err error)
}

// EnsureInterfaces 编译期端口契约自检。
var _ provider.EventPublisher = (*EventPublisher)(nil)
