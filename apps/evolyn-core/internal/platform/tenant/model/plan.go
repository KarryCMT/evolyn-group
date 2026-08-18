package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// 套餐档位（架构文档 26.6；对标简道云 vip.level 形态，键值可扩展）
const (
	PlanFree  = "free"  // 免费版
	PlanTrial = "trial" // 试用版
	PlanPro   = "pro"   // 付费专业版
)

// 配额键（26.6 首批五项；-1 表示不限量，0 表示禁用）
const (
	QuotaApps              = "apps"                // 应用数
	QuotaForms             = "forms"               // 表单数
	QuotaMembers           = "members"             // 成员数
	QuotaStorageGB         = "storage_gb"          // 附件存储容量（GB）
	QuotaWorkflowRunsMonth = "workflow_runs_month" // 月度流程发起量
)

// DefaultQuotas 各套餐默认配额；未知套餐按 free 兜底。
// 配额检查挂创建/上传路径属 P1-7 原计划，本批只落模型与读取
func DefaultQuotas(plan string) Quotas {
	switch plan {
	case PlanTrial:
		return Quotas{
			QuotaApps:              10,
			QuotaForms:             50,
			QuotaMembers:           30, // 对齐简道云试用版 users:30
			QuotaStorageGB:         5,
			QuotaWorkflowRunsMonth: 10000,
		}
	case PlanPro:
		return Quotas{
			QuotaApps:              -1,
			QuotaForms:             -1,
			QuotaMembers:           -1,
			QuotaStorageGB:         -1,
			QuotaWorkflowRunsMonth: -1,
		}
	default: // free
		return Quotas{
			QuotaApps:              3,
			QuotaForms:             10,
			QuotaMembers:           5,
			QuotaStorageGB:         1,
			QuotaWorkflowRunsMonth: 100,
		}
	}
}

// Quotas 租户配额覆盖（JSONB 键值）；空/缺键回落套餐默认值。
// 数值语义：-1 不限量、0 禁用、正数即上限
type Quotas map[string]int64

func (q Quotas) Value() (driver.Value, error) {
	if q == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(q)
}

func (q *Quotas) Scan(v interface{}) error {
	if v == nil {
		*q = Quotas{}
		return nil
	}
	switch data := v.(type) {
	case []byte:
		return json.Unmarshal(data, q)
	case string:
		return json.Unmarshal([]byte(data), q)
	default:
		return fmt.Errorf("cannot scan %T into Quotas", v)
	}
}

// Get 取某配额的生效值：覆盖值优先，缺键回落套餐默认，套餐默认也缺则回落 def
func (q Quotas) Get(plan, key string, def int64) int64 {
	if v, ok := q[key]; ok {
		return v
	}
	if v, ok := DefaultQuotas(plan)[key]; ok {
		return v
	}
	return def
}
