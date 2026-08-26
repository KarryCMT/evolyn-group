package model

import (
	kernel "evolyn/internal/model"
)

// 成员字段 key（前后端统一口径，见《管理后台-成员信息管理开发文档》3.1 预置字段目录）。
// 与既有邀请字段的对应关系：code↔identifier、employeeId↔employeeNo、position↔title、
// employment↔employmentType、hireDate↔hiredAt、workplace↔workLocation，对外只使用本处 key。
const (
	MemberFieldKeyName        = "name"
	MemberFieldKeyCode        = "code"
	MemberFieldKeyMobile      = "mobile"
	MemberFieldKeyEmail       = "email"
	MemberFieldKeyDepartment  = "department"
	MemberFieldKeyRole        = "role"
	MemberFieldKeyAlias       = "alias"
	MemberFieldKeyEmployeeId  = "employeeId"
	MemberFieldKeyGender      = "gender"
	MemberFieldKeyPosition    = "position"
	MemberFieldKeyEmployment  = "employment"
	MemberFieldKeyHireDate    = "hireDate"
	MemberFieldKeyWorkplace   = "workplace"
	MemberFieldKeyBirthday    = "birthday"
	MemberFieldKeyEducation   = "education"
	MemberFieldExtensionCount = 9 // attributes 承载的扩展字段数（alias...education）
)

// 字段类型（接口响应中的展示名，与前端字段目录一致）
const (
	MemberFieldTypeText     = "单行文本"
	MemberFieldTypeDept     = "部门多选"
	MemberFieldTypeRole     = "角色多选"
	MemberFieldTypeDateTime = "日期时间"
)

// MemberFieldDefinition 服务端字段注册表条目：字段元数据、锁定规则与默认配置
// 唯一事实来源。客户端不能提交或修改字段名称、类型及锁定规则（文档 3.1/3.2）。
type MemberFieldDefinition struct {
	Key   string
	Label string
	Type  string
	// VisibilityLocked/EditableLocked/CardLocked 为 true 时对应开关固定，
	// 管理端请求变更即 MEMBER_FIELD_LOCKED
	VisibilityLocked bool
	EditableLocked   bool
	CardLocked       bool
	// 默认配置：新租户 seed 与读取侧幂等兜底共用
	PersonalVisible  bool
	PersonalEditable bool
	CardVisible      bool
}

// memberFieldRegistry 预置字段注册表（顺序即接口返回顺序，与前端目录一致）。
// 锁定策略：姓名/编号/手机/邮箱/部门/角色属于既有账号与关系模型，可见性与
// 编辑权限固定，避免管理员经字段配置绕开账号安全与关系管理接口；
// 姓名为资料卡固定信息（cardLocked），不参与卡片勾选。
var memberFieldRegistry = []MemberFieldDefinition{
	{Key: MemberFieldKeyName, Label: "姓名", Type: MemberFieldTypeText, VisibilityLocked: true, EditableLocked: true, CardLocked: true, PersonalVisible: true, PersonalEditable: false, CardVisible: true},
	{Key: MemberFieldKeyCode, Label: "编号", Type: MemberFieldTypeText, VisibilityLocked: true, EditableLocked: true, PersonalVisible: false, PersonalEditable: false, CardVisible: true},
	{Key: MemberFieldKeyMobile, Label: "手机", Type: MemberFieldTypeText, VisibilityLocked: true, EditableLocked: true, PersonalVisible: true, PersonalEditable: true, CardVisible: true},
	{Key: MemberFieldKeyEmail, Label: "邮箱", Type: MemberFieldTypeText, VisibilityLocked: true, EditableLocked: true, PersonalVisible: true, PersonalEditable: true, CardVisible: true},
	{Key: MemberFieldKeyDepartment, Label: "部门", Type: MemberFieldTypeDept, VisibilityLocked: true, EditableLocked: true, PersonalVisible: false, PersonalEditable: false, CardVisible: true},
	{Key: MemberFieldKeyRole, Label: "角色", Type: MemberFieldTypeRole, VisibilityLocked: true, EditableLocked: true, PersonalVisible: false, PersonalEditable: false, CardVisible: false},
	{Key: MemberFieldKeyAlias, Label: "别名", Type: MemberFieldTypeText, PersonalVisible: false, PersonalEditable: false, CardVisible: false},
	{Key: MemberFieldKeyEmployeeId, Label: "工号", Type: MemberFieldTypeText, PersonalVisible: false, PersonalEditable: false, CardVisible: false},
	{Key: MemberFieldKeyGender, Label: "性别", Type: MemberFieldTypeText, PersonalVisible: false, PersonalEditable: false, CardVisible: false},
	{Key: MemberFieldKeyPosition, Label: "职务", Type: MemberFieldTypeText, PersonalVisible: false, PersonalEditable: false, CardVisible: false},
	{Key: MemberFieldKeyEmployment, Label: "聘用形式", Type: MemberFieldTypeText, PersonalVisible: false, PersonalEditable: false, CardVisible: false},
	{Key: MemberFieldKeyHireDate, Label: "入职日期", Type: MemberFieldTypeDateTime, PersonalVisible: false, PersonalEditable: false, CardVisible: false},
	{Key: MemberFieldKeyWorkplace, Label: "工作地点", Type: MemberFieldTypeText, PersonalVisible: false, PersonalEditable: false, CardVisible: false},
	{Key: MemberFieldKeyBirthday, Label: "出生日期", Type: MemberFieldTypeDateTime, PersonalVisible: false, PersonalEditable: false, CardVisible: false},
	{Key: MemberFieldKeyEducation, Label: "学历", Type: MemberFieldTypeText, PersonalVisible: false, PersonalEditable: false, CardVisible: false},
}

// MemberFieldRegistry 返回预置字段注册表副本（顺序稳定）。
func MemberFieldRegistry() []MemberFieldDefinition {
	out := make([]MemberFieldDefinition, len(memberFieldRegistry))
	copy(out, memberFieldRegistry)
	return out
}

// MemberFieldDefinitionByKey 按字段 key 查注册表；不存在返回 false（用于稳定
// 返回 MEMBER_FIELD_NOT_FOUND）
func MemberFieldDefinitionByKey(key string) (MemberFieldDefinition, bool) {
	for _, def := range memberFieldRegistry {
		if def.Key == key {
			return def, true
		}
	}
	return MemberFieldDefinition{}, false
}

// IsMemberExtensionFieldKey 是否为 member_profiles.attributes 承载的扩展字段
// key（通用成员资料接口唯一可写的字段集合）
func IsMemberExtensionFieldKey(key string) bool {
	switch key {
	case MemberFieldKeyAlias, MemberFieldKeyEmployeeId, MemberFieldKeyGender,
		MemberFieldKeyPosition, MemberFieldKeyEmployment, MemberFieldKeyHireDate,
		MemberFieldKeyWorkplace, MemberFieldKeyBirthday, MemberFieldKeyEducation:
		return true
	}
	return false
}

// MemberFieldSetting 租户级成员字段显示策略：每租户每字段一行。
// revision 为租户配置快照版本（同租户所有行同步递增），前端以整页 revision
// 做乐观锁提交（文档 4.1/5.1）。
// 注意：bool 列不带 gorm default tag——default 会使 GORM 在 INSERT 时跳过
// 零值 false 而落数据库默认 true，破坏注册表默认配置；列默认值仅由迁移
// SQL 表达（生产事实源），seed 路径经 Select 显式写入全部列
type MemberFieldSetting struct {
	ID               uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	FieldKey         string `json:"fieldKey" gorm:"size:64;not null"`
	PersonalVisible  bool   `json:"personalVisible" gorm:"not null"`
	PersonalEditable bool   `json:"personalEditable" gorm:"not null"`
	CardVisible      bool   `json:"cardVisible" gorm:"not null"`
	Revision         int64  `json:"revision" gorm:"not null"`

	kernel.TenantBaseModel
}

func (*MemberFieldSetting) TableName() string { return "tenant_member_field_settings" }

// MemberFieldSettingView 管理端字段配置快照的单字段视图：注册表元数据
// （key/label/type/锁定）+ 租户配置值，前端以响应覆盖本地状态
type MemberFieldSettingView struct {
	Key              string `json:"key"`
	Label            string `json:"label"`
	Type             string `json:"type"`
	PersonalVisible  bool   `json:"personalVisible"`
	PersonalEditable bool   `json:"personalEditable"`
	CardVisible      bool   `json:"cardVisible"`
	VisibilityLocked bool   `json:"visibilityLocked"`
	EditableLocked   bool   `json:"editableLocked"`
	CardLocked       bool   `json:"cardLocked"`
}

// MemberFieldConfigSnapshot 字段配置整页快照：revision 为租户级版本号，
// 字段设置页与卡片展示页共用（同一份服务端配置）
type MemberFieldConfigSnapshot struct {
	Revision int64                    `json:"revision"`
	Fields   []MemberFieldSettingView `json:"fields"`
}
