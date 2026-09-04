/** 前端只描述可被后端解释的存储类型，不能据此拼接或执行 DDL。 */
export type PhysicalStorageType =
  | 'varchar'
  | 'text'
  | 'bigint'
  | 'numeric'
  | 'boolean'
  | 'date'
  | 'timestamptz'
  | 'uuid'
  | 'jsonb';

export type LogicalStorageFieldType =
  | 'shortText'
  | 'longText'
  | 'integer'
  | 'decimal'
  | 'boolean'
  | 'date'
  | 'datetime'
  | 'relation'
  | 'json'
  | 'file';

/** 领域层对字段字典的窄化投影；字段名称可修改，id 必须稳定。 */
export interface LogicalStorageField {
  id: string;
  type: LogicalStorageFieldType;
  required?: boolean;
  indexed?: boolean;
  unique?: boolean;
}

export interface PhysicalColumnDefinition {
  logicalFieldId: string;
  column: string;
  storageType: PhysicalStorageType;
  nullable: boolean;
}

export interface PhysicalIndexDefinition {
  name: string;
  columns: readonly string[];
  unique: boolean;
  method: 'btree';
}

/** 可供设计器预览、后端审查和迁移服务解释的物理模型快照。 */
export interface PhysicalModel {
  table: string;
  columns: readonly PhysicalColumnDefinition[];
  indexes: readonly PhysicalIndexDefinition[];
}

export type PhysicalMigrationOperation =
  | { type: 'addColumn'; column: PhysicalColumnDefinition }
  | { type: 'alterColumn'; before: PhysicalColumnDefinition; after: PhysicalColumnDefinition }
  | { type: 'deprecateColumn'; column: PhysicalColumnDefinition }
  | { type: 'addIndex'; index: PhysicalIndexDefinition }
  | { type: 'deprecateIndex'; index: PhysicalIndexDefinition };

/**
 * 迁移计划是声明式审阅产物：deprecate 表示后端可在确认后处理，绝不代表浏览器
 * 已删除列或索引。
 */
export interface PhysicalMigrationPlan {
  table: string;
  operations: readonly PhysicalMigrationOperation[];
}

export interface PhysicalDiagnostic {
  code: 'PHYSICAL_INVALID_IDENTIFIER' | 'PHYSICAL_DUPLICATE_COLUMN' | 'PHYSICAL_DUPLICATE_INDEX';
  message: string;
  path: string;
}
