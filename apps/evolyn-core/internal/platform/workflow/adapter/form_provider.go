// 业务数据窄端口适配器（ADR-012 Phase 3）：实现引擎
// provider.BusinessDataProvider，桥接表单域 WorkflowRecordStore 窄端口。
//
// 本适配器是第 15.1 章「Workflow 不直接写表单数据」铁律的唯一豁口：
// 内核只感知 BusinessRef 与 map 值，form_records 的读写、发布快照终审
// 全部发生在表单域服务内。
package adapter

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"evolyn/internal/engine/workflow/provider"
	wferrors "evolyn/internal/platform/workflow"

	"gorm.io/gorm"
)

// FormRecordStore 表单域记录数据窄端口（由 form service 实现，装配层注入）。
type FormRecordStore interface {
	RecordData(ctx context.Context, recordID uint) (formID, formVersionID uint, values map[string]any, err error)
	UpdateRecordValues(ctx context.Context, recordID uint, patch map[string]any) error
}

// FormDataProvider 引擎业务数据窄端口的表单域适配。
type FormDataProvider struct {
	store FormRecordStore
}

// NewFormDataProvider 构造业务数据适配器。
func NewFormDataProvider(store FormRecordStore) *FormDataProvider {
	return &FormDataProvider{store: store}
}

// GetData 读取实例绑定的业务数据（form.* 表达式数据源）。
// BusinessID 必须为表单记录 ID（表单型流程的发起约定），并复核记录的
// form/form_version 绑定与实例冻结值一致，防止业务键漂移读到他表数据。
func (p *FormDataProvider) GetData(ctx context.Context, ref provider.BusinessRef) (map[string]any, error) {
	if p.store == nil {
		return nil, wferrors.ErrFormVersionInvalid
	}
	recordID, err := parseRecordID(ref.BusinessID)
	if err != nil {
		return nil, err
	}
	formID, formVersionID, values, err := p.store.RecordData(ctx, recordID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, wferrors.ErrFormVersionInvalid
		}
		return nil, err
	}
	if err := ensureBinding(ref, formID, formVersionID); err != nil {
		return nil, err
	}
	return values, nil
}

// UpdateData 审批编辑写回：先复核绑定一致性，再交由表单域按冻结快照
// 整体校验合并结果（校验失败 FORM_RECORD_INVALID 随事务整体回滚）。
func (p *FormDataProvider) UpdateData(ctx context.Context, ref provider.BusinessRef, values map[string]any) error {
	if p.store == nil || len(values) == 0 {
		return nil
	}
	recordID, err := parseRecordID(ref.BusinessID)
	if err != nil {
		return err
	}
	formID, formVersionID, _, err := p.store.RecordData(ctx, recordID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return wferrors.ErrFormVersionInvalid
		}
		return err
	}
	if err := ensureBinding(ref, formID, formVersionID); err != nil {
		return err
	}
	return p.store.UpdateRecordValues(ctx, recordID, values)
}

// parseRecordID 业务键 → 记录 ID（非数字即表单绑定无效）。
func parseRecordID(businessID string) (uint, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(businessID), 10, 64)
	if err != nil || id == 0 {
		return 0, wferrors.ErrFormVersionInvalid
	}
	return uint(id), nil
}

// ensureBinding 记录归属与实例冻结绑定一致（双版本冻结，第 8.2 章）。
func ensureBinding(ref provider.BusinessRef, formID, formVersionID uint) error {
	if ref.FormID != 0 && formID != ref.FormID {
		return wferrors.ErrFormVersionInvalid
	}
	if ref.FormVersionID != 0 && formVersionID != ref.FormVersionID {
		return wferrors.ErrFormVersionInvalid
	}
	return nil
}

// EnsureInterfaces 编译期端口契约自检。
var _ provider.BusinessDataProvider = (*FormDataProvider)(nil)
