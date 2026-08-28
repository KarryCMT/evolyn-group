package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/notification/model"
	"evolyn/internal/platform/notification/repository"
)

// eventPublisher 业务域窄发布端口实现：在调用方事务内写 Outbox（经
// infrastructure.ResolveDB 加入传播事务），业务提交后由 Worker 物化。
type eventPublisher struct {
	outbox repository.OutboxRepository
}

// PublishInTx 发布结构化事件：事件码与参数先按注册表终审（不合法直接报错，
// 随业务事务回滚，不允许脏事件进入 Worker）；event_id 冲突吞并（幂等键重放
// 不产生第二条待处理事件）。跳转动作不随事件传输，由目录按事件码+参数构造。
func (p *eventPublisher) PublishInTx(ctx context.Context, event EventInput) error {
	if len(event.EventID) == 0 || len(event.EventID) > 128 {
		return httpx.Wrap(httpx.NewBiz(httpx.CodeValidation, "消息事件发布参数不合法", 400),
			fmt.Errorf("event id 长度必须在 1–128 之间"))
	}
	def, ok := LookupEvent(event.EventCode)
	if !ok {
		return httpx.Wrap(httpx.NewBiz(httpx.CodeValidation, "消息事件发布参数不合法", 400),
			fmt.Errorf("未知事件码: %s", event.EventCode))
	}
	if err := ValidateParams(def, event.Parameters); err != nil {
		return httpx.Wrap(httpx.NewBiz(httpx.CodeValidation, "消息事件发布参数不合法", 400), err)
	}

	occurredAt := event.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	parameters, err := json.Marshal(event.Parameters)
	if err != nil {
		return err
	}
	audience := []byte("[]")
	if len(event.AudienceMemberIDs) > 0 {
		if audience, err = json.Marshal(event.AudienceMemberIDs); err != nil {
			return err
		}
	}

	return p.outbox.Insert(ctx, &model.OutboxEvent{
		EventID:           event.EventID,
		EventCode:         event.EventCode,
		ActorMemberID:     event.ActorMemberID,
		AudienceMemberIDs: model.JSONContent(audience),
		Parameters:        model.JSONContent(parameters),
		OccurredAt:        occurredAt,
		Status:            model.OutboxPending,
		NextAttemptAt:     time.Now(),
	})
}
