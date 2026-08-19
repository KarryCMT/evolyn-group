package infrastructure

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"

	"evolyn/migrations"

	"gorm.io/gorm"
)

// Migrator 版本化 SQL 迁移执行器（FIX-009）：migrations/ 目录是数据库
// Schema 唯一事实来源；每个 up 迁移在独立事务内执行并登记版本与校验和，
// 已应用文件被篡改（校验和不一致）时拒绝启动，防止环境间漂移。
// AutoMigrate 仅供开发/测试场景，不参与生产升级
type Migrator struct {
	db *gorm.DB
}

func NewMigrator(db *gorm.DB) *Migrator {
	return &Migrator{db: db}
}

// migrationFile 解析后的迁移文件（版本号 + 方向）
type migrationFile struct {
	version   int64
	name      string
	direction string // up / down
	path      string
	content   string
	checksum  string
}

var migrationNamePattern = regexp.MustCompile(`^(\d{6})_([a-z0-9_]+)\.(up|down)\.sql$`)

// Up 应用嵌入的全部未执行 up 迁移（启动路径）
func (m *Migrator) Up() error {
	return m.UpFS(migrations.FS)
}

// UpFS 对指定文件集执行迁移（嵌入 FS 为默认来源；测试可注入自定义集合）
func (m *Migrator) UpFS(fsys fs.FS) error {
	files, err := loadMigrations(fsys)
	if err != nil {
		return err
	}

	if err := m.ensureTable(); err != nil {
		return err
	}

	applied, err := m.appliedVersions()
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.direction != "up" {
			continue
		}
		if checksum, ok := applied[f.version]; ok {
			// 已应用：校验和防篡改（允许 down 文件无校验记录）
			if checksum != f.checksum {
				return fmt.Errorf("migration %s checksum mismatch: applied %s, file %s", f.path, checksum, f.checksum)
			}
			continue
		}

		if err := m.apply(f); err != nil {
			return fmt.Errorf("apply migration %s: %w", f.path, err)
		}
	}
	return nil
}

// apply 单个迁移：语句逐条在同一事务执行，成功后登记版本记录
func (m *Migrator) apply(f *migrationFile) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		for i, stmt := range SplitSQLStatements(f.content) {
			if err := tx.Exec(stmt).Error; err != nil {
				return fmt.Errorf("statement #%d: %w", i+1, err)
			}
		}
		return tx.Exec(
			"INSERT INTO schema_migrations (version, name, checksum) VALUES (?, ?, ?)",
			f.version, f.name, f.checksum,
		).Error
	})
}

func (m *Migrator) ensureTable() error {
	return m.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version BIGINT PRIMARY KEY,
		name TEXT NOT NULL,
		checksum TEXT NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`).Error
}

func (m *Migrator) appliedVersions() (map[int64]string, error) {
	rows := make([]struct {
		Version  int64
		Checksum string
	}, 0)
	if err := m.db.Raw("SELECT version, checksum FROM schema_migrations ORDER BY version").Scan(&rows).Error; err != nil {
		return nil, err
	}
	applied := make(map[int64]string, len(rows))
	for _, r := range rows {
		applied[r.Version] = r.Checksum
	}
	return applied, nil
}

// loadMigrations 读取并校验迁移文件集：命名规约 + up/down 成对 + 版本升序
func loadMigrations(fsys fs.FS) ([]*migrationFile, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}

	files := make([]*migrationFile, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		match := migrationNamePattern.FindStringSubmatch(e.Name())
		if match == nil {
			return nil, fmt.Errorf("migration file %s does not match NNNNNN_name.(up|down).sql", e.Name())
		}

		data, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return nil, err
		}
		version := int64(0)
		for _, c := range match[1] {
			version = version*10 + int64(c-'0')
		}

		sum := sha256.Sum256(data)
		files = append(files, &migrationFile{
			version:   version,
			name:      match[2],
			direction: match[3],
			path:      e.Name(),
			content:   string(data),
			checksum:  hex.EncodeToString(sum[:]),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].version != files[j].version {
			return files[i].version < files[j].version
		}
		return files[i].direction < files[j].direction
	})

	// up/down 成对校验
	byVersion := make(map[int64]map[string]int)
	for _, f := range files {
		if byVersion[f.version] == nil {
			byVersion[f.version] = map[string]int{}
		}
		byVersion[f.version][f.direction]++
	}
	for v, dirs := range byVersion {
		if dirs["up"] != 1 || dirs["down"] != 1 {
			return nil, fmt.Errorf("migration %06d must have exactly one up and one down file (got up=%d down=%d)", v, dirs["up"], dirs["down"])
		}
	}
	return files, nil
}

// dollarQuoteDelimiter 匹配 PostgreSQL 通用美元引用定界符（FIX-023）：
// $tag$ 或空 tag 的 $$，tag 为字母/下划线开头的标识符（$func$、$body$ 等）。
// 不匹配 $1 之类的参数占位符（后面不紧跟 $）
var dollarQuoteDelimiter = regexp.MustCompile(`^\$([A-Za-z_][A-Za-z0-9_]*)?\$`)

// SplitSQLStatements 把迁移脚本按顶层分号切分为可独立执行的语句：
// 识别 '--' 行注释、块注释、单引号字符串（” 转义）与通用 $tag$ 美元引用，
// 避免注释或字符串字面量中的分号被误判为语句边界
func SplitSQLStatements(script string) []string {
	var (
		statements []string
		current    strings.Builder
		i          = 0
		n          = len(script)
	)

	flush := func() {
		if s := strings.TrimSpace(current.String()); s != "" {
			statements = append(statements, s)
		}
		current.Reset()
	}

	for i < n {
		c := script[i]

		// 行注释：吞到行尾
		if c == '-' && i+1 < n && script[i+1] == '-' {
			for i < n && script[i] != '\n' {
				i++
			}
			current.WriteByte('\n')
			continue
		}

		// 块注释：吞到 */（未闭合到 EOF 则整体视为注释结束）
		if c == '/' && i+1 < n && script[i+1] == '*' {
			i += 2
			for i+1 < n && !(script[i] == '*' && script[i+1] == '/') {
				i++
			}
			i = min(i+2, n)
			current.WriteByte(' ')
			continue
		}

		// 单引号字符串：'' 转义
		if c == '\'' {
			current.WriteByte(c)
			i++
			for i < n {
				current.WriteByte(script[i])
				if script[i] == '\'' {
					if i+1 < n && script[i+1] == '\'' {
						current.WriteByte('\'')
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
			continue
		}

		// 美元引用 $tag$...$tag$（函数体等，FIX-023）：记录开定界符并原样
		// 吞入，仅遇到完全相同的结束定界符才退出——函数体内的分号与其他
		// tag（如嵌套出现的 $other$）都不是语句边界
		if c == '$' {
			if open := dollarQuoteDelimiter.FindString(script[i:]); open != "" {
				current.WriteString(open)
				i += len(open)
				for i < n && !strings.HasPrefix(script[i:], open) {
					current.WriteByte(script[i])
					i++
				}
				// 未闭合到 EOF 也补写闭合定界符，保持语句文本完整可辨
				current.WriteString(open)
				i = min(i+len(open), n)
				continue
			}
		}

		// 顶层分号：语句边界
		if c == ';' {
			flush()
			i++
			continue
		}

		current.WriteByte(c)
		i++
	}
	flush()

	return statements
}
