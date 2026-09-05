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
	compiled, err := compileSystemRecordCondition(SysFieldSubmittedBy, "eq", float64(7))
	require.NoError(t, err)
	assert.Equal(t, "submitted_by_member_id = ?", compiled.Where)
	assert.Equal(t, []any{int64(7)}, compiled.Args)

	compiled, err = compileSystemRecordCondition(SysFieldSubmittedBy, "notIn", []any{float64(1), float64(2)})
	require.NoError(t, err)
	assert.Equal(t, "submitted_by_member_id NOT IN (?, ?)", compiled.Where)
	assert.Equal(t, []any{int64(1), int64(2)}, compiled.Args)
}

func TestCompileSystemRecordConditionFiltersByTimestamp(t *testing.T) {
	compiled, err := compileSystemRecordCondition(SysFieldUpdatedAt, "gte", "2026-09-01 00:00:00")
	require.NoError(t, err)
	assert.Equal(t, "updated_at >= ?", compiled.Where)
	assert.Equal(t, []any{"2026-09-01 00:00:00"}, compiled.Args)

	// date-only 值合法（Postgres 解释为当日零点）；非法格式在服务端拒绝，
	// 不允许落进 timestamptz 比较引发数据库错误
	compiled, err = compileSystemRecordCondition(SysFieldSubmittedAt, "between", []any{"2026-09-01", "2026-09-30"})
	require.NoError(t, err)
	assert.Equal(t, "submitted_at BETWEEN ? AND ?", compiled.Where)

	_, err = compileSystemRecordCondition(SysFieldSubmittedAt, "eq", "2026/09/01")
	assert.Error(t, err)
	_, err = compileSystemRecordCondition(SysFieldUpdatedAt, "contains", "09")
	assert.Error(t, err)
	_, err = compileSystemRecordCondition(SysFieldSubmittedBy, "eq", "not-a-number")
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
