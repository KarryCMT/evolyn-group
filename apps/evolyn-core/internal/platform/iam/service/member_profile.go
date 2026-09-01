package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"evolyn/internal/contextx"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"

	"gorm.io/gorm"
)

// memberProfileService 正式成员扩展档案（成员信息管理二期）：
// 「字段 key → 档案值」的组装与读写裁剪。值来源跨 users/accounts/关系表与
// tn_member_profiles（不重复存储手机/邮箱/部门/角色，文档 4.2）；
// 本人读取按 personalVisible 裁剪、写入仅放行 personalEditable 的扩展字段；
// 管理员读写不裁剪，卡片视图按 cardVisible 服务端裁剪后下发
type memberProfileService struct {
	tx       TxManager
	profiles repository.MemberProfileRepository
	fields   repository.MemberFieldSettingRepository
	users    repository.UserRepository
	audit    auditservice.Recorder
}

func NewMemberProfileService(
	tx TxManager,
	profiles repository.MemberProfileRepository,
	fields repository.MemberFieldSettingRepository,
	users repository.UserRepository,
	audit auditservice.Recorder,
) MemberProfileService {
	return &memberProfileService{tx: tx, profiles: profiles, fields: fields, users: users, audit: audit}
}

// GetMyProfile 本人视图：按 personalVisible 裁剪 Values，EditableKeys 为
// 当前允许经通用资料接口提交的扩展字段（文档 5.2）
func (s *memberProfileService) GetMyProfile(ctx context.Context, memberID uint) (*model.MemberProfileView, error) {
	values, snapshot, err := s.assembleValues(ctx, memberID)
	if err != nil {
		return nil, err
	}
	visible := make(map[string]string, len(values))
	editable := make([]string, 0)
	for _, field := range snapshot.Fields {
		value, ok := values[field.Key]
		if !ok {
			continue
		}
		if field.PersonalVisible {
			visible[field.Key] = value
		}
		// 手机/邮箱/部门/角色/姓名/编号不经通用资料接口写入（各自走专用
		// 接口或管理员维护），个人可编辑仅对扩展字段成立（文档 3.2）
		if field.PersonalEditable && model.IsMemberExtensionFieldKey(field.Key) {
			editable = append(editable, field.Key)
		}
	}
	return &model.MemberProfileView{Values: visible, EditableKeys: editable}, nil
}

// UpdateMyProfile 本人更新：只接受 personalEditable=true 的扩展字段；
// 其余 key（含手机/邮箱等账号安全字段）一律 MEMBER_PROFILE_INVALID，
// 禁止经通用资料接口绕过既有专用流程（文档 3.2/5.2）
func (s *memberProfileService) UpdateMyProfile(ctx context.Context, memberID uint, values map[string]string) (*model.MemberProfileView, error) {
	if len(values) == 0 {
		return s.GetMyProfile(ctx, memberID)
	}
	if _, err := s.users.GetMemberDetail(ctx, memberID); err != nil {
		return nil, err
	}
	snapshot, err := s.loadSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	editableSet := make(map[string]bool)
	for _, field := range snapshot.Fields {
		if field.PersonalEditable && model.IsMemberExtensionFieldKey(field.Key) {
			editableSet[field.Key] = true
		}
	}
	for key := range values {
		if !editableSet[key] {
			return nil, ErrMemberProfileInvalid
		}
	}
	if err := s.applyAttributes(ctx, memberID, values); err != nil {
		return nil, err
	}
	return s.GetMyProfile(ctx, memberID)
}

// GetMemberProfile 管理员视图：全量 Values + cardVisible 裁剪的 CardValues +
// 字段元数据。卡片必须消费服务端裁剪结果，不得前端自行隐藏（文档 5.2）
func (s *memberProfileService) GetMemberProfile(ctx context.Context, memberID uint) (*model.MemberProfileAdminView, error) {
	values, snapshot, err := s.assembleValues(ctx, memberID)
	if err != nil {
		return nil, err
	}
	card := make(map[string]string, len(values))
	for _, field := range snapshot.Fields {
		if field.CardVisible {
			if value, ok := values[field.Key]; ok {
				card[field.Key] = value
			}
		}
	}
	return &model.MemberProfileAdminView{Values: values, CardValues: card, FieldConfig: snapshot.Fields}, nil
}

// UpdateMemberProfile 管理员维护：可写全部扩展字段与企业内编号 identifier
// （租户内有效记录唯一）；不受理手机/邮箱/部门/角色（专用接口，文档 5.2）
func (s *memberProfileService) UpdateMemberProfile(ctx context.Context, memberID uint, req *MemberProfileUpdateRequest) (*model.MemberProfileAdminView, error) {
	if _, err := s.users.GetMemberDetail(ctx, memberID); err != nil {
		return nil, err
	}

	identifierTouched := req.Identifier != nil
	identifier := ""
	if identifierTouched {
		identifier = strings.TrimSpace(*req.Identifier)
		if utf8.RuneCountInString(identifier) > 50 {
			return nil, ErrMemberProfileInvalid
		}
		if identifier != "" {
			exists, err := s.profiles.IdentifierExists(ctx, identifier, memberID)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, ErrMemberProfileInvalid
			}
		}
	}
	if err := validateExtensionValues(req.Values); err != nil {
		return nil, err
	}

	before := map[string]string{}
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		profile, err := s.loadProfile(tctx, memberID)
		if err != nil {
			return err
		}
		if identifierTouched {
			before["identifier"] = profile.Identifier
			profile.Identifier = identifier
		}
		attributes := cloneAttributes(profile.Attributes)
		for key, value := range req.Values {
			before[key] = attributes[key]
			attributes[key] = value
		}
		profile.Attributes = attributes
		_, err = s.profiles.Upsert(tctx, profile)
		return err
	}); err != nil {
		return nil, err
	}

	if s.audit != nil && (len(before) > 0) {
		after := make(map[string]string, len(before))
		for key := range before {
			switch {
			case key == "identifier":
				after[key] = identifier
			default:
				after[key] = req.Values[key]
			}
		}
		s.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "update", ResourceType: "member_profile",
			ResourceID: strconv.FormatUint(uint64(memberID), 10),
			Before:     before,
			After:      after,
		})
	}
	return s.GetMemberProfile(ctx, memberID)
}

// applyAttributes 本人提交的扩展字段合并落库（事务内）：
// 只覆盖提交的 key，未提交的历史值保持不变
func (s *memberProfileService) applyAttributes(ctx context.Context, memberID uint, values map[string]string) error {
	if err := validateExtensionValues(values); err != nil {
		return err
	}
	before := map[string]string{}
	return s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		profile, err := s.loadProfile(tctx, memberID)
		if err != nil {
			return err
		}
		attributes := cloneAttributes(profile.Attributes)
		for key, value := range values {
			before[key] = attributes[key]
			attributes[key] = value
		}
		profile.Attributes = attributes
		_, err = s.profiles.Upsert(tctx, profile)
		return err
	})
}

// assembleValues 组装成员的全量档案值（15 字段恒全量计算，裁剪交给调用方）
// 并返回当前字段配置快照。无档案成员按空档案语义处理（字段值为空串）
func (s *memberProfileService) assembleValues(ctx context.Context, memberID uint) (map[string]string, *model.MemberFieldConfigSnapshot, error) {
	member, err := s.users.GetMemberDetail(ctx, memberID)
	if err != nil {
		return nil, nil, err
	}
	profile, err := s.loadProfile(ctx, memberID)
	if err != nil {
		return nil, nil, err
	}
	snapshot, err := s.loadSnapshot(ctx)
	if err != nil {
		return nil, nil, err
	}

	values := map[string]string{
		model.MemberFieldKeyName: memberDisplayName(member),
		model.MemberFieldKeyCode: profile.Identifier,
	}
	// 扩展字段依注册表逐个取值，保证注册表新增字段自动出网
	for _, def := range model.MemberFieldRegistry() {
		if model.IsMemberExtensionFieldKey(def.Key) {
			values[def.Key] = profile.Attributes[def.Key]
		}
	}
	if member.Account != nil {
		values[model.MemberFieldKeyMobile] = member.Account.Phone
		values[model.MemberFieldKeyEmail] = member.Account.Email
	}
	values[model.MemberFieldKeyDepartment] = joinNames(departmentNames(member), ", ")
	values[model.MemberFieldKeyRole] = joinNames(roleNames(member), ", ")
	return values, snapshot, nil
}

// loadSnapshot 读取租户字段配置；本人/管理员读路径不落库兜底（seed 与
// 管理端 GET 负责），缺失行按注册表默认值兜底渲染
func (s *memberProfileService) loadSnapshot(ctx context.Context) (*model.MemberFieldConfigSnapshot, error) {
	if _, ok := contextx.TenantIDFromContext(ctx); !ok {
		return nil, errors.New("tenant context required")
	}
	settings, err := s.fields.ListByTenant(ctx)
	if err != nil {
		return nil, err
	}
	return buildSnapshot(settings), nil
}

// loadProfile 读取成员档案；无档案（未接受过邀请/未维护过资料）返回空档案
func (s *memberProfileService) loadProfile(ctx context.Context, memberID uint) (*model.MemberProfile, error) {
	profile, err := s.profiles.GetByMember(ctx, memberID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		profile = &model.MemberProfile{MemberID: memberID, Attributes: model.MemberProfileAttributes{}}
		// 租户由 Callback 回填；seed 语义上仍显式带上（Upsert 插入路径依赖）
		if tenantID, ok := contextx.TenantIDFromContext(ctx); ok {
			profile.TenantID = tenantID
		}
		return profile, nil
	}
	return profile, err
}

// validateExtensionValues 扩展字段值校验（文档 4.2）：key 必须在注册表扩展
// 集合内、文本最长 50 字符、日期字段统一 YYYY-MM-DD 且为合法日期
func validateExtensionValues(values map[string]string) error {
	for key, value := range values {
		if !model.IsMemberExtensionFieldKey(key) {
			return ErrMemberProfileInvalid
		}
		if utf8.RuneCountInString(strings.TrimSpace(value)) > 50 {
			return ErrMemberProfileInvalid
		}
		if key == model.MemberFieldKeyHireDate || key == model.MemberFieldKeyBirthday {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				if _, err := time.Parse("2006-01-02", trimmed); err != nil {
					return ErrMemberProfileInvalid
				}
			}
		}
	}
	return nil
}

func cloneAttributes(src model.MemberProfileAttributes) model.MemberProfileAttributes {
	dst := make(model.MemberProfileAttributes, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

// memberDisplayName 姓名取值链：租户内昵称为空时回退账号昵称/登录名（文档 3.1）
func memberDisplayName(member *model.User) string {
	if member.Nickname != "" {
		return member.Nickname
	}
	if member.Account != nil {
		if member.Account.Nickname != "" {
			return member.Account.Nickname
		}
		return member.Account.Name
	}
	return ""
}

func departmentNames(member *model.User) []string {
	names := make([]string, 0, len(member.Departments))
	for _, department := range member.Departments {
		names = append(names, department.Name)
	}
	return names
}

func roleNames(member *model.User) []string {
	names := make([]string, 0, len(member.Roles))
	for _, role := range member.Roles {
		names = append(names, role.Name)
	}
	return names
}

func joinNames(names []string, sep string) string {
	return strings.Join(names, sep)
}
