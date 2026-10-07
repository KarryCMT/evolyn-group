import type { RouteLocationRaw } from 'vue-router';
import type { AppWorkspaceAsset, AppWorkspaceMode } from './appWorkspace.types';

type NavigableWorkspaceAsset = Pick<AppWorkspaceAsset, 'targetCode' | 'type'>;

/** 应用工作区用资产公开编码恢复菜单选中态，表单与仪表盘共用同一参数。 */
export function appWorkspaceRoute(appCode: string, assetCode = ''): RouteLocationRaw {
  return {
    name: 'App',
    params: { appCode, assetCode: assetCode.trim() },
  };
}

/** 递归定位公开编码对应的资产节点，分组节点不会被误选为运行资产。 */
export function findWorkspaceAssetByTargetCode(
  assets: AppWorkspaceAsset[],
  targetCode: string,
): AppWorkspaceAsset | null {
  if (!targetCode) return null;
  for (const asset of assets) {
    if (asset.type !== 'folder' && asset.targetCode === targetCode) return asset;
    const matched = findWorkspaceAssetByTargetCode(asset.children ?? [], targetCode);
    if (matched) return matched;
  }
  return null;
}

/**
 * 将工作区顶部模式和菜单“编辑”动作统一解析为资产工作区路由。
 * dashboard 目前只有独立设计工作区；数据管理仍是表单专属能力。
 */
export function resolveWorkspaceAssetRoute(
  appCode: string,
  asset: NavigableWorkspaceAsset | null,
  mode: Exclude<AppWorkspaceMode, 'fill'>,
): RouteLocationRaw | null {
  const targetCode = asset?.targetCode?.trim();
  if (!asset || !targetCode) return null;

  if (asset.type === 'form') {
    return {
      name: mode === 'design' ? 'form-design' : 'form-data',
      params: { appCode, formCode: targetCode },
    };
  }

  if (asset.type === 'dashboard' && mode === 'design') {
    return {
      name: 'dashboard-design',
      params: { appCode, dashboardCode: targetCode },
    };
  }

  return null;
}
