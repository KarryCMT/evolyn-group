import type { DataQuery } from './types.js';
import { normalizeQuery } from '@evolyn.do/query';

/**
 * 将来自路由、表单或远程适配器的部分查询收敛为可安全执行的分页查询。
 * 该函数不涉及 Vue 响应式状态，因此任意领域都可复用。
 */
export function normalizeDataQuery(query: Partial<DataQuery> = {}): DataQuery {
  const normalizedDsl = normalizeQuery({
    filter: query.filter,
    sorts: query.sorts,
    // QueryDocument 的分页结构完整；缺省值仍会由 Query Engine 继续归一化。
    paging: { page: query.page ?? 1, pageSize: query.pageSize ?? 20 },
    projection: query.projection,
    groupBy: query.groupBy,
    aggregates: query.aggregates,
  });

  return {
    keyword: query.keyword?.trim() ?? '',
    page: normalizedDsl.paging.page,
    pageSize: normalizedDsl.paging.pageSize,
    ...(normalizedDsl.filter ? { filter: normalizedDsl.filter } : {}),
    ...(normalizedDsl.sorts.length ? { sorts: normalizedDsl.sorts } : {}),
    ...(normalizedDsl.projection?.length ? { projection: normalizedDsl.projection } : {}),
    ...(normalizedDsl.groupBy?.length ? { groupBy: normalizedDsl.groupBy } : {}),
    ...(normalizedDsl.aggregates?.length ? { aggregates: normalizedDsl.aggregates } : {}),
  };
}
