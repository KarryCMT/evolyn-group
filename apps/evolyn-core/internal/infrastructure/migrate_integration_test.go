package infrastructure_test

import (
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"evolyn/internal/infrastructure"
	"evolyn/internal/testsupport"
	"evolyn/migrations"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ---- FIX-023 Migration 工程验收（MIGRATE-INT-001~005，真实 PostgreSQL）----
//
// 空库 → 全链迁移 → 幂等重放 → checksum 防篡改 → 中途失败回滚 → 终态
// Schema 与 scripts/db.sql 快照一致。

// MIGRATE-INT-001：空 PostgreSQL 数据库执行全部 Migration，全部成功
func TestMigrateINT001EmptyDatabaseUp(t *testing.T) {
	db := testsupport.NewPostgresRaw(t)

	assert.NoError(t, infrastructure.NewMigrator(db).Up())

	// 版本登记完整：全部版本（当前 36 个）落库
	var count int64
	assert.NoError(t, db.Raw("SELECT COUNT(*) FROM schema_migrations").Scan(&count).Error)
	assert.EqualValues(t, 36, count)

	// 关键业务表已建齐（表名与迁移链一致）
	for _, table := range []string{
		"accounts", "auth_infos", "tenants", "users", "groups", "role_groups", "roles",
		"resources", "departments", "audit_logs", "login_logs",
		"user_roles", "user_groups", "group_roles", "department_users",
		"member_invitations", "tenant_public_invitation_links",
		"tenant_member_field_settings", "member_profiles",
		"admin_groups", "admin_group_members",
		"product_catalogs", "tenant_product_configs",
		"tenant_product_departments", "tenant_product_members",
	} {
		var exists bool
		assert.NoError(t, db.Raw(
			"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = ?)",
			table).Scan(&exists).Error, table)
		assert.True(t, exists, "table %s should exist", table)
	}
}

// MIGRATE-INT-002：全部完成后再次执行，保持幂等（不重复应用、不报错）
func TestMigrateINT002IdempotentReplay(t *testing.T) {
	db := testsupport.NewPostgresRaw(t)
	m := infrastructure.NewMigrator(db)

	assert.NoError(t, m.Up())
	assert.NoError(t, m.Up(), "重复执行必须幂等")

	var count int64
	assert.NoError(t, db.Raw("SELECT COUNT(*) FROM schema_migrations").Scan(&count).Error)
	assert.EqualValues(t, 36, count, "重放不得产生重复版本记录")
}

// MIGRATE-INT-003：已执行迁移内容被篡改（checksum 改变）必须拒绝
func TestMigrateINT003ChecksumTamperRejected(t *testing.T) {
	db := testsupport.NewPostgresRaw(t)
	assert.NoError(t, infrastructure.NewMigrator(db).Up())

	// 动态复制嵌入迁移集，再篡改最后一个版本的内容：UpFS 必须拒绝启动
	tampered := fstest.MapFS{}
	err := fs.WalkDir(migrations.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, readErr := migrations.FS.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		tampered[path] = &fstest.MapFile{Data: data}
		return nil
	})
	assert.NoError(t, err)
	tampered["000007_account_phone_unique.up.sql"] = &fstest.MapFile{Data: []byte("-- tampered content\nSELECT 1;")}

	err = infrastructure.NewMigrator(db).UpFS(tampered)
	assert.ErrorContains(t, err, "checksum mismatch")
}

// MIGRATE-INT-004：迁移中途失败，SQL 与 schema_migrations 记录全部回滚
func TestMigrateINT004MidFailureRollsBack(t *testing.T) {
	db := testsupport.NewPostgresRaw(t)

	// 第一条语句建表成功、第二条失败：整个迁移必须回滚（表不残留、无版本记录）
	fsys := fstest.MapFS{
		"000001_ok.up.sql":   &fstest.MapFile{Data: []byte("SELECT 1;")},
		"000001_ok.down.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
		"000002_boom.up.sql": &fstest.MapFile{Data: []byte(
			"CREATE TABLE tx_rollback_probe (id BIGINT PRIMARY KEY);\nINSERT INTO no_such_table VALUES (1);")},
		"000002_boom.down.sql": &fstest.MapFile{Data: []byte("DROP TABLE IF EXISTS tx_rollback_probe;")},
	}

	err := infrastructure.NewMigrator(db).UpFS(fsys)
	assert.Error(t, err, "第二条语句失败必须使迁移报错")

	var tableExists bool
	assert.NoError(t, db.Raw(
		"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'tx_rollback_probe')",
	).Scan(&tableExists).Error)
	assert.False(t, tableExists, "同事务回滚：已执行语句的建表不得残留")

	var versions int64
	assert.NoError(t, db.Raw("SELECT COUNT(*) FROM schema_migrations WHERE version = 2").Scan(&versions).Error)
	assert.EqualValues(t, 0, versions, "失败迁移不得登记版本")
}

// MIGRATE-INT-005：迁移链终态 Schema 与 scripts/db.sql 快照一致
// （表/列/索引/约束名逐一比对；db.sql 是 migrations 的等价快照，FIX-009）
func TestMigrateINT005SchemaMatchesSnapshot(t *testing.T) {
	migrated := testsupport.NewPostgresRaw(t)
	assert.NoError(t, infrastructure.NewMigrator(migrated).Up())

	snapshot := testsupport.NewPostgresRaw(t)
	execSnapshotSQL(t, snapshot)

	assert.Equal(t, schemaFingerprint(t, snapshot), schemaFingerprint(t, migrated),
		"迁移链终态与 db.sql 快照的表/列/索引/约束必须一致")
}

// execSnapshotSQL 在空库上重放 scripts/db.sql：剥离 psql 元命令
// （\c 等）与 CREATE DATABASE 语句后，复用 SplitSQLStatements 逐条执行
func execSnapshotSQL(t *testing.T, db *gorm.DB) {
	t.Helper()

	raw, err := os.ReadFile("../../scripts/db.sql")
	assert.NoError(t, err)

	var filtered []string
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		// psql 元命令（\c weave）与建库语句在单库会话内不可执行/无意义
		if strings.HasPrefix(trimmed, "\\") || strings.HasPrefix(strings.ToUpper(trimmed), "CREATE DATABASE") {
			continue
		}
		filtered = append(filtered, line)
	}

	for _, stmt := range infrastructure.SplitSQLStatements(strings.Join(filtered, "\n")) {
		assert.NoError(t, db.Exec(stmt).Error, "db.sql 语句执行失败: %.120s", stmt)
	}
}

// schemaFingerprint 提取库结构指纹：public 模式下全部表、列（名称/类型/
// 可空）、索引名、约束名的排序集合。schema_migrations 为迁移器私有表，
// 不属于业务 Schema，两侧统一排除
func schemaFingerprint(t *testing.T, db *gorm.DB) []string {
	t.Helper()

	var fingerprint []string

	var tables []string
	assert.NoError(t, db.Raw(`
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		  AND table_name <> 'schema_migrations'
		ORDER BY table_name`).Scan(&tables).Error)
	fingerprint = append(fingerprint, tables...)

	type columnRow struct {
		TableName  string
		ColumnName string
		DataType   string
		IsNullable string
	}
	var columns []columnRow
	assert.NoError(t, db.Raw(`
		SELECT table_name, column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name <> 'schema_migrations'
		ORDER BY table_name, column_name`).Scan(&columns).Error)
	for _, c := range columns {
		fingerprint = append(fingerprint,
			"col "+c.TableName+"."+c.ColumnName+" "+c.DataType+" "+c.IsNullable)
	}

	var indexes []string
	assert.NoError(t, db.Raw(`
		SELECT indexname FROM pg_indexes
		WHERE schemaname = 'public' AND tablename <> 'schema_migrations'
		ORDER BY indexname`).Scan(&indexes).Error)
	fingerprint = append(fingerprint, indexes...)

	var constraints []string
	// to_regclass 兼容未建 schema_migrations 的快照库（::regclass 直接引用会 42P01）
	assert.NoError(t, db.Raw(`
		SELECT conname FROM pg_constraint
		WHERE connamespace = 'public'::regnamespace
		  AND (to_regclass('public.schema_migrations') IS NULL
		       OR conrelid <> to_regclass('public.schema_migrations'))
		ORDER BY conname`).Scan(&constraints).Error)
	fingerprint = append(fingerprint, constraints...)

	return fingerprint
}
