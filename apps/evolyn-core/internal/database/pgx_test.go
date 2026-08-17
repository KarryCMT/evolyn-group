package database

import (
	"context"
	"testing"
	"time"

	"evolyn/internal/config"
)

func TestPgxConnection(t *testing.T) {
	cfg, err := config.Parse("../../config/app.yaml")
	if err != nil {
		t.Fatalf("parse config failed: %v", err)
	}

	// pgx 指定连接 zcode 数据库
	pgxDBConfig := cfg.DB
	pgxDBConfig.Name = "zcode"

	pool, err := NewPgxPool(&pgxDBConfig)
	if err != nil {
		t.Fatalf("create pgx pool failed: %v", err)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	// 验证当前数据库
	var databaseName string

	err = pool.QueryRow(
		ctx,
		"SELECT current_database()",
	).Scan(&databaseName)

	if err != nil {
		t.Fatalf("query current database failed: %v", err)
	}

	if databaseName != "zcode" {
		t.Fatalf(
			"expected database zcode, got %s",
			databaseName,
		)
	}

	t.Logf(
		"pgx connected successfully, database=%s",
		databaseName,
	)
}

func TestPgxCRUD(t *testing.T) {
	cfg, err := config.Parse("../../config/app.yaml")
	if err != nil {
		t.Fatalf("parse config failed: %v", err)
	}

	// 继续连接 firefly 数据库
	pool, err := NewPgxPool(&cfg.DB)
	if err != nil {
		t.Fatalf("create pgx pool failed: %v", err)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	// 1. 确认当前数据库
	var databaseName string
	err = pool.QueryRow(
		ctx,
		"SELECT current_database()",
	).Scan(&databaseName)

	if err != nil {
		t.Fatalf("query current database failed: %v", err)
	}

	t.Logf("current database=%s", databaseName)

	if databaseName != "firefly" {
		t.Fatalf(
			"expected database firefly, got %s",
			databaseName,
		)
	}

	// 2. 确认 zcode schema 存在
	var schemaExists bool

	err = pool.QueryRow(
		ctx,
		`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.schemata
			WHERE schema_name = $1
		)
		`,
		"zcode",
	).Scan(&schemaExists)

	if err != nil {
		t.Fatalf("check zcode schema failed: %v", err)
	}

	if !schemaExists {
		t.Fatal("schema zcode does not exist")
	}

	t.Log("schema zcode exists")

	// // 3. 为避免重复执行测试报错，先删除测试表
	// _, err = pool.Exec(ctx, `
	// 	DROP TABLE IF EXISTS zcode.pgx_crud_test
	// `)
	// if err != nil {
	// 	t.Fatalf("drop old test table failed: %v", err)
	// }

	// // 4. 创建真实物理表
	// _, err = pool.Exec(ctx, `
	// 	CREATE TABLE zcode.pgx_crud_test (
	// 		id BIGSERIAL PRIMARY KEY,
	// 		name VARCHAR(100) NOT NULL,
	// 		amount NUMERIC(18, 2),
	// 		status VARCHAR(50),
	// 		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	// 	)
	// `)
	// if err != nil {
	// 	t.Fatalf("create table failed: %v", err)
	// }

	// t.Log("create zcode.pgx_crud_test success")

	// 5. INSERT
	var id int64

	err = pool.QueryRow(
		ctx,
		`
		INSERT INTO zcode.pgx_crud_test (
			name,
			amount,
			status
		)
		VALUES ($1, $2, $3)
		RETURNING id
		`,
		"Evolyn Test",
		1000.50,
		"draft",
	).Scan(&id)

	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	t.Logf("insert success, id=%d", id)

	// 6. SELECT
	var (
		name   string
		amount float64
		status string
	)

	err = pool.QueryRow(
		ctx,
		`
		SELECT
			name,
			amount,
			status
		FROM zcode.pgx_crud_test
		WHERE id = $1
		`,
		id,
	).Scan(
		&name,
		&amount,
		&status,
	)

	if err != nil {
		t.Fatalf("select failed: %v", err)
	}

	if name != "Evolyn Test" {
		t.Fatalf("unexpected name: %s", name)
	}

	if status != "draft" {
		t.Fatalf("unexpected status: %s", status)
	}

	t.Logf(
		"select success: name=%s amount=%.2f status=%s",
		name,
		amount,
		status,
	)

	// 7. UPDATE
	tag, err := pool.Exec(
		ctx,
		`
		UPDATE zcode.pgx_crud_test
		SET
			name = $1,
			status = $2
		WHERE id = $3
		`,
		"Evolyn Updated",
		"published",
		id,
	)

	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	if tag.RowsAffected() != 1 {
		t.Fatalf(
			"expected update 1 row, got %d",
			tag.RowsAffected(),
		)
	}

	t.Log("update success")

	// 8. 再查一次确认 UPDATE
	err = pool.QueryRow(
		ctx,
		`
		SELECT
			name,
			status
		FROM zcode.pgx_crud_test
		WHERE id = $1
		`,
		id,
	).Scan(
		&name,
		&status,
	)

	if err != nil {
		t.Fatalf("select after update failed: %v", err)
	}

	if name != "Evolyn Updated" {
		t.Fatalf("update name not effective: %s", name)
	}

	if status != "published" {
		t.Fatalf("update status not effective: %s", status)
	}

	// 9. DELETE
	// tag, err = pool.Exec(
	// 	ctx,
	// 	`
	// 	DELETE FROM zcode.pgx_crud_test
	// 	WHERE id = $1
	// 	`,
	// 	id,
	// )

	// if err != nil {
	// 	t.Fatalf("delete failed: %v", err)
	// }

	// if tag.RowsAffected() != 1 {
	// 	t.Fatalf(
	// 		"expected delete 1 row, got %d",
	// 		tag.RowsAffected(),
	// 	)
	// }

	// t.Log("delete success")

	// 10. 验证删除
	// var count int

	// err = pool.QueryRow(
	// 	ctx,
	// 	`
	// 	SELECT COUNT(*)
	// 	FROM zcode.pgx_crud_test
	// 	WHERE id = $1
	// 	`,
	// 	id,
	// ).Scan(&count)

	// if err != nil {
	// 	t.Fatalf("count after delete failed: %v", err)
	// }

	// if count != 0 {
	// 	t.Fatalf("expected count 0, got %d", count)
	// }

	t.Log("pgx CRUD test passed")
}
