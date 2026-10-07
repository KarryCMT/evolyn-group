import type { Ref } from 'vue';
import { onScopeDispose, watch } from 'vue';
import { onBeforeRouteLeave } from 'vue-router';

export interface UnsavedChangesGuardOptions {
  dirty: Readonly<Ref<boolean>>;
  confirmLeave: () => Promise<boolean>;
  target?: Pick<Window, 'addEventListener' | 'removeEventListener'>;
}

/** 路由与浏览器离开共用 dirty 事实源；beforeunload 仅在 dirty 窗口内注册。 */
export function useUnsavedChangesGuard(options: UnsavedChangesGuardOptions) {
  const target = options.target ?? window;
  let registered = false;
  const onBeforeUnload = (event: BeforeUnloadEvent) => {
    event.preventDefault();
    event.returnValue = '';
  };

  function syncBrowserGuard(dirty: boolean) {
    if (dirty && !registered) {
      target.addEventListener('beforeunload', onBeforeUnload as EventListener);
      registered = true;
    } else if (!dirty && registered) {
      target.removeEventListener('beforeunload', onBeforeUnload as EventListener);
      registered = false;
    }
  }

  watch(options.dirty, syncBrowserGuard, { immediate: true });
  onScopeDispose(() => syncBrowserGuard(false));
  onBeforeRouteLeave(async () => {
    if (!options.dirty.value) return true;
    return options.confirmLeave();
  });

  return { syncBrowserGuard };
}
