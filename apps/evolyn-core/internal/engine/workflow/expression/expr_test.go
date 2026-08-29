package expression

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileAndEvalWhitelist(t *testing.T) {
	engine := NewExprEngine()

	// 第 16.1 章示例表达式
	p, err := engine.Compile("form.amount > 10000 && starter.department_id != \"finance\"")
	require.NoError(t, err)

	out, err := p.Eval(map[string]any{
		"form":    map[string]any{"amount": 20000},
		"starter": map[string]any{"department_id": "sales"},
	})
	require.NoError(t, err)
	assert.Equal(t, true, out)

	out, err = p.Eval(map[string]any{
		"form":    map[string]any{"amount": 5000},
		"starter": map[string]any{"department_id": "finance"},
	})
	require.NoError(t, err)
	assert.Equal(t, false, out)

	// variables 白名单组
	p2, err := engine.Compile("variables.passed == true")
	require.NoError(t, err)
	out, err = p2.Eval(map[string]any{"variables": map[string]any{"passed": true}})
	require.NoError(t, err)
	assert.Equal(t, true, out)
}

func TestCompileRejectsNonWhitelistAndCalls(t *testing.T) {
	engine := NewExprEngine()

	// 白名单外变量（第 16.2 章：仅暴露 form/starter/variables）
	_, err := engine.Compile("secret_key")
	assert.Error(t, err)

	// 函数调用禁止（内置函数同样拒绝）
	_, err = engine.Compile("len(form.amount) > 1")
	assert.Error(t, err)

	// 语法错误
	_, err = engine.Compile("form.amount >")
	assert.Error(t, err)

	// 空表达式
	_, err = engine.Compile("  ")
	assert.Error(t, err)

	// 长度上限
	long := make([]byte, MaxExpressionLength+1)
	for i := range long {
		long[i] = 'a'
	}
	_, err = engine.Compile(string(long))
	assert.Error(t, err)
}

func TestEvalIsolatesEnv(t *testing.T) {
	engine := NewExprEngine()
	p, err := engine.Compile("form.amount")
	require.NoError(t, err)

	// env 缺失键以空集参与求值（不panic、不出错）
	out, err := p.Eval(nil)
	require.NoError(t, err)
	assert.Nil(t, out)

	// 非 map 值被忽略，保持零值语义
	out, err = p.Eval(map[string]any{"form": "not-a-map"})
	require.NoError(t, err)
	assert.Nil(t, out)
}
