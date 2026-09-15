import { computed, readonly, shallowRef, watch, type Ref } from 'vue';
import { ApiError, ERROR_CODES } from '@evolyn.do/utils';
import { getAppByCode } from '~/api/apps';
import type { AppItem } from '~/types';

export type AppHomeStatus = 'loading' | 'ready' | 'not-found' | 'error';

/**
 * 根据公开应用编码加载应用首页需要的最小元数据。
 *
 * 路由使用公开 appCode，直接通过按编码查询接口获取应用，避免为定位单个应用
 * 拉取应用列表及其分页数据。
 */
export function useAppHome(appCode: Readonly<Ref<string>>) {
  const app = shallowRef<AppItem | null>(null);
  const status = shallowRef<AppHomeStatus>('loading');
  const errorMessage = shallowRef('');
  let requestVersion = 0;

  async function load(code = appCode.value) {
    const version = ++requestVersion;
    app.value = null;
    errorMessage.value = '';

    if (!code) {
      status.value = 'not-found';
      return;
    }

    status.value = 'loading';
    try {
      const matched = await getAppByCode(code);
      if (version !== requestVersion) return;

      app.value = matched;
      status.value = 'ready';
    } catch (error) {
      if (version !== requestVersion) return;

      if (error instanceof ApiError && error.errCode === ERROR_CODES.APP_NOT_FOUND) {
        status.value = 'not-found';
        return;
      }

      errorMessage.value = '应用信息加载失败，请稍后重试';
      status.value = 'error';
    }
  }

  watch(
    appCode,
    () => {
      void load();
    },
    { immediate: true },
  );

  return {
    app: readonly(app),
    status: readonly(status),
    errorMessage: readonly(errorMessage),
    appName: computed(() => app.value?.name ?? '应用'),
    reload: load,
  };
}
