import { computed, readonly, shallowRef, watch, type Ref } from 'vue';
import { listApplications } from '~/api/applications';
import type { ApplicationItem } from '~/types';

export type ApplicationHomeStatus = 'loading' | 'ready' | 'not-found' | 'error';

/**
 * 根据公开应用编码加载应用首页需要的最小元数据。
 *
 * 当前服务端只有按内部 ID 查询的接口，而路由使用公开 appCode；因此临时用
 * 游标列表查找，保证多页应用列表也能命中。后端提供按 code 查询后，只需替换
 * 此处实现，页面和视图组件无需感知接口变化。
 */
export function useApplicationHome(appCode: Readonly<Ref<string>>) {
  const application = shallowRef<ApplicationItem | null>(null);
  const status = shallowRef<ApplicationHomeStatus>('loading');
  const errorMessage = shallowRef('');
  let requestVersion = 0;

  async function findApplication(code: string): Promise<ApplicationItem | null> {
    let cursor: string | undefined;

    do {
      const page = await listApplications({ status: 'active', limit: 100, cursor });
      const matched = page.items.find((item) => item.code === code);
      if (matched) return matched;
      cursor = page.hasMore ? page.nextCursor : undefined;
    } while (cursor);

    return null;
  }

  async function load(code = appCode.value) {
    const version = ++requestVersion;
    application.value = null;
    errorMessage.value = '';

    if (!code) {
      status.value = 'not-found';
      return;
    }

    status.value = 'loading';
    try {
      const matched = await findApplication(code);
      if (version !== requestVersion) return;

      application.value = matched;
      status.value = matched ? 'ready' : 'not-found';
    } catch {
      if (version !== requestVersion) return;
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
    application: readonly(application),
    status: readonly(status),
    errorMessage: readonly(errorMessage),
    applicationName: computed(() => application.value?.name ?? '应用'),
    reload: load,
  };
}
