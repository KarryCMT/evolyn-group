package repository

import (
	"context"
	"fmt"

	"evolyn/internal/infrastructure"
)

// allocateInstanceNumber 由数据库按租户/东八区日期原子分配，不读取主键或 MAX+1。
// 计数器与实例加入同一事务；请求幂等在引擎创建实例之前裁决。
func (r *runtimeInstances) allocateInstanceNumber(ctx context.Context, tenantID uint) (string, error) {
	if !infrastructure.InTransaction(ctx) {
		return "", fmt.Errorf("instance numbering requires transaction")
	}
	var allocated struct {
		NumberDate string
		LastValue  int64
	}
	err := infrastructure.ResolveDB(ctx, r.base).Raw(`INSERT INTO wf_instance_number_counters (tenant_id, number_date, last_value)
 VALUES (?, to_char(statement_timestamp() AT TIME ZONE 'Asia/Shanghai', 'YYYYMMDD'), 1)
 ON CONFLICT (tenant_id, number_date)
 DO UPDATE SET last_value = wf_instance_number_counters.last_value + 1
 RETURNING number_date, last_value`, tenantID).Scan(&allocated).Error
	if err != nil {
		return "", err
	}
	return formatInstanceNumber(allocated.NumberDate, allocated.LastValue), nil
}

// 六位是最小宽度，超过 999999 自动扩展，不截断或循环使用。
func formatInstanceNumber(day string, sequence int64) string {
	return fmt.Sprintf("WF-%s-%06d", day, sequence)
}
