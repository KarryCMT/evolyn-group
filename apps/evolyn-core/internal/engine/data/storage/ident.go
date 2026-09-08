// Package storage 表单物理存储纯模型（ADR：docs/低代码平台/表单设计器/
// 物理表存储后端实施方案.md §13）。
//
// 本包是物理值表的唯一权威模型层：StorageModel / Diff / Plan、控件→物理
// 类型映射与 DML 值编解码全部在此定义。铁律：
//   - 不得依赖 Gin、GORM、pgx、HTTP 或 Form/Workflow 域的具体模型；
//   - 只描述结构与语义，不生成 PostgreSQL 方言 SQL（DDL 语句由
//     infrastructure/dynamicddl 执行器翻译，DML 由 platform/form 仓储编译）；
//   - 所有表名/列名/索引名必须经本包标识符白名单校验后才能进入执行层。
package storage

import (
	"fmt"
	"regexp"
)

// 标识符白名单：动态表名（tn_fd_/tn_fc_ + 应用段 + 随机段）、列名（f_ + fieldId）
// 与索引名（ix_/ux_ + 表名 + 语义段）。只允许小写字母、数字、下划线，总长
// 不超过 PostgreSQL 63 字节标识符上限；服务端生成与消费两侧共用本规则。
var (
	dynamicTableNamePattern = regexp.MustCompile(`^tn_f[dc]_[0-9a-z]{4}_[0-9a-z]{6}$`)
	columnNamePattern       = regexp.MustCompile(`^f_[0-9a-z]{10}$`)
	fieldIDPattern          = regexp.MustCompile(`^[0-9a-z]{10}$`)
	indexNamePattern        = regexp.MustCompile(`^(ix|ux)_[0-9a-z_]+$`)
)

// MaxIdentifierLength PostgreSQL 标识符上限（NAMEDATALEN-1）。
const MaxIdentifierLength = 63

// MaxFieldIDLength fieldId 长度（10 位小写 base32，组合空间 ≈1.1e15，
// 单表单字段上限 500，碰撞概率可忽略；服务端与设计器生成侧同规则）。
const MaxFieldIDLength = 10

// ValidateFieldID 校验字段不可变标识（fieldId）。
func ValidateFieldID(id string) error {
	if len(id) != MaxFieldIDLength || !fieldIDPattern.MatchString(id) {
		return fmt.Errorf("fieldId %q 必须是 %d 位小写字母/数字", id, MaxFieldIDLength)
	}
	return nil
}

// ValidateDynamicTableName 校验动态物理表名（服务端分配，客户端永不提交）。
func ValidateDynamicTableName(name string) error {
	if len(name) > MaxIdentifierLength || !dynamicTableNamePattern.MatchString(name) {
		return fmt.Errorf("动态表名 %q 不符合 tn_f[dc]_<app4>_<rand6> 规则", name)
	}
	return nil
}

// ValidateColumnName 校验用户字段物理列名（f_ + fieldId，服务端从不可变
// 发布快照推导，禁止任何客户端输入直达）。
func ValidateColumnName(name string) error {
	if len(name) > MaxIdentifierLength || !columnNamePattern.MatchString(name) {
		return fmt.Errorf("物理列名 %q 不符合 f_<fieldId> 规则", name)
	}
	return nil
}

// ValidateIndexName 校验动态表索引名。
func ValidateIndexName(name string) error {
	if len(name) > MaxIdentifierLength || !indexNamePattern.MatchString(name) {
		return fmt.Errorf("索引名 %q 不符合命名白名单", name)
	}
	return nil
}

// IsParentTableName 报告表名是否为父数据表（tn_fd_ 前缀）。
func IsParentTableName(name string) bool {
	return len(name) > 6 && name[:6] == "tn_fd_"
}

// IsChildTableName 报告表名是否为子表单明细表（tn_fc_ 前缀）。
func IsChildTableName(name string) bool {
	return len(name) > 6 && name[:6] == "tn_fc_"
}
