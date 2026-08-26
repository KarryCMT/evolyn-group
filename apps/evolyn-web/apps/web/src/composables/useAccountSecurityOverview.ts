import { shallowRef } from 'vue';
import { getMySecurityOverview } from '~/api/account';
import type { AccountSecurityOverview } from '~/types';

/**
 * 账号安全概览的请求状态收口。
 * 安全面板挂载即加载；失败时保留已展示的数据，并允许用户手动重试。
 */
export function useAccountSecurityOverview() {
  const overview = shallowRef<AccountSecurityOverview | null>(null);
  const loading = shallowRef(false);
  const loadError = shallowRef('');

  async function loadOverview() {
    loading.value = true;
    loadError.value = '';
    try {
      overview.value = await getMySecurityOverview();
    } catch {
      loadError.value = '账号安全概览加载失败，请稍后重试';
    } finally {
      loading.value = false;
    }
  }

  return {
    overview,
    loading,
    loadError,
    loadOverview,
  };
}
