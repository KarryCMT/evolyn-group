package service

import (
	"testing"

	"evolyn/internal/platform/form/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 系统字段编译器（000067）：物理列直映射，值只进绑定参数；操作符矩阵与
// 前端 @evolyn.do/query 的 enum/datetime 字典镜像。
func TestCompileSystemRecordConditionFiltersBySubmitter(t *testing.T) {
	compiled, err := compileSystemRecordCondition(SysFieldSubmittedBy, "eq", float64(7), "")
	require.NoError(t, err)
	assert.Equal(t, "submitted_by_member_id = ?", compiled.Where)
	assert.Equal(t, []any{int64(7)}, compiled.Args)

	compiled, err = compileSystemRecordCondition(SysFieldSubmittedBy, "notIn", []any{float64(1), float64(2)}, "")
	require.NoError(t, err)
	assert.Equal(t, "submitted_by_member_id NOT IN (?, ?)", compiled.Where)
	assert.Equal(t, []any{int64(1), int64(2)}, compiled.Args)
}

func TestCompileSystemRecordConditionFiltersByRecordID(t *testing.T) {
	compiled, err := compileSystemRecordCondition(SysFieldRecordID, "eq", float64(42), "r.")
	require.NoError(t, err)
	assert.Equal(t, "r.id = ?", compiled.Where)
	assert.Equal(t, []any{int64(42)}, compiled.Args)

	_, err = compileSystemRecordCondition(SysFieldRecordID, "eq", float64(0), "r.")
	assert.Error(t, err)
	_, err = compileSystemRecordCondition(SysFieldRecordID, "contains", "42", "r.")
	assert.Error(t, err)
}

func TestCompileSystemRecordConditionFiltersByTimestamp(t *testing.T) {
	compiled, err := compileSystemRecordCondition(SysFieldUpdatedAt, "gte", "2026-09-01 00:00:00", "")
	require.NoError(t, err)
	assert.Equal(t, "updated_at >= ?", compiled.Where)
	assert.Equal(t, []any{"2026-09-01 00:00:00"}, compiled.Args)

	// date-only 值合法（Postgres 解释为当日零点）；非法格式在服务端拒绝，
	// 不允许落进 timestamptz 比较引发数据库错误
	compiled, err = compileSystemRecordCondition(SysFieldSubmittedAt, "between", []any{"2026-09-01", "2026-09-30"}, "")
	require.NoError(t, err)
	assert.Equal(t, "submitted_at BETWEEN ? AND ?", compiled.Where)

	_, err = compileSystemRecordCondition(SysFieldSubmittedAt, "eq", "2026/09/01", "")
	assert.Error(t, err)
	_, err = compileSystemRecordCondition(SysFieldUpdatedAt, "contains", "09", "")
	assert.Error(t, err)
	_, err = compileSystemRecordCondition(SysFieldSubmittedBy, "eq", "not-a-number", "")
	assert.Error(t, err)
}

func TestCompileRecordListQueryRoutesSystemFields(t *testing.T) {
	document := model.RecordQueryDocument{
		Version: 1,
		Filter:  &model.RecordQueryExpression{Type: "condition", Field: " " + SysFieldSubmittedBy + " ", Operator: "isNotNull"},
	}
	compiled, err := CompileRecordListQuery(document, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "TRUE", compiled.Where)
}

func TestCompileRecordListSortsOnlyAllowsSystemFields(t *testing.T) {
	order, err := CompileRecordListSorts([]model.RecordQuerySort{
		{Field: SysFieldUpdatedAt, Direction: "DESC"},
		{Field: SysFieldSubmittedAt, Direction: "asc"},
	})
	require.NoError(t, err)
	assert.Equal(t, "updated_at DESC, submitted_at ASC", order)

	order, err = CompileRecordListSorts(nil)
	require.NoError(t, err)
	assert.Empty(t, order)

	_, err = CompileRecordListSorts([]model.RecordQuerySort{{Field: "_widget_name", Direction: "asc"}})
	assert.Error(t, err)
	_, err = CompileRecordListSorts([]model.RecordQuerySort{{Field: SysFieldUpdatedAt, Direction: "random"}})
	assert.Error(t, err)
}

// 物理模式系统字段前缀按字段分派：流程三字段（单号/状态/更新时间）挂物理
// 表 d 前缀以命中预置复合索引（方案 §5.3/§11），其余（信封物理属性）恒挂
// 信封表 r；排序与筛选共用同一分派。
func TestCompileSystemFieldPhysicalPrefixDispatch(t *testing.T) {
	opts := RecordQueryCompileOptions{Physical: true, PhysicalColumns: map[string]string{}}

	document := model.RecordQueryDocument{Version: 1}

	document.Filter = &model.RecordQueryExpression{Type: "condition", Field: SysFieldWorkflowStatus, Operator: "eq", Value: "RUNNING"}
	compiled, err := CompileRecordListQuery(document, nil, nil, opts)
	require.NoError(t, err)
	assert.Equal(t, "d.workflow_status = ?", compiled.Where)
	assert.Equal(t, []any{"RUNNING"}, compiled.Args)

	document.Filter = &model.RecordQueryExpression{Type: "condition", Field: SysFieldWorkflowInstanceNo, Operator: "startsWith", Value: "WF-2026"}
	compiled, err = CompileRecordListQuery(document, nil, nil, opts)
	require.NoError(t, err)
	assert.Equal(t, "d.workflow_instance_no LIKE ? ESCAPE '\\'", compiled.Where)

	document.Filter = &model.RecordQueryExpression{Type: "condition", Field: SysFieldSubmittedAt, Operator: "gte", Value: "2026-09-01"}
	compiled, err = CompileRecordListQuery(document, nil, nil, opts)
	require.NoError(t, err)
	assert.Equal(t, "r.submitted_at >= ?", compiled.Where)

	document.Filter = &model.RecordQueryExpression{Type: "condition", Field: SysFieldRecordID, Operator: "eq", Value: float64(42)}
	compiled, err = CompileRecordListQuery(document, nil, nil, opts)
	require.NoError(t, err)
	assert.Equal(t, "r.id = ?", compiled.Where)

	order, err := CompileRecordListSorts([]model.RecordQuerySort{
		{Field: SysFieldWorkflowUpdatedAt, Direction: "desc"},
		{Field: SysFieldUpdatedAt, Direction: "desc"},
	}, opts)
	require.NoError(t, err)
	assert.Equal(t, "d.workflow_updated_at DESC, r.updated_at DESC", order)

	// legacy 单表模式恒无前缀（既有断言口径不变）
	compiled, err = CompileRecordListQuery(model.RecordQueryDocument{
		Version: 1,
		Filter:  &model.RecordQueryExpression{Type: "condition", Field: SysFieldWorkflowStatus, Operator: "eq", Value: "NONE"},
	}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "workflow_status = ?", compiled.Where)
}
