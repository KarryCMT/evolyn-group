// Package adapter Workflow 引擎平台能力适配器（ADR-012）：实现引擎内核
// provider/ 窄端口，桥接平台设施。域内不反向依赖内核以外的任何具体实现。
package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	wfevent "evolyn/internal/engine/workflow/event"
	enginemodel "evolyn/internal/engine/workflow/model"
	"evolyn/internal/engine/workflow/provider"
	"evolyn/internal/infrastructure"
	notificationservice "evolyn/internal/platform/notification/service"
	wfmodel "evolyn/internal/platform/workflow/model"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// EventPublisher 引擎事件发布适配器（Phase 6 Existing Outbox Integration，
// 第 18 章）：把引擎内核经窄端口发布的 workflow.* 事件桥接到既有 notification
// 域 EventPublisher.PublishInTx——同一业务事务内写 tn_notification_outbox_events，
// 由既有 Dispatcher 异步扇出；不新建 domain_outbox。
//
// 边界语义：
//   - 事件与审批状态同事务落库（18.3），但发布本身是 best-effort：适配器
//     内部失败（元数据缺失/目录校验拒绝/写 Outbox 失败）只记日志并吞掉，
//     绝不回滚审批事务——流程状态迁移才是权威事实；
//   - 仅通知目录注册的事件进 Outbox（task.created/transferred/reminder 与
//     实例终态/退回）；其余冻结事件（task.approved/rejected/cancelled、
//     node.entered/completed）V1 无消费方，只落日志流水；
//   - 展示元数据（流程名/节点名/参与人受众）在发布事务内按 wf_* 行解析，
//     不由引擎携带（引擎信封只含 ID 维度）；消息展示名随事件固化，
//     流程改名不回写历史消息。
//
// 租户隔离：wf_* 查询经 Model 路径由 GORM 租户 Callback 自动过滤，并以事件
// 携带的 TenantID 显式兜底；Outbox 行 tenant_id 同样由 Callback 固化。
type EventPublisher struct {
	notifier notificationservice.EventPublisher
	db       *gorm.DB
}

// NewEventPublisher 构造事件发布适配器（notifier 为 notification 域事务
// Outbox 发布端口；base 连接经 ResolveDB 加入调用方事务）。
func NewEventPublisher(notifier notificationservice.EventPublisher, base *gorm.DB) *EventPublisher {
	return &EventPublisher{notifier: notifier, db: base}
}

func (p *EventPublisher) PublishInTx(ctx context.Context, event provider.Event) error {
	switch event.EventName {
	case wfevent.TaskCreated, wfevent.TaskTransferred, wfevent.TaskReminder:
		return p.publishTaskEvent(ctx, event)
	case wfevent.InstanceCompleted, wfevent.InstanceRejected,
		wfevent.InstanceCancelled, wfevent.InstanceReturned:
		return p.publishInstanceEvent(ctx, event)
	default:
		// 未注册通知消费的冻结事件：Phase 2 日志流水语义保留（Phase 7 起
		// Webhook 等消费方按事件名扩展接入）
		logrus.WithFields(eventLogFields(event)).Debug("workflow domain event (no notification consumer)")
		return nil
	}
}

// publishTaskEvent 任务级通知事件：待办/转办/催办的接收人 = 任务参与人快照
// （event_audience），跳转动作 open_workflow_task 直达待办。
func (p *EventPublisher) publishTaskEvent(ctx context.Context, event provider.Event) error {
	task, err := p.findTask(ctx, event.TenantID, event.TaskID)
	if err != nil {
		return p.drop(ctx, event, fmt.Errorf("task %d not found: %w", event.TaskID, err))
	}
	instance, err := p.findInstance(ctx, event.TenantID, task.InstanceID)
	if err != nil {
		return p.drop(ctx, event, fmt.Errorf("instance %d not found: %w", task.InstanceID, err))
	}
	nodeName, err := p.nodeName(ctx, instance.DefinitionVersionID, task.NodeKey)
	if err != nil {
		return p.drop(ctx, event, err)
	}
	actors, err := p.taskActorIDs(ctx, task.ID)
	if err != nil {
		return p.drop(ctx, event, err)
	}
	flowName, err := p.flowName(ctx, instance.DefinitionID)
	if err != nil {
		return p.drop(ctx, event, err)
	}
	return p.emit(ctx, event, notificationservice.EventInput{
		EventID:           fmt.Sprintf("wf:task:%d:%s", task.ID, eventSuffix(event.EventName)),
		EventCode:         event.EventName,
		ActorMemberID:     event.ActorMemberID,
		AudienceMemberIDs: actors,
		Parameters: map[string]string{
			"flowName": flowName,
			"nodeName": nodeName,
			"taskId":   fmt.Sprintf("%d", task.ID),
		},
		OccurredAt: event.OccurredAt,
	})
}

// publishInstanceEvent 实例级通知事件：终态/退回的接收人 = 发起人
// （event_audience），跳转动作 open_workflow_instance 进流程详情；退回事件
// 幂等键携带重提交节点实例 ID（同一实例可能多次退回）。
func (p *EventPublisher) publishInstanceEvent(ctx context.Context, event provider.Event) error {
	instance, err := p.findInstance(ctx, event.TenantID, event.InstanceID)
	if err != nil {
		return p.drop(ctx, event, fmt.Errorf("instance %d not found: %w", event.InstanceID, err))
	}
	flowName, err := p.flowName(ctx, instance.DefinitionID)
	if err != nil {
		return p.drop(ctx, event, err)
	}
	eventID := fmt.Sprintf("wf:instance:%d:%s", instance.ID, eventSuffix(event.EventName))
	if event.EventName == wfevent.InstanceReturned {
		eventID = fmt.Sprintf("%s:%d", eventID, event.NodeInstanceID)
	}
	return p.emit(ctx, event, notificationservice.EventInput{
		EventID:           eventID,
		EventCode:         event.EventName,
		ActorMemberID:     event.ActorMemberID,
		AudienceMemberIDs: []uint{instance.StarterMemberID},
		Parameters: map[string]string{
			"flowName":   flowName,
			"instanceId": fmt.Sprintf("%d", instance.ID),
		},
		OccurredAt: event.OccurredAt,
	})
}

// emit 写入通知域事务 Outbox；失败只记日志吞掉（best-effort 边界，见类型注释）。
func (p *EventPublisher) emit(ctx context.Context, event provider.Event, input notificationservice.EventInput) error {
	if err := p.notifier.PublishInTx(ctx, input); err != nil {
		logrus.WithFields(eventLogFields(event)).WithError(err).
			Warn("workflow domain event outbox publish failed (best-effort, approval transaction keeps)")
		return nil
	}
	return nil
}

// drop 事件因元数据缺失被丢弃（best-effort）：只记日志，不阻断审批事务。
func (p *EventPublisher) drop(ctx context.Context, event provider.Event, cause error) error {
	logrus.WithFields(eventLogFields(event)).WithError(cause).
		Warn("workflow domain event dropped (metadata unavailable)")
	return nil
}

func eventLogFields(event provider.Event) logrus.Fields {
	return logrus.Fields{
		"eventName":  event.EventName,
		"tenantId":   event.TenantID,
		"instanceId": event.InstanceID,
		"taskId":     event.TaskID,
		"actor":      event.ActorMemberID,
	}
}

// eventSuffix 事件码 → 幂等键后缀（去掉 workflow. 前缀后的稳定短名）。
func eventSuffix(eventCode string) string {
	for _, name := range []string{
		wfevent.TaskCreated, wfevent.TaskTransferred, wfevent.TaskReminder,
		wfevent.InstanceCompleted, wfevent.InstanceRejected,
		wfevent.InstanceCancelled, wfevent.InstanceReturned,
	} {
		if name == eventCode {
			return name[len("workflow."):]
		}
	}
	return eventCode
}

// ---- 发布事务内的展示元数据解析（wf_* 行级只读查询，租户 Callback 过滤） ----

func (p *EventPublisher) findTask(ctx context.Context, tenantID, taskID uint) (*wfmodel.WfTask, error) {
	var row wfmodel.WfTask
	if err := infrastructure.ResolveDB(ctx, p.db).
		Where("id = ? AND tenant_id = ?", taskID, tenantID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (p *EventPublisher) findInstance(ctx context.Context, tenantID, instanceID uint) (*wfmodel.WfInstance, error) {
	var row wfmodel.WfInstance
	if err := infrastructure.ResolveDB(ctx, p.db).
		Where("id = ? AND tenant_id = ?", instanceID, tenantID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// flowName 流程展示名（设计态定义行，改名不影响运行态历史消息——名称随事件
// 固化进消息快照）。
func (p *EventPublisher) flowName(ctx context.Context, definitionID uint) (string, error) {
	var row wfmodel.WfDefinition
	if err := infrastructure.ResolveDB(ctx, p.db).Select("name").
		Where("id = ?", definitionID).First(&row).Error; err != nil {
		return "", err
	}
	return row.Name, nil
}

// nodeName 按发布快照解析节点展示名（任务 NodeKey → dsl_snapshot 节点名）。
func (p *EventPublisher) nodeName(ctx context.Context, definitionVersionID uint, nodeKey string) (string, error) {
	var row wfmodel.WfDefinitionVersion
	if err := infrastructure.ResolveDB(ctx, p.db).Select("dsl_snapshot").
		Where("id = ?", definitionVersionID).First(&row).Error; err != nil {
		return "", err
	}
	var doc enginemodel.Document
	if err := json.Unmarshal(row.DSLSnapshot, &doc); err != nil {
		return "", err
	}
	if node, ok := doc.NodeOf(nodeKey); ok && node.Name != "" {
		return node.Name, nil
	}
	return nodeKey, nil
}

// taskActorIDs 任务参与人快照成员 ID（通知受众；快照在任务创建事务内固化）。
func (p *EventPublisher) taskActorIDs(ctx context.Context, taskID uint) ([]uint, error) {
	var rows []wfmodel.WfTaskActor
	if err := infrastructure.ResolveDB(ctx, p.db).
		Where("task_id = ?", taskID).Find(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].MemberID)
	}
	return ids, nil
}

// FormDirectory 表单目录只读窄端口（装配层由 form 域仓储适配）：流程发起时
// 把 (formCode, formVersionNo) 解析为内部 form_id / form_version_id 落库；
// 引擎运行期按实例冻结的 FormVersionID 取数（Phase 3 接 BusinessDataProvider）。
type FormDirectory interface {
	ResolveFormVersion(ctx context.Context, formCode string, versionNo int) (formID, formVersionID uint, err error)
	// GetVersionContent 按表单版本行 ID 读取发布快照全文与版本元数据
	//（Phase 4 任务详情上下文：Form Schema Snapshot 出网投影）
	GetVersionContent(ctx context.Context, formVersionID uint) (content json.RawMessage, formCode string, versionNo int, err error)
}

// EnsureInterfaces 编译期端口契约自检。
var _ provider.EventPublisher = (*EventPublisher)(nil)
