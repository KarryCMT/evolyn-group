// 物理 DML 仓储（物理表存储方案 §9/§11）：动态 tn_fd_*/tn_fc_* 表的受控
// 原生 SQL。所有表名/列名来自已应用 StorageModel（服务端唯一事实源，经
// engine/data/storage 白名单校验），所有值一律绑定参数；语句经
// infrastructure.ResolveDB 加入 ctx 传播事务，保证与记录信封写入同一原子
// 边界（绝不另开 pgxpool 第二事务）。
package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"evolyn/internal/engine/data/storage"
	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/form/model"

	"gorm.io/gorm"
)

// PhysicalColumn 列表查询投影列（服务端从已应用模型生成）。
type PhysicalColumn struct {
	WidgetName string
	Column     storage.ColumnSpec
}

// PhysicalListBinding 物理模式列表查询绑定（表 + 投影列白名单）。
type PhysicalListBinding struct {
	TableName string
	Columns   []PhysicalColumn
	// Children 是当前已应用物理模型中的非弃用子表单；列表页在父记录分页后
	// 按每张子表批量回填明细，避免将一对多 JOIN 混入父记录分页语义。
	Children []storage.ChildTableSpec
}

// PhysicalRecordRow 物理行：信封列 + 按 widgetName 解码后的业务值。
type PhysicalRecordRow struct {
	Record model.FormRecord
	Values map[string]any
}

// PhysicalValueRepository 动态物理表 DML。写入方法必须在外层
// TxManager 事务内调用（调用方保证）。
type PhysicalValueRepository interface {
	// InsertParentRow 写入父表业务值行（record_id/tenant_id 复合绑定防跨租户）。
	InsertParentRow(ctx context.Context, tableName string, tenantID, recordID uint, columns []storage.ColumnSpec, values map[string]any) error
	// ReadParentRow 读父表行并解码为 widgetName 键值（NotFound 返回 gorm.ErrRecordNotFound）。
	ReadParentRow(ctx context.Context, tableName string, tenantID, recordID uint, columns []storage.ColumnSpec) (map[string]any, error)
	// UpdateParentRowValues 整体替换父表业务值（只 UPDATE 提交值包含的列与
	// NULL 收网列；审批写回路径使用）。
	UpdateParentRowValues(ctx context.Context, tableName string, tenantID, recordID uint, columns []storage.ColumnSpec, values map[string]any) error
	// ReplaceChildRows 子表单集合替换：同一事务内删除旧行后按客户端顺序批量
	// 写入新子行（方案 §5.4；父记录锁定由调用方在信封行上先行完成）。
	ReplaceChildRows(ctx context.Context, tableName string, tenantID, parentRecordID uint, columns []storage.ColumnSpec, rows []map[string]any) error
	// ReadChildRows 按父记录读取子行（sort_order 升序，返回每行 widgetName 键值）。
	ReadChildRows(ctx context.Context, tableName string, tenantID, parentRecordID uint, columns []storage.ColumnSpec) ([]map[string]any, error)
	// ReadChildRowsByParentIDs 按当前页父记录批量读取同一子表单的明细行，避免
	// 列表页对每条记录逐个查询子表造成 N×M 次往返。
	ReadChildRowsByParentIDs(ctx context.Context, tableName string, tenantID uint, parentRecordIDs []uint, columns []storage.ColumnSpec) (map[uint][]map[string]any, error)
	// DeleteParentRows/DeleteChildRows 在删除记录信封前清理物理值。动态表都以
	// tn_form_records 为外键父级，必须按子表 → 父表 → 信封的顺序执行。
	DeleteParentRows(ctx context.Context, tableName string, tenantID uint, recordIDs []uint) error
	DeleteChildRows(ctx context.Context, tableName string, tenantID uint, recordIDs []uint) error
	// UpdateWorkflowProjection 同事务更新物理表流程投影列（信封列由
	// FormRecordRepository 负责）。
	UpdateWorkflowProjection(ctx context.Context, tableName string, tenantID, recordID uint, instanceNo, status string, updatedAt time.Time) error
	// ListJoinControlled 物理模式列表：以信封表 r 为锚 JOIN 物理表 d，共用
	// 与 JSONB 模式完全相同的编译谓词/排序/分页。
	ListJoinControlled(ctx context.Context, params RecordListParams, binding PhysicalListBinding) ([]PhysicalRecordRow, int64, error)
}

func (r *physicalValueRepository) DeleteParentRows(ctx context.Context, tableName string, tenantID uint, recordIDs []uint) error {
	if err := storage.ValidateDynamicTableName(tableName); err != nil {
		return err
	}
	if len(recordIDs) == 0 {
		return nil
	}
	placeholders, args := recordIDPlaceholders(tenantID, recordIDs)
	return r.withContext(ctx).Exec(
		fmt.Sprintf("DELETE FROM %q WHERE tenant_id = ? AND record_id IN (%s)", tableName, placeholders), args...,
	).Error
}

func (r *physicalValueRepository) DeleteChildRows(ctx context.Context, tableName string, tenantID uint, recordIDs []uint) error {
	if err := storage.ValidateDynamicTableName(tableName); err != nil {
		return err
	}
	if len(recordIDs) == 0 {
		return nil
	}
	placeholders, args := recordIDPlaceholders(tenantID, recordIDs)
	return r.withContext(ctx).Exec(
		fmt.Sprintf("DELETE FROM %q WHERE tenant_id = ? AND parent_record_id IN (%s)", tableName, placeholders), args...,
	).Error
}

// recordIDPlaceholders 为动态值表删除生成纯参数化 IN 条件；表名/列名不来自
// 请求，而 ID 仍绝不拼接进 SQL，避免删除路径退化为裸原生语句。
func recordIDPlaceholders(tenantID uint, recordIDs []uint) (string, []any) {
	placeholders := make([]string, len(recordIDs))
	args := make([]any, 0, len(recordIDs)+1)
	args = append(args, tenantID)
	for i, id := range recordIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	return strings.Join(placeholders, ", "), args
}

type physicalValueRepository struct{ db *gorm.DB }

// NewPhysicalValueRepository 构造物理 DML 仓储。
func NewPhysicalValueRepository(db *gorm.DB) PhysicalValueRepository {
	return &physicalValueRepository{db: db}
}

func (r *physicalValueRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

// encodeValues 按列规格编码提交值（widgetName 键 → SQL 参数序列）。
func encodeValues(columns []storage.ColumnSpec, values map[string]any) ([]string, []any, error) {
	names := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns))
	for _, column := range columns {
		arg, err := storage.EncodeSQLValue(column.Kind, values[column.WidgetName])
		if err != nil {
			return nil, nil, fmt.Errorf("字段 %s: %w", column.WidgetName, err)
		}
		names = append(names, column.ColumnName())
		args = append(args, arg)
	}
	return names, args, nil
}

func (r *physicalValueRepository) InsertParentRow(ctx context.Context, tableName string, tenantID, recordID uint, columns []storage.ColumnSpec, values map[string]any) error {
	if err := storage.ValidateDynamicTableName(tableName); err != nil {
		return err
	}
	names, args, err := encodeValues(columns, values)
	if err != nil {
		return err
	}
	placeholders := make([]string, 0, len(names)+2)
	columnsSQL := make([]string, 0, len(names)+2)
	allArgs := make([]any, 0, len(args)+2)
	columnsSQL = append(columnsSQL, "tenant_id", "record_id")
	placeholders = append(placeholders, "?", "?")
	allArgs = append(allArgs, tenantID, recordID)
	for i, name := range names {
		if err := storage.ValidateColumnName(name); err != nil {
			return err
		}
		columnsSQL = append(columnsSQL, `"`+name+`"`)
		placeholders = append(placeholders, "?")
		allArgs = append(allArgs, args[i])
	}
	sql := fmt.Sprintf("INSERT INTO %q (%s) VALUES (%s)",
		tableName, strings.Join(columnsSQL, ", "), strings.Join(placeholders, ", "))
	return r.withContext(ctx).Exec(sql, allArgs...).Error
}

func (r *physicalValueRepository) ReadParentRow(ctx context.Context, tableName string, tenantID, recordID uint, columns []storage.ColumnSpec) (map[string]any, error) {
	if err := storage.ValidateDynamicTableName(tableName); err != nil {
		return nil, err
	}
	// 零列父表（纯子表单表单/无字段表单）：空 SELECT 列表是非法 SQL，
	// 行存在性由记录信封与复合外键保证，直接返回空值集。
	if len(columns) == 0 {
		return map[string]any{}, nil
	}
	selects := make([]string, 0, len(columns))
	scanTargets := make([]any, 0, len(columns))
	rawValues := make([]any, len(columns))
	for i, column := range columns {
		if err := storage.ValidateColumnName(column.ColumnName()); err != nil {
			return nil, err
		}
		selects = append(selects, `"`+column.ColumnName()+`"`)
		rawValues[i] = new(any)
		scanTargets = append(scanTargets, rawValues[i])
	}
	sql := fmt.Sprintf("SELECT %s FROM %q WHERE tenant_id = ? AND record_id = ?",
		strings.Join(selects, ", "), tableName)
	row := r.withContext(ctx).Raw(sql, tenantID, recordID).Row()
	if err := row.Scan(scanTargets...); err != nil {
		return nil, err
	}
	return decodeRow(columns, rawValues)
}

// decodeRow 扫描值 → widgetName 键的 JSON 值形态。
func decodeRow(columns []storage.ColumnSpec, rawValues []any) (map[string]any, error) {
	values := make(map[string]any, len(columns))
	for i, column := range columns {
		ptr := rawValues[i].(*any)
		decoded, err := storage.DecodeSQLValue(column.Kind, *ptr)
		if err != nil {
			return nil, fmt.Errorf("字段 %s: %w", column.WidgetName, err)
		}
		values[column.WidgetName] = decoded
	}
	return values, nil
}

func (r *physicalValueRepository) UpdateParentRowValues(ctx context.Context, tableName string, tenantID, recordID uint, columns []storage.ColumnSpec, values map[string]any) error {
	if err := storage.ValidateDynamicTableName(tableName); err != nil {
		return err
	}
	names, args, err := encodeValues(columns, values)
	if err != nil {
		return err
	}
	// 零值列（零列表单提交/不含业务字段的写回）：空 SET 子句是非法 SQL，
	// 信封 updated_at 由调用方（persistResolvedValues 的 TouchUpdatedAt）单独刷新。
	if len(names) == 0 {
		return nil
	}
	sets := make([]string, 0, len(names))
	allArgs := make([]any, 0, len(args)+3)
	for i, name := range names {
		if err := storage.ValidateColumnName(name); err != nil {
			return err
		}
		sets = append(sets, `"`+name+`" = ?`)
		allArgs = append(allArgs, args[i])
	}
	sql := fmt.Sprintf("UPDATE %q SET %s WHERE tenant_id = ? AND record_id = ?",
		tableName, strings.Join(sets, ", "))
	allArgs = append(allArgs, tenantID, recordID)
	return r.withContext(ctx).Exec(sql, allArgs...).Error
}

func (r *physicalValueRepository) ReplaceChildRows(ctx context.Context, tableName string, tenantID, parentRecordID uint, columns []storage.ColumnSpec, rows []map[string]any) error {
	if err := storage.ValidateDynamicTableName(tableName); err != nil {
		return err
	}
	// 集合替换：先删旧行（复合租户条件），再按提交顺序整批写入（sort_order
	// 从 0 起）；父记录 updated_at 由调用方（信封侧）同步刷新。
	if err := r.withContext(ctx).Exec(
		fmt.Sprintf("DELETE FROM %q WHERE tenant_id = ? AND parent_record_id = ?", tableName),
		tenantID, parentRecordID).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	var sb strings.Builder
	var allArgs []any
	sb.WriteString(fmt.Sprintf("INSERT INTO %q (tenant_id, parent_record_id, sort_order", tableName))
	for _, column := range columns {
		if err := storage.ValidateColumnName(column.ColumnName()); err != nil {
			return err
		}
		sb.WriteString(`, "` + column.ColumnName() + `"`)
	}
	sb.WriteString(") VALUES ")
	for rowOrder, row := range rows {
		if rowOrder > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(?, ?, ?")
		allArgs = append(allArgs, tenantID, parentRecordID, rowOrder)
		for _, column := range columns {
			arg, err := storage.EncodeSQLValue(column.Kind, row[column.WidgetName])
			if err != nil {
				return fmt.Errorf("子表行 %d 字段 %s: %w", rowOrder, column.WidgetName, err)
			}
			sb.WriteString(", ?")
			allArgs = append(allArgs, arg)
		}
		sb.WriteString(")")
	}
	return r.withContext(ctx).Exec(sb.String(), allArgs...).Error
}

func (r *physicalValueRepository) ReadChildRows(ctx context.Context, tableName string, tenantID, parentRecordID uint, columns []storage.ColumnSpec) ([]map[string]any, error) {
	if err := storage.ValidateDynamicTableName(tableName); err != nil {
		return nil, err
	}
	// 零列子表（子表单暂无子字段但存在行）：以行数查询替代空 SELECT 列表，
	// 按行数返回空 map，保留「每子行一个空对象」的行序语义（集合替换/回显
	// 依赖行数一致）。
	if len(columns) == 0 {
		var count int
		if err := r.withContext(ctx).Raw(
			fmt.Sprintf("SELECT count(*) FROM %q WHERE tenant_id = ? AND parent_record_id = ?", tableName),
			tenantID, parentRecordID).Scan(&count).Error; err != nil {
			return nil, err
		}
		rows := make([]map[string]any, count)
		for i := range rows {
			rows[i] = map[string]any{}
		}
		return rows, nil
	}
	selects := make([]string, 0, len(columns))
	for _, column := range columns {
		if err := storage.ValidateColumnName(column.ColumnName()); err != nil {
			return nil, err
		}
		selects = append(selects, `"`+column.ColumnName()+`"`)
	}
	sql := fmt.Sprintf("SELECT %s FROM %q WHERE tenant_id = ? AND parent_record_id = ? ORDER BY sort_order ASC",
		strings.Join(selects, ", "), tableName)
	rows, err := r.withContext(ctx).Raw(sql, tenantID, parentRecordID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := make([]map[string]any, 0)
	for rows.Next() {
		rawValues := make([]any, len(columns))
		scanTargets := make([]any, len(columns))
		for i := range rawValues {
			rawValues[i] = new(any)
			scanTargets[i] = rawValues[i]
		}
		if err := rows.Scan(scanTargets...); err != nil {
			return nil, err
		}
		decoded, err := decodeRow(columns, rawValues)
		if err != nil {
			return nil, err
		}
		results = append(results, decoded)
	}
	return results, rows.Err()
}

// ReadChildRowsByParentIDs 为列表页按子表单批量水合明细。结果中的每个输入父
// 记录 ID 均有键（没有明细时为空切片），确保调用方可稳定出网 `[]` 而非遗漏键。
func (r *physicalValueRepository) ReadChildRowsByParentIDs(ctx context.Context, tableName string, tenantID uint, parentRecordIDs []uint, columns []storage.ColumnSpec) (map[uint][]map[string]any, error) {
	if err := storage.ValidateDynamicTableName(tableName); err != nil {
		return nil, err
	}
	results := make(map[uint][]map[string]any, len(parentRecordIDs))
	if len(parentRecordIDs) == 0 {
		return results, nil
	}
	placeholders := make([]string, 0, len(parentRecordIDs))
	args := make([]any, 0, len(parentRecordIDs)+1)
	args = append(args, tenantID)
	for _, parentRecordID := range parentRecordIDs {
		// 同一页不会出现重复信封记录；即使调用方传入重复 ID，也只保留一次
		// 结果桶，SQL 的重复占位不影响正确性。
		results[parentRecordID] = []map[string]any{}
		placeholders = append(placeholders, "?")
		args = append(args, parentRecordID)
	}

	selects := []string{"parent_record_id"}
	for _, column := range columns {
		if err := storage.ValidateColumnName(column.ColumnName()); err != nil {
			return nil, err
		}
		selects = append(selects, `"`+column.ColumnName()+`"`)
	}
	sql := fmt.Sprintf(
		"SELECT %s FROM %q WHERE tenant_id = ? AND parent_record_id IN (%s) ORDER BY parent_record_id ASC, sort_order ASC",
		strings.Join(selects, ", "), tableName, strings.Join(placeholders, ", "),
	)
	rows, err := r.withContext(ctx).Raw(sql, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var parentRecordID uint
		rawValues := make([]any, len(columns))
		scanTargets := make([]any, 0, len(columns)+1)
		scanTargets = append(scanTargets, &parentRecordID)
		for i := range rawValues {
			rawValues[i] = new(any)
			scanTargets = append(scanTargets, rawValues[i])
		}
		if err := rows.Scan(scanTargets...); err != nil {
			return nil, err
		}
		decoded, err := decodeRow(columns, rawValues)
		if err != nil {
			return nil, err
		}
		results[parentRecordID] = append(results[parentRecordID], decoded)
	}
	return results, rows.Err()
}

func (r *physicalValueRepository) UpdateWorkflowProjection(ctx context.Context, tableName string, tenantID, recordID uint, instanceNo, status string, updatedAt time.Time) error {
	if err := storage.ValidateDynamicTableName(tableName); err != nil {
		return err
	}
	return r.withContext(ctx).Exec(
		fmt.Sprintf("UPDATE %q SET workflow_instance_no = ?, workflow_status = ?, workflow_updated_at = ? WHERE tenant_id = ? AND record_id = ?", tableName),
		instanceNo, status, updatedAt, tenantID, recordID).Error
}

// envelopeSelectColumns 信封表 r 的固定投影列（与 model.FormRecord 字段对齐）。
var envelopeSelectColumns = []string{
	"r.id", "r.form_id", "r.form_version_id", "r.data_op_id", "r.menu_code",
	"r.values", "r.submitted_by_member_id", "r.submitted_by_name",
	"r.submitted_at", "r.updated_at", "r.created_at",
	"r.updated_by_member_id", "r.updated_by_name",
	"r.tenant_id", "r.workflow_instance_no", "r.workflow_status", "r.workflow_updated_at",
}

func (r *physicalValueRepository) ListJoinControlled(ctx context.Context, params RecordListParams, binding PhysicalListBinding) ([]PhysicalRecordRow, int64, error) {
	if err := storage.ValidateDynamicTableName(binding.TableName); err != nil {
		return nil, 0, err
	}
	join := fmt.Sprintf("FROM tn_form_records r JOIN %q d ON d.record_id = r.id AND d.tenant_id = r.tenant_id", binding.TableName)
	where := "WHERE r.tenant_id = ? AND r.form_id = ?"
	args := append([]any{params.TenantID, params.FormID}, params.Args...)
	if strings.TrimSpace(params.Where) != "" && params.Where != "TRUE" {
		where += " AND (" + params.Where + ")"
	}
	orderBy := params.OrderBy
	if orderBy == "" {
		orderBy = "r.id DESC"
	} else {
		// 物理模式排序片段带 r./d. 前缀（系统字段按字段分派编译），稳定尾
		// 排序恒追加
		orderBy += ", r.id DESC"
	}

	var total int64
	countSQL := "SELECT count(*) " + join + " " + where
	if err := r.withContext(ctx).Raw(countSQL, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []PhysicalRecordRow{}, 0, nil
	}

	selects := append([]string{}, envelopeSelectColumns...)
	for _, column := range binding.Columns {
		if err := storage.ValidateColumnName(column.Column.ColumnName()); err != nil {
			return nil, 0, err
		}
		selects = append(selects, `d."`+column.Column.ColumnName()+`"`)
	}
	offset := (params.Page - 1) * params.PageSize
	listSQL := fmt.Sprintf("SELECT %s %s %s ORDER BY %s LIMIT %d OFFSET %d",
		strings.Join(selects, ", "), join, where, orderBy, params.PageSize, offset)

	rows, err := r.withContext(ctx).Raw(listSQL, args...).Rows()
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	results := make([]PhysicalRecordRow, 0, params.PageSize)
	for rows.Next() {
		var record model.FormRecord
		var valuesJSON []byte
		physicalRaw := make([]any, len(binding.Columns))
		physicalScan := make([]any, len(binding.Columns))
		for i := range physicalRaw {
			physicalRaw[i] = new(any)
			physicalScan[i] = physicalRaw[i]
		}
		scanTargets := []any{
			&record.ID, &record.FormID, &record.FormVersionID, &record.DataOpID, &record.MenuCode,
			&valuesJSON, &record.SubmittedByMemberID, &record.SubmittedByName,
			&record.SubmittedAt, &record.UpdatedAt, &record.CreatedAt,
			&record.UpdatedByMemberID, &record.UpdatedByName,
			&record.TenantID, &record.WorkflowInstanceNo, &record.WorkflowStatus, &record.WorkflowUpdatedAt,
		}
		scanTargets = append(scanTargets, physicalScan...)
		if err := rows.Scan(scanTargets...); err != nil {
			return nil, 0, err
		}
		if len(valuesJSON) > 0 {
			record.Values = model.JSONContent(valuesJSON)
		}
		values, err := decodePhysicalBinding(binding.Columns, physicalRaw)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, PhysicalRecordRow{Record: record, Values: values})
	}
	return results, total, rows.Err()
}

func decodePhysicalBinding(columns []PhysicalColumn, rawValues []any) (map[string]any, error) {
	values := make(map[string]any, len(columns))
	for i, column := range columns {
		ptr := rawValues[i].(*any)
		decoded, err := storage.DecodeSQLValue(column.Column.Kind, *ptr)
		if err != nil {
			return nil, fmt.Errorf("字段 %s: %w", column.WidgetName, err)
		}
		values[column.WidgetName] = decoded
	}
	return values, nil
}
