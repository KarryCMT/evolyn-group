package adapter

import (
	"context"
	"errors"
	"testing"

	wfevent "evolyn/internal/engine/workflow/event"
	"evolyn/internal/engine/workflow/provider"
	notificationservice "evolyn/internal/platform/notification/service"

	"github.com/stretchr/testify/assert"
)

// fakeNotifier 通知域发布端口替身：记录调用并可注入失败。
type fakeNotifier struct {
	calls   []notificationservice.EventInput
	failErr error
}

func (f *fakeNotifier) PublishInTx(ctx context.Context, input notificationservice.EventInput) error {
	if f.failErr != nil {
		return f.failErr
	}
	f.calls = append(f.calls, input)
	return nil
}

// TestPublishUnregisteredEventsAreLogOnly 未在通知目录注册消费的冻结事件
// （task 终态、节点流转）不进 Outbox，只落日志流水且不报错。
func TestPublishUnregisteredEventsAreLogOnly(t *testing.T) {
	notifier := &fakeNotifier{}
	publisher := &EventPublisher{notifier: notifier}

	for _, name := range []string{
		wfevent.TaskApproved, wfevent.TaskRejected, wfevent.TaskCancelled,
		wfevent.NodeEntered, wfevent.NodeCompleted,
	} {
		err := publisher.PublishInTx(context.Background(), provider.Event{EventName: name})
		assert.NoError(t, err, name)
	}
	assert.Empty(t, notifier.calls)
}

// TestEventSuffix 幂等键后缀：去掉 workflow. 前缀的稳定短名。
func TestEventSuffix(t *testing.T) {
	assert.Equal(t, "task.created", eventSuffix(wfevent.TaskCreated))
	assert.Equal(t, "instance.returned", eventSuffix(wfevent.InstanceReturned))
	assert.Equal(t, "custom.code", eventSuffix("custom.code"))
}

// TestEmitBestEffort best-effort 边界：通知端口失败只吞掉（审批事务不回滚），
// 引擎侧对发布返回值本就不感知。
func TestEmitBestEffort(t *testing.T) {
	notifier := &fakeNotifier{failErr: errors.New("outbox unavailable")}
	publisher := &EventPublisher{notifier: notifier}

	err := publisher.emit(context.Background(),
		provider.Event{EventName: wfevent.TaskCreated},
		notificationservice.EventInput{})
	assert.NoError(t, err)
	assert.Empty(t, notifier.calls)
}

// TestDropBestEffort 元数据缺失丢弃：只记日志，不向引擎传播错误。
func TestDropBestEffort(t *testing.T) {
	publisher := &EventPublisher{notifier: &fakeNotifier{}}
	err := publisher.drop(context.Background(),
		provider.Event{EventName: wfevent.TaskCreated, TaskID: 1}, errors.New("task not found"))
	assert.NoError(t, err)
}
