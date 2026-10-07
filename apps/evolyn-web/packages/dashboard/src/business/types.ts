/** 业务仪表盘协议版本，与后端 internal/engine/dashboard 同步。 */
export const BUSINESS_DASHBOARD_VERSION = 1 as const;
export const BUSINESS_DASHBOARD_COLUMNS = 12 as const;

export type BusinessDashboardWidgetType = 'chart' | 'table';

export interface BusinessDashboardLayout {
  x: number;
  y: number;
  w: number;
  h: number;
}

/**
 * 阶段二只固化组件身份、布局和基础展示属性；数据绑定配置在阶段三扩展。
 * settings 必须保持纯 JSON，禁止放入 Vue、GridStack 或图表运行时实例。
 */
export interface BusinessDashboardWidget {
  id: string;
  type: BusinessDashboardWidgetType;
  title?: string;
  layout: BusinessDashboardLayout;
  datasetId?: string;
  settings?: Record<string, unknown>;
}

export interface BusinessDashboardDocument {
  version: typeof BUSINESS_DASHBOARD_VERSION;
  settings: {
    desktop: {
      columns: typeof BUSINESS_DASHBOARD_COLUMNS;
      rowHeight: number;
    };
  };
  datasets: Array<{ id: string }>;
  widgets: BusinessDashboardWidget[];
  filters: Array<{ id: string }>;
  interactions: Array<{ id: string }>;
  publishScope: { type: 'all' };
}

export interface BusinessDashboardIssue {
  path: string;
  code: string;
  message: string;
}

/** 描述器定义设计器可创建的组件，不进入持久化文档。 */
export interface BusinessDashboardWidgetDescriptor {
  type: BusinessDashboardWidgetType;
  label: string;
  description: string;
  defaultTitle: string;
  defaultLayout: Pick<BusinessDashboardLayout, 'w' | 'h'>;
  defaultSettings: Record<string, unknown>;
}

export interface BusinessDashboardNormalizationResult {
  document: BusinessDashboardDocument | null;
  issues: BusinessDashboardIssue[];
}
