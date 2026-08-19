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

// TestLoadMigrationsEmbedded 校验嵌入迁移集：命名规约、up/down 成对、版本连续排序
func TestLoadMigrationsEmbedded(t *testing.T) {
	files, err := loadMigrations(migrations.FS)
	assert.NoError(t, err)

	// 12 个文件 = 6 版本 × up/down
	assert.Len(t, files, 12)
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
