import type { Component } from 'vue';
import type { AppMenuType } from '~/types';
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
import { markRaw } from 'vue';

/**
 * 菜单图标受控映射表：后端稳定图标键 → Remix Fill 组件。未知键回退到
 * 节点类型默认图标并记录可观测事件（console.warn），消费组件只拿
 * Component，不理解图标键语义。应用行（收藏选择器的应用级节点）无
 * 菜单类型，按 app 兜底 bookmark。
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

const iconByType: Record<AppMenuType | 'app', Component> = {
  group: markRaw(RiFolder3Fill),
  form: markRaw(RiFileList3Fill),
  dashboard: markRaw(RiBarChartBoxFill),
  page: markRaw(RiArticleFill),
  app: markRaw(RiBookmark3Fill),
};

/** 按图标键 + 节点类型解析图标组件：键缺失或未登记时回退类型默认图标。 */
export function resolveMenuIcon(type: AppMenuType | 'app', iconKey: string | null): Component {
  if (iconKey) {
    const matched = iconByKey[iconKey];
    if (matched) {
      return matched;
    }
    console.warn(`[menu-icon] unknown icon key: ${iconKey}`);
  }
  return iconByType[type];
}
