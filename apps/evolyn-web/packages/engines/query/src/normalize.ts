import {
  QUERY_DSL_VERSION,
  type QueryAggregate,
  type QueryDocument,
  type QueryExpression,
  type QueryPaging,
  type QuerySort,
} from './types.js';

const DEFAULT_PAGING: QueryPaging = { page: 1, pageSize: 20 };

/** 将部分查询转换为稳定、可缓存和可序列化的 DSL v1 文档。 */
export function normalizeQuery(query: Partial<QueryDocument> = {}): QueryDocument {
  const filter = query.filter ? normalizeExpression(query.filter) : undefined;
  const projection = normalizeFields(query.projection);
  const groupBy = normalizeFields(query.groupBy);
  const aggregates = normalizeAggregates(query.aggregates);

  return {
    version: QUERY_DSL_VERSION,
    ...(filter ? { filter } : {}),
    sorts: normalizeSorts(query.sorts),
    paging: normalizePaging(query.paging),
    ...(projection.length ? { projection } : {}),
    ...(groupBy.length ? { groupBy } : {}),
    ...(aggregates.length ? { aggregates } : {}),
  };
}

/** 生成稳定 JSON，供请求去重、缓存键或审计快照使用。 */
export function serializeQuery(query: Partial<QueryDocument> = {}): string {
  return JSON.stringify(normalizeQuery(query));
}

function normalizeExpression(expression: QueryExpression): QueryExpression | undefined {
  if (expression.type === 'condition') {
    const field = expression.field.trim();
    return field ? { ...expression, field } : undefined;
  }

  const children = expression.children
    .map(normalizeExpression)
    .filter((child): child is QueryExpression => Boolean(child));
  return children.length
    ? { type: 'group', conjunction: expression.conjunction, children }
    : undefined;
}

function normalizeSorts(sorts: readonly QuerySort[] | undefined): readonly QuerySort[] {
  return (sorts ?? [])
    .map((sort) => ({ ...sort, field: sort.field.trim() }))
    .filter((sort) => Boolean(sort.field));
}

function normalizePaging(paging: Partial<QueryPaging> | undefined): QueryPaging {
  return {
    page: normalizePositiveInteger(paging?.page, DEFAULT_PAGING.page),
    pageSize: normalizePositiveInteger(paging?.pageSize, DEFAULT_PAGING.pageSize),
  };
}

function normalizeFields(fields: readonly string[] | undefined): readonly string[] {
  return [...new Set((fields ?? []).map((field) => field.trim()).filter(Boolean))];
}

function normalizeAggregates(
  aggregates: readonly QueryAggregate[] | undefined,
): readonly QueryAggregate[] {
  return (aggregates ?? [])
    .map((aggregate) => ({
      ...aggregate,
      field: aggregate.field.trim(),
      alias: aggregate.alias.trim(),
    }))
    .filter((aggregate) => Boolean(aggregate.field && aggregate.alias));
}

function normalizePositiveInteger(value: number | undefined, fallback: number): number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 1
    ? Math.floor(value)
    : fallback;
}
