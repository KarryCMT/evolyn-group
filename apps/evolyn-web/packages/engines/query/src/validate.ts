import {
  QUERY_DSL_VERSION,
  QUERY_OPERATORS_BY_FIELD_TYPE,
  type QueryCondition,
  type QueryDiagnostic,
  type QueryDocument,
  type QueryExpression,
  type QueryOperator,
  type QueryValidationOptions,
  type QueryValidationResult,
} from './types.js';
import { normalizeQuery } from './normalize.js';

const VALUELESS_OPERATORS = new Set<QueryOperator>(['isNull', 'isNotNull']);
const VALUE_OPERATORS = new Set<QueryOperator>([
  'eq', 'neq', 'contains', 'notContains', 'startsWith', 'endsWith', 'gt', 'gte', 'lt', 'lte', 'in', 'notIn', 'between',
]);

/** 校验 Query DSL 形状与可选字段类型约束；不注入权限，也不执行任何查询。 */
export function validateQuery(
  query: Partial<QueryDocument>,
  options: QueryValidationOptions = {},
): QueryValidationResult {
  const diagnostics: QueryDiagnostic[] = [];

  if (query.version !== undefined && query.version !== QUERY_DSL_VERSION) {
    diagnostics.push({
      code: 'QUERY_INVALID_VERSION',
      message: `仅支持 Query DSL v${QUERY_DSL_VERSION}。`,
      path: 'version',
    });
  }
  if (query.filter) validateExpression(query.filter, 'filter', options, diagnostics);
  query.sorts?.forEach((sort, index) => {
    if (!sort.field.trim() || !['asc', 'desc'].includes(sort.direction)) {
      diagnostics.push({ code: 'QUERY_INVALID_SORT', message: '排序字段或方向无效。', path: `sorts[${index}]` });
    }
  });
  query.projection?.forEach((field, index) => {
    if (!field.trim()) {
      diagnostics.push({ code: 'QUERY_INVALID_PROJECTION', message: '投影字段不能为空。', path: `projection[${index}]` });
    }
  });
  query.aggregates?.forEach((aggregate, index) => {
    if (!aggregate.field.trim() || !aggregate.alias.trim()) {
      diagnostics.push({ code: 'QUERY_INVALID_AGGREGATE', message: '聚合字段和别名不能为空。', path: `aggregates[${index}]` });
    }
  });

  return Object.freeze({
    document: diagnostics.length ? null : normalizeQuery(query),
    diagnostics: Object.freeze(diagnostics),
  });
}

/** 提供给字段设计器和适配器的操作符能力查询。 */
export function isQueryOperatorAllowed(
  fieldType: keyof typeof QUERY_OPERATORS_BY_FIELD_TYPE,
  operator: QueryOperator,
): boolean {
  return QUERY_OPERATORS_BY_FIELD_TYPE[fieldType].includes(operator);
}

function validateExpression(
  expression: QueryExpression,
  path: string,
  options: QueryValidationOptions,
  diagnostics: QueryDiagnostic[],
) {
  if (expression.type === 'group') {
    if (!expression.children.length) {
      diagnostics.push({ code: 'QUERY_EMPTY_GROUP', message: '条件组至少需要一个子条件。', path });
      return;
    }
    expression.children.forEach((child, index) =>
      validateExpression(child, `${path}.children[${index}]`, options, diagnostics),
    );
    return;
  }

  validateCondition(expression, path, options, diagnostics);
}

function validateCondition(
  condition: QueryCondition,
  path: string,
  options: QueryValidationOptions,
  diagnostics: QueryDiagnostic[],
) {
  if (!condition.field.trim()) {
    diagnostics.push({ code: 'QUERY_EMPTY_FIELD', message: '筛选字段不能为空。', path: `${path}.field` });
  }
  if (!VALUE_OPERATORS.has(condition.operator) && !VALUELESS_OPERATORS.has(condition.operator)) {
    diagnostics.push({ code: 'QUERY_INVALID_OPERATOR', message: '筛选操作符无效。', path: `${path}.operator` });
    return;
  }
  const fieldType = options.fieldTypes?.[condition.field];
  if (fieldType && !isQueryOperatorAllowed(fieldType, condition.operator)) {
    diagnostics.push({
      code: 'QUERY_OPERATOR_NOT_ALLOWED',
      message: `字段类型 ${fieldType} 不支持操作符 ${condition.operator}。`,
      path: `${path}.operator`,
    });
  }
  if (VALUELESS_OPERATORS.has(condition.operator)) return;

  if (condition.value === undefined || condition.value === null) {
    diagnostics.push({ code: 'QUERY_INVALID_VALUE', message: '该筛选操作符必须提供值。', path: `${path}.value` });
  } else if (
    (condition.operator === 'in' || condition.operator === 'notIn') &&
    (!Array.isArray(condition.value) || !condition.value.length)
  ) {
    diagnostics.push({ code: 'QUERY_INVALID_VALUE', message: '集合筛选至少需要一个值。', path: `${path}.value` });
  } else if (condition.operator === 'between' && (!Array.isArray(condition.value) || condition.value.length !== 2)) {
    diagnostics.push({ code: 'QUERY_INVALID_VALUE', message: '区间筛选必须提供两个值。', path: `${path}.value` });
  }
}
