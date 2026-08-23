// Package model 应用管理域数据模型（M2-A，docs/低代码平台/应用管理/开发文档.md）
package model

import (
	"database/sql/driver"
	"encoding/json"

	kernel "evolyn/internal/model"
)

// 可见业务状态（§7.1）：仅表达可见状态；删除只写 deleted_at，不设 deleted 值
const (
	ApplicationStatusActive   = "active"
	ApplicationStatusArchived = "archived"
)

// 实例化状态（§7.1）：与 status 独立演进；M2-A 空白应用同步创建即 ready，
// pending/running/failed 留给 M2-C 异步模板安装
const (
	ProvisionStatusReady   = "ready"
	ProvisionStatusPending = "pending"
	ProvisionStatusRunning = "running"
	ProvisionStatusFailed  = "failed"
)

// 创建来源与安装渠道（§6.5）
const (
	SourceTypeBlank    = "blank"
	SourceTypeTemplate = "template"

	InstallChannelSelf = "self"
)

// Application 应用实例（租户内一级资源）：所有业务读写经 GORM 租户
// Callback 过滤（嵌入 TenantBaseModel）。owner/creator 引用租户成员，
// 同租户约束由 Service 层加载校验，禁止裸 ID 写入（§9.3）
type Application struct {
	ID                uint   `json:"id" gorm:"autoIncrement;primaryKey"`
	Code              string `json:"code" gorm:"size:64;not null"`  // 服务端生成，租户内唯一（部分唯一索引兜底）
	Name              string `json:"name" gorm:"size:128;not null"` // 展示名，允许同租户重名
	Icon              string `json:"icon" gorm:"size:32;not null;default:bookmark"`
	Color             string `json:"color" gorm:"size:32;not null;default:primary"`
	OwnerMemberID     uint   `json:"ownerMemberId" gorm:"not null"` // 应用所有者（租户成员）
	CreatorMemberID   uint   `json:"creatorMemberId" gorm:"not null"`
	SourceType        string `json:"sourceType" gorm:"size:16;not null"`            // blank / template
	Status            string `json:"status" gorm:"size:16;not null;default:active"` // active / archived
	ProvisionStatus   string `json:"provisionStatus" gorm:"size:16;not null;default:ready"`
	DefinitionVersion int    `json:"definitionVersion" gorm:"not null;default:1"` // 应用定义版本，非乐观锁
	MenuRevision      int64  `json:"menuRevision" gorm:"not null;default:1"`      // 菜单修订号：菜单写入乐观并发口令（000016），与发布演进独立
	SortOrder         int64  `json:"sortOrder" gorm:"not null;default:0"`
	Config            Config `json:"config" gorm:"type:jsonb;not null;default:'{}'"` // 小型应用级配置

	kernel.TenantBaseModel
}

func (*Application) TableName() string { return "applications" }

// Installation 安装记录：应用创建来源快照（§6.5），一应用一条、追加写。
// 刻意不嵌 TenantBaseModel（无软删/更新语义）：读取一律经 application_id
// 关联（应用本身已租户过滤），不做独立租户路径查询
type Installation struct {
	ID                  uint            `json:"id" gorm:"autoIncrement;primaryKey"`
	TenantID            uint            `json:"tenantId" gorm:"index;not null;default:1"`
	ApplicationID       uint            `json:"applicationId" gorm:"uniqueIndex;not null"`
	SourceType          string          `json:"sourceType" gorm:"size:16;not null"`
	TemplateID          *uint           `json:"templateId"`        // M2-A 恒 NULL
	TemplateVersionID   *uint           `json:"templateVersionId"` // M2-A 恒 NULL
	Channel             string          `json:"channel" gorm:"size:32;not null"`
	BlueprintChecksum   *string         `json:"blueprintChecksum" gorm:"size:64"` // M2-A 恒 NULL
	InstalledByMemberID uint            `json:"installedByMemberId" gorm:"not null"`
	InstalledAt         kernel.JSONTime `json:"installedAt"`
}

func (*Installation) TableName() string { return "application_installations" }

// Config 应用级小型配置 JSONB 载体：空值落 '{}'（列 NOT NULL DEFAULT '{}'
// 与迁移一致）。仅承载开关/偏好类小配置，严禁混入表单/页面/流程大定义（§6.2）
type Config map[string]any

func (c Config) Value() (driver.Value, error) {
	if len(c) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(c)
	return string(b), err
}

func (c *Config) Scan(value interface{}) error {
	switch data := value.(type) {
	case nil:
		*c = Config{}
	case []byte:
		return json.Unmarshal(data, c)
	case string:
		return json.Unmarshal([]byte(data), c)
	}
	return nil
}
