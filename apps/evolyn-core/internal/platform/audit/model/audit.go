package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	kernel "evolyn/internal/model"
)

// AuditLog 业务审计日志（FIX-013）：记录「谁在什么租户对什么资源做了什么修改」。
// 追加写流水，不做软删/更新；技术链路日志（log/trace/monitor）不在本表范围
type AuditLog struct {
	ID           uint     `json:"id" gorm:"autoIncrement;primaryKey"`
	TenantID     uint     `json:"tenantId" gorm:"index;not null;default:0"` // 0 = 平台级操作（运营域）
	AccountID    uint     `json:"accountId" gorm:"index;default:0"`         // 操作者账号（登录身份）
	MemberID     uint     `json:"memberId" gorm:"index;default:0"`          // 操作者成员（租户内身份）
	Module       string   `json:"module" gorm:"size:64;index;not null"`     // 业务域：tenant/iam/...
	Action       string   `json:"action" gorm:"size:64;index;not null"`     // 动作：create/update/delete/bind/...
	ResourceType string   `json:"resourceType" gorm:"size:64;index;not null"`
	ResourceID   string   `json:"resourceId" gorm:"size:128;index"`
	RequestID    string   `json:"requestId" gorm:"size:64"`
	IP           string   `json:"ip" gorm:"size:64"`
	UserAgent    string   `json:"userAgent" gorm:"size:256"`
	BeforeData   JSONText `json:"beforeData" gorm:"type:jsonb"` // 变更前快照，可空
	AfterData    JSONText `json:"afterData" gorm:"type:jsonb"`  // 变更后快照，可空
	// 企业日志展示投影（000036）：事件码/分类由审计服务按事件注册表生成，
	// 快照与摘要写时固化；存量历史行为空串，展示降级为「历史操作记录」。
	// before/after 仍是受控内部审计快照，不直接出网
	EventCode          string          `json:"eventCode" gorm:"size:100;not null;default:''"`          // 稳定事件码：模块.资源类型.动作
	CategoryCode       string          `json:"categoryCode" gorm:"size:64;not null;default:''"`        // 稳定日志范围码（见 service 分类常量）
	ActorNameSnapshot  string          `json:"actorNameSnapshot" gorm:"size:128;not null;default:''"`  // 操作人显示名快照
	TargetNameSnapshot string          `json:"targetNameSnapshot" gorm:"size:256;not null;default:''"` // 目标资源展示名快照
	Summary            string          `json:"summary" gorm:"size:1000;not null;default:''"`           // 服务端生成且脱敏的操作详情
	CreatedAt          kernel.JSONTime `json:"createdAt"`
}

func (*AuditLog) TableName() string {
	return "tn_audit_logs"
}

// JSONText JSONB 列载体：空串落 NULL，序列化失败在写入前拦截。
// 与 TenantConfig 等既有 JSONB 载手的 Value/Scan 口径一致
type JSONText string

func (j JSONText) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

func (j *JSONText) Scan(v interface{}) error {
	if v == nil {
		*j = ""
		return nil
	}
	switch data := v.(type) {
	case []byte:
		*j = JSONText(data)
	case string:
		*j = JSONText(data)
	default:
		return fmt.Errorf("cannot scan %T into audit.JSONText", v)
	}
	return nil
}

// MarshalJSONText 任意对象序列化为 JSONText；nil 返回空串（落 NULL）
func MarshalJSONText(v interface{}) (JSONText, error) {
	if v == nil {
		return "", nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return JSONText(data), nil
}
