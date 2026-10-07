import type { DashboardFormDataSource, DashboardFormFieldCatalog } from '~/types';
import { ApiError } from '@evolyn.do/utils';
import { shallowRef } from 'vue';
import { getDashboardFormFieldCatalog, listDashboardFormDataSources } from '~/api/dashboard';

/** 设计器数据目录只缓存当前仪表盘会话，切换资产时整体清空，避免跨租户复用。 */
export function useDashboardDataCatalog() {
  const sources = shallowRef<DashboardFormDataSource[]>([]);
  const catalogs = shallowRef<Record<string, DashboardFormFieldCatalog>>({});
  const loading = shallowRef(false);
  const errorMessage = shallowRef('');
  let dashboardCode = '';

  async function load(code: string) {
    dashboardCode = code;
    loading.value = true;
    errorMessage.value = '';
    catalogs.value = {};
    try {
      sources.value = await listDashboardFormDataSources(code);
    } catch (error) {
      sources.value = [];
      errorMessage.value = error instanceof ApiError ? error.message : '数据源加载失败';
    } finally {
      loading.value = false;
    }
  }

  async function ensureCatalog(formCode: string) {
    if (!dashboardCode || catalogs.value[formCode]) return catalogs.value[formCode] ?? null;
    try {
      const catalog = await getDashboardFormFieldCatalog(dashboardCode, formCode);
      catalogs.value = { ...catalogs.value, [formCode]: catalog };
      errorMessage.value = '';
      return catalog;
    } catch (error) {
      errorMessage.value = error instanceof ApiError ? error.message : '字段目录加载失败';
      return null;
    }
  }

  return { sources, catalogs, loading, errorMessage, load, ensureCatalog };
}
