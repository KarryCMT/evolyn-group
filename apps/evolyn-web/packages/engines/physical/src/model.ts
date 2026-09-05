import type {
  LogicalStorageField,
  LogicalStorageFieldType,
  PhysicalColumnDefinition,
  PhysicalDiagnostic,
  PhysicalIndexDefinition,
  PhysicalModel,
  PhysicalStorageType,
} from './types.js';

const STORAGE_TYPES: Readonly<Record<LogicalStorageFieldType, PhysicalStorageType>> = {
  shortText: 'varchar',
  longText: 'text',
  integer: 'bigint',
  decimal: 'numeric',
  boolean: 'boolean',
  date: 'date',
  datetime: 'timestamptz',
  relation: 'uuid',
  json: 'jsonb',
  file: 'jsonb',
};

/**
 * 以稳定 field id 推导列名，而非使用可变显示名称。field_01J82 永远映射为
 * f_01j82，字段改名不会触发物理列改名。
 */
export function derivePhysicalColumnName(fieldId: string): string {
  const normalized = fieldId
    .trim()
    .toLowerCase()
    .replace(/^field_/, '');
  // 列名前缀始终是 f_，因此 field id 去前缀后的稳定段可合法地以数字开头。
  if (!/^[a-z0-9][a-z0-9_]*$/.test(normalized)) {
    throw new Error(`Invalid logical field id: ${fieldId}`);
  }
  return `f_${normalized}`;
}

/** 建立不可变的物理模型快照，供设计器预览或后端迁移服务审阅。 */
export function createPhysicalModel(
  table: string,
  fields: readonly LogicalStorageField[],
): PhysicalModel {
  const normalizedTable = normalizeIdentifier(table, 'table');
  const columns = fields.map(toPhysicalColumn);
  const indexes = fields.flatMap((field) => toIndexes(normalizedTable, field));
  const diagnostics = validatePhysicalModel({ table: normalizedTable, columns, indexes });
  if (diagnostics.length)
    throw new Error(diagnostics.map((diagnostic) => diagnostic.message).join(' '));

  return Object.freeze({
    table: normalizedTable,
    columns: Object.freeze(columns),
    indexes: Object.freeze(indexes),
  });
}

/** 校验来自外部系统或历史快照的物理模型，读取侧不必依赖创建入口。 */
export function validatePhysicalModel(model: PhysicalModel): readonly PhysicalDiagnostic[] {
  const diagnostics: PhysicalDiagnostic[] = [];
  if (!isIdentifier(model.table)) {
    diagnostics.push({
      code: 'PHYSICAL_INVALID_IDENTIFIER',
      message: '物理表名必须是小写字母、数字和下划线组成的标识符。',
      path: 'table',
    });
  }
  collectDuplicates(
    model.columns.map((column) => column.column),
    'columns',
    'PHYSICAL_DUPLICATE_COLUMN',
    diagnostics,
  );
  collectDuplicates(
    model.indexes.map((index) => index.name),
    'indexes',
    'PHYSICAL_DUPLICATE_INDEX',
    diagnostics,
  );
  return Object.freeze(diagnostics);
}

function toPhysicalColumn(field: LogicalStorageField): PhysicalColumnDefinition {
  return {
    logicalFieldId: field.id,
    column: derivePhysicalColumnName(field.id),
    storageType: STORAGE_TYPES[field.type],
    nullable: !field.required,
  };
}

function toIndexes(table: string, field: LogicalStorageField): PhysicalIndexDefinition[] {
  if (!field.indexed && !field.unique) return [];
  const column = derivePhysicalColumnName(field.id);
  return [
    {
      name: `idx_${table}_${column}`,
      columns: [column],
      unique: Boolean(field.unique),
      method: 'btree',
    },
  ];
}

function normalizeIdentifier(value: string, kind: string): string {
  const normalized = value.trim().toLowerCase();
  if (!isIdentifier(normalized)) throw new Error(`Invalid physical ${kind} identifier: ${value}`);
  return normalized;
}

function isIdentifier(value: string): boolean {
  return /^[a-z][a-z0-9_]*$/.test(value);
}

function collectDuplicates(
  values: readonly string[],
  path: string,
  code: Extract<
    PhysicalDiagnostic['code'],
    'PHYSICAL_DUPLICATE_COLUMN' | 'PHYSICAL_DUPLICATE_INDEX'
  >,
  diagnostics: PhysicalDiagnostic[],
) {
  const seen = new Set<string>();
  values.forEach((value, index) => {
    if (seen.has(value)) {
      diagnostics.push({ code, message: `物理标识符 ${value} 重复。`, path: `${path}[${index}]` });
      return;
    }
    seen.add(value);
  });
}
