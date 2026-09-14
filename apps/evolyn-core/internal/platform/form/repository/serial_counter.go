package repository

import (
	"context"
	"fmt"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/form/model"

	"gorm.io/gorm"
)

type formSerialCounterRepository struct{ db *gorm.DB }

func NewFormSerialCounterRepository(db *gorm.DB) FormSerialCounterRepository {
	return &formSerialCounterRepository{db: db}
}

// Allocate 把 next_value 原子推进一位并返回推进前的值。UPSERT 的 RETURNING 避免
// 先读后写造成竞态；外层提交事务回滚时计数推进也会随之回滚。
func (r *formSerialCounterRepository) Allocate(
	ctx context.Context, tenantID, formID uint, fieldID, cycleKey string, initialValue int64,
) (int64, error) {
	if initialValue < 1 || fieldID == "" || cycleKey == "" {
		return 0, fmt.Errorf("invalid form serial counter scope")
	}
	var allocated int64
	err := infrastructure.ResolveDB(ctx, r.db).Raw(`
		INSERT INTO tn_form_serial_counters (tenant_id, form_id, field_id, cycle_key, next_value, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, LOCALTIMESTAMP, LOCALTIMESTAMP)
		ON CONFLICT (tenant_id, form_id, field_id, cycle_key)
		DO UPDATE SET next_value = tn_form_serial_counters.next_value + 1, updated_at = LOCALTIMESTAMP
		RETURNING next_value - 1`, tenantID, formID, fieldID, cycleKey, initialValue+1).Scan(&allocated).Error
	return allocated, err
}

func (r *formSerialCounterRepository) Migrate() error {
	return r.db.AutoMigrate(&model.FormSerialCounter{})
}
