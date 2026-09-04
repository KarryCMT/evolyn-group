import { computed, shallowRef } from 'vue';
import type { DataQuery } from '../types';

export interface UseDataWorkspaceOptions {
  initialQuery?: Partial<DataQuery>;
}

/**
 * 管理跨页面均可复用的数据工作台查询状态；接口请求、权限和数据源由应用侧接入。
 */
export function useDataWorkspace(options: UseDataWorkspaceOptions = {}) {
  const query = shallowRef<DataQuery>({
    keyword: options.initialQuery?.keyword ?? '',
    page: Math.max(options.initialQuery?.page ?? 1, 1),
    pageSize: Math.max(options.initialQuery?.pageSize ?? 20, 1),
  });
  const hasKeyword = computed(() => Boolean(query.value.keyword.trim()));

  function updateQuery(patch: Partial<DataQuery>) {
    const nextPage = patch.page ?? query.value.page;
    const nextPageSize = patch.pageSize ?? query.value.pageSize;
    query.value = {
      ...query.value,
      ...patch,
      page: Math.max(nextPage, 1),
      pageSize: Math.max(nextPageSize, 1),
    };
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
