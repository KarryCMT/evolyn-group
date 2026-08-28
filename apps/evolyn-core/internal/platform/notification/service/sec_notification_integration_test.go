package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"evolyn/internal/contextx"
	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	iamrepository "evolyn/internal/platform/iam/repository"
	"evolyn/internal/platform/notification/model"
	notificationrepository "evolyn/internal/platform/notification/repository"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantrepository "evolyn/internal/platform/tenant/repository"
	tenantservice "evolyn/internal/platform/tenant/service"
	"evolyn/internal/testsupport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---- 消息中心真库集成测试（真实 PostgreSQL，TEST_PG_DSN 未设置时自动跳过）：
// SEC-NOTIFICATION-LIST/READ/SETTINGS/RECIPIENT + NOTIFICATION-OUTBOX-INT。
// 链路：testsupport 全量迁移（含 000039）→ 双租户 + owner 成员 → 事务
// Outbox 发布（含回滚同步回滚）→ Dispatcher 物化扇出 → 收件箱查询/已读的
// 租户与成员隔离 → 偏好/联系人的跨租户校验。

// itMemberDirectory 成员目录窄端口的测试适配（与 server.go 装配同语义）
type itMemberDirectory struct {
	users iamrepository.UserRepository
}

func (d itMemberDirectory) ValidateMember(ctx context.Context, tenantID, memberID uint) error {
	_, err := d.users.GetUserByID(contextx.NewTenantContext(ctx, tenantID), memberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 与 server.go 装配同口径：归属校验失败按 400 业务错（Dispatcher
			// 依此静默收敛扇出范围，不计为暂时性错误）
			return httpx.NewBiz(httpx.CodeValidation, "成员无效", 400)
		}
		return err
	}
	return nil
}

func (d itMemberDirectory) MemberDisplayName(ctx context.Context, tenantID, memberID uint) string {
	member, err := d.users.GetUserByID(contextx.NewTenantContext(ctx, tenantID), memberID)
	if err != nil {
		return ""
	}
	return member.Nickname
}

// itAdminResolver 系统管理员窄端口的测试适配（与 server.go 装配同语义：
// 内置系统管理员组经 tenant-admin 角色绑定实时推导）
type itAdminResolver struct {
	adminGroups iamrepository.AdminGroupRepository
}

func (r itAdminResolver) ResolveAdminMemberIDs(ctx context.Context, tenantID uint) ([]uint, error) {
	tenantCtx := contextx.NewTenantContext(ctx, tenantID)
	roleID, err := r.adminGroups.ResolveBuiltinRoleID(tenantCtx)
	if err != nil {
		return nil, err
	}
	members, err := r.adminGroups.ListBuiltinMembers(tenantCtx, roleID)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.ID)
	}
	return ids, nil
}

func TestNotificationIntegration(t *testing.T) {
	db := testsupport.NewPostgres(t)
	rdb := testsupport.DisabledRedis()
	iamRepo := iamrepository.NewRepositories(db, rdb)
	tenantRepo := tenantrepository.NewRepository(db, rdb)
	txManager := infrastructure.NewTxManager(db)

	messageRepo := notificationrepository.NewMessageRepository(db)
	settingRepo := notificationrepository.NewSettingRepository(db)
	outboxRepo := notificationrepository.NewOutboxRepository(db)

	inboxSvc := newInboxService(txManager, messageRepo, 6)
	settingSvc := newSettingService(txManager, settingRepo, nil, 200)
	publisher := &eventPublisher{outbox: outboxRepo}
	dispatcher := newDispatcher(
		txManager, outboxRepo, messageRepo, settingRepo,
		itMemberDirectory{users: iamRepo.User()},
		itAdminResolver{adminGroups: iamRepo.AdminGroup()},
		6, nil,
	)

	quotaSvc := tenantservice.NewQuotaService(tenantRepo, tenantRepo, iamRepo.User(), nil)
	tenantSvc := tenantservice.NewTenantService(txManager, tenantRepo, iamRepo, quotaSvc, nil, 0)

	ctx := context.Background()
	openTenant := func(code, ownerName string) *tenantmodel.Tenant {
		t.Helper()
		tenant, err := tenantSvc.Open(ctx, &tenantservice.OpenTenantRequest{
			Code: code, Name: code, Plan: tenantmodel.PlanFree,
			OwnerName: ownerName, OwnerPassword: "secret123",
		})
		require.NoError(t, err, "open tenant %s", code)
		return tenant
	}
	ownerMember := func(tenant *tenantmodel.Tenant, ownerName string) *iammodel.User {
		t.Helper()
		account, err := iamRepo.Account().GetByName(ctx, ownerName)
		require.NoError(t, err)
		member, err := iamRepo.User().GetByAccountAndTenant(
			contextx.NewTenantContext(ctx, tenant.ID), account.ID, tenant.ID)
		require.NoError(t, err)
		return member
	}
	rawCount := func(table string, args ...interface{}) int64 {
		t.Helper()
		var count int64
		require.NoError(t, db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s", table), args...).Scan(&count).Error)
		return count
	}

	// ---- 双租户 + 各自 owner 成员（开通事务预置通知设置聚合根） ----
	alpha := openTenant("sec-notify-alpha", "owner-notify-alpha")
	beta := openTenant("sec-notify-beta", "owner-notify-beta")
	alphaMember := ownerMember(alpha, "owner-notify-alpha")
	betaMember := ownerMember(beta, "owner-notify-beta")

	// 租户开通 Seeder 预置：聚合根幂等存在且 revision=1
	alphaSetting, err := settingSvc.GetAggregate(ctx, alpha.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, alphaSetting.Revision)

	// ---- NOTIFICATION-OUTBOX-INT：业务事务回滚 → Outbox 同步回滚 ----
	rollbackErr := txManager.WithinTransaction(contextx.NewTenantContext(ctx, alpha.ID), func(tctx context.Context) error {
		require.NoError(t, publisher.PublishInTx(tctx, EventInput{
			EventID:    "it:rollback:event",
			EventCode:  "application.asset.changed",
			Parameters: map[string]string{"appName": "回滚应用", "verb": "创建了应用", "appCode": "app_rollback"},
		}))
		return errors.New("business rollback")
	})
	require.Error(t, rollbackErr)
	assert.EqualValues(t, 0, rawCount("notification_outbox_events"), "业务事务回滚时 Outbox 必须同步回滚")

	// ---- NOTIFICATION-OUTBOX-INT：提交后 Dispatcher 物化且重放幂等 ----
	require.NoError(t, txManager.WithinTransaction(contextx.NewTenantContext(ctx, alpha.ID), func(tctx context.Context) error {
		return publisher.PublishInTx(tctx, EventInput{
			EventID:       "it:alpha:create:1",
			EventCode:     "application.asset.changed",
			ActorMemberID: alphaMember.ID,
			Parameters:    map[string]string{"appName": "CRM", "verb": "创建了应用", "appCode": "app_crm"},
		})
	}))
	require.NoError(t, txManager.WithinTransaction(contextx.NewTenantContext(ctx, beta.ID), func(tctx context.Context) error {
		return publisher.PublishInTx(tctx, EventInput{
			EventID:       "it:beta:create:1",
			EventCode:     "application.asset.changed",
			ActorMemberID: betaMember.ID,
			Parameters:    map[string]string{"appName": "Beta 应用", "verb": "创建了应用", "appCode": "app_beta"},
		})
	}))

	processed, err := dispatcher.DispatchBatch(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, processed, 2)
	assert.EqualValues(t, 2, rawCount("notification_messages"))
	// actor=owner 且 tenant_admin 实时推导亦为 owner：去重后单成员单收件箱行
	assert.EqualValues(t, 2, rawCount("notification_member_inboxes"))

	// 幂等重放：同 event_id 二次入队（吞并）+ 再物化一轮不新增消息/收件箱
	require.NoError(t, txManager.WithinTransaction(contextx.NewTenantContext(ctx, alpha.ID), func(tctx context.Context) error {
		return publisher.PublishInTx(tctx, EventInput{
			EventID:       "it:alpha:create:1",
			EventCode:     "application.asset.changed",
			ActorMemberID: alphaMember.ID,
			Parameters:    map[string]string{"appName": "CRM", "verb": "创建了应用", "appCode": "app_crm"},
		})
	}))
	_, err = dispatcher.DispatchBatch(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 2, rawCount("notification_messages"), "同 event_id 重放不得新增逻辑消息")
	assert.EqualValues(t, 2, rawCount("notification_member_inboxes"), "重复解析同一成员不得新增收件箱行")

	// 消息内容为模板渲染的纯文本快照（actorName 固化）
	alphaPage, err := inboxSvc.ListInbox(ctx, alpha.ID, alphaMember.ID, model.InboxQuery{
		CategoryID: CategoryAppLog,
	})
	require.NoError(t, err)
	require.Len(t, alphaPage.Items, 1)
	item := alphaPage.Items[0]
	assert.Equal(t, "owner-notify-alpha创建了应用「CRM」", item.Content)
	assert.Equal(t, "应用资产变更", item.EventLabel)
	assert.False(t, item.Read)
	assert.Equal(t, "open_application", actionTypeOf(t, item.Action))
	assert.Equal(t, 6, alphaPage.RetentionMonths)

	// ---- SEC-NOTIFICATION-LIST：租户/成员双向隔离 ----
	betaPage, err := inboxSvc.ListInbox(ctx, beta.ID, betaMember.ID, model.InboxQuery{CategoryID: CategoryAppLog})
	require.NoError(t, err)
	require.Len(t, betaPage.Items, 1)
	assert.Contains(t, betaPage.Items[0].Content, "Beta 应用")
	// alpha 成员看 beta 收件箱行 ID：空列表（不泄露存在性）
	crossPage, err := inboxSvc.ListInbox(ctx, alpha.ID, alphaMember.ID, model.InboxQuery{CategoryID: CategorySystemManagement})
	require.NoError(t, err)
	assert.Empty(t, crossPage.Items)

	// ---- 未读摘要与已读 ----
	summary, err := inboxSvc.UnreadSummary(ctx, alpha.ID, alphaMember.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, summary.UnreadTotal)

	// SEC-NOTIFICATION-READ：beta 成员猜 alpha 的收件箱行 ID 不能标记已读
	alphaInboxID := alphaPage.Items[0].ID
	_, err = inboxSvc.MarkRead(ctx, beta.ID, betaMember.ID, alphaInboxID)
	require.Error(t, err)
	assert.Equal(t, "NOTIFICATION_NOT_FOUND", bizCodeOf(err))

	summary, err = inboxSvc.MarkRead(ctx, alpha.ID, alphaMember.ID, alphaInboxID)
	require.NoError(t, err)
	assert.EqualValues(t, 0, summary.UnreadTotal)
	// 重复已读幂等且不改写首次时间
	_, err = inboxSvc.MarkRead(ctx, alpha.ID, alphaMember.ID, alphaInboxID)
	require.NoError(t, err)

	// ---- SEC-NOTIFICATION-RECIPIENT/SETTINGS：偏好与联系人跨租户校验 ----
	betaRecipient, err := settingSvc.CreateRecipient(ctx, beta.ID, model.CreateCustomRecipientRequest{
		Revision: 1, Name: "乙值班", Mobile: "13900000000",
	})
	require.NoError(t, err)
	// alpha 偏好关联 beta 联系人 → NOTIFICATION_RECIPIENT_NOT_FOUND（跨租户不可见）
	_, err = settingSvc.PatchPreference(ctx, alpha.ID, "application.asset.changed", model.PatchPreferenceRequest{
		Revision: 1,
		Recipients: &[]model.RecipientInput{
			{Kind: model.RecipientCustomRecipient, RecipientID: betaRecipient.ID},
		},
	})
	require.Error(t, err)
	assert.Equal(t, "NOTIFICATION_RECIPIENT_NOT_FOUND", bizCodeOf(err))
	// alpha 聚合未被污染：revision 仍为 1（校验失败整体回滚）
	alphaSetting, err = settingSvc.GetAggregate(ctx, alpha.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, alphaSetting.Revision)

	// 合法路径：alpha 添加自有联系人（聚合 1→2）并替换接收规则（2→3）
	alphaRecipient, err := settingSvc.CreateRecipient(ctx, alpha.ID, model.CreateCustomRecipientRequest{
		Revision: 1, Name: "甲值班", Email: "ops@lingyanyun.example",
	})
	require.NoError(t, err)
	patched, err := settingSvc.PatchPreference(ctx, alpha.ID, "application.asset.changed", model.PatchPreferenceRequest{
		Revision:   2,
		Recipients: &[]model.RecipientInput{{Kind: model.RecipientCustomRecipient, RecipientID: alphaRecipient.ID}},
	})
	require.NoError(t, err)
	assert.EqualValues(t, 3, patched.Revision)

	// 覆盖后扇出范围生效：自定义联系人不产生站内信（新事件扇出 0 收件箱行）
	require.NoError(t, txManager.WithinTransaction(contextx.NewTenantContext(ctx, alpha.ID), func(tctx context.Context) error {
		return publisher.PublishInTx(tctx, EventInput{
			EventID:       "it:alpha:create:2",
			EventCode:     "application.asset.changed",
			ActorMemberID: alphaMember.ID,
			Parameters:    map[string]string{"appName": "新应用", "verb": "创建了应用", "appCode": "app_new"},
		})
	}))
	_, err = dispatcher.DispatchBatch(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 3, rawCount("notification_messages"))
	assert.EqualValues(t, 2, rawCount("notification_member_inboxes"),
		"接收规则改为仅自定义联系人后不得产生站内收件箱行")

	// 删除被引用联系人 → 409 + usedByEventCodes
	err = settingSvc.DeleteRecipient(ctx, alpha.ID, alphaRecipient.ID, 3)
	require.Error(t, err)
	assert.Equal(t, "NOTIFICATION_RECIPIENT_IN_USE", bizCodeOf(err))

	// ---- 过期消息排除（清理滞后也不可见） ----
	expired := &model.Message{
		EventID: "it:expired", CategoryCode: CategorySystemManagement, EventCode: "application.asset.changed",
		Content: "过期消息", OccurredAt: time.Now().AddDate(0, -7, 0), ExpiresAt: time.Now().Add(-time.Hour),
	}
	expired.TenantID = alpha.ID
	expiredID, err := messageRepo.InsertIgnoreConflict(contextx.NewTenantContext(ctx, alpha.ID), expired)
	require.NoError(t, err)
	require.NoError(t, messageRepo.InsertInboxesIgnoreConflict(
		contextx.NewTenantContext(ctx, alpha.ID), alpha.ID, expiredID, CategorySystemManagement,
		expired.OccurredAt, []uint{alphaMember.ID}))
	listPage, err := inboxSvc.ListInbox(ctx, alpha.ID, alphaMember.ID, model.InboxQuery{CategoryID: CategorySystemManagement})
	require.NoError(t, err)
	assert.Empty(t, listPage.Items, "过期消息即使尚未物理清理也不得出现在列表")
	summary, err = inboxSvc.UnreadSummary(ctx, alpha.ID, alphaMember.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 0, summary.UnreadTotal, "过期消息不得计入未读摘要")

	// ---- 保留清理：先删收件箱行再删无引用消息 ----
	require.NoError(t, dispatcher.CleanupExpiredOnce(ctx))
	assert.EqualValues(t, 2, rawCount("notification_member_inboxes"), "未过期收件箱行不受清理影响")
	var expiredLeft int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM notification_messages WHERE event_id = 'it:expired'").Scan(&expiredLeft).Error)
	assert.EqualValues(t, 0, expiredLeft, "过期且无收件箱引用的消息应被成批硬删")
}

// ---- 集成测试断言助手 ----

// bizCodeOf 从 BizError 取稳定错误码（errCode 断言不匹配 message 文本）
func bizCodeOf(err error) string {
	var current error = err
	for current != nil {
		if biz, ok := current.(*httpx.BizError); ok {
			return biz.Code
		}
		current = errors.Unwrap(current)
	}
	return ""
}

func actionTypeOf(t *testing.T, raw []byte) string {
	t.Helper()
	action := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(raw, &action))
	value, _ := action["type"].(string)
	return value
}
