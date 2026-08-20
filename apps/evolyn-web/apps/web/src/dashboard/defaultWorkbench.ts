import type { DashboardWidget } from '~/types/dashboard';

/**
 * 工作台首次进入时使用的成员端默认布局。
 * 设计器与首页都从这里取值，避免两端初始卡片和坐标逐渐偏离。
 */
const defaultWidgets: DashboardWidget[] = [
  widget('onboarding', '新手引导', 0, 0, 12, 2, { noResize: true }),
  widget('greeting', '问候语', 0, 2, 3, 1, { minW: 3, minH: 1 }),
  widget('favorites', '最近使用', 3, 2, 9, 2, { minW: 4, minH: 2 }),
  widget('shortcut', '未命名快捷入口', 0, 4, 12, 2, { minH: 2 }),
  widget('todo', '流程中心', 0, 6, 3, 4, { minW: 3, minH: 3 }),
  widget('favorites', '我的收藏', 3, 6, 9, 2, { minW: 4, minH: 2 }),
  widget('apps', '我的应用', 3, 8, 9, 3, { minW: 4, minH: 3 }),
  widget('charts', '我的图表', 3, 11, 9, 2, { minW: 4, minH: 2 }),
];

/** 返回新对象，避免 GridStack 更新坐标时修改默认布局常量。 */
export function createDefaultWorkbenchWidgets(): DashboardWidget[] {
  return defaultWidgets.map((item) => ({ ...item }));
}

function widget(
  type: DashboardWidget['type'],
  title: string,
  x: number,
  y: number,
  w: number,
  h: number,
  constraints: Partial<DashboardWidget> = {},
): DashboardWidget {
  return {
    id: `${type}-${x}-${y}`,
    type,
    title,
    x,
    y,
    w,
    h,
    component: 'DashboardWidgetHost',
    ...constraints,
  };
}
