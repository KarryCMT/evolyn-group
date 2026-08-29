package service

import (
	"context"
	"testing"

	apperrors "evolyn/internal/platform/notification"
	"evolyn/internal/platform/notification/model"

	auditservice "evolyn/internal/platform/audit/service"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ---- 测试替身（照 form 域手写 fake 风格） ----

// passThroughTx 事务桩：直接执行 fn（单测无真库事务）
type passThroughTx struct{}

func (passThroughTx) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// fakeSettingRepo 内存版设置聚合仓储（无并发语义，覆盖 Service 分支足够）
type fakeSettingRepo struct {
	settings    map[uint]*model.Setting              // tenantID → 聚合根
	preferences map[uint][]*model.Preference         // tenantID → 覆盖行
	rules       map[uint][]model.PreferenceRecipient // preferenceID → 接收规则
	recipients  map[uint][]*model.CustomRecipient    // tenantID → 联系人
	nextID      uint
}

func newFakeSettingRepo() *fakeSettingRepo {
	return &fakeSettingRepo{
		settings:    map[uint]*model.Setting{},
		preferences: map[uint][]*model.Preference{},
		rules:       map[uint][]model.PreferenceRecipient{},
		recipients:  map[uint][]*model.CustomRecipient{},
		nextID:      100,
	}
}

func (f *fakeSettingRepo) Migrate() error { return nil }

func (f *fakeSettingRepo) EnsureSetting(ctx context.Context, tenantID uint) (*model.Setting, error) {
	if s, ok := f.settings[tenantID]; ok {
		return s, nil
	}
	f.nextID++
	setting := &model.Setting{ID: f.nextID, Revision: 1}
	setting.TenantID = tenantID
	f.settings[tenantID] = setting
	return setting, nil
}

func (f *fakeSettingRepo) BumpRevision(ctx context.Context, settingID uint, fromRevision int64) (bool, error) {
	for _, s := range f.settings {
		if s.ID == settingID && s.Revision == fromRevision {
			s.Revision++
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeSettingRepo) ListPreferences(ctx context.Context, tenantID uint) ([]model.Preference, error) {
	rows := f.preferences[tenantID]
	out := make([]model.Preference, 0, len(rows))
	for _, row := range rows {
		out = append(out, *row)
	}
	return out, nil
}

func (f *fakeSettingRepo) UpsertPreference(ctx context.Context, pref *model.Preference) error {
	for _, row := range f.preferences[pref.TenantID] {
		if row.EventCode == pref.EventCode {
			row.SystemEnabled = pref.SystemEnabled
			row.EmailEnabled = pref.EmailEnabled
			row.SMSEnabled = pref.SMSEnabled
			row.RecipientsOverridden = pref.RecipientsOverridden
			pref.ID = row.ID
			return nil
		}
	}
	f.nextID++
	pref.ID = f.nextID
	f.preferences[pref.TenantID] = append(f.preferences[pref.TenantID], pref)
	return nil
}

func (f *fakeSettingRepo) ListPreferenceRecipients(
	ctx context.Context, tenantID uint, preferenceIDs []uint,
) (map[uint][]model.PreferenceRecipient, error) {
	out := map[uint][]model.PreferenceRecipient{}
	for _, id := range preferenceIDs {
		if rules, ok := f.rules[id]; ok {
			out[id] = rules
		}
	}
	return out, nil
}

func (f *fakeSettingRepo) ReplaceRecipients(
	ctx context.Context, tenantID, preferenceID uint, items []model.PreferenceRecipient,
) error {
	f.rules[preferenceID] = items
	return nil
}

func (f *fakeSettingRepo) ListCustomRecipients(ctx context.Context, tenantID uint) ([]model.CustomRecipient, error) {
	rows := make([]model.CustomRecipient, 0, len(f.recipients[tenantID]))
	for _, row := range f.recipients[tenantID] {
		rows = append(rows, *row)
	}
	return rows, nil
}

func (f *fakeSettingRepo) GetCustomRecipient(ctx context.Context, tenantID, id uint) (*model.CustomRecipient, error) {
	for _, row := range f.recipients[tenantID] {
		if row.ID == id {
			return row, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeSettingRepo) CountCustomRecipients(ctx context.Context, tenantID uint) (int64, error) {
	return int64(len(f.recipients[tenantID])), nil
}

func (f *fakeSettingRepo) InsertCustomRecipient(ctx context.Context, recipient *model.CustomRecipient) (*model.CustomRecipient, error) {
	f.nextID++
	recipient.ID = f.nextID
	f.recipients[recipient.TenantID] = append(f.recipients[recipient.TenantID], recipient)
	return recipient, nil
}

func (f *fakeSettingRepo) SoftDeleteCustomRecipient(ctx context.Context, tenantID, id uint) error {
	kept := f.recipients[tenantID][:0]
	for _, row := range f.recipients[tenantID] {
		if row.ID != id {
			kept = append(kept, row)
		}
	}
	f.recipients[tenantID] = kept
	return nil
}

func (f *fakeSettingRepo) FindRecipientUsage(ctx context.Context, tenantID, recipientID uint) ([]string, error) {
	var codes []string
	for _, prefs := range f.preferences[tenantID] {
		for _, rule := range f.rules[prefs.ID] {
			if rule.CustomRecipientID != nil && *rule.CustomRecipientID == recipientID {
				codes = append(codes, prefs.EventCode)
			}
		}
	}
	return codes, nil
}

// fakeRecorder 收集审计条目
type fakeRecorder struct{ entries []auditservice.Entry }

func (r *fakeRecorder) Record(ctx context.Context, e auditservice.Entry) {
	r.entries = append(r.entries, e)
}

// ---- 构造助手 ----

const testTenantID = uint(7)

func newTestSettingService(repo *fakeSettingRepo, limit int) *settingService {
	return newSettingService(passThroughTx{}, repo, &fakeRecorder{}, limit)
}

func boolPtr(v bool) *bool { return &v }

// seedRecipient 预置一个自定义联系人并返回视图形态
func seedRecipient(t *testing.T, repo *fakeSettingRepo, name string) uint {
	t.Helper()
	recipient := &model.CustomRecipient{Name: name, Mobile: "13800000000", Revision: 1}
	recipient.TenantID = testTenantID
	created, err := repo.InsertCustomRecipient(context.Background(), recipient)
	assert.NoError(t, err)
	return created.ID
}

// ---- GetAggregate 投影 ----

func TestSettingAggregateProjection(t *testing.T) {
	repo := newFakeSettingRepo()
	svc := newTestSettingService(repo, 200)

	aggregate, err := svc.GetAggregate(context.Background(), testTenantID)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, aggregate.Revision)
	assert.Len(t, aggregate.Categories, 9)
	// smsBudget 未接入计费事实源时为 null（前端隐藏数值摘要）
	assert.Nil(t, aggregate.SmsBudget)
	// 渠道能力：站内信可用，邮件/短信不可用且带可展示原因
	assert.True(t, aggregate.ChannelCapabilities[ChannelSystem].Available)
	assert.False(t, aggregate.ChannelCapabilities[ChannelEmail].Available)

	var appLog *model.SettingCategoryView
	for i := range aggregate.Categories {
		if aggregate.Categories[i].ID == CategoryAppLog {
			appLog = &aggregate.Categories[i]
		}
	}
	assert.NotNil(t, appLog)
	assert.True(t, appLog.Configurable)
	assert.Len(t, appLog.Events, 5)

	// 无覆盖行投影注册表默认：system 开、email/sms 关、默认接收对象
	first := appLog.Events[0]
	assert.Equal(t, "application.asset.changed", first.Code)
	assert.True(t, first.Channels[ChannelSystem])
	assert.False(t, first.Channels[ChannelEmail])
	assert.Equal(t, []model.RecipientView{
		{Kind: model.RecipientEventActor, Label: "创建者"},
		{Kind: model.RecipientTenantAdmin, Label: "系统管理员"},
	}, first.Recipients)

	// 尚无事件注册的分类：仍返回目录但 configurable=false 且 events 为空
	var dataReminder *model.SettingCategoryView
	for i := range aggregate.Categories {
		if aggregate.Categories[i].ID == CategoryDataReminder {
			dataReminder = &aggregate.Categories[i]
		}
	}
	assert.NotNil(t, dataReminder)
	assert.False(t, dataReminder.Configurable)
	assert.Empty(t, dataReminder.Events)
}

// ---- PatchPreference ----

func TestPatchPreferenceChannelAndRecipients(t *testing.T) {
	repo := newFakeSettingRepo()
	svc := newTestSettingService(repo, 200)
	ctx := context.Background()
	recipientID := seedRecipient(t, repo, "运维值班")

	// 部分更新渠道 + 全量替换接收规则（含自定义联系人）
	result, err := svc.PatchPreference(ctx, testTenantID, "application.asset.changed", model.PatchPreferenceRequest{
		Revision: 1,
		Channels: &model.ChannelPatch{Email: boolPtr(false)}, // 缺省键保持默认
		Recipients: &[]model.RecipientInput{
			{Kind: model.RecipientEventActor},
			{Kind: model.RecipientCustomRecipient, RecipientID: recipientID},
		},
	})
	assert.NoError(t, err)
	assert.EqualValues(t, 2, result.Revision)
	assert.Equal(t, "application.asset.changed", result.Event.Code)
	// 接收规则投影：创建者 + 自定义联系人姓名标签
	assert.Len(t, result.Event.Recipients, 2)
	assert.Equal(t, "运维值班", result.Event.Recipients[1].Label)

	// 聚合 revision 已递增：旧 revision 再写返回 409 冲突
	_, err = svc.PatchPreference(ctx, testTenantID, "application.asset.changed", model.PatchPreferenceRequest{
		Revision: 1, Channels: &model.ChannelPatch{Email: boolPtr(true)},
	})
	assert.ErrorIs(t, err, apperrors.ErrSettingsConflict)

	// 显式清空接收规则（recipients_overridden 区分「默认」与「显式为空」）
	result, err = svc.PatchPreference(ctx, testTenantID, "application.asset.changed", model.PatchPreferenceRequest{
		Revision: 2, Recipients: &[]model.RecipientInput{},
	})
	assert.NoError(t, err)
	assert.Empty(t, result.Event.Recipients)
}

func TestPatchPreferenceValidations(t *testing.T) {
	repo := newFakeSettingRepo()
	svc := newTestSettingService(repo, 200)
	ctx := context.Background()

	// 未知事件码
	_, err := svc.PatchPreference(ctx, testTenantID, "not.an.event", model.PatchPreferenceRequest{Revision: 1})
	assert.ErrorIs(t, err, apperrors.ErrEventUnknown)

	// 关闭必选渠道（站内信）
	_, err = svc.PatchPreference(ctx, testTenantID, "application.asset.changed", model.PatchPreferenceRequest{
		Revision: 1, Channels: &model.ChannelPatch{System: boolPtr(false)},
	})
	assert.ErrorIs(t, err, apperrors.ErrChannelRequired)

	// 开启能力未就绪渠道（P3 前邮件/短信恒不可用，不保存虚假开启状态）
	_, err = svc.PatchPreference(ctx, testTenantID, "application.asset.changed", model.PatchPreferenceRequest{
		Revision: 1, Channels: &model.ChannelPatch{Email: boolPtr(true)},
	})
	assert.ErrorIs(t, err, apperrors.ErrChannelUnavailable)
	_, err = svc.PatchPreference(ctx, testTenantID, "application.asset.changed", model.PatchPreferenceRequest{
		Revision: 1, Channels: &model.ChannelPatch{SMS: boolPtr(true)},
	})
	assert.ErrorIs(t, err, apperrors.ErrChannelUnavailable)

	// 接收规则：未知类型 / 重复动态规则 / 异租户联系人
	_, err = svc.PatchPreference(ctx, testTenantID, "application.asset.changed", model.PatchPreferenceRequest{
		Revision: 1, Recipients: &[]model.RecipientInput{{Kind: "someone_else"}},
	})
	assert.ErrorIs(t, err, apperrors.ErrRecipientInvalid)
	_, err = svc.PatchPreference(ctx, testTenantID, "application.asset.changed", model.PatchPreferenceRequest{
		Revision: 1,
		Recipients: &[]model.RecipientInput{
			{Kind: model.RecipientEventActor}, {Kind: model.RecipientEventActor},
		},
	})
	assert.ErrorIs(t, err, apperrors.ErrRecipientInvalid)
	_, err = svc.PatchPreference(ctx, testTenantID, "application.asset.changed", model.PatchPreferenceRequest{
		Revision: 1, Recipients: &[]model.RecipientInput{{Kind: model.RecipientCustomRecipient, RecipientID: 999}},
	})
	assert.ErrorIs(t, err, apperrors.ErrRecipientNotFound)
}

// ---- 自定义提醒对象 ----

func TestCustomRecipientLifecycle(t *testing.T) {
	repo := newFakeSettingRepo()
	svc := newTestSettingService(repo, 200)
	ctx := context.Background()

	// 手机/邮箱至少一项
	_, err := svc.CreateRecipient(ctx, testTenantID, model.CreateCustomRecipientRequest{
		Revision: 1, Name: "无联系方式",
	})
	assert.ErrorIs(t, err, apperrors.ErrRecipientInvalid)
	// 手机号格式
	_, err = svc.CreateRecipient(ctx, testTenantID, model.CreateCustomRecipientRequest{
		Revision: 1, Name: "坏手机", Mobile: "abc",
	})
	assert.ErrorIs(t, err, apperrors.ErrRecipientInvalid)
	// 邮箱大小写规范化
	created, err := svc.CreateRecipient(ctx, testTenantID, model.CreateCustomRecipientRequest{
		Revision: 1, Name: "运维值班", Mobile: "13800000000", Email: "OPS@lingyanyun.example",
	})
	assert.NoError(t, err)
	assert.Equal(t, "ops@lingyanyun.example", created.Email)

	// revision 冲突
	_, err = svc.CreateRecipient(ctx, testTenantID, model.CreateCustomRecipientRequest{
		Revision: 1, Name: "并发新增", Mobile: "13900000000",
	})
	assert.ErrorIs(t, err, apperrors.ErrSettingsConflict)

	// 上限：已有 1 个联系人时上限 1 即拒绝（上限由服务端配置约束，不信任前端）
	limited := newTestSettingService(repo, 1)
	_, err = limited.CreateRecipient(ctx, testTenantID, model.CreateCustomRecipientRequest{
		Revision: 2, Name: "达到上限", Email: "a@lingyanyun.example",
	})
	assert.ErrorIs(t, err, apperrors.ErrRecipientLimitExceeded)

	// 删除：先被偏好引用 → 409 + usedByEventCodes；解除引用后成功
	_, err = svc.PatchPreference(ctx, testTenantID, "application.asset.changed", model.PatchPreferenceRequest{
		Revision:   2,
		Recipients: &[]model.RecipientInput{{Kind: model.RecipientCustomRecipient, RecipientID: created.ID}},
	})
	assert.NoError(t, err)
	err = svc.DeleteRecipient(ctx, testTenantID, created.ID, 3)
	assert.ErrorIs(t, err, apperrors.ErrRecipientInUse)

	_, err = svc.PatchPreference(ctx, testTenantID, "application.asset.changed", model.PatchPreferenceRequest{
		Revision: 3, Recipients: &[]model.RecipientInput{{Kind: model.RecipientTenantAdmin}},
	})
	assert.NoError(t, err)
	assert.NoError(t, svc.DeleteRecipient(ctx, testTenantID, created.ID, 4))

	// 删除不存在的联系人（revision 动态读取，避免与前置步骤的递增耦合）
	current, err := repo.EnsureSetting(ctx, testTenantID)
	assert.NoError(t, err)
	err = svc.DeleteRecipient(ctx, testTenantID, 4242, current.Revision)
	assert.ErrorIs(t, err, apperrors.ErrRecipientNotFound)
}

func TestSeedDefaultsIdempotent(t *testing.T) {
	repo := newFakeSettingRepo()
	svc := newTestSettingService(repo, 200)
	ctx := context.Background()

	assert.NoError(t, svc.SeedDefaults(ctx, testTenantID))
	assert.NoError(t, svc.SeedDefaults(ctx, testTenantID))
	setting, err := repo.EnsureSetting(ctx, testTenantID)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, setting.Revision)
}

// ---- 脱敏 ----

func TestContactMasking(t *testing.T) {
	assert.Equal(t, "138****0000", MaskMobile("13800000000"))
	assert.Equal(t, "", MaskMobile(""))
	assert.Equal(t, "****", MaskMobile("123"))
	assert.Equal(t, "o***@lingyanyun.example", MaskEmail("ops@lingyanyun.example"))
	assert.Equal(t, "", MaskEmail(""))
}
