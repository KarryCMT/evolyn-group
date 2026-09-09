// RecordValueStore 与物理值路径（方案 §9）：Form Service 只依赖面向业务值
// 的窄接口——新记录由 PhysicalRecordValueStore（经 repository 的受控动态
// DML）承载；历史 JSONB 兼容读取保留在既有 values 路径，但绝不作为新记录
// 写入实现。提交终审、字段权限、值决议、审计与 WorkflowRecordStore 的语义
// 不在多个实现中复制：本文件只做「校验后的值 ↔ 物理行」的搬运与编解码。
package service

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"evolyn/internal/contextx"
	storagepkg "evolyn/internal/engine/data/storage"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/httpx"

	"gorm.io/gorm"
)

// RecordValueStore 业务值存取窄接口（方案 §9.1）：Create/Replace 的 values
// 键为 widgetName（提交终审后的干净值），实现负责按已应用物理模型落行。
// List 由列表管线（ListRecords）单独编译（谓词/权限/排序共享同一编译计划），
// 不在此接口内复制。
type RecordValueStore interface {
	Create(ctx context.Context, recordID uint, values map[string]any) error
	Read(ctx context.Context, recordID uint) (map[string]any, error)
	Replace(ctx context.Context, recordID uint, values map[string]any) error
}

// RecordWorkflowStatusNone 普通表单/未绑定流程的投影状态。
const RecordWorkflowStatusNone = "NONE"

// physicalWriteContext 单次物理写入的模型绑定快照。
type physicalWriteContext struct {
	binding *model.FormStorage
	model   *storagepkg.StorageModel
}

// resolvePhysicalContext 加载表单的物理存储绑定与已应用模型；无绑定返回
// nil（存量 JSONB 表单）。结构变更在途（PUBLISHING）/最近失败（FAILED）时
// 按已应用模型继续服务；仅从未应用过模型（首次发布 DDL 在途）才以
// FORM_STORAGE_NOT_READY 拒绝。
func (s *formService) resolvePhysicalContext(ctx context.Context, form *model.Form) (*physicalWriteContext, error) {
	binding, err := s.loadStorageByForm(ctx, form.ID)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, nil
	}
	if s.physical == nil || s.schemaVersions == nil {
		return nil, fmt.Errorf("physical storage pipeline is not configured")
	}
	// 结构变更在途（PUBLISHING）或最近发布失败（FAILED）时按已应用模型继续
	// 服务：物理表结构是历次已应用字段的并集，发布指针仍指向的旧快照字段
	// 集 ⊆ 已应用模型，读写以已应用模型为列映射即可安全放行——与 202 发布
	// 语义一致（DDL 完成前旧快照继续可运行）。仅从未应用过模型（首次发布
	// DDL 在途）才真正不可用，此时也不存在任何已发布运行时。
	applied, err := s.loadAppliedModel(ctx, binding)
	if err != nil {
		return nil, err
	}
	if applied == nil {
		return nil, httpx.Wrap(apperrors.ErrStorageNotReady,
			fmt.Errorf("form %s has no applied storage model", form.Code))
	}
	return &physicalWriteContext{binding: binding, model: applied}, nil
}

// writeColumns 返回值中出现的字段对应的物理列（含弃用列：旧版本快照仍
// 含已删字段，其提交与审批写回必须继续落列；新快照提交不含弃用字段，
// 未携带的新列/弃用列保持 NULL）。物理结构是历次已应用字段的并集。
func (pc *physicalWriteContext) writeColumns(values map[string]any) []storagepkg.ColumnSpec {
	columns := make([]storagepkg.ColumnSpec, 0, len(pc.model.Columns))
	for _, column := range pc.model.Columns {
		if _, present := values[column.WidgetName]; present {
			columns = append(columns, column)
		}
	}
	return columns
}

// createPhysicalValues 写入物理父行与子表行（提交路径；调用方保证已在
// TxManager 事务内且信封行已创建）。
func (s *formService) createPhysicalValues(ctx context.Context, pc *physicalWriteContext, recordID uint, values map[string]any) error {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("tenant context required")
	}
	if err := s.physical.InsertParentRow(ctx, pc.binding.PhysicalTable, tenantID, recordID, pc.writeColumns(values), values); err != nil {
		return err
	}
	return s.replacePhysicalChildren(ctx, pc, recordID, values)
}

// replacePhysicalChildren 按模型子表替换子行（集合替换语义：方案 §5.4）。
func (s *formService) replacePhysicalChildren(ctx context.Context, pc *physicalWriteContext, recordID uint, values map[string]any) error {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("tenant context required")
	}
	for _, child := range pc.model.Children {
		if child.Deprecated {
			continue
		}
		raw, present := values[child.WidgetName]
		if !present {
			continue
		}
		rows, err := childRowsOf(raw)
		if err != nil {
			return err
		}
		if err := s.physical.ReplaceChildRows(ctx, child.TableName, tenantID, recordID, child.Columns, rows); err != nil {
			return err
		}
	}
	return nil
}

// childRowsOf 子表单协议值（对象数组）→ 行 map 序列。
func childRowsOf(raw any) ([]map[string]any, error) {
	if raw == nil {
		return nil, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("子表单值必须是对象数组")
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("子表单行必须是对象")
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// readPhysicalValues 读取物理父行 + 子表行，组装为与 JSONB 记录同构的
// values map（widgetName 键；子表单键值为对象数组）。
func (s *formService) readPhysicalValues(ctx context.Context, pc *physicalWriteContext, recordID uint) (map[string]any, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	// 全列读取（含弃用列）：记录绑定历史快照时，已删字段仍参与表达式取数
	// 与审批写回的合并基线（方案 §6「旧版本仍可解释和提交」）。
	values, err := s.physical.ReadParentRow(ctx, pc.binding.PhysicalTable, tenantID, recordID, pc.model.Columns)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	if values == nil {
		return nil, nil
	}
	for _, child := range pc.model.Children {
		if child.Deprecated {
			continue
		}
		rows, err := s.physical.ReadChildRows(ctx, child.TableName, tenantID, recordID, child.Columns)
		if err != nil {
			return nil, err
		}
		items := make([]any, 0, len(rows))
		for _, row := range rows {
			items = append(items, row)
		}
		values[child.WidgetName] = items
	}
	return values, nil
}

// activeColumns 非弃用列全集（仅列表投影使用：最新快照不含弃用字段，
// 出网值域按其白名单过滤）。
func activeColumns(columns []storagepkg.ColumnSpec) []storagepkg.ColumnSpec {
	active := make([]storagepkg.ColumnSpec, 0, len(columns))
	for _, column := range columns {
		if column.Deprecated {
			continue
		}
		active = append(active, column)
	}
	return active
}

// replacePhysicalValues 流程写回路径：整体替换父行值列与子行（信封
// updated_at 由调用方刷新）。
func (s *formService) replacePhysicalValues(ctx context.Context, pc *physicalWriteContext, recordID uint, values map[string]any) error {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("tenant context required")
	}
	if err := s.physical.UpdateParentRowValues(ctx, pc.binding.PhysicalTable, tenantID, recordID, pc.writeColumns(values), values); err != nil {
		return err
	}
	return s.replacePhysicalChildren(ctx, pc, recordID, values)
}

// samePhysicalValues 幂等重放的物理值比较（信封 values 为 NULL，不能复用
// JSONB 字节比较）。
func samePhysicalValues(left, right map[string]any) bool {
	return reflect.DeepEqual(left, right)
}

// setWorkflowProjection 同事务更新物理表的流程投影（单号/状态/时间三列齐
// 写；信封侧单号仍走 SetWorkflowInstanceNo 的仅空可写约束，由调用方保证
// 先后语义——提交发起与实例状态变更共用）。
func (s *formService) setWorkflowProjection(ctx context.Context, pc *physicalWriteContext, recordID uint, instanceNo, status string, updatedAt time.Time) error {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("tenant context required")
	}
	if pc != nil {
		if err := s.physical.UpdateWorkflowProjection(ctx, pc.binding.PhysicalTable, tenantID, recordID, instanceNo, status, updatedAt); err != nil {
			return err
		}
	}
	return nil
}
