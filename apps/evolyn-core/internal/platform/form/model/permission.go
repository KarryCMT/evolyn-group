// 权限组数据模型（表单权限 P1，docs/低代码平台/表单权限/表单权限组后端设计方案.md §6）：
// 资产权限组 = 主体范围（成员/部门/角色）× 操作集 × 字段矩阵 × 数据范围的
// 整体授权单元，四要素整体提交、整体生效（S1）。operations/field_permissions/
// data_scope 以 JSONB 类型化容器存储，写入前由 Service 层校验器整体校验。
package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	kernel "evolyn/internal/model"
)

// 资产类型（类型白名单由 Service 层注册表承载，不设数据库 CHECK——仪表盘
// 扩展无需 DDL，约束与代码同源）
const PermissionAssetTypeForm = "form"

// 权限组主体类型稳定枚举（tn_asset_permission_group_subjects.subject_type CHECK 同源）
const (
	PermissionSubjectMember     = "member"
	PermissionSubjectDepartment = "department"
	PermissionSubjectRole       = "role"
)

// ValidPermissionSubjectType 主体类型是否属于稳定枚举
func ValidPermissionSubjectType(t string) bool {
	switch t {
	case PermissionSubjectMember, PermissionSubjectDepartment, PermissionSubjectRole:
		return true
	}
	return false
}

// 操作权限键稳定字典（设计 §3；出网/存储均用键，中文文案由前端字典渲染）。
// 普通表单合法集为 standard 9 项，流程表单在此基础上追加 workflow_ 3 项。
const (
	PermissionOpView       = "view"
	PermissionOpAdd        = "add"
	PermissionOpCopy       = "copy"
	PermissionOpEdit       = "edit"
	PermissionOpDelete     = "delete"
	PermissionOpBatchPrint = "batch_print"

	PermissionOpBatchModify = "batch_modify"
	PermissionOpImport      = "import"
	PermissionOpExport      = "export"

	PermissionOpWorkflowOwnerTransfer = "workflow_owner_transfer"
	PermissionOpWorkflowTerminate     = "workflow_terminate"
	PermissionOpWorkflowActivate      = "workflow_activate"
)

// standardPermissionOperations 普通表单（form_type=standard）合法操作集
var standardPermissionOperations = []string{
	PermissionOpView, PermissionOpAdd, PermissionOpCopy, PermissionOpEdit,
	PermissionOpDelete, PermissionOpBatchPrint, PermissionOpBatchModify,
	PermissionOpImport, PermissionOpExport,
}

// workflowPermissionOperations 流程表单（form_type=workflow）追加的流程操作集
var workflowPermissionOperations = []string{
	PermissionOpWorkflowOwnerTransfer, PermissionOpWorkflowTerminate, PermissionOpWorkflowActivate,
}

// ValidPermissionOperation 键是否属于操作字典（不含类型分派）
func ValidPermissionOperation(op string) bool {
	for _, key := range append(append([]string{}, standardPermissionOperations...), workflowPermissionOperations...) {
		if key == op {
			return true
		}
	}
	return false
}

// IsWorkflowPermissionOperation 流程专属操作键（standard 表单出现即整体拒绝）
func IsWorkflowPermissionOperation(op string) bool {
	for _, key := range workflowPermissionOperations {
		if key == op {
			return true
		}
	}
	return false
}

// LegalPermissionOperations 返回指定表单类型的合法操作键集（新建副本防外部改写）
func LegalPermissionOperations(formType FormType) []string {
	keys := append([]string{}, standardPermissionOperations...)
	if formType == FormTypeWorkflow {
		keys = append(keys, workflowPermissionOperations...)
	}
	return keys
}

// 数据范围 match 语义（S6：空条件=全部数据）
const (
	PermissionScopeMatchAll = "all" // 且（默认）
	PermissionScopeMatchAny = "any" // 或
)

// PermissionFieldRule 字段矩阵条目：字段键 + 可见/可编辑（JSONB 存储与出网同形）。
// deny-by-default（S7）：矩阵中缺失的字段默认不可见不可编辑，由管理员显式放行。
type PermissionFieldRule struct {
	Field    string `json:"field"`
	Visible  bool   `json:"visible"`
	Editable bool   `json:"editable"`
}

// PermissionDataCondition 数据范围条件：字段 + operator（§5.1 字典）+ 比较值。
// value 统一为数组形态（标量取单元素）；empty/not_empty 须为空数组。
type PermissionDataCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    []any  `json:"value"`
}

// PermissionDataScopeSpec 数据范围：match 决定组内条件的合取方式（S6）。
type PermissionDataScopeSpec struct {
	Match      string                    `json:"match"`
	Conditions []PermissionDataCondition `json:"conditions"`
}

// Normalize 归一缺省：match 空回落 all（默认且），conditions 空数组兜底
func (s *PermissionDataScopeSpec) Normalize() {
	if s.Match == "" {
		s.Match = PermissionScopeMatchAll
	}
	if s.Conditions == nil {
		s.Conditions = []PermissionDataCondition{}
	}
}

// ---- JSONB 类型化容器（口径同 iam model.Rules：Value 出入字符串形态） ----

// PermissionOperations 操作键数组容器
type PermissionOperations []string

// Value 实现 driver.Valuer
func (o PermissionOperations) Value() (driver.Value, error) {
	if o == nil {
		o = PermissionOperations{}
	}
	b, err := json.Marshal(o)
	return string(b), err
}

// Scan 实现 sql.Scanner
func (o *PermissionOperations) Scan(value interface{}) error {
	data, ok := value.([]byte)
	if !ok {
		if text, isText := value.(string); isText {
			data = []byte(text)
		} else if value == nil {
			*o = PermissionOperations{}
			return nil
		} else {
			return fmt.Errorf("form: cannot scan PermissionOperations from %T", value)
		}
	}
	return json.Unmarshal(data, o)
}

// PermissionFieldRules 字段矩阵容器
type PermissionFieldRules []PermissionFieldRule

// Value 实现 driver.Valuer
func (r PermissionFieldRules) Value() (driver.Value, error) {
	if r == nil {
		r = PermissionFieldRules{}
	}
	b, err := json.Marshal(r)
	return string(b), err
}

// Scan 实现 sql.Scanner
func (r *PermissionFieldRules) Scan(value interface{}) error {
	data, ok := value.([]byte)
	if !ok {
		if text, isText := value.(string); isText {
			data = []byte(text)
		} else if value == nil {
			*r = PermissionFieldRules{}
			return nil
		} else {
			return fmt.Errorf("form: cannot scan PermissionFieldRules from %T", value)
		}
	}
	return json.Unmarshal(data, r)
}

// PermissionDataScopeValue 数据范围容器
type PermissionDataScopeValue PermissionDataScopeSpec

// Value 实现 driver.Valuer
func (s PermissionDataScopeValue) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	return string(b), err
}

// Scan 实现 sql.Scanner
func (s *PermissionDataScopeValue) Scan(value interface{}) error {
	data, ok := value.([]byte)
	if !ok {
		if text, isText := value.(string); isText {
			data = []byte(text)
		} else if value == nil {
			*s = PermissionDataScopeValue{}
			return nil
		} else {
			return fmt.Errorf("form: cannot scan PermissionDataScopeValue from %T", value)
		}
	}
	return json.Unmarshal(data, s)
}

// ---- GORM 模型 ----

// AssetPermissionGroup 资产权限组：组删除走 TenantBaseModel 软删；禁用组
// 同样维持收口（S5）但不授权。code 为 fpg_ 前缀服务端生成的出网稳定标识。
type AssetPermissionGroup struct {
	ID               uint                     `json:"id" gorm:"autoIncrement;primaryKey"`
	ApplicationID    uint                     `json:"applicationId" gorm:"not null"` // 冗余归属，Service 校验与资产一致
	AssetType        string                   `json:"assetType" gorm:"size:16;not null;default:form"`
	AssetID          uint                     `json:"assetId" gorm:"not null"` // form → tn_forms.id（内部主键）
	Code             string                   `json:"code" gorm:"size:64;not null"`
	Name             string                   `json:"name" gorm:"size:64;not null"`
	Description      string                   `json:"description" gorm:"size:200;not null;default:''"`
	Enabled          bool                     `json:"enabled" gorm:"not null;default:true"`
	Operations       PermissionOperations     `json:"operations" gorm:"type:jsonb;not null;default:[]"`
	FieldPermissions PermissionFieldRules     `json:"fieldPermissions" gorm:"type:jsonb;not null;default:[]"`
	DataScope        PermissionDataScopeValue `json:"dataScope" gorm:"type:jsonb;not null;default:{}"`
	Revision         int64                    `json:"revision" gorm:"not null;default:1"` // 整组乐观锁（PUT 全量提交口令）

	kernel.TenantBaseModel
}

func (*AssetPermissionGroup) TableName() string { return "tn_asset_permission_groups" }

// AssetPermissionGroupSubject 权限组主体：随组软删/硬删联动（DELETE Service
// 同事务显式硬删，外键仅兜底物理清理路径）。subject_id 无外键，判定侧容错。
type AssetPermissionGroupSubject struct {
	ID          uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	GroupID     uint            `json:"groupId" gorm:"not null;index"`
	SubjectType string          `json:"subjectType" gorm:"size:16;not null"`
	SubjectID   uint            `json:"subjectId" gorm:"not null"`
	CreatedAt   kernel.JSONTime `json:"createdAt"`

	TenantID uint `json:"tenantId" gorm:"index;not null;default:1"`
}

func (*AssetPermissionGroupSubject) TableName() string { return "tn_asset_permission_group_subjects" }

// ---- 出入网 DTO ----

// PermissionSubjectInput 主体写入项
type PermissionSubjectInput struct {
	Type string `json:"type" binding:"required" example:"member"`
	ID   uint   `json:"id" binding:"required" example:"12"`
}

// CreatePermissionGroupRequest 创建权限组（POST /forms/:code/permission-groups）：
// 四要素整体提交（S1）；dataScope 传 null 等价全部数据（S6）。
type CreatePermissionGroupRequest struct {
	Name             string                   `json:"name" binding:"required" example:"管理全部数据"`
	Description      string                   `json:"description" example:"此分组内的成员可以管理全部数据。"`
	Enabled          *bool                    `json:"enabled"`
	Operations       []string                 `json:"operations" example:"view,add,edit"`
	FieldPermissions []PermissionFieldRule    `json:"fieldPermissions"`
	DataScope        *PermissionDataScopeSpec `json:"dataScope"`
	SubjectIds       []PermissionSubjectInput `json:"subjectIds"`
}

// UpdatePermissionGroupRequest 更新权限组（PUT 全量提交，baseRevision 为整组
// 乐观锁口令；冲突返回 FORM_PERMISSION_REVISION_CONFLICT 409）。
type UpdatePermissionGroupRequest struct {
	CreatePermissionGroupRequest
	BaseRevision int64 `json:"baseRevision" binding:"required" example:"4"`
}

// PermissionSubjectView 主体出网项（name 为读取时解析的展示名，解析不到为空串）
type PermissionSubjectView struct {
	Type string `json:"type" example:"member"`
	ID   uint   `json:"id" example:"12"`
	Name string `json:"name" example:"张三"`
}

// PermissionGroupView 权限组出网视图（组四要素 + 主体清单）
type PermissionGroupView struct {
	Code             string                  `json:"code" example:"fpg_01940af83bcc"`
	Name             string                  `json:"name"`
	Description      string                  `json:"description"`
	Enabled          bool                    `json:"enabled"`
	Operations       []string                `json:"operations"`
	FieldPermissions []PermissionFieldRule   `json:"fieldPermissions"`
	DataScope        PermissionDataScopeSpec `json:"dataScope"`
	Revision         int64                   `json:"revision" example:"4"`
	Subjects         []PermissionSubjectView `json:"subjects"`
}

// PermissionFieldView 权限配置字段清单条目（GET /forms/:code/permission-fields）：
// 字段清单事实源为最新发布版本（未发布回落草稿），required 取快照 allowBlank 反相。
type PermissionFieldView struct {
	Field    string `json:"field" example:"name"`
	Label    string `json:"label" example:"姓名"`
	Type     string `json:"type" example:"text"`
	Required bool   `json:"required"`
}

// String 辅助：操作键数组的日志形态
func (o PermissionOperations) String() string {
	return strings.Join(o, ",")
}
