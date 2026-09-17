import type { MenuFavoriteItem, MenuFavoriteMutation } from '~/types';
import { addMenuFavorite, listMenuFavorites, removeMenuFavorite } from '~/api/menuFavorites';
import { readonly, shallowRef } from 'vue';

export type MenuFavoriteStatus = 'idle' | 'loading' | 'ready' | 'error';

/** 单页拉取量：与后端默认一致（上限 100），工作台卡片与抽屉共用第一页。 */
const FAVORITE_PAGE_LIMIT = 20;

/**
 * 菜单个人收藏全局状态（ADR-011 P2）：收藏入口同时存在于工作台卡片、
 * 「我的收藏」抽屉与应用侧栏右键菜单，使用模块级单例保证三处展示即时
 * 一致；服务端是唯一事实源——本地只做「请求期间锁定 + 以接口返回的
 * favorited 覆盖」，不从本地数组猜测状态（后端方案 §5.3/§7.1）。
 */
const items = shallowRef<MenuFavoriteItem[]>([]);
const status = shallowRef<MenuFavoriteStatus>('idle');
const errorMessage = shallowRef('');
const nextCursor = shallowRef('');
/** 请求进行中的节点编码集合：同一节点的收藏/取消在请求期间禁用重复触发。 */
const pendingMenuIds = shallowRef<ReadonlySet<string>>(new Set());
let loadingPromise: Promise<void> | null = null;

function setPending(menuId: string, pending: boolean) {
  const next = new Set(pendingMenuIds.value);
  if (pending) {
    next.add(menuId);
  } else {
    next.delete(menuId);
  }
  pendingMenuIds.value = next;
}

/** 拉取第一页（懒加载：首个消费组件挂载时触发；force 用于收藏变更后重取）。 */
function load(force = false): Promise<void> {
  if (loadingPromise && !force) return loadingPromise;
  if (status.value === 'ready' && !force) return Promise.resolve();
  status.value = 'loading';
  errorMessage.value = '';
  loadingPromise = listMenuFavorites({ limit: FAVORITE_PAGE_LIMIT })
    .then((page) => {
      items.value = page.items;
      nextCursor.value = page.nextCursor;
      status.value = 'ready';
    })
    .catch((error) => {
      console.warn('[menu-favorites] load failed', error);
      status.value = 'error';
      errorMessage.value = '收藏加载失败，请稍后重试';
    })
    .finally(() => {
      loadingPromise = null;
    });
  return loadingPromise;
}

/** 续拉下一页：短页由服务端可见性过滤产生，nextCursor 为空即末页。 */
async function loadMore() {
  if (!nextCursor.value || status.value === 'loading') return;
  try {
    const page = await listMenuFavorites({ limit: FAVORITE_PAGE_LIMIT, cursor: nextCursor.value });
    // 去重防御：收藏/取消并发续拉时以 menuId 去重，避免出现重复入口。
    const seen = new Set(items.value.map((item) => item.node.menuId));
    items.value = [...items.value, ...page.items.filter((item) => !seen.has(item.node.menuId))];
    nextCursor.value = page.nextCursor;
  } catch (error) {
    console.warn('[menu-favorites] load more failed', error);
  }
}

/** 收藏成功后新条目应出现在列表头部：静默重取第一页保持与事实源一致。 */
async function refreshAfterMutation() {
  if (status.value !== 'ready') return;
  try {
    const page = await listMenuFavorites({ limit: FAVORITE_PAGE_LIMIT });
    items.value = page.items;
    nextCursor.value = page.nextCursor;
  } catch {
    // 刷新失败不打断主流程：收藏动作本身已成功，下次打开面板会重取。
  }
}

/** 收藏菜单节点：成功后刷新共享列表并返回服务端状态（幂等）。 */
async function favorite(appCode: string, menuCode: string): Promise<MenuFavoriteMutation | null> {
  if (pendingMenuIds.value.has(menuCode)) return null;
  setPending(menuCode, true);
  try {
    const result = await addMenuFavorite({ appCode, menuCode });
    void refreshAfterMutation();
    return result;
  } finally {
    setPending(menuCode, false);
  }
}

/** 取消收藏：本地即时移除条目（乐观），失败时重取回滚。 */
async function unfavorite(menuCode: string): Promise<MenuFavoriteMutation | null> {
  if (pendingMenuIds.value.has(menuCode)) return null;
  setPending(menuCode, true);
  const snapshot = items.value;
  items.value = snapshot.filter((item) => item.node.menuId !== menuCode);
  try {
    const result = await removeMenuFavorite(menuCode);
    return result;
  } catch (error) {
    items.value = snapshot;
    throw error;
  } finally {
    setPending(menuCode, false);
  }
}

/** 判断收藏条目当前是否可打开：仅表单资产域已落地运行态。 */
export function isOpenableFavorite(item: MenuFavoriteItem): boolean {
  return item.node.target?.type === 'form' && Boolean(item.node.target.code);
}

/**
 * 消费入口：返回共享状态与动作。首个调用方可传 ensureLoaded 触发懒加载
 * （组件挂载即取数）；侧栏右键等纯写入方不预取列表。
 */
export function useMenuFavorites(options: { ensureLoaded?: boolean } = {}) {
  if (options.ensureLoaded) {
    void load();
  }
  return {
    items: readonly(items),
    status: readonly(status),
    errorMessage: readonly(errorMessage),
    nextCursor: readonly(nextCursor),
    pendingMenuIds: readonly(pendingMenuIds),
    load,
    loadMore,
    favorite,
    unfavorite,
  };
}
