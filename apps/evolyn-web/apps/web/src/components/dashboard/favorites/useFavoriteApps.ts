import type { FavoriteApp } from './favoriteCatalog';
import { computed, ref } from 'vue';
import { defaultFavoriteAppIds, flattenFavoriteApps } from './favoriteCatalog';

const allApps = flattenFavoriteApps();
const appsById = new Map(allApps.map((app) => [app.id, app]));

// 收藏入口同时存在于个人菜单和工作台卡片中，使用模块级状态使两处展示即时保持一致。
const selectedAppIds = ref<string[]>([...defaultFavoriteAppIds]);

export function useFavoriteApps() {
  const favoriteApps = computed<FavoriteApp[]>(() =>
    selectedAppIds.value
      .map((id) => appsById.get(id))
      .filter((app): app is FavoriteApp => Boolean(app)),
  );

  function replaceFavoriteApps(ids: string[]) {
    // 目录搜索和多选可能产生重复项，统一在写入时过滤无效及重复 id。
    selectedAppIds.value = [...new Set(ids)].filter((id) => appsById.has(id));
  }

  return {
    favoriteApps,
    selectedAppIds,
    replaceFavoriteApps,
  };
}
