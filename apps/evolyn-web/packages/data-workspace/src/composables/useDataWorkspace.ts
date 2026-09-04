import { computed, shallowRef } from 'vue';
import { normalizeDataQuery, type DataQuery } from '@evolyn.do/data';
import type { UseDataWorkspaceOptions } from '../types.js';

/**
 * Vue 工作台仅持有查询交互状态；查询归一化规则始终由 Data Engine 提供。
 */
export function useDataWorkspace(options: UseDataWorkspaceOptions = {}) {
  const query = shallowRef<DataQuery>(normalizeDataQuery(options.initialQuery));
  const hasKeyword = computed(() => Boolean(query.value.keyword));

  function updateQuery(patch: Partial<DataQuery>) {
    query.value = normalizeDataQuery({ ...query.value, ...patch });
  }

  function search(keyword: string) {
    updateQuery({ keyword, page: 1 });
  }

  function changePage(page: number) {
    updateQuery({ page });
  }

  function changePageSize(pageSize: number) {
    updateQuery({ page: 1, pageSize });
  }

  return {
    query,
    hasKeyword,
    updateQuery,
    search,
    changePage,
    changePageSize,
  };
}
