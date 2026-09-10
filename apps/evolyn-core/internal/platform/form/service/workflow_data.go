// 记录数据窄端口（流程引擎 Phase 3，ADR-012 第 15 章）。
//
// Workflow 禁止直接 UPDATE tn_form_records（第 15.1 章铁律）：审批编辑与
// 表达式取数一律经本端口由表单域完成——合并结果按记录绑定的发布快照
// 整体终审（ValidateRecordValues），校验失败整体报错（同事务回滚），
// 与提交记录共用同一套校验与错误协议（FORM_RECORD_INVALID + fieldErrors）。
// 物理表存储方案 §9.2 起，physical 记录的读写经 PhysicalRecordValueStore
// 分派：values 只从物理父/子表进出，信封 values 保持 NULL。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/httpx"
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

// WorkflowProjection 更新载荷（物理表存储方案 §10.2）：状态直接复制流程
// 实例状态枚举，不自行重定义业务状态；普通表单不使用本端口。
type WorkflowProjection struct {
	InstanceNo string
	Status     string
	UpdatedAt  time.Time
}

// WorkflowProjectionUpdater 流程投影窄端口：流程平台适配层在实例状态变更
// 的同一事务内调用；流程内核不得直接更新 tn_form_records 或任何 tn_fd_* 表。
type WorkflowProjectionUpdater interface {
	UpdateWorkflowProjection(ctx context.Context, recordID uint, projection WorkflowProjection) error
}

// recordWriteOperator 记录写回操作人（随 ctx 传播；workflow 侧审批编辑/
// 发起人修改入口经 WithRecordWriteOperator 注入）。
type recordWriteOperator struct {
	MemberID uint
	Name     string
}

type recordWriteOperatorKey struct{}

// WithRecordWriteOperator 注入记录写回操作人（000072）：form 域在写回路径
// 读取并随 updated_at 同语句刷新信封 updated_by_* 双列。未注入（系统自动
// 路径）时只推进时间、最后写人人保持原值。
func WithRecordWriteOperator(ctx context.Context, memberID uint, name string) context.Context {
	return context.WithValue(ctx, recordWriteOperatorKey{}, recordWriteOperator{MemberID: memberID, Name: name})
}

// recordWriteOperatorFromContext 读取写回操作人；MemberID 为 0 视同未注入。
func recordWriteOperatorFromContext(ctx context.Context) (uint, string) {
	if op, ok := ctx.Value(recordWriteOperatorKey{}).(recordWriteOperator); ok && op.MemberID != 0 {
		return op.MemberID, op.Name
	}
	return 0, ""
}

// RecordData 实现 WorkflowRecordStore：physical 记录读物理行（信封 values
// 恒 NULL），legacy 记录读 values JSONB。
func (s *formService) RecordData(ctx context.Context, recordID uint) (uint, uint, map[string]any, error) {
	record, err := s.records.GetByID(ctx, recordID)
	if err != nil {
		return 0, 0, nil, err
	}
	values := make(map[string]any)
	if len(record.Values) > 0 && string(record.Values) != "null" {
		if err := json.Unmarshal([]byte(record.Values), &values); err != nil {
			return 0, 0, nil, fmt.Errorf("record %d values decode: %w", recordID, err)
		}
		return record.FormID, record.FormVersionID, values, nil
	}
	// values 为空 = physical 记录（新记录不写 JSONB）：经物理行读取
	pc, err := s.resolvePhysicalContext(ctx, &model.Form{ID: record.FormID})
	if err != nil {
		return 0, 0, nil, err
	}
	if pc != nil {
		physicalValues, err := s.readPhysicalValues(ctx, pc, recordID)
		if err != nil {
			return 0, 0, nil, err
		}
		return record.FormID, record.FormVersionID, physicalValues, nil
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

	// 基线分派：physical 记录以物理行为底，legacy 以 values JSONB 为底。
	var baseline map[string]any
	var physical *physicalWriteContext
	if len(record.Values) > 0 && string(record.Values) != "null" {
		baseline = make(map[string]any)
		if err := json.Unmarshal([]byte(record.Values), &baseline); err != nil {
			return fmt.Errorf("record %d values decode: %w", recordID, err)
		}
	} else {
		physical, err = s.resolvePhysicalContext(ctx, &model.Form{ID: record.FormID})
		if err != nil {
			return err
		}
		if physical != nil {
			baseline, err = s.readPhysicalValues(ctx, physical, recordID)
			if err != nil {
				return err
			}
		} else {
			baseline = map[string]any{}
		}
	}

	// 合并：以既有值为底，patch 覆盖（显式 null 即清空语义，与字典 1.2 一致）
	merged := make(map[string]any, len(baseline)+len(patch))
	for key, value := range baseline {
		merged[key] = value
	}
	for key, value := range patch {
		merged[key] = value
	}

	// 复用提交校验：把合并结果回转 RawMessage 交给同一套快照终审。
	// v6 起快照经 ResolveMergedRecordValues 按记录基线执行不可见字段赋值
	// 策略（§6.2：两条路径共用同一值决议器）；v6 前快照保持旧静态可见语义。
	var cleaned map[string]any
	if version.ProtocolVersion >= model.InvisibleValuePolicyVersion {
		cleaned, fieldErrors := ResolveMergedRecordValues(content, merged, baseline)
		if len(fieldErrors) > 0 {
			return fmt.Errorf("record %d merge: %w", recordID,
				apperrors.ErrRecordInvalid.WithData(map[string]any{"fieldErrors": fieldErrors}))
		}
		return s.persistResolvedValues(ctx, physical, recordID, cleaned)
	}
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
	return s.persistResolvedValues(ctx, physical, recordID, cleaned)
}

// persistResolvedValues 落库决议后的记录值：physical 记录整体替换物理父行
// 值列与子行并刷新信封写回元数据（updated_at + 最后写人人，000067/000072）；
// legacy 记录替换 values JSONB（同语句刷 updated_at/updated_by_*）。
func (s *formService) persistResolvedValues(ctx context.Context, physical *physicalWriteContext, recordID uint, cleaned map[string]any) error {
	operatorID, operatorName := recordWriteOperatorFromContext(ctx)
	if physical != nil {
		if err := s.replacePhysicalValues(ctx, physical, recordID, cleaned); err != nil {
			return err
		}
		return s.records.TouchWriteMeta(ctx, recordID, operatorID, operatorName)
	}
	valuesJSON, err := json.Marshal(cleaned)
	if err != nil {
		return err
	}
	return s.records.UpdateValues(ctx, recordID, model.JSONContent(valuesJSON), operatorID, operatorName)
}

// UpdateWorkflowProjection 实现 WorkflowProjectionUpdater（物理表存储方案
// §10.2）：同事务更新信封投影列（单号/状态/时间）与物理表投影列。状态为
// 冻结枚举，非法值以 FORM_WORKFLOW_PROJECTION_INVALID 拒绝。
func (s *formService) UpdateWorkflowProjection(ctx context.Context, recordID uint, projection WorkflowProjection) error {
	if !workflowStatusValuePattern.MatchString(projection.Status) {
		return httpx.Wrap(apperrors.ErrWorkflowProjectionInvalid,
			fmt.Errorf("invalid projection status %q for record %d", projection.Status, recordID))
	}
	record, err := s.records.GetByID(ctx, recordID)
	if err != nil {
		return err
	}
	pc, err := s.resolvePhysicalContext(ctx, &model.Form{ID: record.FormID})
	if err != nil {
		return err
	}
	if err := s.records.SetWorkflowProjection(ctx, recordID, projection.Status, projection.UpdatedAt); err != nil {
		return err
	}
	if projection.InstanceNo != "" && record.WorkflowInstanceNo == "" {
		if err := s.records.SetWorkflowInstanceNo(ctx, recordID, projection.InstanceNo); err != nil {
			return err
		}
	}
	// 物理表三列齐写：单号跟随信封（record 已复核为空才由上方写入，此处
	// 传最终单号——重复投影同值幂等）
	instanceNo := record.WorkflowInstanceNo
	if projection.InstanceNo != "" {
		instanceNo = projection.InstanceNo
	}
	return s.setWorkflowProjection(ctx, pc, recordID, instanceNo, projection.Status, projection.UpdatedAt)
}
