// Package dynamicddl PostgreSQL 动态表 DDL 执行器（方案 §12/§13）。
//
// 唯一允许执行 CREATE/ALTER 动态物理表的层：Controller 与 Service 只能提交
// engine/data/storage 产出的结构化 Plan，本执行器负责标识符白名单复核、
// 引用转义、advisory lock、RLS 安装与中文注释。禁止删列/删索引/窄化类型等
// 破坏性操作——Plan 动作枚举已在模型层封闭。
package dynamicddl

import (
	"context"
	"fmt"
	"strings"

	"evolyn/internal/engine/data/storage"
	"evolyn/internal/infrastructure"

	"gorm.io/gorm"
)

// Executor 动态 DDL 执行器：所有方法必须在调用方事务 ctx 内执行
// （经 infrastructure.ResolveDB 加入 TxManager 传播事务），保证
// 「领取 Job → 执行 DDL → 回写元数据」同事务、crash 自动整体回滚。
type Executor struct {
	db *gorm.DB
}

// NewExecutor 构造执行器（db 为基连接；实际语句经 ResolveDB 绑定 ctx 事务）。
func NewExecutor(db *gorm.DB) *Executor { return &Executor{db: db} }

// TableCommentContext 动态表注释上下文（运维定位用，不含业务值）。
type TableCommentContext struct {
	TenantID  uint
	StorageID uint
	FormCode  string
}

// QuoteIdentifier 标识符引用（白名单字符集下等价于原文包裹；双层防线，
// 任何未经白名单校验的标识符在此仍会被拒绝）。
func QuoteIdentifier(name string) (string, error) {
	if err := validateIdentifierChars(name); err != nil {
		return "", err
	}
	return `"` + name + `"`, nil
}

// validateIdentifierChars 动态 DDL 涉及的标识符统一字符白名单。
func validateIdentifierChars(name string) error {
	if name == "" || len(name) > storage.MaxIdentifierLength {
		return fmt.Errorf("标识符 %q 长度非法", name)
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return fmt.Errorf("标识符 %q 含非法字符", name)
	}
	if name[0] >= '0' && name[0] <= '9' {
		return fmt.Errorf("标识符 %q 不得以数字开头", name)
	}
	return nil
}

// AcquireAdvisoryLock 在当前事务内获取表单级咨询锁（key 为租户+存储的
// 稳定复合键），防止并发发布/重试对同一物理表交错执行 DDL。事务提交或
// 回滚自动释放，绝不显式解锁。
func (e *Executor) AcquireAdvisoryLock(ctx context.Context, tenantID, storageID uint) error {
	key := advisoryLockKey(tenantID, storageID)
	return infrastructure.ResolveDB(ctx, e.db).Exec(
		"SELECT pg_advisory_xact_lock(?)", key,
	).Error
}

func advisoryLockKey(tenantID, storageID uint) int64 {
	return int64(tenantID)<<32 ^ int64(storageID)&0xFFFFFFFF
}

// ExecutePlan 执行结构计划：按动作封闭翻译为受控 DDL。所有语句在当前事务
// 内逐条执行，任一失败由调用方整体回滚（crash recovery 与重试幂等由
// Job 状态机保证：失败回 PENDING 重新执行，已存在对象按 IF NOT EXISTS 收敛）。
func (e *Executor) ExecutePlan(ctx context.Context, plan storage.Plan, model *storage.StorageModel, commentCtx TableCommentContext) error {
	if err := model.Validate(); err != nil {
		return fmt.Errorf("动态 DDL 拒绝非法模型: %w", err)
	}
	db := infrastructure.ResolveDB(ctx, e.db)
	for _, action := range plan.Actions {
		var err error
		switch action.Kind {
		case storage.ActionCreateParent:
			err = e.createParentTable(db, model, commentCtx)
		case storage.ActionAddParentColumn:
			err = e.addColumn(db, action.Table, *action.Column, false)
		case storage.ActionCreateChild:
			err = e.createChildTable(db, action.Table, *action.Child, commentCtx)
		case storage.ActionAddChildColumn:
			err = e.addColumn(db, action.Table, *action.Column, true)
		default:
			err = fmt.Errorf("未知 DDL 动作 %q", action.Kind)
		}
		if err != nil {
			return fmt.Errorf("执行 DDL 动作 %s(%s) 失败: %w", action.Kind, action.Table, err)
		}
	}
	return nil
}

// createParentTable 首次建父表：预置系统列 + 用户列 + 复合外键 + 预置索引
// + RLS。表为空，索引瞬时完成；并发索引（CONCURRENTLY）属 Phase 5 大表
// 场景，首期不涉及。
func (e *Executor) createParentTable(db *gorm.DB, model *storage.StorageModel, commentCtx TableCommentContext) error {
	table, err := QuoteIdentifier(model.TableName)
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("CREATE TABLE IF NOT EXISTS " + table + " (\n")
	sb.WriteString("    tenant_id BIGINT NOT NULL,\n")
	sb.WriteString("    record_id BIGINT NOT NULL,\n")
	sb.WriteString("    workflow_instance_no TEXT,\n")
	sb.WriteString("    workflow_status TEXT NOT NULL DEFAULT 'NONE',\n")
	sb.WriteString("    workflow_updated_at TIMESTAMP,\n")
	for _, column := range model.Columns {
		def, err := columnDefinition(column)
		if err != nil {
			return err
		}
		sb.WriteString("    " + def + ",\n")
	}
	pkName, err := QuoteIdentifier("pk_" + model.TableName)
	if err != nil {
		return err
	}
	fkName, err := QuoteIdentifier("fk_" + model.TableName + "_record")
	if err != nil {
		return err
	}
	sb.WriteString("    CONSTRAINT " + pkName + " PRIMARY KEY (record_id),\n")
	sb.WriteString("    CONSTRAINT " + fkName +
		" FOREIGN KEY (tenant_id, record_id) REFERENCES tn_form_records (tenant_id, id)\n")
	sb.WriteString(")")
	if err := db.Exec(sb.String()).Error; err != nil {
		return err
	}

	// 预置索引：流程状态复合（筛选/keyset 导出）与单号部分索引（方案 §5.3）。
	statusIndex, err := QuoteIdentifier("ix_" + model.TableName + "_workflow_status")
	if err != nil {
		return err
	}
	if err := db.Exec(fmt.Sprintf(
		"CREATE INDEX IF NOT EXISTS %s ON %s (tenant_id, workflow_status, workflow_updated_at DESC, record_id DESC)",
		statusIndex, table)).Error; err != nil {
		return err
	}
	noIndex, err := QuoteIdentifier("ix_" + model.TableName + "_workflow_no")
	if err != nil {
		return err
	}
	if err := db.Exec(fmt.Sprintf(
		"CREATE INDEX IF NOT EXISTS %s ON %s (tenant_id, workflow_instance_no) WHERE workflow_instance_no IS NOT NULL",
		noIndex, table)).Error; err != nil {
		return err
	}

	if err := installRLS(db, model.TableName); err != nil {
		return err
	}
	return db.Exec(fmt.Sprintf(
		"COMMENT ON TABLE %s IS '表单物理值表：storage_id=%d form=%s tenant_id=%d（记录信封见 tn_form_records，本表仅存业务字段值与流程查询投影）'",
		table, commentCtx.StorageID, commentCtx.FormCode, commentCtx.TenantID)).Error
}

// createChildTable 首次建子表单明细表（方案 §5.4）：预置行信封列 + 子字段列
// + 父记录复合外键 + 顺序唯一 + 父记录索引 + RLS。
func (e *Executor) createChildTable(db *gorm.DB, tableName string, child storage.ChildTableSpec, commentCtx TableCommentContext) error {
	table, err := QuoteIdentifier(tableName)
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("CREATE TABLE IF NOT EXISTS " + table + " (\n")
	sb.WriteString("    tenant_id BIGINT NOT NULL,\n")
	sb.WriteString("    row_id BIGSERIAL,\n")
	sb.WriteString("    parent_record_id BIGINT NOT NULL,\n")
	sb.WriteString("    sort_order INTEGER NOT NULL,\n")
	sb.WriteString("    row_revision BIGINT NOT NULL DEFAULT 1,\n")
	for _, column := range child.Columns {
		def, err := columnDefinition(column)
		if err != nil {
			return err
		}
		sb.WriteString("    " + def + ",\n")
	}
	sb.WriteString("    created_at TIMESTAMP NOT NULL DEFAULT LOCALTIMESTAMP,\n")
	sb.WriteString("    updated_at TIMESTAMP NOT NULL DEFAULT LOCALTIMESTAMP,\n")
	fkName, err := QuoteIdentifier("fk_" + tableName + "_parent")
	if err != nil {
		return err
	}
	uqName, err := QuoteIdentifier("uq_" + tableName + "_sort")
	if err != nil {
		return err
	}
	pkName, err := QuoteIdentifier("pk_" + tableName)
	if err != nil {
		return err
	}
	sb.WriteString("    CONSTRAINT " + pkName + " PRIMARY KEY (row_id),\n")
	sb.WriteString("    CONSTRAINT " + fkName +
		" FOREIGN KEY (tenant_id, parent_record_id) REFERENCES tn_form_records (tenant_id, id),\n")
	sb.WriteString("    CONSTRAINT " + uqName + " UNIQUE (parent_record_id, sort_order)\n")
	sb.WriteString(")")
	if err := db.Exec(sb.String()).Error; err != nil {
		return err
	}

	parentIndex, err := QuoteIdentifier("ix_" + tableName + "_parent")
	if err != nil {
		return err
	}
	if err := db.Exec(fmt.Sprintf(
		"CREATE INDEX IF NOT EXISTS %s ON %s (tenant_id, parent_record_id, sort_order)",
		parentIndex, table)).Error; err != nil {
		return err
	}

	if err := installRLS(db, tableName); err != nil {
		return err
	}
	return db.Exec(fmt.Sprintf(
		"COMMENT ON TABLE %s IS '子表单明细表：storage_id=%d form=%s tenant_id=%d（集合替换语义：随父记录同事务整批重写）'",
		table, commentCtx.StorageID, commentCtx.FormCode, commentCtx.TenantID)).Error
}

// addColumn 加用户列（父表/子表通用）：用户列始终可空（required 是提交校验
// 语义，历史行没有值），并写入中文列注释（widgetName+fieldId 不可变标识，
// 不固化可变 label）。
func (e *Executor) addColumn(db *gorm.DB, tableName string, column storage.ColumnSpec, child bool) error {
	if child {
		if err := storage.ValidateDynamicTableName(tableName); err != nil {
			return err
		}
	} else if err := storage.ValidateDynamicTableName(tableName); err != nil {
		return err
	}
	def, err := columnDefinition(column)
	if err != nil {
		return err
	}
	table, err := QuoteIdentifier(tableName)
	if err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE " + table + " ADD COLUMN IF NOT EXISTS " + def).Error; err != nil {
		return err
	}
	name, err := QuoteIdentifier(column.ColumnName())
	if err != nil {
		return err
	}
	return db.Exec(fmt.Sprintf(
		"COMMENT ON COLUMN %s.%s IS '表单字段 %s（fieldId %s，类型 %s）'",
		table, name, column.WidgetName, column.FieldID, column.WidgetType)).Error
}

// columnDefinition 用户列定义片段（含列注释所需的不可变标识信息在 addColumn
// 单独写；此处只产出 DDL 片段）。
func columnDefinition(column storage.ColumnSpec) (string, error) {
	if err := storage.ValidateColumnName(column.ColumnName()); err != nil {
		return "", err
	}
	if storage.ColumnTypeOf(column.Kind) != column.Type {
		return "", fmt.Errorf("字段 %s 类型与语义不一致", column.WidgetName)
	}
	name, err := QuoteIdentifier(column.ColumnName())
	if err != nil {
		return "", err
	}
	return name + " " + string(column.Type), nil
}

// installRLS 启用并强制行级安全，策略以 app.current_tenant 会话变量为准；
// 未设置（NULLIF 空串）时比较结果为 NULL → 全行不可见（fail-closed），
// 与 GORM 租户 Callback 的应用层隔离形成纵深防御（方案 §12）。
// CREATE POLICY 无 IF NOT EXISTS 语法，幂等经 pg_policy 存在性判定实现
// （Job 重试路径可能对已装策略的表重复执行）。
func installRLS(db *gorm.DB, tableName string) error {
	if err := storage.ValidateDynamicTableName(tableName); err != nil {
		return err
	}
	table, err := QuoteIdentifier(tableName)
	if err != nil {
		return err
	}
	policyName := "p_" + tableName + "_tenant"
	quotedPolicy, err := QuoteIdentifier(policyName)
	if err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE " + table + " ENABLE ROW LEVEL SECURITY").Error; err != nil {
		return err
	}
	if err := db.Exec("ALTER TABLE " + table + " FORCE ROW LEVEL SECURITY").Error; err != nil {
		return err
	}
	// tableName 已通过白名单（仅小写字母/数字/下划线），嵌入字符串字面量安全；
	// plpgsql 块内直接执行 DDL（$$ 美元引用内单引号无需转义）。
	predicate := "tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::BIGINT"
	return db.Exec(fmt.Sprintf(`DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policy WHERE polname = '%s' AND polrelid = '%s'::regclass) THEN
        CREATE POLICY %s ON %s USING (%s) WITH CHECK (%s);
    END IF;
END $$;`, policyName, tableName, quotedPolicy, table, predicate, predicate)).Error
}
