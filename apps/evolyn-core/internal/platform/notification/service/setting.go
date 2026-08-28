package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	apperrors "evolyn/internal/platform/notification"
	"evolyn/internal/platform/notification/model"
	"evolyn/internal/platform/notification/repository"

	auditservice "evolyn/internal/platform/audit/service"

	"gorm.io/gorm"
)

// 自定义联系人约束（上限经服务端配置注入，默认 200；不信任前端）
const defaultCustomRecipientLimit = 200

var (
	mobilePattern = regexp.MustCompile(`^\+?[0-9]{5,20}$`)
	emailPattern  = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

type settingService struct {
	tx             TxManager
	settings       repository.SettingRepository
	audit          auditservice.Recorder
	recipientLimit int
}

func newSettingService(
	tx TxManager, settings repository.SettingRepository, audit auditservice.Recorder, recipientLimit int,
) *settingService {
	if recipientLimit <= 0 {
		recipientLimit = defaultCustomRecipientLimit
	}
	return &settingService{tx: tx, settings: settings, audit: audit, recipientLimit: recipientLimit}
}

// GetAggregate 设置聚合读取：聚合根幂等补齐（兼容存量租户）后按目录 + 覆盖行
// 投影有效偏好；无覆盖行的事件投影注册表默认值，前端无需感知覆盖语义。
func (s *settingService) GetAggregate(ctx context.Context, tenantID uint) (*model.SettingAggregateView, error) {
	setting, err := s.settings.EnsureSetting(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	view, _, err := s.buildAggregateView(ctx, tenantID, setting.Revision)
	if err != nil {
		return nil, err
	}
	return view, nil
}

// buildAggregateView 组装设置聚合视图：返回视图与偏好覆盖索引（PATCH 复用）
func (s *settingService) buildAggregateView(
	ctx context.Context, tenantID uint, revision int64,
) (*model.SettingAggregateView, map[string]*model.Preference, error) {
	preferences, err := s.settings.ListPreferences(ctx, tenantID)
	if err != nil {
		return nil, nil, err
	}
	prefIndex := make(map[string]*model.Preference, len(preferences))
	ids := make([]uint, 0, len(preferences))
	for i := range preferences {
		prefIndex[preferences[i].EventCode] = &preferences[i]
		ids = append(ids, preferences[i].ID)
	}
	recipients, err := s.settings.ListPreferenceRecipients(ctx, tenantID, ids)
	if err != nil {
		return nil, nil, err
	}
	// 自定义联系人索引：接收规则 custom 项的展示标签
	customIndex := make(map[uint]model.CustomRecipient)
	if customs, err := s.settings.ListCustomRecipients(ctx, tenantID); err == nil {
		for _, custom := range customs {
			customIndex[custom.ID] = custom
		}
	} else {
		return nil, nil, err
	}

	categories := make([]model.SettingCategoryView, 0, len(Categories()))
	for _, category := range Categories() {
		categoryView := model.SettingCategoryView{
			ID:           category.Code,
			Label:        category.Label,
			Group:        category.Group,
			Configurable: category.Configurable,
			Events:       []model.SettingEventView{},
		}
		for _, def := range EventsOfCategory(category.Code) {
			categoryView.Events = append(categoryView.Events, projectEventView(def, prefIndex[def.Code], recipients, customIndex))
		}
		categories = append(categories, categoryView)
	}

	return &model.SettingAggregateView{
		Revision:            revision,
		Categories:          categories,
		ChannelCapabilities: channelCapabilities(),
		SmsBudget:           nil, // P3 计费事实源接入前不下发数值，前端隐藏摘要
	}, prefIndex, nil
}

// projectEventView 事件有效偏好投影：渠道与接收规则均按「覆盖行 ?? 注册表默认」
func projectEventView(
	def EventDef,
	pref *model.Preference,
	recipients map[uint][]model.PreferenceRecipient,
	customs map[uint]model.CustomRecipient,
) model.SettingEventView {
	channels := make(map[string]bool, len(def.DefaultChannels))
	for channel, enabled := range def.DefaultChannels {
		channels[channel] = enabled
	}
	recipientViews := make([]model.RecipientView, 0, len(def.DefaultRecipients))
	if pref != nil {
		channels[ChannelSystem] = pref.SystemEnabled
		channels[ChannelEmail] = pref.EmailEnabled
		channels[ChannelSMS] = pref.SMSEnabled
	}
	if pref != nil && pref.RecipientsOverridden {
		for _, rule := range recipients[pref.ID] {
			view := model.RecipientView{Kind: rule.TargetKind, Label: RecipientLabel(rule.TargetKind)}
			if rule.TargetKind == model.RecipientCustomRecipient && rule.CustomRecipientID != nil {
				view.RecipientID = *rule.CustomRecipientID
				if custom, ok := customs[*rule.CustomRecipientID]; ok {
					view.Label = custom.Name
				}
			}
			recipientViews = append(recipientViews, view)
		}
	} else {
		for _, kind := range def.DefaultRecipients {
			recipientViews = append(recipientViews, model.RecipientView{Kind: kind, Label: RecipientLabel(kind)})
		}
	}
	return model.SettingEventView{
		Code:              def.Code,
		Label:             def.Label,
		Severity:          def.Severity,
		SupportedChannels: def.SupportedChannels,
		LockedChannels:    def.LockedChannels,
		Channels:          channels,
		Recipients:        recipientViews,
	}
}

// PatchPreference 事件偏好更新（文档 9.3）：事务内先比对 revision，再校验
// 事件/渠道/接收规则，upsert 覆盖行 + 全量替换接收规则，最后以旧 revision
// 为条件原子递增（零行影响=并发修改，409 拒绝伪成功）。
func (s *settingService) PatchPreference(
	ctx context.Context, tenantID uint, eventCode string, req model.PatchPreferenceRequest,
) (*model.PatchPreferenceResponse, error) {
	def, ok := LookupEvent(eventCode)
	if !ok {
		return nil, apperrors.ErrEventUnknown
	}
	if req.Channels == nil && req.Recipients == nil {
		// 空更新幂等：返回当前有效偏好，不消耗 revision
		view, _, err := s.buildAggregateView(ctx, tenantID, req.Revision)
		if err != nil {
			return nil, err
		}
		for _, category := range view.Categories {
			for _, event := range category.Events {
				if event.Code == eventCode {
					return &model.PatchPreferenceResponse{Revision: view.Revision, Event: event}, nil
				}
			}
		}
		return nil, apperrors.ErrEventUnknown
	}

	var result *model.PatchPreferenceResponse
	err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		setting, err := s.settings.EnsureSetting(tctx, tenantID)
		if err != nil {
			return err
		}
		if setting.Revision != req.Revision {
			return apperrors.ErrSettingsConflict
		}

		preferences, err := s.settings.ListPreferences(tctx, tenantID)
		if err != nil {
			return err
		}
		var current *model.Preference
		for i := range preferences {
			if preferences[i].EventCode == eventCode {
				current = &preferences[i]
				break
			}
		}

		// 目标渠道 = 覆盖行 ?? 默认，再叠加请求的部分更新（默认 map 必须
		// 深拷贝——注册表是全局只读事实源，禁止就地改写）
		target := make(map[string]bool, len(def.DefaultChannels))
		for channel, enabled := range def.DefaultChannels {
			target[channel] = enabled
		}
		if current != nil {
			target = map[string]bool{
				ChannelSystem: current.SystemEnabled,
				ChannelEmail:  current.EmailEnabled,
				ChannelSMS:    current.SMSEnabled,
			}
		}
		if req.Channels != nil {
			if req.Channels.System != nil {
				target[ChannelSystem] = *req.Channels.System
			}
			if req.Channels.Email != nil {
				target[ChannelEmail] = *req.Channels.Email
			}
			if req.Channels.SMS != nil {
				target[ChannelSMS] = *req.Channels.SMS
			}
		}
		if err := validateTargetChannels(def, target); err != nil {
			return err
		}

		// 接收规则：缺省不修改；出现即全量替换
		recipientsOverridden := current != nil && current.RecipientsOverridden
		var newRules []model.PreferenceRecipient
		if req.Recipients != nil {
			newRules, err = s.normalizeRecipientInputs(tctx, tenantID, *req.Recipients)
			if err != nil {
				return err
			}
			recipientsOverridden = true
		}

		pref := &model.Preference{
			SettingID:            setting.ID,
			EventCode:            eventCode,
			SystemEnabled:        target[ChannelSystem],
			EmailEnabled:         target[ChannelEmail],
			SMSEnabled:           target[ChannelSMS],
			RecipientsOverridden: recipientsOverridden,
			TenantID:             tenantID,
		}
		if current != nil {
			pref.ID = current.ID
			pref.CreatedAt = current.CreatedAt
		}
		if err := s.settings.UpsertPreference(tctx, pref); err != nil {
			return err
		}
		if req.Recipients != nil {
			if err := s.settings.ReplaceRecipients(tctx, tenantID, pref.ID, newRules); err != nil {
				return err
			}
		}

		// 聚合乐观锁收口：并发修改时零行影响，整体回滚（伪成功禁止）
		ok, err := s.settings.BumpRevision(tctx, setting.ID, req.Revision)
		if err != nil {
			return err
		}
		if !ok {
			return apperrors.ErrSettingsConflict
		}

		// 响应投影：新偏好 + 接收规则（custom 标签需联系人名）
		customIndex := make(map[uint]model.CustomRecipient)
		if customs, err := s.settings.ListCustomRecipients(tctx, tenantID); err == nil {
			for _, custom := range customs {
				customIndex[custom.ID] = custom
			}
		} else {
			return err
		}
		ruleMap := map[uint][]model.PreferenceRecipient{}
		if req.Recipients != nil {
			ruleMap[pref.ID] = newRules
		} else if current != nil && current.RecipientsOverridden {
			loaded, err := s.settings.ListPreferenceRecipients(tctx, tenantID, []uint{pref.ID})
			if err != nil {
				return err
			}
			ruleMap = loaded
		}
		event := projectEventView(def, pref, ruleMap, customIndex)
		result = &model.PatchPreferenceResponse{Revision: req.Revision + 1, Event: event}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 审计在事务提交后 best-effort 记录：摘要只含事件码与渠道开关
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module:       "notification",
			Action:       "update",
			ResourceType: "notification_setting",
			ResourceID:   eventCode,
			Summary:      "更新通知偏好「" + def.Label + "」",
			After:        map[string]interface{}{"eventCode": eventCode},
		})
	}
	return result, nil
}

// validateTargetChannels 渠道目标值校验：必选渠道不可关、不支持渠道不可开、
// 能力未就绪（P3 前 email/sms）不允许新开启——不保存「已开启但永远无法发送」
// 的虚假状态
func validateTargetChannels(def EventDef, target map[string]bool) error {
	capabilities := channelCapabilities()
	for _, channel := range []string{ChannelSystem, ChannelEmail, ChannelSMS} {
		enabled := target[channel]
		if enabled && !channelSupported(def, channel) {
			return apperrors.ErrChannelNotSupported
		}
		if !enabled && channelLocked(def, channel) {
			return apperrors.ErrChannelRequired
		}
		if enabled && !capabilities[channel].Available {
			return apperrors.ErrChannelUnavailable
		}
	}
	return nil
}

// normalizeRecipientInputs 接收规则输入校验与持久化形态转换：动态规则同一
// 偏好内至多一条；自定义联系人不能重复关联且必须同租户、未删除
func (s *settingService) normalizeRecipientInputs(
	ctx context.Context, tenantID uint, inputs []model.RecipientInput,
) ([]model.PreferenceRecipient, error) {
	rules := make([]model.PreferenceRecipient, 0, len(inputs))
	seenDynamic := make(map[string]bool, len(inputs))
	seenCustom := make(map[uint]bool, len(inputs))
	for _, input := range inputs {
		switch input.Kind {
		case model.RecipientEventActor, model.RecipientEventAudience, model.RecipientTenantAdmin:
			if seenDynamic[input.Kind] {
				return nil, apperrors.ErrRecipientInvalid
			}
			seenDynamic[input.Kind] = true
			rules = append(rules, model.PreferenceRecipient{TargetKind: input.Kind})
		case model.RecipientCustomRecipient:
			if input.RecipientID == 0 {
				return nil, apperrors.ErrRecipientInvalid
			}
			if seenCustom[input.RecipientID] {
				return nil, apperrors.ErrRecipientInvalid
			}
			seenCustom[input.RecipientID] = true
			recipientID := input.RecipientID
			if _, err := s.settings.GetCustomRecipient(ctx, tenantID, recipientID); err != nil {
				if err == gorm.ErrRecordNotFound {
					return nil, apperrors.ErrRecipientNotFound
				}
				return nil, err
			}
			rules = append(rules, model.PreferenceRecipient{
				TargetKind:        model.RecipientCustomRecipient,
				CustomRecipientID: &recipientID,
			})
		default:
			return nil, apperrors.ErrRecipientInvalid
		}
	}
	return rules, nil
}

// ListRecipients 自定义提醒对象列表（完整联系方式仅设置管理员可达）
func (s *settingService) ListRecipients(ctx context.Context, tenantID uint) ([]model.CustomRecipientView, error) {
	recipients, err := s.settings.ListCustomRecipients(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	views := make([]model.CustomRecipientView, 0, len(recipients))
	for _, recipient := range recipients {
		views = append(views, projectCustomRecipient(recipient))
	}
	return views, nil
}

// CreateRecipient 新增提醒对象：聚合 revision 乐观锁 + 手机/邮箱至少一项 +
// 租户上限三重校验；事务提交后审计摘要仅记录脱敏联系方式
func (s *settingService) CreateRecipient(
	ctx context.Context, tenantID uint, req model.CreateCustomRecipientRequest,
) (*model.CustomRecipientView, error) {
	name := strings.TrimSpace(req.Name)
	mobile := strings.TrimSpace(req.Mobile)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if len([]rune(name)) < 1 || len([]rune(name)) > 80 {
		return nil, apperrors.ErrRecipientInvalid
	}
	if mobile == "" && email == "" {
		return nil, apperrors.ErrRecipientInvalid
	}
	if mobile != "" && !mobilePattern.MatchString(mobile) {
		return nil, apperrors.ErrRecipientInvalid
	}
	if email != "" && (len(email) > 254 || !emailPattern.MatchString(email)) {
		return nil, apperrors.ErrRecipientInvalid
	}

	var created *model.CustomRecipient
	err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		setting, err := s.settings.EnsureSetting(tctx, tenantID)
		if err != nil {
			return err
		}
		if setting.Revision != req.Revision {
			return apperrors.ErrSettingsConflict
		}
		count, err := s.settings.CountCustomRecipients(tctx, tenantID)
		if err != nil {
			return err
		}
		if count >= int64(s.recipientLimit) {
			return apperrors.ErrRecipientLimitExceeded
		}
		// 嵌入字段显式赋值（与租户域 seed 同口径，不依赖 Callback 兜底）
		recipient := &model.CustomRecipient{Name: name, Mobile: mobile, Email: email, Revision: 1}
		recipient.TenantID = tenantID
		recipient, err = s.settings.InsertCustomRecipient(tctx, recipient)
		if err != nil {
			return err
		}
		created = recipient
		ok, err := s.settings.BumpRevision(tctx, setting.ID, req.Revision)
		if err != nil {
			return err
		}
		if !ok {
			return apperrors.ErrSettingsConflict
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	view := projectCustomRecipient(*created)
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module:       "notification",
			Action:       "create",
			ResourceType: "custom_recipient",
			ResourceID:   fmt.Sprintf("%d", created.ID),
			TargetName:   created.Name,
			Summary:      "添加提醒对象「" + created.Name + "」（" + maskContact(mobile, email) + "）",
			After: map[string]interface{}{
				"name":   created.Name,
				"mobile": MaskMobile(created.Mobile),
				"email":  MaskEmail(created.Email),
			},
		})
	}
	return &view, nil
}

// DeleteRecipient 删除提醒对象：仍被偏好引用时 409 并携带脱敏
// usedByEventCodes（不默认级联改变投递范围）；聚合 revision 乐观锁同上
func (s *settingService) DeleteRecipient(ctx context.Context, tenantID, id uint, revision int64) error {
	var name string
	err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		setting, err := s.settings.EnsureSetting(tctx, tenantID)
		if err != nil {
			return err
		}
		if setting.Revision != revision {
			return apperrors.ErrSettingsConflict
		}
		recipient, err := s.settings.GetCustomRecipient(tctx, tenantID, id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperrors.ErrRecipientNotFound
			}
			return err
		}
		usage, err := s.settings.FindRecipientUsage(tctx, tenantID, id)
		if err != nil {
			return err
		}
		if len(usage) > 0 {
			return apperrors.ErrRecipientInUse.WithData(map[string]interface{}{"usedByEventCodes": usage})
		}
		if err := s.settings.SoftDeleteCustomRecipient(tctx, tenantID, id); err != nil {
			return err
		}
		ok, err := s.settings.BumpRevision(tctx, setting.ID, revision)
		if err != nil {
			return err
		}
		if !ok {
			return apperrors.ErrSettingsConflict
		}
		name = recipient.Name
		return nil
	})
	if err != nil {
		return err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module:       "notification",
			Action:       "delete",
			ResourceType: "custom_recipient",
			ResourceID:   fmt.Sprintf("%d", id),
			TargetName:   name,
			Summary:      "删除提醒对象「" + name + "」",
		})
	}
	return nil
}

// SeedDefaults 租户开通事务预置设置聚合根（幂等；读取侧 EnsureSetting 兜底）
func (s *settingService) SeedDefaults(ctx context.Context, tenantID uint) error {
	_, err := s.settings.EnsureSetting(ctx, tenantID)
	return err
}

// resolveRecipientKinds 事件在租户的有效接收规则（Dispatcher 扇出使用）：
// 自定义联系人无成员身份，不进入站内信扇出（仅 P3 外部渠道）
func (s *settingService) resolveRecipientKinds(
	ctx context.Context, tenantID uint, def EventDef,
) ([]string, error) {
	preferences, err := s.settings.ListPreferences(ctx, tenantID)
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
	rules, err := s.settings.ListPreferenceRecipients(ctx, tenantID, []uint{current.ID})
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

func projectCustomRecipient(recipient model.CustomRecipient) model.CustomRecipientView {
	return model.CustomRecipientView{
		ID:       recipient.ID,
		Name:     recipient.Name,
		Mobile:   recipient.Mobile,
		Email:    recipient.Email,
		Revision: recipient.Revision,
	}
}

// MaskMobile 手机号脱敏（日志/审计/错误信息只保留脱敏快照）
func MaskMobile(mobile string) string {
	if len(mobile) < 7 {
		if mobile == "" {
			return ""
		}
		return "****"
	}
	return mobile[:3] + "****" + mobile[len(mobile)-4:]
}

// MaskEmail 邮箱脱敏：保留首字符与域名
func MaskEmail(email string) string {
	at := strings.Index(email, "@")
	if at <= 0 {
		if email == "" {
			return ""
		}
		return "****"
	}
	return email[:1] + "***" + email[at:]
}

// maskContact 组合脱敏（审计摘要用）
func maskContact(mobile, email string) string {
	parts := make([]string, 0, 2)
	if masked := MaskMobile(mobile); masked != "" {
		parts = append(parts, masked)
	}
	if masked := MaskEmail(email); masked != "" {
		parts = append(parts, masked)
	}
	if len(parts) == 0 {
		return "未填写联系方式"
	}
	return strings.Join(parts, " / ")
}
