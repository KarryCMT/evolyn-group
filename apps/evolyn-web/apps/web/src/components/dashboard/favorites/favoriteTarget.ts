import type { RouteLocationRaw } from 'vue-router';
import type { MenuFavoriteItem } from '~/types';
import { appWorkspaceRoute } from '~/components/app/workspace/appWorkspaceNavigation';

/**
 * 收藏条目的打开语义：收藏挂菜单节点，打开时回到所属应用工作区并以资产
 * 公开编码恢复选中态（路由 /app/:appCode/:assetCode?）。仪表盘/页面资产域
 * 未落地运行态，返回 null 由调用方提示。
 */
export function favoriteTargetRoute(item: MenuFavoriteItem): RouteLocationRaw | null {
  if (item.node.target?.type === 'form' && item.node.target.code) {
    return appWorkspaceRoute(item.app.code, item.node.target.code);
  }
  return null;
}
