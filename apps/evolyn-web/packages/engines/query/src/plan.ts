import type { QueryDocument, QueryExpression, QueryLogicalPlan } from './types.js';

/**
 * 将已规范化且校验通过的 DSL 转换为存储无关逻辑计划。SQL、租户条件、字段
 * 物理映射与权限谓词由平台适配层负责，不能进入这个纯函数。
 */
export function buildQueryLogicalPlan(document: QueryDocument): QueryLogicalPlan {
  const complexity = measureExpression(document.filter);
  return Object.freeze({
    ...(document.filter ? { filter: document.filter } : {}),
    projection: Object.freeze([...(document.projection ?? [])]),
    groupBy: Object.freeze([...(document.groupBy ?? [])]),
    aggregates: Object.freeze([...(document.aggregates ?? [])]),
    sorts: Object.freeze([...document.sorts]),
    paging: Object.freeze({ ...document.paging }),
    aggregate: Boolean(document.groupBy?.length || document.aggregates?.length),
    complexity: Object.freeze(complexity),
  });
}

export function measureQueryExpression(
  expression: QueryExpression | undefined,
): QueryLogicalPlan['complexity'] {
  return measureExpression(expression);
}

function measureExpression(expression: QueryExpression | undefined): {
  depth: number;
  conditions: number;
} {
  if (!expression) return { depth: 0, conditions: 0 };
  if (expression.type === 'condition') return { depth: 1, conditions: 1 };
  const children = expression.children.map(measureExpression);
  return {
    depth: 1 + Math.max(0, ...children.map((child) => child.depth)),
    conditions: children.reduce((sum, child) => sum + child.conditions, 0),
  };
}
