// 记录数据窄端口（流程引擎 Phase 3，ADR-012 第 15 章）。
//
// Workflow 禁止直接 UPDATE tn_form_records（第 15.1 章铁律）：审批编辑与
// 表达式取数一律经本端口由表单域完成——合并结果按记录绑定的发布快照
// 整体终审（ValidateRecordValues），校验失败整体报错（同事务回滚），
// 与提交记录共用同一套校验与错误协议（FORM_RECORD_INVALID + fieldErrors）。
package service

import (
	"context"
	"encoding/json"
	"fmt"

	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
)

// WorkflowRecordStore 表单记录数据窄端口：由 platform/workflow 适配器
// 实现 engine provider.BusinessDataProvider 时消费（装配层桥接）。
type WorkflowRecordStore interface {
	// RecordData 读取记录当前值与绑定关系（form_id / form_version_id），
	// 供流程引擎填充 WorkflowContext.form.* 表达式数据源。
	// 记录不存在返回 gorm.ErrRecordNotFound。
	RecordData(ctx context.Context, recordID uint) (formID, formVersionID uint, values map[string]any, err error)
	// UpdateRecordValues 审批编辑合并更新：patch 键=widgetName，值与既有
	// values 合并后按记录绑定快照整体校验（未知键/隐藏字段/类型范围/必填
	// 逐项复核）。注意「只传 patch 键」语义：未出现字段保持原值不清空，
	// 需要清空的字段显式传 null。
	UpdateRecordValues(ctx context.Context, recordID uint, patch map[string]any) error
}

// RecordData 实现 WorkflowRecordStore。
func (s *formService) RecordData(ctx context.Context, recordID uint) (uint, uint, map[string]any, error) {
	record, err := s.records.GetByID(ctx, recordID)
	if err != nil {
		return 0, 0, nil, err
	}
	values := make(map[string]any)
	if err := json.Unmarshal([]byte(record.Values), &values); err != nil {
		return 0, 0, nil, fmt.Errorf("record %d values decode: %w", recordID, err)
	}
	return record.FormID, record.FormVersionID, values, nil
}

// UpdateRecordValues 实现 WorkflowRecordStore。
func (s *formService) UpdateRecordValues(ctx context.Context, recordID uint, patch map[string]any) error {
	record, err := s.records.GetByID(ctx, recordID)
	if err != nil {
		return err
	}
	version, err := s.versions.GetByID(ctx, record.FormVersionID)
	if err != nil {
		return err
	}
	content := make(map[string]any)
	if err := json.Unmarshal([]byte(version.Content), &content); err != nil {
		return fmt.Errorf("record %d snapshot decode: %w", recordID, err)
	}

	// 合并：以既有值为底，patch 覆盖（显式 null 即清空语义，与字典 1.2 一致）
	merged := make(map[string]any)
	if err := json.Unmarshal([]byte(record.Values), &merged); err != nil {
		return fmt.Errorf("record %d values decode: %w", recordID, err)
	}
	for key, value := range patch {
		merged[key] = value
	}

	// 复用提交校验：把合并结果回转 RawMessage 交给同一套快照终审
	rawValues := make(map[string]json.RawMessage, len(merged))
	for key, value := range merged {
		raw, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("record %d value %s encode: %w", recordID, key, err)
		}
		rawValues[key] = raw
	}
	cleaned, fieldErrors := ValidateRecordValues(content, rawValues)
	if len(fieldErrors) > 0 {
		// 与提交记录同一套稳定码与回填协议：FORM_RECORD_INVALID + fieldErrors
		return fmt.Errorf("record %d merge: %w", recordID,
			apperrors.ErrRecordInvalid.WithData(map[string]any{"fieldErrors": fieldErrors}))
	}

	valuesJSON, err := json.Marshal(cleaned)
	if err != nil {
		return err
	}
	return s.records.UpdateValues(ctx, recordID, model.JSONContent(valuesJSON))
}
