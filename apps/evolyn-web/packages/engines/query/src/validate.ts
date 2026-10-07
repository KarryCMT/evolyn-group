import {
  DEFAULT_QUERY_COMPLEXITY_BUDGET,
  QUERY_DSL_VERSION,
  QUERY_OPERATORS_BY_FIELD_TYPE,
  type QueryAggregateOperator,
  type QueryComplexityBudget,
  type QueryDiagnostic,
  type QueryDocument,
  type QueryFieldCapability,
  type QueryFieldType,
  type QueryOperator,
  type QueryValidationOptions,
  type QueryValidationResult,
} from './types.js';
import { normalizeQuery } from './normalize.js';

const VALUELESS_OPERATORS = new Set<QueryOperator>(['isNull', 'isNotNull']);
const VALUE_OPERATORS = new Set<QueryOperator>([
  'eq',
  'neq',
  'contains',
  'notContains',
  'startsWith',
  'endsWith',
  'gt',
  'gte',
  'lt',
  'lte',
  'in',
  'notIn',
  'between',
]);
const AGGREGATES = new Set<QueryAggregateOperator>(['count', 'sum', 'avg', 'min', 'max']);
const AGGREGATE_ALIAS = /^[A-Za-z][A-Za-z0-9_]{0,63}$/;

/** 校验 Query DSL 形状、字段能力与复杂度；不注入权限，也不执行任何查询。 */
export function validateQuery(
  input: unknown,
  options: QueryValidationOptions = {},
): QueryValidationResult {
  const diagnostics: QueryDiagnostic[] = [];
  if (!isRecord(input)) {
    add(diagnostics, '$', 'QUERY_DOCUMENT_INVALID', '查询文档必须是对象。');
    return freezeResult(null, diagnostics);
  }

  validateKnownKeys(
    input,
    new Set(['version', 'filter', 'sorts', 'paging', 'projection', 'groupBy', 'aggregates']),
    '$',
    diagnostics,
  );
  if (input.version !== QUERY_DSL_VERSION) {
    add(
      diagnostics,
      'version',
      'QUERY_INVALID_VERSION',
      `仅支持 Query DSL v${QUERY_DSL_VERSION}。`,
    );
  }

  const budget = resolveBudget(options.budget);
  const counter = { conditions: 0, depth: 0 };
  if (input.filter !== undefined) {
    validateExpression(input.filter, 'filter', 1, options, budget, counter, diagnostics);
  }

  const groupBy = validateFields(
    input.groupBy,
    'groupBy',
    'QUERY_INVALID_GROUP_BY',
    budget.maxGroupBy,
    diagnostics,
    (field, path) => validateFieldCapability(field, path, 'group', options, diagnostics),
  );
  const aggregateAliases = validateAggregates(input.aggregates, options, budget, diagnostics);
  validateSorts(
    input.sorts,
    options,
    budget,
    new Set([...groupBy, ...aggregateAliases]),
    diagnostics,
  );
  validateFields(
    input.projection,
    'projection',
    'QUERY_INVALID_PROJECTION',
    budget.maxProjection,
    diagnostics,
    (field, path) => {
      if (!aggregateAliases.has(field)) {
        validateFieldCapability(field, path, 'project', options, diagnostics);
      }
    },
  );
  validatePaging(input.paging, budget, diagnostics);

  if (counter.conditions > budget.maxConditions) {
    add(
      diagnostics,
      'filter',
      'QUERY_COMPLEXITY_EXCEEDED',
      `筛选条件不能超过 ${budget.maxConditions} 个。`,
    );
  }

  if (diagnostics.length) return freezeResult(null, diagnostics);
  return freezeResult(normalizeQuery(input as unknown as QueryDocument), diagnostics);
}

/** 提供给字段设计器和适配器的操作符能力查询。 */
export function isQueryOperatorAllowed(
  fieldType: keyof typeof QUERY_OPERATORS_BY_FIELD_TYPE,
  operator: QueryOperator,
): boolean {
  return QUERY_OPERATORS_BY_FIELD_TYPE[fieldType].includes(operator);
}

function validateExpression(
  input: unknown,
  path: string,
  depth: number,
  options: QueryValidationOptions,
  budget: QueryComplexityBudget,
  counter: { conditions: number; depth: number },
  diagnostics: QueryDiagnostic[],
) {
  counter.depth = Math.max(counter.depth, depth);
  if (depth > budget.maxDepth) {
    add(diagnostics, path, 'QUERY_COMPLEXITY_EXCEEDED', `筛选嵌套不能超过 ${budget.maxDepth} 层。`);
    return;
  }
  if (!isRecord(input)) {
    add(diagnostics, path, 'QUERY_INVALID_EXPRESSION', '筛选表达式必须是对象。');
    return;
  }
  if (input.type === 'group') {
    validateKnownKeys(input, new Set(['type', 'conjunction', 'children']), path, diagnostics);
    if (input.conjunction !== 'and' && input.conjunction !== 'or') {
      add(
        diagnostics,
        `${path}.conjunction`,
        'QUERY_INVALID_EXPRESSION',
        '条件组连接词必须是 and 或 or。',
      );
    }
    if (!Array.isArray(input.children) || input.children.length === 0) {
      add(diagnostics, path, 'QUERY_EMPTY_GROUP', '条件组至少需要一个子条件。');
      return;
    }
    input.children.forEach((child, index) =>
      validateExpression(
        child,
        `${path}.children[${index}]`,
        depth + 1,
        options,
        budget,
        counter,
        diagnostics,
      ),
    );
    return;
  }
  if (input.type !== 'condition') {
    add(diagnostics, `${path}.type`, 'QUERY_INVALID_EXPRESSION', '未知的筛选表达式类型。');
    return;
  }

  counter.conditions += 1;
  validateKnownKeys(input, new Set(['type', 'field', 'operator', 'value']), path, diagnostics);
  const field = readTrimmedString(input.field);
  if (!field) add(diagnostics, `${path}.field`, 'QUERY_EMPTY_FIELD', '筛选字段不能为空。');

  const operator = typeof input.operator === 'string' ? input.operator : '';
  if (
    !VALUE_OPERATORS.has(operator as QueryOperator) &&
    !VALUELESS_OPERATORS.has(operator as QueryOperator)
  ) {
    add(diagnostics, `${path}.operator`, 'QUERY_INVALID_OPERATOR', '筛选操作符无效。');
    return;
  }
  if (field) {
    const capability = resolveCapability(field, options, `${path}.field`, diagnostics);
    if (
      capability &&
      (!capability.filterable ||
        !isQueryOperatorAllowed(capability.type, operator as QueryOperator))
    ) {
      add(
        diagnostics,
        `${path}.operator`,
        'QUERY_OPERATOR_NOT_ALLOWED',
        `字段类型 ${capability.type} 不支持操作符 ${operator}。`,
      );
    }
  }
  validateConditionValue(operator as QueryOperator, input.value, `${path}.value`, diagnostics);
}

function validateConditionValue(
  operator: QueryOperator,
  value: unknown,
  path: string,
  diagnostics: QueryDiagnostic[],
) {
  if (VALUELESS_OPERATORS.has(operator)) {
    if (value !== undefined)
      add(diagnostics, path, 'QUERY_INVALID_VALUE', '空值操作符不得携带值。');
    return;
  }
  if (value === undefined || value === null || !isQueryValue(value)) {
    add(diagnostics, path, 'QUERY_INVALID_VALUE', '该筛选操作符必须提供合法值。');
    return;
  }
  if (
    (operator === 'in' || operator === 'notIn') &&
    (!Array.isArray(value) || value.length === 0)
  ) {
    add(diagnostics, path, 'QUERY_INVALID_VALUE', '集合筛选至少需要一个值。');
  } else if (operator === 'between' && (!Array.isArray(value) || value.length !== 2)) {
    add(diagnostics, path, 'QUERY_INVALID_VALUE', '区间筛选必须提供两个值。');
  }
}

function validateSorts(
  input: unknown,
  options: QueryValidationOptions,
  budget: QueryComplexityBudget,
  outputFields: ReadonlySet<string>,
  diagnostics: QueryDiagnostic[],
) {
  if (!Array.isArray(input)) {
    add(diagnostics, 'sorts', 'QUERY_INVALID_SORT', 'sorts 必须是数组。');
    return;
  }
  if (input.length > budget.maxSorts) {
    add(
      diagnostics,
      'sorts',
      'QUERY_COMPLEXITY_EXCEEDED',
      `排序字段不能超过 ${budget.maxSorts} 个。`,
    );
  }
  input.forEach((item, index) => {
    const path = `sorts[${index}]`;
    if (!isRecord(item)) {
      add(diagnostics, path, 'QUERY_INVALID_SORT', '排序必须是对象。');
      return;
    }
    validateKnownKeys(item, new Set(['field', 'direction']), path, diagnostics);
    const field = readTrimmedString(item.field);
    if (!field || (item.direction !== 'asc' && item.direction !== 'desc')) {
      add(diagnostics, path, 'QUERY_INVALID_SORT', '排序字段或方向无效。');
      return;
    }
    if (!outputFields.has(field)) {
      validateFieldCapability(field, `${path}.field`, 'sort', options, diagnostics);
    }
  });
}

function validateFields(
  input: unknown,
  path: 'projection' | 'groupBy',
  invalidCode: 'QUERY_INVALID_PROJECTION' | 'QUERY_INVALID_GROUP_BY',
  maximum: number,
  diagnostics: QueryDiagnostic[],
  validateCapability: (field: string, path: string) => void,
): Set<string> {
  if (input === undefined) return new Set();
  if (!Array.isArray(input)) {
    add(diagnostics, path, invalidCode, `${path} 必须是数组。`);
    return new Set();
  }
  if (input.length > maximum) {
    add(diagnostics, path, 'QUERY_COMPLEXITY_EXCEEDED', `${path} 不能超过 ${maximum} 项。`);
  }
  const seen = new Set<string>();
  input.forEach((value, index) => {
    const field = readTrimmedString(value);
    const itemPath = `${path}[${index}]`;
    if (!field || seen.has(field)) {
      add(diagnostics, itemPath, invalidCode, '字段不能为空或重复。');
      return;
    }
    seen.add(field);
    validateCapability(field, itemPath);
  });
  return seen;
}

function validateAggregates(
  input: unknown,
  options: QueryValidationOptions,
  budget: QueryComplexityBudget,
  diagnostics: QueryDiagnostic[],
): Set<string> {
  if (input === undefined) return new Set();
  if (!Array.isArray(input)) {
    add(diagnostics, 'aggregates', 'QUERY_INVALID_AGGREGATE', 'aggregates 必须是数组。');
    return new Set();
  }
  if (input.length > budget.maxAggregates) {
    add(
      diagnostics,
      'aggregates',
      'QUERY_COMPLEXITY_EXCEEDED',
      `聚合指标不能超过 ${budget.maxAggregates} 个。`,
    );
  }
  const aliases = new Set<string>();
  input.forEach((item, index) => {
    const path = `aggregates[${index}]`;
    if (!isRecord(item)) {
      add(diagnostics, path, 'QUERY_INVALID_AGGREGATE', '聚合必须是对象。');
      return;
    }
    validateKnownKeys(item, new Set(['field', 'operator', 'alias']), path, diagnostics);
    const field = readTrimmedString(item.field);
    const alias = readTrimmedString(item.alias);
    const operator = typeof item.operator === 'string' ? item.operator : '';
    if (
      !field ||
      !alias ||
      !AGGREGATE_ALIAS.test(alias) ||
      !AGGREGATES.has(operator as QueryAggregateOperator)
    ) {
      add(diagnostics, path, 'QUERY_INVALID_AGGREGATE', '聚合字段、函数或别名无效。');
      return;
    }
    if (aliases.has(alias)) {
      add(diagnostics, `${path}.alias`, 'QUERY_DUPLICATE_ALIAS', '聚合别名不能重复。');
    } else {
      aliases.add(alias);
    }
    const capability = resolveCapability(field, options, `${path}.field`, diagnostics);
    if (capability && !capability.aggregates.includes(operator as QueryAggregateOperator)) {
      add(
        diagnostics,
        `${path}.operator`,
        'QUERY_AGGREGATE_NOT_ALLOWED',
        `字段 ${field} 不支持聚合 ${operator}。`,
      );
    }
  });
  return aliases;
}

function validatePaging(
  input: unknown,
  budget: QueryComplexityBudget,
  diagnostics: QueryDiagnostic[],
) {
  if (!isRecord(input)) {
    add(diagnostics, 'paging', 'QUERY_INVALID_PAGING', 'paging 必须是对象。');
    return;
  }
  validateKnownKeys(input, new Set(['page', 'pageSize']), 'paging', diagnostics);
  if (!isPositiveInteger(input.page)) {
    add(diagnostics, 'paging.page', 'QUERY_INVALID_PAGING', '页码必须是正整数。');
  }
  if (!isPositiveInteger(input.pageSize)) {
    add(diagnostics, 'paging.pageSize', 'QUERY_INVALID_PAGING', '每页数量必须是正整数。');
  } else if (input.pageSize > budget.maxPageSize) {
    add(
      diagnostics,
      'paging.pageSize',
      'QUERY_COMPLEXITY_EXCEEDED',
      `每页数量不能超过 ${budget.maxPageSize}。`,
    );
  }
}

function validateFieldCapability(
  field: string,
  path: string,
  operation: 'project' | 'group' | 'sort',
  options: QueryValidationOptions,
  diagnostics: QueryDiagnostic[],
) {
  const capability = resolveCapability(field, options, path, diagnostics);
  if (!capability) return;
  const allowed =
    operation === 'project'
      ? capability.projectable
      : operation === 'group'
        ? capability.groupable
        : capability.sortable;
  if (allowed) return;
  const code =
    operation === 'project'
      ? 'QUERY_FIELD_NOT_PROJECTABLE'
      : operation === 'group'
        ? 'QUERY_FIELD_NOT_GROUPABLE'
        : 'QUERY_FIELD_NOT_SORTABLE';
  add(diagnostics, path, code, `字段 ${field} 不支持该查询能力。`);
}

function resolveCapability(
  field: string,
  options: QueryValidationOptions,
  path: string,
  diagnostics: QueryDiagnostic[],
): QueryFieldCapability | null {
  if (options.fields) {
    const capability = options.fields[field];
    if (!capability) {
      add(diagnostics, path, 'QUERY_UNKNOWN_FIELD', `字段 ${field} 不在字段目录中。`);
      return null;
    }
    return capability;
  }
  const type = options.fieldTypes?.[field];
  if (!type) return null;
  return permissiveCapability(type);
}

function permissiveCapability(type: QueryFieldType): QueryFieldCapability {
  const numeric = type === 'number' || type === 'decimal';
  return {
    type,
    filterable: true,
    sortable: true,
    projectable: true,
    groupable: true,
    aggregates: numeric ? ['count', 'sum', 'avg', 'min', 'max'] : ['count', 'min', 'max'],
  };
}

function resolveBudget(input: Partial<QueryComplexityBudget> | undefined): QueryComplexityBudget {
  return { ...DEFAULT_QUERY_COMPLEXITY_BUDGET, ...input };
}

function validateKnownKeys(
  value: Record<string, unknown>,
  allowed: ReadonlySet<string>,
  path: string,
  diagnostics: QueryDiagnostic[],
) {
  for (const key of Object.keys(value)) {
    if (!allowed.has(key)) {
      add(
        diagnostics,
        path === '$' ? key : `${path}.${key}`,
        'QUERY_DOCUMENT_INVALID',
        '查询文档包含未受支持的配置。',
      );
    }
  }
}

function isQueryValue(value: unknown): boolean {
  if (typeof value === 'string' || typeof value === 'boolean') return true;
  if (typeof value === 'number') return Number.isFinite(value);
  return Array.isArray(value) && value.every((item) => isQueryScalar(item));
}

function isQueryScalar(value: unknown): boolean {
  return (
    value === null ||
    typeof value === 'string' ||
    typeof value === 'boolean' ||
    (typeof value === 'number' && Number.isFinite(value))
  );
}

function readTrimmedString(value: unknown): string {
  return typeof value === 'string' ? value.trim() : '';
}

function isPositiveInteger(value: unknown): value is number {
  return typeof value === 'number' && Number.isInteger(value) && value > 0;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function add(
  diagnostics: QueryDiagnostic[],
  path: string,
  code: QueryDiagnostic['code'],
  message: string,
) {
  diagnostics.push({ path, code, message });
}

function freezeResult(
  document: QueryDocument | null,
  diagnostics: QueryDiagnostic[],
): QueryValidationResult {
  return Object.freeze({ document, diagnostics: Object.freeze(diagnostics) });
}
