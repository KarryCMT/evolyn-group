// Package model 流程引擎平台层持久化模型（ADR-012，Phase 1 Definition Engine）：
// wf_definition（DSL 草稿 + draft_revision 乐观锁）与 wf_definition_version
// （不可变发布快照）。DSL v1 协议类型唯一事实源在 internal/engine/workflow/model，
// 本包只承载 JSONB 持久化形态与出网 DTO。
package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	kernel "evolyn/internal/model"
)

// WfDefinition 流程定义（租户级资产，从属于租户而非单个应用；审批中心为
// Tenant 级平台能力，应用侧仅展示关联，V1.1 定版 §22）。
type WfDefinition struct {
	ID               uint       `json:"id" gorm:"autoIncrement;primaryKey"`
	Code             string     `json:"code" gorm:"size:64;not null"` // wf_ 前缀稳定公开编码（路由/API 使用）
	Name             string     `json:"name" gorm:"size:128;not null"`
	Description      string     `json:"description" gorm:"size:512;not null;default:''"`
	FormCode         string     `json:"formCode" gorm:"size:64;not null;default:''"` // 绑定表单公开编码（000060）：流程型表单的流程设计页定位口令；空串=独立定义
	DraftContent     DSLContent `json:"draft" gorm:"type:jsonb;not null"`
	DraftRevision    int64      `json:"draftRevision" gorm:"not null;default:1"` // 草稿乐观锁：保存条件递增
	LatestVersionID  *uint      `json:"latestVersionId"`                         // 最新发布版本；NULL=从未发布
	PublishedVersion int        `json:"publishedVersion" gorm:"not null;default:0"`
	CreatorMemberID  uint       `json:"creatorMemberId" gorm:"not null"`

	kernel.TenantBaseModel
}

func (*WfDefinition) TableName() string { return "wf_definition" }

// WfDefinitionVersion 不可变发布快照：发布事务内一次写入，之后不存在更新
// 路径；DSL 全文整体冻结（Node/Edge/Config 内嵌其中）。
type WfDefinitionVersion struct {
	ID                  uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	DefinitionID        uint            `json:"definitionId" gorm:"not null"`
	VersionNo           int             `json:"versionNo" gorm:"not null"` // 定义内递增发布号 1,2,3…
	DSLSnapshot         DSLContent      `json:"dsl" gorm:"type:jsonb;not null"`
	PublishedByMemberID uint            `json:"publishedByMemberId" gorm:"not null"`
	PublishedAt         kernel.JSONTime `json:"publishedAt"`

	TenantID  uint            `json:"tenantId" gorm:"index;not null;default:1"`
	CreatedAt kernel.JSONTime `json:"createdAt"`
}

func (*WfDefinitionVersion) TableName() string { return "wf_definition_version" }

// DSLContent DSL JSONB 原文载体：与表单域 JSONContent 同构——协议要求
// 「未编辑属性不丢失」，草稿/快照一律原样字节存取（校验在 Service 层经引擎
// 严格校验器完成），不经 map 往返避免键序与空值失真。
type DSLContent json.RawMessage

// Value 实现 driver.Valuer：空值落 '{}'，其余以字符串形态交给 pgx 写入 jsonb 列
func (j DSLContent) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	return string(j), nil
}

// Scan 实现 sql.Scanner：接受 pgx 返回的 []byte / string
func (j *DSLContent) Scan(value interface{}) error {
	switch data := value.(type) {
	case nil:
		*j = DSLContent("null")
	case []byte:
		*j = append((*j)[:0], data...)
	case string:
		*j = DSLContent(data)
	default:
		return fmt.Errorf("workflow: cannot scan DSLContent from %T", value)
	}
	return nil
}

// MarshalJSON 原样出网（json.RawMessage 语义，避免二次转义）
func (j DSLContent) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON 原样收网
func (j *DSLContent) UnmarshalJSON(data []byte) error {
	*j = append((*j)[:0], data...)
	return nil
}
