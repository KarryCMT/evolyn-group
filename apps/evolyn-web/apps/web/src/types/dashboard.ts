import type { EvolynGridItem } from '@evolyn.do/ui';

export type DashboardWidgetType =
  | 'onboarding'
  | 'greeting'
  | 'shortcut'
  | 'todo'
  | 'favorites'
  | 'apps'
  | 'charts';

/** 设计器组件面板中的组件预设，包含拖入后所需的业务数据与初始尺寸。 */
export interface DashboardWidgetPreset {
  key: string;
  type: DashboardWidgetType;
  title: string;
  w: number;
  h: number;
  minW: number;
  minH: number;
  maxW?: number;
  maxH?: number;
  config?: Record<string, unknown>;
}

/** 快捷入口与图表看板可按需添加多个实例，其余页面组件在同一工作台中只能保留一个。 */
const repeatableDashboardWidgetPresetKeys = new Set(['shortcut', 'charts']);

export function isDashboardWidgetPresetRepeatable(preset: Pick<DashboardWidgetPreset, 'key'>) {
  return repeatableDashboardWidgetPresetKeys.has(preset.key);
}

/** 卡片内容所需的最小业务数据，不包含网格坐标。 */
export interface DashboardWidgetContent {
  id: string;
  type: DashboardWidgetType;
  title: string;
  config?: Record<string, unknown>;
}

/** 工作台持久化卡片：网格坐标由 EvolynGrid 管理，业务内容由 type/config 决定。 */
export interface DashboardWidget extends EvolynGridItem, DashboardWidgetContent {
  /** 来源组件面板的预设键，用于限制非重复组件只能添加一次。 */
  presetKey?: string;
}

/** 同类组件以业务类型和展示标题确定唯一性，避免配置项差异导致重复添加。 */
export function isDashboardWidgetPresetInLayout(
  preset: Pick<DashboardWidgetPreset, 'type' | 'title'>,
  widgets: DashboardWidget[],
) {
  return widgets.some((widget) => widget.type === preset.type && widget.title === preset.title);
}
