// Package model 表单资产域数据模型（后端契约 §1）。
package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	kernel "evolyn/internal/model"
)

// FormType 表单资产类型：标准表单只承载字段与数据，流程表单额外开放流程
// 设计能力。ADR-011 起类型可经 form-actions:switch-type 动作切换
// （standard↔workflow），切换后原类型流程数据保留；切换裁决在 Service 层。
type FormType string

const (
	FormTypeStandard FormType = "standard"
	FormTypeWorkflow FormType = "workflow"
)

// Valid 判断表单类型是否属于稳定枚举，禁止未知字符串进入持久化层。
func (t FormType) Valid() bool {
	return t == FormTypeStandard || t == FormTypeWorkflow
}

// Form 表单资产（含草稿）：租户内从属于应用；草稿全文为目标保存协议文档，
// draft_revision 为草稿乐观锁口令；发布快照另存 form_versions（不可变）。
type Form struct {
	ID               uint        `json:"id" gorm:"autoIncrement;primaryKey"`
	ApplicationID    uint        `json:"applicationId" gorm:"not null"`                     // 所属应用（同租户，Service 层校验）
	Code             string      `json:"code" gorm:"size:64;not null"`                      // form_ 前缀稳定公开编码（路由/API 使用）
	Name             string      `json:"name" gorm:"size:128;not null"`                     // 表单名称（不进入协议 content）
	FormType         FormType    `json:"formType" gorm:"size:16;not null;default:standard"` // 表单类型（ADR-011 起可经动作切换）
	DraftContent     JSONContent `json:"draft" gorm:"type:jsonb;not null"`
	DraftRevision    int64       `json:"draftRevision" gorm:"not null;default:1"`   // 草稿乐观锁：保存条件递增
	ProtocolVersion  int         `json:"protocolVersion" gorm:"not null;default:1"` // 协议版本外部承载（文档内无版本字段）
	LatestVersionID  *uint       `json:"latestVersionId"`                           // 最新发布版本；NULL=从未发布
	PublishedVersion int         `json:"publishedVersion" gorm:"not null;default:0"`
	CreatorMemberID  uint        `json:"creatorMemberId" gorm:"not null"`

	kernel.TenantBaseModel
}

func (*Form) TableName() string { return "forms" }

// FormVersion 不可变发布快照：发布事务内一次写入，之后不存在更新路径；
// schema_revision 即行 id（出网字符串），提交校验与记录归属的双口令之一。
type FormVersion struct {
	ID                  uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	FormID              uint            `json:"formId" gorm:"not null"`
	VersionNo           int             `json:"versionNo" gorm:"not null"` // 表单内递增发布号 1,2,3…
	SchemaRevision      int64           `json:"schemaRevision" gorm:"not null;default:0"`
	Content             JSONContent     `json:"content" gorm:"type:jsonb;not null"`
	FieldKeys           JSONContent     `json:"fieldKeys" gorm:"type:jsonb;not null"` // 顶层字段键有序数组（提交未知键快速拒绝）
	ProtocolVersion     int             `json:"protocolVersion" gorm:"not null;default:1"`
	PublishedByMemberID uint            `json:"publishedByMemberId" gorm:"not null"`
	PublishedAt         kernel.JSONTime `json:"publishedAt"`

	TenantID  uint            `json:"tenantId" gorm:"index;not null;default:1"`
	CreatedAt kernel.JSONTime `json:"createdAt"`
}

func (*FormVersion) TableName() string { return "form_versions" }

// FormRecord 记录提交：追加写，values 为服务端按发布快照校验通过后的值
// （键=widgetName）；form_version_id 固定受理时所依据的版本（历史版本合法）。
type FormRecord struct {
	ID                  uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	FormID              uint            `json:"formId" gorm:"not null"`
	FormVersionID       uint            `json:"formVersionId" gorm:"not null"`
	Values              JSONContent     `json:"values" gorm:"type:jsonb;not null"`
	SubmittedByMemberID uint            `json:"submittedByMemberId" gorm:"not null"`
	SubmittedAt         kernel.JSONTime `json:"submittedAt"`

	TenantID  uint            `json:"tenantId" gorm:"index;not null;default:1"`
	CreatedAt kernel.JSONTime `json:"createdAt"`
}

func (*FormRecord) TableName() string { return "form_records" }

// JSONContent JSONB 原文载体：保存协议要求「未编辑属性不丢失」，因此草稿/快照
// 一律原样字节存取（校验在 Service 层完成），不经 map 往返避免键序与空值失真。
type JSONContent json.RawMessage

// Value 实现 driver.Valuer：空值落 '{}'，其余以字符串形态交给 pgx 写入 jsonb 列
func (j JSONContent) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	return string(j), nil
}

// Scan 实现 sql.Scanner：接受 pgx 返回的 []byte / string
func (j *JSONContent) Scan(value interface{}) error {
	switch data := value.(type) {
	case nil:
		*j = JSONContent("null")
	case []byte:
		*j = append((*j)[:0], data...)
	case string:
		*j = JSONContent(data)
	default:
		return fmt.Errorf("form: cannot scan JSONContent from %T", value)
	}
	return nil
}

// MarshalJSON 原样出网（json.RawMessage 语义，避免二次转义）
func (j JSONContent) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON 原样收网
func (j *JSONContent) UnmarshalJSON(data []byte) error {
	*j = append((*j)[:0], data...)
	return nil
}
