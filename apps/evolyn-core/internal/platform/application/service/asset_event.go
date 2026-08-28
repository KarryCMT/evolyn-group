package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"evolyn/internal/platform/application/model"
	iammodel "evolyn/internal/platform/iam/model"
)

// 消息中心 P1：应用资产变更事件的稳定动词（模板 {verb} 参数值，与事件
// 注册表 application.asset.changed 的模板「{actorName}{verb}「{appName}」」配套）
const (
	assetVerbCreated = "创建了应用"
	assetVerbDeleted = "删除了应用"
)

// AssetEventNotifier 应用资产变更事件发布端口（消息中心 P1）：由装配层
// 适配 notification 域的事务 Outbox（窄端口，域间不直接耦合）；发布必须
// 发生在业务事务内——业务回滚则事件同步回滚，不允许「应用已提交但事件
// 永久丢失」。未注入时静默跳过（事件是可选增强，不阻断业务主流程）。
type AssetEventNotifier interface {
	NotifyAssetChanged(ctx context.Context, eventID, verb, appCode, appName string, actorMemberID uint) error
}

// AssetNotifierInjector 支持事后注入资产事件发布器的应用服务
type AssetNotifierInjector interface {
	UseAssetNotifier(notifier AssetEventNotifier)
}

// UseAssetNotifier 注入资产事件发布器（消息中心 P1）
func (s *applicationService) UseAssetNotifier(notifier AssetEventNotifier) {
	s.notifier = notifier
}

// publishAssetChangedInTx 在业务事务内发布应用资产变更事件；事件编码为
// 「application:asset:{应用ID}:{动词}:{随机}」——业务事务重试（回滚后重放）
// 生成新事件 ID，同一业务提交只物化一条消息
func (s *applicationService) publishAssetChangedInTx(
	ctx context.Context, verb string, app *model.Application, actor *iammodel.User,
) error {
	if s.notifier == nil || app == nil {
		return nil
	}
	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		return fmt.Errorf("generate asset event id: %w", err)
	}
	actorID := uint(0)
	if actor != nil {
		actorID = actor.ID
	}
	return s.notifier.NotifyAssetChanged(ctx,
		fmt.Sprintf("application:asset:%d:%s", app.ID, hex.EncodeToString(suffix)),
		verb, app.Code, app.Name, actorID)
}
