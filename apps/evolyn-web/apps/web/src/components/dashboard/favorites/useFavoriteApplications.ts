import type { FavoriteApplication } from './favoriteCatalog';
import { computed, ref } from 'vue';
import {
  defaultFavoriteApplicationIds,
  flattenFavoriteApplications,
} from './favoriteCatalog';

const allApplications = flattenFavoriteApplications();
const applicationsById = new Map(
  allApplications.map((application) => [application.id, application]),
);

// 收藏入口同时存在于个人菜单和工作台卡片中，使用模块级状态使两处展示即时保持一致。
const selectedApplicationIds = ref<string[]>([...defaultFavoriteApplicationIds]);

export function useFavoriteApplications() {
  const favoriteApplications = computed<FavoriteApplication[]>(() =>
    selectedApplicationIds.value
      .map((id) => applicationsById.get(id))
      .filter((application): application is FavoriteApplication => Boolean(application)),
  );

  function replaceFavoriteApplications(ids: string[]) {
    // 目录搜索和多选可能产生重复项，统一在写入时过滤无效及重复 id。
    selectedApplicationIds.value = [...new Set(ids)].filter((id) => applicationsById.has(id));
  }

  return {
    favoriteApplications,
    selectedApplicationIds,
    replaceFavoriteApplications,
  };
}
