import type { EvolynGridItem } from '@evolyn.do/ui';

export type DashboardWidgetType = 'onboarding' | 'greeting' | 'shortcut' | 'todo' | 'favorites' | 'apps' | 'charts';

/** 卡片内容所需的最小业务数据，不包含网格坐标。 */
export interface DashboardWidgetContent {
  id: string;
  type: DashboardWidgetType;
  title: string;
  config?: Record<string, unknown>;
}

/** 工作台持久化卡片：网格坐标由 EvolynGrid 管理，业务内容由 type/config 决定。 */
export interface DashboardWidget extends EvolynGridItem, DashboardWidgetContent {}
