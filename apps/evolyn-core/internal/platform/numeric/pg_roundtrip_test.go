package numeric

import (
	"testing"

	"evolyn/internal/testsupport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 数据库契约测试（设计 §54）：decimal string 经 Go → PostgreSQL NUMERIC
// → Go 往返后数值保持一致（无 float 中转、无指数形态、无精度损失）。
// 真库依赖：TEST_PG_DSN 未设置时按惯例跳过
func TestNumericPostgresRoundtrip(t *testing.T) {
	db := testsupport.NewPostgres(t)

	require.NoError(t, db.Exec(`CREATE TEMP TABLE numeric_contract (
		id int PRIMARY KEY,
		v NUMERIC
	)`).Error)

	vectors := []string{
		"9999999999999999.123456", // 设计 §54 原例
		"0.1",
		"-9007199254740993",    // 超 float64 安全整数
		"0.000000000000000001", // 1e-18（maxScale 边界内）
		"9999999999999999999999999999999999999999", // 40 位大整数
		"-12345.6789",
		"0",
	}

	for i, input := range vectors {
		expected, err := Parse(input)
		require.NoError(t, err)
		expectedText, err := expected.Serialize()
		require.NoError(t, err)

		require.NoError(t, db.Exec(`INSERT INTO numeric_contract (id, v) VALUES ($1, $2)`, i, input).Error,
			"decimal string 原文入参 NUMERIC")

		var raw string
		require.NoError(t, db.Raw(`SELECT v::text FROM numeric_contract WHERE id = $1`, i).Scan(&raw).Error,
			"NUMERIC 以 ::text 读出避免驱动侧 float 转换")

		got, err := Parse(raw)
		require.NoError(t, err, "PG ::text 形态应可回解析")
		gotText, err := got.Serialize()
		require.NoError(t, err)

		assert.Equal(t, expectedText, gotText, "往返数值一致性: %s", input)

		cmp, err := got.Compare(expected)
		require.NoError(t, err)
		assert.Zero(t, cmp, "往返数值等值: %s", input)
	}
}
