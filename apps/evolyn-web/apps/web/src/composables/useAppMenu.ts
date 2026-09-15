import type { Component, Ref } from 'vue';
import type { AppWorkspaceAsset } from '~/components/app/workspace/appWorkspace.types';
import type { AppMenu, AppMenuNode, AppMenuType } from '~/types';
import {
  RiArticleFill,
  RiBarChartBoxFill,
  RiBookmark3Fill,
  RiBriefcase4Fill,
  RiCalendarCheckFill,
  RiCheckboxCircleFill,
  RiContactsBook3Fill,
  RiFileList3Fill,
  RiFolder3Fill,
  RiShoppingCart2Fill,
} from '@remixicon/vue';
import { markRaw, readonly, shallowRef, watch } from 'vue';
import { getAppMenuByCode } from '~/api/apps';

export type AppMenuStatus = 'loading' | 'ready' | 'error';

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
  bookmark: markRaw(RiBookmark3Fill),
  briefcase: markRaw(RiBriefcase4Fill),
  calendar: markRaw(RiCalendarCheckFill),
  check: markRaw(RiCheckboxCircleFill),
  contacts: markRaw(RiContactsBook3Fill),
  'shopping-cart': markRaw(RiShoppingCart2Fill),
};

const iconByType: Record<AppMenuType, Component> = {
  group: markRaw(RiFolder3Fill),
  form: markRaw(RiFileList3Fill),
  dashboard: markRaw(RiBarChartBoxFill),
  page: markRaw(RiArticleFill),
};

function resolveIcon(node: AppMenuNode): Component {
  if (node.icon) {
    const matched = iconByKey[node.icon];
    if (matched) {
      return matched;
    }
    console.warn(`[app-menu] unknown icon key: ${node.icon}`);
  }
  return iconByType[node.type];
}

/**
 * 后端 rootMenuIds + nodeMap → 侧栏资产树。只保留 capabilities.view 节点，
 * 同父顺序按 (sortOrder, menuId) 排序，与后端 §6.2 契约一致；
 * 分组携带 children，叶子节点 code 即 menuId（选中态定位键）。
 */
function buildAssets(menu: AppMenu): AppWorkspaceAsset[] {
  const visible = Object.values(menu.nodeMap).filter((node) => node.capabilities.view);
  const byParent = new Map<string | null, AppMenuNode[]>();
  for (const node of visible) {
    const key = node.parentMenuId;
    const siblings = byParent.get(key) ?? [];
    siblings.push(node);
    byParent.set(key, siblings);
  }
  for (const siblings of byParent.values()) {
    siblings.sort((a, b) =>
      a.sortOrder === b.sortOrder ? a.menuId.localeCompare(b.menuId) : a.sortOrder - b.sortOrder,
    );
  }

  const toAsset = (node: AppMenuNode): AppWorkspaceAsset => {
    const type: AppWorkspaceAsset['type'] = node.type === 'group' ? 'folder' : node.type;
    const asset: AppWorkspaceAsset = {
      code: node.menuId,
      label: node.name,
      icon: resolveIcon(node),
      iconKey: node.icon,
      type,
      // 菜单 menuId 仅用于树节点定位；设计器路由必须使用资产公开编码。
      targetCode: node.target?.code ?? null,
      formType: node.target?.type === 'form' ? node.target.formType : null,
      capabilities: node.capabilities,
    };
    if (node.type === 'group') {
      const children = (byParent.get(node.menuId) ?? []).map(toAsset);
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
export function useAppMenu(appCode: Readonly<Ref<string>>) {
  const assets = shallowRef<AppWorkspaceAsset[]>([]);
  const status = shallowRef<AppMenuStatus>('loading');
  const errorMessage = shallowRef('');
  const menuRevision = shallowRef(0);
  let requestVersion = 0;
  /** 最近一次成功加载的应用编码；切换应用时才允许清空旧菜单。 */
  let loadedAppCode = '';

  async function load(code = appCode.value) {
    const version = ++requestVersion;
    errorMessage.value = '';

    if (!code) {
      assets.value = [];
      status.value = 'error';
      errorMessage.value = '应用编码缺失，无法加载菜单';
      return;
    }

    // 同一应用的刷新（例如修改名称/图标后的菜单修订同步）必须保留现有
    // 资产树与 ready 状态，否则 activeAsset 会暂时变为 null，运行态表单被
    // v-if 卸载并重新请求，用户会感知为“整页重新加载”。
    const switchingApp = code !== loadedAppCode;
    if (switchingApp) {
      assets.value = [];
      status.value = 'loading';
    }
    try {
      const menu = await getAppMenuByCode(code);
      if (version !== requestVersion) return;

      assets.value = buildAssets(menu);
      menuRevision.value = menu.menuRevision;
      loadedAppCode = code;
      status.value = 'ready';
    } catch (error) {
      if (version !== requestVersion) return;

      // 后台刷新失败时继续呈现最后一个可用菜单，避免用户正在填写的数据被
      // 加载占位替换；首次加载与切换应用仍展示完整错误态。
      if (switchingApp) {
        status.value = 'error';
        errorMessage.value = '应用菜单加载失败，请稍后重试';
      }
      console.warn('[app-menu] load menu failed', error);
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
    menuRevision: readonly(menuRevision),
    status: readonly(status),
    errorMessage: readonly(errorMessage),
    reload: () => load(),
  };
}
