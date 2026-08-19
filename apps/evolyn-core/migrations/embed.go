// Package migrations 数据库版本化迁移（FIX-009）：本目录是 Schema 唯一事实
// 来源，SQL 文件随代码编译嵌入（生产单二进制即可升级）。命名规约
// NNNNNN_name.(up|down).sql，版本号只增不复用。
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
