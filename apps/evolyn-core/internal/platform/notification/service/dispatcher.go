package service

import (
	"context"
	"encoding/json"
	"time"

	"evolyn/internal/contextx"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/notification/model"
	"evolyn/internal/platform/notification/repository"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Dispatcher 扇出参数默认值（可经配置覆盖）
const (
	DefaultOutboxBatchSize = 20
	DefaultMaxAttempts     = 5
	DefaultRetentionBatch  = 1000
	baseRetryBackoff       = 10 * time.Second
	// cleanupMaxRounds 保留清理单轮最大批次数：防异常数据量下的长循环
	cleanupMaxRounds = 200
)

// 稳定内部错误码（last_error_code，不存原始敏感错误）
const (
	outboxErrEventUnknown = "EVENT_UNKNOWN"
	outboxErrParams       = "PARAMS_INVALID"
	outboxErrDispatch     = "DISPATCH_ERROR"
)

// Dispatcher Outbox 消费器：领取 → 目录/参数终审 → 接收人解析 → 模板渲染 →
// 同事务写消息+收件箱 → 标记 done。单条失败按类别退避重试（有界指数），
// 进程崩溃时事务回滚，事件自然回到 pending（重放幂等）。
type Dispatcher struct {
	tx              TxManager
	outbox          repository.OutboxRepository
	messages        repository.MessageRepository
	settings        repository.SettingRepository
	members         MemberDirectory
	admins          AdminRecipientResolver
	retentionMonths int
	batchSize       int
	maxAttempts     int
	retentionBatch  int
	logger          *logrus.Logger
}

func newDispatcher(
	tx TxManager,
	outbox repository.OutboxRepository,
	messages repository.MessageRepository,
	settings repository.SettingRepository,
	members MemberDirectory,
	admins AdminRecipientResolver,
	retentionMonths int,
	logger *logrus.Logger,
) *Dispatcher {
	if retentionMonths <= 0 {
		retentionMonths = defaultRetentionMonths
	}
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &Dispatcher{
		tx:              tx,
		outbox:          outbox,
		messages:        messages,
		settings:        settings,
		members:         members,
		admins:          admins,
		retentionMonths: retentionMonths,
		batchSize:       DefaultOutboxBatchSize,
		maxAttempts:     DefaultMaxAttempts,
		retentionBatch:  DefaultRetentionBatch,
		logger:          logger,
	}
}

// SetBatchSize 覆盖单轮领取批量（装配层按配置注入；非法值回落默认）
func (d *Dispatcher) SetBatchSize(size int) {
	if size > 0 {
		d.batchSize = size
	}
}

// DispatchBatch 领取并物化一批事件：整批同事务，成功条目随批提交；单条处理
// 错误已按 retry/failed 落库，不阻断同批其他条目。返回本批处理条数。
func (d *Dispatcher) DispatchBatch(ctx context.Context) (int, error) {
	processed := 0
	err := d.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		events, err := d.outbox.ClaimBatch(tctx, d.batchSize)
		if err != nil {
			return err
		}
		for i := range events {
			processed++
			if dispatchErr := d.dispatchOne(tctx, &events[i]); dispatchErr != nil {
				// 单条失败已在 dispatchOne 内落库（retry/failed），仅告警；
				// 事务级故障（连接中断）会随整批回滚，事件回到 pending
				d.logger.WithError(dispatchErr).Warnf(
					"notification outbox dispatch event %s failed", events[i].EventID)
			}
		}
		return nil
	})
	return processed, err
}

// dispatchOne 物化单个事件；返回的 error 仅用于日志，状态迁移已落库
func (d *Dispatcher) dispatchOne(ctx context.Context, event *model.OutboxEvent) error {
	def, ok := LookupEvent(event.EventCode)
	if !ok {
		return d.outbox.MarkFailed(ctx, event.ID, outboxErrEventUnknown)
	}
	var params map[string]string
	if err := json.Unmarshal(event.Parameters, &params); err != nil {
		return d.outbox.MarkFailed(ctx, event.ID, outboxErrParams)
	}
	if err := ValidateParams(def, params); err != nil {
		return d.outbox.MarkFailed(ctx, event.ID, outboxErrParams)
	}

	// 接收成员解析（同租户复核）；解析层故障按暂时错误退避重试
	memberIDs, err := d.resolveMembers(ctx, event, def)
	if err != nil {
		return d.scheduleRetry(ctx, event, err)
	}

	// 展示快照固化：actorName 从成员目录解析，失败回退「系统」不阻断扇出
	actorName := "系统"
	if event.ActorMemberID != 0 && d.members != nil {
		if name := d.members.MemberDisplayName(ctx, event.TenantID, event.ActorMemberID); name != "" {
			actorName = name
		}
	}
	content := RenderContent(def, params, actorName)

	// 物化以租户上下文写入（Message/MemberInbox 带租户 Callback 与显式条件）
	tenantCtx := contextx.NewTenantContext(ctx, event.TenantID)
	action, err := json.Marshal(map[string]string{})
	if err != nil {
		return d.scheduleRetry(ctx, event, err)
	}
	if built := BuildAction(def, params); built != nil {
		if action, err = json.Marshal(built); err != nil {
			return d.scheduleRetry(ctx, event, err)
		}
	}
	message := &model.Message{
		TenantID:     event.TenantID,
		EventID:      event.EventID,
		CategoryCode: def.Category,
		EventCode:    def.Code,
		Severity:     def.Severity,
		Content:      content,
		Action:       model.JSONContent(action),
		SourceRef:    model.JSONContent("{}"),
		OccurredAt:   event.OccurredAt,
		ExpiresAt:    event.OccurredAt.AddDate(0, d.retentionMonths, 0),
	}
	messageID, err := d.messages.InsertIgnoreConflict(tenantCtx, message)
	if err != nil {
		return d.scheduleRetry(ctx, event, err)
	}
	if len(memberIDs) > 0 {
		if err = d.messages.InsertInboxesIgnoreConflict(
			tenantCtx, event.TenantID, messageID, def.Category, event.OccurredAt, memberIDs,
		); err != nil {
			return d.scheduleRetry(ctx, event, err)
		}
	}
	return d.outbox.MarkDone(ctx, event.ID)
}

// resolveMembers 按租户有效接收规则解析站内信成员（去重）：
//   - event_actor：事件发起成员，须仍属于当前租户且有效；
//   - event_audience：事件显式受众，批量复核同租户；
//   - tenant_admin：经窄端口按内置系统管理员组实时推导；
//   - custom_recipient：无成员身份，不产生站内收件箱（P3 外部渠道）。
func (d *Dispatcher) resolveMembers(
	ctx context.Context, event *model.OutboxEvent, def EventDef,
) ([]uint, error) {
	tenantCtx := contextx.NewTenantContext(ctx, event.TenantID)
	kinds, err := d.resolveRecipientKinds(tenantCtx, event.TenantID, def)
	if err != nil {
		return nil, err
	}

	seen := make(map[uint]bool)
	add := func(memberID uint) {
		if memberID != 0 {
			seen[memberID] = true
		}
	}
	for _, kind := range kinds {
		switch kind {
		case model.RecipientEventActor:
			if event.ActorMemberID == 0 {
				continue
			}
			if err := d.validateMemberQuietly(tenantCtx, event.TenantID, event.ActorMemberID); err != nil {
				return nil, err
			}
			add(event.ActorMemberID)
		case model.RecipientEventAudience:
			var audience []uint
			if err := json.Unmarshal(event.AudienceMemberIDs, &audience); err != nil {
				return nil, err
			}
			for _, memberID := range audience {
				if err := d.validateMemberQuietly(tenantCtx, event.TenantID, memberID); err != nil {
					return nil, err
				}
				add(memberID)
			}
		case model.RecipientTenantAdmin:
			if d.admins == nil {
				continue
			}
			adminIDs, err := d.admins.ResolveAdminMemberIDs(ctx, event.TenantID)
			if err != nil {
				return nil, err
			}
			for _, memberID := range adminIDs {
				add(memberID)
			}
		case model.RecipientCustomRecipient:
			// 外部联系人：仅进入邮件/短信投递任务（P3），站内信跳过
		}
	}
	memberIDs := make([]uint, 0, len(seen))
	for memberID := range seen {
		memberIDs = append(memberIDs, memberID)
	}
	return memberIDs, nil
}

// validateMemberQuietly 成员复核：NotFound（离职/跨租户/已删）静默跳过该成员
// （扇出范围收敛，不是错误）；目录服务故障返回错误交由退避重试
func (d *Dispatcher) validateMemberQuietly(ctx context.Context, tenantID, memberID uint) error {
	if d.members == nil {
		return nil
	}
	err := d.members.ValidateMember(ctx, tenantID, memberID)
	if err == nil {
		return nil
	}
	if biz, ok := err.(*httpx.BizError); ok && biz.HTTP == 400 {
		// 归属校验不通过（ErrMemberInvalid 同口径）：成员不在扇出范围
		return nil
	}
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	return err
}

// resolveRecipientKinds 事件在租户的有效接收规则（覆盖行 ?? 注册表默认）
func (d *Dispatcher) resolveRecipientKinds(ctx context.Context, tenantID uint, def EventDef) ([]string, error) {
	preferences, err := d.settings.ListPreferences(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	var current *model.Preference
	for i := range preferences {
		if preferences[i].EventCode == def.Code {
			current = &preferences[i]
			break
		}
	}
	if current == nil || !current.RecipientsOverridden {
		kinds := make([]string, 0, len(def.DefaultRecipients))
		for _, kind := range def.DefaultRecipients {
			if kind != model.RecipientCustomRecipient {
				kinds = append(kinds, kind)
			}
		}
		return kinds, nil
	}
	rules, err := d.settings.ListPreferenceRecipients(ctx, tenantID, []uint{current.ID})
	if err != nil {
		return nil, err
	}
	kinds := make([]string, 0, len(rules[current.ID]))
	for _, rule := range rules[current.ID] {
		if rule.TargetKind != model.RecipientCustomRecipient {
			kinds = append(kinds, rule.TargetKind)
		}
	}
	return kinds, nil
}

// scheduleRetry 暂时失败退避：有界指数（base * 2^attempt），超过上限进入
// failed（不再自动重试，等待人工或后续补偿任务）
func (d *Dispatcher) scheduleRetry(ctx context.Context, event *model.OutboxEvent, cause error) error {
	if event.AttemptCount+1 >= d.maxAttempts {
		return d.outbox.MarkFailed(ctx, event.ID, errorCodeOf(cause))
	}
	backoff := baseRetryBackoff << uint(event.AttemptCount)
	return d.outbox.MarkRetry(ctx, event.ID, errorCodeOf(cause), time.Now().Add(backoff))
}

// errorCodeOf 原因 → 稳定内部错误码（BizError 取其 errCode，其余统一 DISPATCH_ERROR）
func errorCodeOf(cause error) string {
	if biz, ok := cause.(*httpx.BizError); ok && biz.Code != "" {
		return biz.Code
	}
	return outboxErrDispatch
}

// CleanupExpiredOnce 保留清理单轮：先分批删过期收件箱行，再分批删无引用的
// 过期消息；每批行数受控避免长事务与表膨胀（保留期口径：occurred_at 早于
// now() - retentionMonths，列表/未读统计在 SQL 侧已排除过期，不依赖本任务）
func (d *Dispatcher) CleanupExpiredOnce(ctx context.Context) error {
	for round := 0; round < cleanupMaxRounds; round++ {
		deleted, err := d.messages.DeleteExpiredInboxes(ctx, d.retentionBatch)
		if err != nil {
			return err
		}
		if deleted < int64(d.retentionBatch) {
			break
		}
	}
	for round := 0; round < cleanupMaxRounds; round++ {
		deleted, err := d.messages.DeleteOrphanExpiredMessages(ctx, d.retentionBatch)
		if err != nil {
			return err
		}
		if deleted < int64(d.retentionBatch) {
			break
		}
	}
	return nil
}
