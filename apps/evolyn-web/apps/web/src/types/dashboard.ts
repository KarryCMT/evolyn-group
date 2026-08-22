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

/** 卡片内容所需的最小业务数据，不包含网格坐标。 */
export interface DashboardWidgetContent {
  id: string;
  type: DashboardWidgetType;
  title: string;
  config?: Record<string, unknown>;
}

/** 工作台持久化卡片：网格坐标由 EvolynGrid 管理，业务内容由 type/config 决定。 */
export interface DashboardWidget extends EvolynGridItem, DashboardWidgetContent {}
