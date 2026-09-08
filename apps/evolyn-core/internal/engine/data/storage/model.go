// StorageModel：一次发布快照对应的完整物理结构描述（含历次已应用字段的
// 并集与弃用标记），以 JSONB 持久化于 tn_form_storage_schema_versions.model，
// 是动态表名/列名/类型/Index 的唯一服务端事实源。浏览器、Controller 与
// Repository 均不得在其外构造或猜测这些标识符。
package storage

import (
	"fmt"
	"sort"
)

// SystemPresetColumns 所有物理父表预置的流程查询投影列（方案 §5.3/§10）。
// 单号/状态/更新时间在信封表与物理表同名同语义；物理表侧仅作高频筛选与
// 导出投影，唯一性仍由 tn_form_records / wf_instance 保证。
var SystemPresetColumns = []PresetColumn{
	{Name: "tenant_id", Type: ColumnTypeBigint, Nullable: false},
	{Name: "record_id", Type: ColumnTypeBigint, Nullable: false},
	{Name: "workflow_instance_no", Type: ColumnTypeText, Nullable: true},
	{Name: "workflow_status", Type: ColumnTypeText, Nullable: false},
	{Name: "workflow_updated_at", Type: ColumnTypeTimestamp, Nullable: true},
}

// PresetColumn 物理表预置系统列（非用户字段，不经 fieldId 推导）。
type PresetColumn struct {
	Name     string     `json:"name"`
	Type     ColumnType `json:"type"`
	Nullable bool       `json:"nullable"`
}

// ChildPresetColumns 子表单明细表预置列（sort_order 是行顺序，INTEGER）。
var ChildPresetColumns = []PresetColumn{
	{Name: "tenant_id", Type: ColumnTypeBigint, Nullable: false},
	{Name: "row_id", Type: ColumnTypeBigint, Nullable: false},
	{Name: "parent_record_id", Type: ColumnTypeBigint, Nullable: false},
	{Name: "sort_order", Type: ColumnTypeInteger, Nullable: false},
	{Name: "row_revision", Type: ColumnTypeBigint, Nullable: false},
	{Name: "created_at", Type: ColumnTypeTimestamp, Nullable: false},
	{Name: "updated_at", Type: ColumnTypeTimestamp, Nullable: false},
}

// ColumnSpec 用户字段物理列（父表列与子表列同构；Deprecated 仅是模型层
// 标记——新版本不再展示/写入，物理列与历史值保留，清理属人工维护任务）。
type ColumnSpec struct {
	FieldID    string     `json:"fieldId"`
	WidgetName string     `json:"widgetName"`
	WidgetType string     `json:"widgetType"`
	Kind       FieldKind  `json:"kind"`
	Type       ColumnType `json:"type"`
	Deprecated bool       `json:"deprecated"`
}

// ColumnName 由不可变 fieldId 推导规范列名（f_<fieldId>，永不变更）。
func (c ColumnSpec) ColumnName() string { return "f_" + c.FieldID }

// ChildTableSpec 子表单物理子表：父记录的一对多明细（方案 §5.4）。列名由
// 子字段各自的 fieldId 推导，不与父表共享命名空间。
type ChildTableSpec struct {
	FieldID    string       `json:"fieldId"`
	WidgetName string       `json:"widgetName"`
	TableName  string       `json:"table"`
	Columns    []ColumnSpec `json:"columns"`
	Deprecated bool         `json:"deprecated"`
}

// StorageModel 物理存储模型：父表 + 子表集合。每次成功发布冻结一份完整
// 快照；表名自分配后永久不变。
type StorageModel struct {
	TableName string           `json:"table"`
	Columns   []ColumnSpec     `json:"columns"`
	Children  []ChildTableSpec `json:"children"`
}

// Validate 结构自检：表名/列名/fieldId 全部过白名单，列类型与值语义一致，
// fieldId 与列名在各自作用域内唯一。执行层（DDL/DML）消费模型前必须调用。
func (m *StorageModel) Validate() error {
	if m == nil {
		return fmt.Errorf("storage model is nil")
	}
	if err := ValidateDynamicTableName(m.TableName); err != nil {
		return err
	}
	if !IsParentTableName(m.TableName) {
		return fmt.Errorf("父表名 %q 必须以 tn_fd_ 开头", m.TableName)
	}
	fieldIDs := make(map[string]bool, len(m.Columns))
	widgetNames := make(map[string]bool, len(m.Columns))
	for _, column := range m.Columns {
		if err := ValidateFieldID(column.FieldID); err != nil {
			return err
		}
		if err := ValidateColumnName(column.ColumnName()); err != nil {
			return err
		}
		if fieldIDs[column.FieldID] {
			return fmt.Errorf("fieldId %q 在父表中重复", column.FieldID)
		}
		if widgetNames[column.WidgetName] {
			return fmt.Errorf("widgetName %q 在父表中重复", column.WidgetName)
		}
		fieldIDs[column.FieldID] = true
		widgetNames[column.WidgetName] = true
		if ColumnTypeOf(column.Kind) != column.Type {
			return fmt.Errorf("字段 %s 的类型 %s 与值语义 %s 不一致", column.WidgetName, column.Type, column.Kind)
		}
	}
	childIDs := make(map[string]bool, len(m.Children))
	for _, child := range m.Children {
		if err := ValidateFieldID(child.FieldID); err != nil {
			return err
		}
		if err := ValidateDynamicTableName(child.TableName); err != nil {
			return err
		}
		if !IsChildTableName(child.TableName) {
			return fmt.Errorf("子表名 %q 必须以 tn_fc_ 开头", child.TableName)
		}
		if childIDs[child.FieldID] {
			return fmt.Errorf("子表字段 fieldId %q 重复", child.FieldID)
		}
		childIDs[child.FieldID] = true
		if fieldIDs[child.FieldID] {
			return fmt.Errorf("子表 fieldId %q 与父表字段冲突", child.FieldID)
		}
		childColumnIDs := make(map[string]bool, len(child.Columns))
		for _, column := range child.Columns {
			if err := ValidateFieldID(column.FieldID); err != nil {
				return err
			}
			if err := ValidateColumnName(column.ColumnName()); err != nil {
				return err
			}
			if childColumnIDs[column.FieldID] {
				return fmt.Errorf("子表 %s 的 fieldId %q 重复", child.TableName, column.FieldID)
			}
			childColumnIDs[column.FieldID] = true
			if ColumnTypeOf(column.Kind) != column.Type {
				return fmt.Errorf("子表字段 %s 的类型 %s 与值语义 %s 不一致", column.WidgetName, column.Type, column.Kind)
			}
		}
	}
	return nil
}

// ColumnByFieldID 按 fieldId 定位父表列。
func (m *StorageModel) ColumnByFieldID(fieldID string) (ColumnSpec, bool) {
	for _, column := range m.Columns {
		if column.FieldID == fieldID {
			return column, true
		}
	}
	return ColumnSpec{}, false
}

// ChildByFieldID 按 fieldId 定位子表。
func (m *StorageModel) ChildByFieldID(fieldID string) (ChildTableSpec, bool) {
	for _, child := range m.Children {
		if child.FieldID == fieldID {
			return child, true
		}
	}
	return ChildTableSpec{}, false
}

// sortColumnsByKey 稳定排序（模型 JSON 序列化确定性的前提，checksum 依赖）。
func sortColumnsByKey(columns []ColumnSpec) {
	sort.Slice(columns, func(i, j int) bool { return columns[i].FieldID < columns[j].FieldID })
}
