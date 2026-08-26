package service

import (
	"context"
	"errors"
	"strconv"

	"evolyn/internal/contextx"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"
)

// memberFieldService 成员字段配置（成员信息管理一期）：维护租户级显示策略。
// 字段元数据与锁定规则唯一来源是 model 包的字段注册表，本服务负责：
// 补齐租户默认配置（幂等）、组装配置快照、以及单字段即时更新（乐观锁）
type memberFieldService struct {
	tx     TxManager
	fields repository.MemberFieldSettingRepository
	audit  auditservice.Recorder
}

func NewMemberFieldService(
	tx TxManager,
	fields repository.MemberFieldSettingRepository,
	audit auditservice.Recorder,
) MemberFieldService {
	return &memberFieldService{tx: tx, fields: fields, audit: audit}
}

// GetSnapshot 读取当前租户的完整字段配置快照。首次读取（或注册表新增字段后）
// 幂等补齐缺失的默认行，保证旧租户也能得到完整配置（文档 4.1）
func (s *memberFieldService) GetSnapshot(ctx context.Context) (*model.MemberFieldConfigSnapshot, error) {
	settings, err := s.ensureDefaults(ctx)
	if err != nil {
		return nil, err
	}
	return buildSnapshot(settings), nil
}

// UpdateField 单字段即时更新（文档 5.1）：请求只携带本次变更的开关与页面
// revision，成功后返回整页最新快照供前端覆盖本地状态。
// 校验顺序：字段存在 → 锁定规则 → 可见/可编辑联动 → 乐观锁
func (s *memberFieldService) UpdateField(ctx context.Context, fieldKey string, req *MemberFieldSettingUpdateRequest) (*model.MemberFieldConfigSnapshot, error) {
	def, ok := model.MemberFieldDefinitionByKey(fieldKey)
	if !ok {
		return nil, ErrMemberFieldNotFound
	}

	settings, err := s.ensureDefaults(ctx)
	if err != nil {
		return nil, err
	}
	setting, ok := settingOfKey(settings, fieldKey)
	if !ok {
		return nil, ErrMemberFieldNotFound
	}

	// 请求只提交本次变更的开关：指针为 nil 表示沿用当前值
	nextVisible := setting.PersonalVisible
	nextEditable := setting.PersonalEditable
	nextCard := setting.CardVisible
	if req.PersonalVisible != nil {
		// 关闭可见必须同步关闭可编辑（文档 3.2）；前端已先联动，服务端兜底
		nextVisible = *req.PersonalVisible
		if !nextVisible {
			nextEditable = false
		}
	}
	if req.PersonalEditable != nil {
		nextEditable = *req.PersonalEditable
	}
	if req.CardVisible != nil {
		nextCard = *req.CardVisible
	}

	// 锁定规则：固定项与当前值比较（注册表默认即为锁定值），请求试图变更即拒绝
	if def.VisibilityLocked && nextVisible != setting.PersonalVisible {
		return nil, ErrMemberFieldLocked
	}
	if def.EditableLocked && nextEditable != setting.PersonalEditable {
		return nil, ErrMemberFieldLocked
	}
	if def.CardLocked && nextCard != setting.CardVisible {
		return nil, ErrMemberFieldLocked
	}
	// 联动规则（文档 3.2）：personalEditable=true 必须同时 personalVisible=true
	if nextEditable && !nextVisible {
		return nil, ErrMemberFieldConfigInvalid
	}

	before := map[string]string{
		"personalVisible":  strconv.FormatBool(setting.PersonalVisible),
		"personalEditable": strconv.FormatBool(setting.PersonalEditable),
		"cardVisible":      strconv.FormatBool(setting.CardVisible),
	}
	var updated bool
	err = s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		// 目标行按乐观锁条件更新开关值；版本推进统一交给 BumpRevision：
		// 同租户所有行同步 +1，保证整页 revision 是真正的租户配置快照版本
		ok, err := s.fields.UpdateWithRevision(tctx, setting.ID, req.Revision, map[string]interface{}{
			"personal_visible":  nextVisible,
			"personal_editable": nextEditable,
			"card_visible":      nextCard,
		})
		updated = ok
		if err != nil || !ok {
			return err
		}
		return s.fields.BumpRevision(tctx)
	})
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, ErrMemberFieldConfigConflict
	}

	// 审计在事务提交后 best-effort 写入：记录字段 key 与变更前后值
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "update", ResourceType: "member_field_setting",
			ResourceID: fieldKey,
			Before:     before,
			After: map[string]string{
				"personalVisible":  strconv.FormatBool(nextVisible),
				"personalEditable": strconv.FormatBool(nextEditable),
				"cardVisible":      strconv.FormatBool(nextCard),
			},
		})
	}

	latest, err := s.fields.ListByTenant(ctx)
	if err != nil {
		return nil, err
	}
	return buildSnapshot(latest), nil
}

// SeedDefaults 为指定租户写入全部预置字段默认配置（幂等）：租户开通事务内
// 预置调用；bctx 需携带目标租户上下文（与 seedTenantBaseline 同口径）
func (s *memberFieldService) SeedDefaults(ctx context.Context, tenantID uint) error {
	rows := defaultSettingRows(tenantID)
	return s.fields.CreateBatch(ctx, rows)
}

// ensureDefaults 读取当前租户配置并补齐缺失默认行（幂等兜底），返回补齐后的全量行
func (s *memberFieldService) ensureDefaults(ctx context.Context) ([]model.MemberFieldSetting, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required")
	}
	settings, err := s.fields.ListByTenant(ctx)
	if err != nil {
		return nil, err
	}
	existing := make(map[string]struct{}, len(settings))
	for _, setting := range settings {
		existing[setting.FieldKey] = struct{}{}
	}
	missing := make([]model.MemberFieldSetting, 0)
	for _, def := range model.MemberFieldRegistry() {
		if _, ok := existing[def.Key]; !ok {
			row := defaultSettingRow(def, tenantID)
			missing = append(missing, row)
		}
	}
	if len(missing) > 0 {
		if err := s.fields.CreateBatch(ctx, missing); err != nil {
			return nil, err
		}
		// 补齐后重读，保证 revision 等列与库内一致
		return s.fields.ListByTenant(ctx)
	}
	return settings, nil
}

// defaultSettingRows 按注册表生成某租户的默认配置行集
func defaultSettingRows(tenantID uint) []model.MemberFieldSetting {
	registry := model.MemberFieldRegistry()
	rows := make([]model.MemberFieldSetting, 0, len(registry))
	for _, def := range registry {
		rows = append(rows, defaultSettingRow(def, tenantID))
	}
	return rows
}

func defaultSettingRow(def model.MemberFieldDefinition, tenantID uint) model.MemberFieldSetting {
	setting := model.MemberFieldSetting{
		FieldKey:         def.Key,
		PersonalVisible:  def.PersonalVisible,
		PersonalEditable: def.PersonalEditable,
		CardVisible:      def.CardVisible,
		Revision:         1,
	}
	// seed 路径 ctx 可能无租户上下文，显式写 TenantID 与 Callback 回填口径一致
	setting.TenantID = tenantID
	return setting
}

// buildSnapshot 按注册表顺序组装快照；未落库的行（理论不会出现，ensureDefaults
// 已兜底）回落注册表默认值，保证 15 个字段恒完整
func buildSnapshot(settings []model.MemberFieldSetting) *model.MemberFieldConfigSnapshot {
	byKey := make(map[string]model.MemberFieldSetting, len(settings))
	var revision int64 = 1
	for _, setting := range settings {
		byKey[setting.FieldKey] = setting
		if setting.Revision > revision {
			revision = setting.Revision
		}
	}
	registry := model.MemberFieldRegistry()
	fields := make([]model.MemberFieldSettingView, 0, len(registry))
	for _, def := range registry {
		view := model.MemberFieldSettingView{
			Key:              def.Key,
			Label:            def.Label,
			Type:             def.Type,
			PersonalVisible:  def.PersonalVisible,
			PersonalEditable: def.PersonalEditable,
			CardVisible:      def.CardVisible,
			VisibilityLocked: def.VisibilityLocked,
			EditableLocked:   def.EditableLocked,
			CardLocked:       def.CardLocked,
		}
		if setting, ok := byKey[def.Key]; ok {
			view.PersonalVisible = setting.PersonalVisible
			view.PersonalEditable = setting.PersonalEditable
			view.CardVisible = setting.CardVisible
		}
		fields = append(fields, view)
	}
	return &model.MemberFieldConfigSnapshot{Revision: revision, Fields: fields}
}

func settingOfKey(settings []model.MemberFieldSetting, fieldKey string) (model.MemberFieldSetting, bool) {
	for _, setting := range settings {
		if setting.FieldKey == fieldKey {
			return setting, true
		}
	}
	return model.MemberFieldSetting{}, false
}
