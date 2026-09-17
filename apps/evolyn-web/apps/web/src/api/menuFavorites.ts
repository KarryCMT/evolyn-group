import type {
  CreateMenuFavoritePayload,
  MenuFavoriteListQuery,
  MenuFavoriteMutation,
  MenuFavoritePage,
} from '~/types';
// 菜单个人收藏接口：与后端 /api/v1/menu-favorites 一一对应
// （见 evolyn-core internal/platform/app/controller/menu.go）
import { http } from '@evolyn.do/utils';

/**
 * 收藏菜单节点（POST /menu-favorites）：服务端按 canFavorite 统一策略复核
 * （仅资产叶子节点 ∧ 应用可用 ∧ 节点在当前成员有效可见集内），分组/隐藏/
 * 无表单入口权限的节点返回 errCode=APP_MENU_FAVORITE_INVALID；重复收藏幂等
 */
export function addMenuFavorite(payload: CreateMenuFavoritePayload): Promise<MenuFavoriteMutation> {
  return http.post('/menu-favorites', payload);
}

/**
 * 取消收藏（DELETE /menu-favorites/:menuCode）：幂等，目标收藏不存在同样
 * 返回 favorited=false；取消不要求目标仍可见（可清除失效入口的个人状态）
 */
export function removeMenuFavorite(menuCode: string): Promise<MenuFavoriteMutation> {
  return http.delete(`/menu-favorites/${menuCode}`);
}

/**
 * 我的收藏跨应用列表（GET /menu-favorites）：按收藏时间倒序游标分页；
 * 应用归档、节点隐藏、资产权限失效的记录由服务端读侧过滤（只过滤不删除，
 * 恢复后自然恢复展示），因此可能出现短页，以 nextCursor 判断是否续拉
 */
export function listMenuFavorites(query: MenuFavoriteListQuery = {}): Promise<MenuFavoritePage> {
  return http.get('/menu-favorites', {
    cursor: query.cursor,
    limit: query.limit,
  });
}
