import type {
  PhysicalIndexDefinition,
  PhysicalMigrationOperation,
  PhysicalMigrationPlan,
  PhysicalModel,
} from './types.js';

/**
 * 对比两个已审阅的模型快照并生成非执行性计划。移除字段或索引统一标记为
 * deprecate，避免浏览器侧把“设计器删除”误判为可立即 DROP。
 */
export function planPhysicalMigration(
  previous: PhysicalModel,
  next: PhysicalModel,
): PhysicalMigrationPlan {
  if (previous.table !== next.table) {
    throw new Error('Physical migration plans cannot change the target table.');
  }

  const operations: PhysicalMigrationOperation[] = [];
  const previousColumns = new Map(previous.columns.map((column) => [column.logicalFieldId, column]));
  const nextColumns = new Map(next.columns.map((column) => [column.logicalFieldId, column]));

  next.columns.forEach((column) => {
    const before = previousColumns.get(column.logicalFieldId);
    if (!before) operations.push({ type: 'addColumn', column });
    else if (!isSameColumn(before, column)) operations.push({ type: 'alterColumn', before, after: column });
  });
  previous.columns.forEach((column) => {
    if (!nextColumns.has(column.logicalFieldId)) operations.push({ type: 'deprecateColumn', column });
  });

  appendIndexOperations(previous.indexes, next.indexes, operations);
  return Object.freeze({ table: next.table, operations: Object.freeze(operations) });
}

function isSameColumn(
  left: PhysicalModel['columns'][number],
  right: PhysicalModel['columns'][number],
): boolean {
  return left.column === right.column && left.storageType === right.storageType && left.nullable === right.nullable;
}

function appendIndexOperations(
  previous: readonly PhysicalIndexDefinition[],
  next: readonly PhysicalIndexDefinition[],
  operations: PhysicalMigrationOperation[],
) {
  const previousIndexes = new Map(previous.map((index) => [index.name, index]));
  const nextIndexes = new Map(next.map((index) => [index.name, index]));
  next.forEach((index) => {
    const before = previousIndexes.get(index.name);
    if (!before || !isSameIndex(before, index)) operations.push({ type: 'addIndex', index });
  });
  previous.forEach((index) => {
    const after = nextIndexes.get(index.name);
    if (!after || !isSameIndex(index, after)) operations.push({ type: 'deprecateIndex', index });
  });
}

function isSameIndex(left: PhysicalIndexDefinition, right: PhysicalIndexDefinition): boolean {
  return left.unique === right.unique && left.method === right.method && left.columns.join('|') === right.columns.join('|');
}
