import type { Component } from 'vue';
import type { AppMenuType, FormType } from '~/types';
import {
  RiAlarmWarningFill,
  RiArticleFill,
  RiBarChartBoxFill,
  RiBookmark3Fill,
  RiBriefcase4Fill,
  RiCalendarCheckFill,
  RiCheckboxCircleFill,
  RiContactsBook3Fill,
  RiDashboardFill,
  RiDatabase2Fill,
  RiFileList3Fill,
  RiFileTextFill,
  RiFolder3Fill,
  RiGiftFill,
  RiGlobalFill,
  RiHome5Fill,
  RiInboxArchiveFill,
  RiLineChartFill,
  RiMapPin2Fill,
  RiMoneyCnyBoxFill,
  RiPriceTag3Fill,
  RiProductHuntFill,
  RiProjector2Fill,
  RiQuestionAnswerFill,
  RiRobot2Fill,
  RiRouteFill,
  RiShoppingBag3Fill,
  RiShoppingCart2Fill,
  RiStore2Fill,
  RiTeamFill,
  RiTruckFill,
  RiUserStarFill,
  RiVipCrown2Fill,
  RiWallet3Fill,
} from '@remixicon/vue';
import { markRaw } from 'vue';

/**
 * 菜单图标受控映射表：后端稳定图标键 → Remix Fill 组件。未知键回退到
 * 节点类型默认图标并记录可观测事件（console.warn），消费组件只拿
 * Component，不理解图标键语义。应用行（收藏选择器的应用级节点）无
 * 菜单类型，按 app 兜底 bookmark。
 */
export interface MenuIconOption {
  key: string;
  label: string;
  icon: Component;
}

/** 菜单图标候选集：key 是已持久化的稳定值，选择器和菜单展示共用本表。 */
const menuIconEntries: readonly (readonly [string, string, Component])[] = [
  ['file-list', '表单', RiFileList3Fill],
  ['chart', '图表', RiBarChartBoxFill],
  ['dashboard', '仪表盘', RiDashboardFill],
  ['line-chart', '趋势图', RiLineChartFill],
  ['inbox', '收件箱', RiInboxArchiveFill],
  ['home', '主页', RiHome5Fill],
  ['database', '数据', RiDatabase2Fill],
  ['report', '报表', RiFileTextFill],
  ['calendar', '日程', RiCalendarCheckFill],
  ['bookmark', '收藏', RiBookmark3Fill],
  ['gift', '礼品', RiGiftFill],
  ['tag', '标签', RiPriceTag3Fill],
  ['question', '问答', RiQuestionAnswerFill],
  ['contacts', '联系人', RiContactsBook3Fill],
  ['user-star', '成员', RiUserStarFill],
  ['team', '团队', RiTeamFill],
  ['store', '店铺', RiStore2Fill],
  ['product', '产品', RiProductHuntFill],
  ['shopping-cart', '购物车', RiShoppingCart2Fill],
  ['shopping-bag', '采购', RiShoppingBag3Fill],
  ['truck', '配送', RiTruckFill],
  ['briefcase', '业务', RiBriefcase4Fill],
  ['project', '项目', RiProjector2Fill],
  ['route', '流程', RiRouteFill],
  ['wallet', '钱包', RiWallet3Fill],
  ['money', '金额', RiMoneyCnyBoxFill],
  ['vip', '会员', RiVipCrown2Fill],
  ['robot', '智能助手', RiRobot2Fill],
  ['global', '全球', RiGlobalFill],
  ['location', '位置', RiMapPin2Fill],
  ['alarm', '提醒', RiAlarmWarningFill],
  ['article', '文档', RiArticleFill],
  ['check', '任务', RiCheckboxCircleFill],
  ['folder', '分组', RiFolder3Fill],
];

export const menuIconOptions: readonly MenuIconOption[] = menuIconEntries.map(
  ([key, label, icon]) => ({
    key,
    label,
    icon: markRaw(icon),
  }),
);

const iconByKey: Record<string, Component> = Object.fromEntries(
  menuIconOptions.map((option) => [option.key, option.icon]),
);

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

/** 菜单展示色只由节点类型派生，表单需进一步按 standard/workflow 区分。 */
export type MenuIconTone = 'dashboard' | 'form' | 'workflow' | 'group' | 'page';

export function resolveMenuIconTone(type: AppMenuType, formType: FormType | null): MenuIconTone {
  if (type === 'form') return formType === 'workflow' ? 'workflow' : 'form';
  if (type === 'group') return 'group';
  return type;
}

/** 无历史图标时按菜单类型提供稳定回退，避免选择器和侧栏展示不一致。 */
export function defaultMenuIconKey(type: AppMenuType, formType: FormType | null): string {
  switch (resolveMenuIconTone(type, formType)) {
    case 'dashboard':
      return 'dashboard';
    case 'workflow':
      return 'route';
    case 'group':
      return 'folder';
    case 'page':
      return 'article';
    default:
      return 'file-list';
  }
}
