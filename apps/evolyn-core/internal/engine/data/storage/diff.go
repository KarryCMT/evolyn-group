// Diff 与 Plan：发布时由「上一个已应用物理模型」与「新发布快照目标模型」
// 产生结构计划（方案 §6/§8）。首期允许的操作封闭为：首次建父/子表、加
// 父/子字段列、弃用列元数据标记；禁止自动删列/删索引/窄化类型/不可逆数据
// 修复——类型变化直接以稳定错误拒绝（FORM_STORAGE_TYPE_CHANGE_UNSUPPORTED）。
package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// PlanActionKind DDL 计划动作种类（封闭枚举）。
type PlanActionKind string

const (
	// ActionCreateParent 首次创建父表（含预置系统列、RLS 与预置索引）。
	ActionCreateParent PlanActionKind = "create_parent"
	// ActionAddParentColumn 父表加用户列（用户列始终可空，required 只是提交
	// 校验语义，绝不转 SQL NOT NULL——新增必填字段会遇到历史 NULL 行）。
	ActionAddParentColumn PlanActionKind = "add_parent_column"
	// ActionCreateChild 首次创建子表单明细表（含预置列、RLS 与父记录索引）。
	ActionCreateChild PlanActionKind = "create_child"
	// ActionAddChildColumn 子表加用户列。
	ActionAddChildColumn PlanActionKind = "add_child_column"
)

// PlanAction 单条结构动作；执行层据此翻译受控 DDL，携带物即白名单产物。
type PlanAction struct {
	Kind    PlanActionKind  `json:"kind"`
	Table   string          `json:"table"`
	Column  *ColumnSpec     `json:"column,omitempty"`
	Child   *ChildTableSpec `json:"child,omitempty"`
	Comment string          `json:"comment,omitempty"`
}

// Plan 结构计划：Empty=true 表示无任何物理变更（同步发布路径）。
type Plan struct {
	Actions  []PlanAction `json:"actions"`
	Checksum string       `json:"checksum"`
	Empty    bool         `json:"empty"`
}

// ErrTypeChange 字段类型变化（fieldId 相同而值语义/列类型不同），发布拒绝。
type ErrTypeChange struct {
	FieldID string
	Detail  string
}

func (e *ErrTypeChange) Error() string {
	return fmt.Sprintf("字段 %s 的物理类型不可变更：%s", e.FieldID, e.Detail)
}

// DiffResult Diff 产物：可直接执行的 Plan 与合并弃用列后的「完整目标模型」
// （= 新快照字段 ∪ 已应用字段的并集，弃用列保留元数据与物理列）。
type DiffResult struct {
	Plan         Plan
	MergedModel  StorageModel
	TypeConflict *ErrTypeChange
	// AddedColumns / DeprecatedColumns 供审计与出网诊断（widgetName 列表）。
	AddedColumns      []string
	DeprecatedColumns []string
}

// Diff 对比已应用模型（applied 为 nil 表示首次发布）与目标模型（target，
// 由新发布快照推导、不含弃用列）。返回执行计划与合并模型；类型变化不返回
// error 而是携带 TypeConflict，由调用方映射稳定错误码。
func Diff(applied, target *StorageModel) (*DiffResult, error) {
	if err := target.Validate(); err != nil {
		return nil, fmt.Errorf("目标模型非法: %w", err)
	}
	result := &DiffResult{MergedModel: StorageModel{TableName: target.TableName}}

	// 合并基准：目标字段按 fieldId 索引（目标内 fieldId 已由 Validate 保证唯一）。
	targetByID := make(map[string]ColumnSpec, len(target.Columns))
	for _, column := range target.Columns {
		targetByID[column.FieldID] = column
	}

	if applied != nil {
		if err := applied.Validate(); err != nil {
			return nil, fmt.Errorf("已应用模型非法: %w", err)
		}
		if applied.TableName != target.TableName {
			return nil, fmt.Errorf("表名不可变更: %s -> %s", applied.TableName, target.TableName)
		}
		// 已应用列：仍在目标中 → 校验类型一致；不在目标中 → 保留为弃用列。
		for _, column := range applied.Columns {
			next, exists := targetByID[column.FieldID]
			if exists {
				if next.Kind != column.Kind || next.Type != column.Type {
					result.TypeConflict = &ErrTypeChange{
						FieldID: column.FieldID,
						Detail:  fmt.Sprintf("%s/%s -> %s/%s", column.Kind, column.Type, next.Kind, next.Type),
					}
					return result, nil
				}
				continue
			}
			deprecated := column
			deprecated.Deprecated = true
			result.MergedModel.Columns = append(result.MergedModel.Columns, deprecated)
			result.DeprecatedColumns = append(result.DeprecatedColumns, column.WidgetName)
		}
	}
	// 目标列全部进入合并模型（按 fieldId 稳定排序，弃用列与目标列合并后统一排序）。
	result.MergedModel.Columns = append(result.MergedModel.Columns, target.Columns...)
	sortColumnsByKey(result.MergedModel.Columns)

	// 子表合并：同 fieldId 子表校验既有列类型；既有列消失 → 弃用标记。
	appliedChildren := make(map[string]ChildTableSpec)
	if applied != nil {
		for _, child := range applied.Children {
			appliedChildren[child.FieldID] = child
		}
	}
	mergedChildren, err := mergeChildTables(appliedChildren, target.Children, result)
	if err != nil {
		return nil, err
	}
	if result.TypeConflict != nil {
		return result, nil //nolint:nilerr // 类型冲突经 result.TypeConflict 传递，非执行错误
	}
	result.MergedModel.Children = mergedChildren
	if err := result.MergedModel.Validate(); err != nil {
		return nil, fmt.Errorf("合并模型非法: %w", err)
	}

	// 生成动作：首次发布建父表；否则按合并模型与已应用模型的差集加列。
	if applied == nil {
		result.Plan.Actions = append(result.Plan.Actions, PlanAction{
			Kind: ActionCreateParent, Table: target.TableName,
			Comment: "首次创建物理父表",
		})
		result.AddedColumns = append(result.AddedColumns, widgetNames(target.Columns)...)
	} else {
		appliedByID := make(map[string]ColumnSpec, len(applied.Columns))
		for _, column := range applied.Columns {
			appliedByID[column.FieldID] = column
		}
		for _, column := range result.MergedModel.Columns {
			if _, exists := appliedByID[column.FieldID]; exists {
				continue
			}
			spec := column
			result.Plan.Actions = append(result.Plan.Actions, PlanAction{
				Kind: ActionAddParentColumn, Table: target.TableName, Column: &spec,
				Comment: "新增字段列",
			})
			result.AddedColumns = append(result.AddedColumns, column.WidgetName)
		}
	}
	// 子表动作：新子表整表创建；既有子表差集加列。
	for _, child := range result.MergedModel.Children {
		previous, exists := appliedChildren[child.FieldID]
		if applied == nil || !exists {
			spec := child
			result.Plan.Actions = append(result.Plan.Actions, PlanAction{
				Kind: ActionCreateChild, Table: child.TableName, Child: &spec,
				Comment: "首次创建子表单明细表",
			})
			continue
		}
		previousChildIDs := make(map[string]bool, len(previous.Columns))
		for _, column := range previous.Columns {
			previousChildIDs[column.FieldID] = true
		}
		for _, column := range child.Columns {
			if previousChildIDs[column.FieldID] {
				continue
			}
			spec := column
			result.Plan.Actions = append(result.Plan.Actions, PlanAction{
				Kind: ActionAddChildColumn, Table: child.TableName, Column: &spec,
				Comment: "子表新增字段列",
			})
		}
	}

	result.Plan.Empty = len(result.Plan.Actions) == 0
	checksum, err := ModelChecksum(&result.MergedModel)
	if err != nil {
		return nil, err
	}
	result.Plan.Checksum = checksum
	return result, nil
}

// mergeChildTables 子表合并：目标子表继承/校验既有列，既有列消失 → 弃用
// 标记；已应用但不在目标的子表整体弃用（物理表保留，仅元数据标记）。
// 类型冲突经 result.TypeConflict 返回（调用方映射稳定错误码）。
func mergeChildTables(appliedChildren map[string]ChildTableSpec, targetChildren []ChildTableSpec, result *DiffResult) ([]ChildTableSpec, error) {
	merged := make([]ChildTableSpec, 0, len(targetChildren))
	for _, child := range targetChildren {
		previous, exists := appliedChildren[child.FieldID]
		if exists {
			if previous.TableName != child.TableName {
				return nil, fmt.Errorf("子表名不可变更: %s -> %s", previous.TableName, child.TableName)
			}
			currentByID := make(map[string]ColumnSpec, len(child.Columns))
			for _, column := range child.Columns {
				currentByID[column.FieldID] = column
			}
			for _, column := range previous.Columns {
				if next, ok := currentByID[column.FieldID]; ok {
					if next.Kind != column.Kind || next.Type != column.Type {
						result.TypeConflict = &ErrTypeChange{
							FieldID: child.FieldID + ":" + column.FieldID,
							Detail:  fmt.Sprintf("子表列 %s/%s -> %s/%s", column.Kind, column.Type, next.Kind, next.Type),
						}
						return nil, nil
					}
				} else {
					deprecated := column
					deprecated.Deprecated = true
					child.Columns = append(child.Columns, deprecated)
				}
			}
			sortColumnsByKey(child.Columns)
		}
		merged = append(merged, child)
	}
	targetChildIDs := make(map[string]bool, len(targetChildren))
	for _, child := range targetChildren {
		targetChildIDs[child.FieldID] = true
	}
	for fieldID, child := range appliedChildren {
		if !targetChildIDs[fieldID] {
			deprecated := child
			deprecated.Deprecated = true
			merged = append(merged, deprecated)
			result.DeprecatedColumns = append(result.DeprecatedColumns, child.WidgetName)
		}
	}
	return merged, nil
}

// ModelChecksum 模型规范化序列化的 sha256（字段序已排序，序列化确定）。
// 用于 DDL Worker 领取后的防篡改复核与审计留痕。
func ModelChecksum(model *StorageModel) (string, error) {
	if model == nil {
		return "", fmt.Errorf("model is nil")
	}
	sorted := &StorageModel{TableName: model.TableName}
	sorted.Columns = append([]ColumnSpec(nil), model.Columns...)
	sorted.Children = append([]ChildTableSpec(nil), model.Children...)
	sortColumnsByKey(sorted.Columns)
	for i := range sorted.Children {
		sortColumnsByKey(sorted.Children[i].Columns)
	}
	encoded, err := json.Marshal(sorted)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func widgetNames(columns []ColumnSpec) []string {
	names := make([]string, 0, len(columns))
	for _, column := range columns {
		names = append(names, column.WidgetName)
	}
	return names
}
