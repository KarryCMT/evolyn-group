import { markRaw, readonly, shallowRef, watch, type Component, type Ref } from 'vue';
import { RiArticleFill, RiBarChartBoxFill, RiFileList3Fill, RiFolder3Fill } from '@remixicon/vue';
import { getApplicationMenuByCode } from '~/api/applications';
import type { ApplicationMenu, ApplicationMenuEntry, ApplicationMenuEntryType } from '~/types';
import type { ApplicationWorkspaceAsset } from '~/components/application/workspace/applicationWorkspace.types';

export type ApplicationMenuStatus = 'loading' | 'ready' | 'error';

/**
 * 图标受控映射表：后端稳定图标键 → Remix Fill 组件。未知键回退到节点
 * 类型默认图标并记录可观测事件（console.warn），侧栏组件只消费 Component，
 * 不理解图标键语义。
 */
const iconByKey: Record<string, Component> = {
  folder: markRaw(RiFolder3Fill),
  'file-list': markRaw(RiFileList3Fill),
  chart: markRaw(RiBarChartBoxFill),
  article: markRaw(RiArticleFill),
};

const iconByType: Record<ApplicationMenuEntryType, Component> = {
  group: markRaw(RiFolder3Fill),
  form: markRaw(RiFileList3Fill),
  dashboard: markRaw(RiBarChartBoxFill),
  page: markRaw(RiArticleFill),
};

function resolveIcon(entry: ApplicationMenuEntry): Component {
  if (entry.icon) {
    const matched = iconByKey[entry.icon];
    if (matched) {
      return matched;
    }
    console.warn(`[application-menu] unknown icon key: ${entry.icon}`);
  }
  return iconByType[entry.type];
}

/**
 * 后端 rootEntryIds + entryMap → 侧栏资产树。只保留 capabilities.view 节点，
 * 同父顺序按 (sortOrder, entryId) 排序，与后端 §6.2 契约一致；
 * 分组携带 children，叶子节点 code 即 entryId（选中态定位键）。
 */
function buildAssets(menu: ApplicationMenu): ApplicationWorkspaceAsset[] {
  const visible = Object.values(menu.entryMap).filter((entry) => entry.capabilities.view);
  const byParent = new Map<string | null, ApplicationMenuEntry[]>();
  for (const entry of visible) {
    const key = entry.parentEntryId;
    const siblings = byParent.get(key) ?? [];
    siblings.push(entry);
    byParent.set(key, siblings);
  }
  for (const siblings of byParent.values()) {
    siblings.sort((a, b) =>
      a.sortOrder === b.sortOrder ? a.entryId.localeCompare(b.entryId) : a.sortOrder - b.sortOrder,
    );
  }

  const toAsset = (entry: ApplicationMenuEntry): ApplicationWorkspaceAsset => {
    const type: ApplicationWorkspaceAsset['type'] = entry.type === 'group' ? 'folder' : entry.type;
    const asset: ApplicationWorkspaceAsset = {
      code: entry.entryId,
      label: entry.name,
      icon: resolveIcon(entry),
      type,
      capabilities: entry.capabilities,
    };
    if (entry.type === 'group') {
      const children = (byParent.get(entry.entryId) ?? []).map(toAsset);
      if (children.length > 0) {
        asset.children = children;
      }
    }
    return asset;
  };

  return (byParent.get(null) ?? []).map(toAsset);
}

/**
 * 应用菜单数据源（M2-菜单-2）：随 appCode 变化加载菜单接口，产出侧栏
 * 可直接消费的资产树；错误处理与 UI 数据适配收敛在本 composable，
 * 侧栏组件保持纯展示（方案 §11）。
 */
export function useApplicationMenu(appCode: Readonly<Ref<string>>) {
  const assets = shallowRef<ApplicationWorkspaceAsset[]>([]);
  const status = shallowRef<ApplicationMenuStatus>('loading');
  const errorMessage = shallowRef('');
  let requestVersion = 0;

  async function load(code = appCode.value) {
    const version = ++requestVersion;
    assets.value = [];
    errorMessage.value = '';

    if (!code) {
      status.value = 'error';
      errorMessage.value = '应用编码缺失，无法加载菜单';
      return;
    }

    status.value = 'loading';
    try {
      const menu = await getApplicationMenuByCode(code);
      if (version !== requestVersion) return;

      assets.value = buildAssets(menu);
      status.value = 'ready';
    } catch (error) {
      if (version !== requestVersion) return;

      status.value = 'error';
      errorMessage.value = '应用菜单加载失败，请稍后重试';
      console.warn('[application-menu] load menu failed', error);
    }
  }

  watch(
    appCode,
    () => {
      void load();
    },
    { immediate: true },
  );

  // assets 不包 readonly：消费方（Shell/Sidebar props）需要可变数组类型；
  // 数据源由本 composable 整体重建，外部无 mutate 场景
  return {
    assets,
    status: readonly(status),
    errorMessage: readonly(errorMessage),
    reload: () => void load(),
  };
}
