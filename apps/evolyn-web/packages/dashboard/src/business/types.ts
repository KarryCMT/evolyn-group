import type { QueryDocument, QuerySortDirection } from '@evolyn.do/query';

/** 业务仪表盘协议版本，与后端 internal/engine/dashboard 同步。 */
export const BUSINESS_DASHBOARD_VERSION = 1 as const;
export const BUSINESS_DASHBOARD_COLUMNS = 12 as const;
export const BUSINESS_DASHBOARD_MAX_DATASETS = 50 as const;
export const BUSINESS_DASHBOARD_MAX_WIDGETS = 100 as const;

export type BusinessDashboardWidgetType = 'chart' | 'table';

export interface BusinessDashboardLayout {
  x: number;
  y: number;
  w: number;
  h: number;
}

/** 所有字段绑定只保存发布 schema 的不可变 fieldId，不保存 label 或物理列名。 */
export interface BusinessDashboardFieldRef {
  fieldId: string;
}

export interface BusinessDashboardFormDataset {
  id: string;
  name: string;
  source: {
    type: 'form';
    formCode: string;
  };
  /** Query DSL 中的字段字符串同样解释为发布 schema fieldId。 */
  query: QueryDocument;
}

export type BusinessDashboardDataset = BusinessDashboardFormDataset;

export type BusinessDashboardChartVariant = 'bar' | 'line' | 'pie';
export type BusinessDashboardLegendPosition = 'top' | 'right' | 'bottom' | 'left';

export interface BusinessDashboardChartDimension {
  field: BusinessDashboardFieldRef;
  label?: string;
}

export interface BusinessDashboardChartMetric {
  /** 引用 Dataset query.aggregates 中的稳定 alias。 */
  aggregateAlias: string;
  label?: string;
}

export interface BusinessDashboardChartSettings {
  encoding: {
    dimensions: BusinessDashboardChartDimension[];
    metrics: BusinessDashboardChartMetric[];
  };
  display: {
    variant: BusinessDashboardChartVariant;
    orientation: 'vertical' | 'horizontal';
    stack: 'none' | 'normal';
    legend: { visible: boolean; position: BusinessDashboardLegendPosition };
    labels: { visible: boolean };
  };
}

export type BusinessDashboardTableFormat =
  | 'auto'
  | 'text'
  | 'decimal'
  | 'money'
  | 'percent'
  | 'date'
  | 'datetime';

export interface BusinessDashboardTableColumn {
  id: string;
  field: BusinessDashboardFieldRef;
  label?: string;
  width?: number;
  align: 'left' | 'center' | 'right';
  format: BusinessDashboardTableFormat;
}

export interface BusinessDashboardTableSort {
  field: BusinessDashboardFieldRef;
  direction: QuerySortDirection;
}

export interface BusinessDashboardTableSettings {
  columns: BusinessDashboardTableColumn[];
  sorts: BusinessDashboardTableSort[];
  pagination: { pageSize: number };
  display: {
    density: 'compact' | 'default' | 'comfortable';
    striped: boolean;
    bordered: boolean;
    showHeader: boolean;
    emptyText: string;
  };
}

interface BusinessDashboardWidgetBase {
  id: string;
  title?: string;
  layout: BusinessDashboardLayout;
  datasetId?: string;
}

export interface BusinessDashboardChartWidget extends BusinessDashboardWidgetBase {
  type: 'chart';
  settings: BusinessDashboardChartSettings;
}

export interface BusinessDashboardTableWidget extends BusinessDashboardWidgetBase {
  type: 'table';
  settings: BusinessDashboardTableSettings;
}

export type BusinessDashboardWidget = BusinessDashboardChartWidget | BusinessDashboardTableWidget;

export interface BusinessDashboardDocument {
  version: typeof BUSINESS_DASHBOARD_VERSION;
  settings: {
    desktop: {
      columns: typeof BUSINESS_DASHBOARD_COLUMNS;
      rowHeight: number;
    };
  };
  datasets: BusinessDashboardDataset[];
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

export type BusinessDashboardWidgetPatch = {
  title?: string;
  datasetId?: string;
  settings?: BusinessDashboardChartSettings | BusinessDashboardTableSettings;
};

export type BusinessDashboardDatasetPatch = {
  name?: string;
  query?: QueryDocument;
};

/** 描述器定义设计器可创建的组件，不进入持久化文档。 */
export interface BusinessDashboardWidgetDescriptor {
  type: BusinessDashboardWidgetType;
  label: string;
  description: string;
  defaultTitle: string;
  defaultLayout: Pick<BusinessDashboardLayout, 'w' | 'h'>;
  defaultSettings: BusinessDashboardChartSettings | BusinessDashboardTableSettings;
}

export interface BusinessDashboardNormalizationResult {
  document: BusinessDashboardDocument | null;
  issues: BusinessDashboardIssue[];
}

export interface BusinessDashboardResultColumn {
  key: string;
  label: string;
  type: string;
}

export interface BusinessDashboardDatasetResult {
  datasetId: string;
  columns: BusinessDashboardResultColumn[];
  rows: Array<Record<string, unknown>>;
  total: number;
  page: number;
  pageSize: number;
}

export interface BusinessDashboardWidgetRuntime {
  status: 'idle' | 'loading' | 'success' | 'error';
  result?: BusinessDashboardDatasetResult;
  message?: string;
}
