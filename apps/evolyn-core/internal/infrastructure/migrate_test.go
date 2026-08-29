package infrastructure

import (
	"testing"
	"testing/fstest"

	"evolyn/migrations"

	"github.com/stretchr/testify/assert"
)

func TestSplitSQLStatements(t *testing.T) {
	script := `-- 注释里有分号;不应切分
CREATE TABLE a (id BIGINT PRIMARY KEY, note TEXT); /* 块注释;同样忽略 */
INSERT INTO a (note) VALUES ('字符串里的;分号'), ('转义''引号'';继续');
-- 美元引用函数体（后续动态 DDL 会用到）
CREATE FUNCTION f() RETURNS void AS $$
BEGIN
	PERFORM 1; -- 函数体内的分号
END
$$ LANGUAGE plpgsql;
`

	// 注释内容剥离，可执行语句 3 条：DDL / DML / 函数定义
	stmts := SplitSQLStatements(script)
	assert.Len(t, stmts, 3)
	assert.Contains(t, stmts[0], "CREATE TABLE a")
	assert.NotContains(t, stmts[0], "注释里有分号") // 行注释剥离
	// 字符串字面量中的分号与转义引号保持完整
	assert.Contains(t, stmts[1], "'字符串里的;分号'")
	assert.Contains(t, stmts[1], "转义''引号'';继续")
	// 美元引用函数体整体保留（内部分号不切分）
	assert.Contains(t, stmts[2], "$$")
	assert.Contains(t, stmts[2], "PERFORM 1;")
}

func TestSplitSQLEmptyAndWhitespace(t *testing.T) {
	assert.Empty(t, SplitSQLStatements(""))
	assert.Empty(t, SplitSQLStatements("\n\n-- only comment\n"))
	assert.Equal(t, []string{"SELECT 1"}, SplitSQLStatements("  SELECT 1 ;\n"))
}

// TestSplitSQLTaggedDollarQuotes 通用 $tag$ 美元引用（FIX-023）：
// 记录开 tag，仅完全相同的结束 tag 退出引用状态。
// 注意 PG 语义：美元引用无转义，体内出现相同 tag 即闭合——因此函数体
// 内只放 $$ 与 $other$ 这类「不同定界符」验证不误闭合
func TestSplitSQLTaggedDollarQuotes(t *testing.T) {
	script := `-- $func$ 形态：tag 必须完全匹配才闭合
CREATE FUNCTION f1() RETURNS void AS $func$
BEGIN
	PERFORM 1; -- 函数体内的分号
	RAISE NOTICE '$$ 与 $other$ 都不是结束定界符';
END
$func$ LANGUAGE plpgsql;

CREATE FUNCTION f2() RETURNS void AS $body$
BEGIN
	PERFORM 2;
END
$body$ LANGUAGE plpgsql;

SELECT '$1 占位符不受影响', 1;
`

	stmts := SplitSQLStatements(script)
	assert.Len(t, stmts, 3, "三条语句：两个函数定义 + 一条查询")

	// $func$ 函数体完整保留：内部分号与 $$/$other$ 定界符均不切分/不闭合
	assert.Contains(t, stmts[0], "$func$")
	assert.Contains(t, stmts[0], "PERFORM 1;")
	assert.Contains(t, stmts[0], "'$$ 与 $other$ 都不是")
	assert.Contains(t, stmts[0], "$func$ LANGUAGE plpgsql")

	// 不同 tag（$body$）同样整体保留
	assert.Contains(t, stmts[1], "PERFORM 2;")
	assert.Contains(t, stmts[1], "$body$ LANGUAGE plpgsql")

	// $1 参数占位符不进入美元引用状态，语句正常切分
	assert.Contains(t, stmts[2], "$1 占位符不受影响")
}

// TestSplitSQLDollarQuoteTagMismatch 开闭 tag 不一致：不闭合引用，
// 后续分号都在引用体内，整段为一条语句
func TestSplitSQLDollarQuoteTagMismatch(t *testing.T) {
	script := `SELECT $a$ body with ; semicolons $b$ mismatch; SELECT 2;`
	stmts := SplitSQLStatements(script)
	assert.Len(t, stmts, 1, "$b$ 不是 $a$ 的闭合，引用吞到 EOF")
	assert.Contains(t, stmts[0], "$a$")
	assert.Contains(t, stmts[0], "$b$ mismatch")
}

// TestSplitSQLUnterminatedBlockComment 块注释未闭合到 EOF：不越界，
// 整体按注释吞掉
func TestSplitSQLUnterminatedBlockComment(t *testing.T) {
	stmts := SplitSQLStatements("SELECT 1; /* unterminated comment")
	assert.Equal(t, []string{"SELECT 1"}, stmts)
}

// TestLoadMigrationsEmbedded 校验嵌入迁移集：命名规约、up/down 成对、版本连续排序
func TestLoadMigrationsEmbedded(t *testing.T) {
	files, err := loadMigrations(migrations.FS)
	assert.NoError(t, err)

	// 106 个文件 = 53 版本 × up/down（000053 流程变量 + job 类型约束扩展，随链顺延更新）
	assert.Len(t, files, 106)
	for i := 0; i+1 < len(files); i += 2 {
		assert.Equal(t, files[i].version, files[i+1].version, "up/down 应相邻成对")
		assert.Equal(t, "down", files[i].direction)
		assert.Equal(t, "up", files[i+1].direction)
		assert.Equal(t, files[i].name, files[i+1].name)
	}

	// 首版本即初始基线
	assert.Equal(t, int64(1), files[0].version)
	assert.Equal(t, "init", files[0].name)
}

func TestLoadMigrationsBadNaming(t *testing.T) {
	fsys := fstest.MapFS{
		"init.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
	}
	_, err := loadMigrations(fsys)
	assert.ErrorContains(t, err, "does not match")
}

func TestLoadMigrationsMissingDown(t *testing.T) {
	fsys := fstest.MapFS{
		"000001_init.up.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
	}
	_, err := loadMigrations(fsys)
	assert.ErrorContains(t, err, "exactly one up and one down")
}
