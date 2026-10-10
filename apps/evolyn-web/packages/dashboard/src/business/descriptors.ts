import type { BusinessDashboardWidgetDescriptor } from './types.js';

/** 组件目录只提供平台领域默认值，不携带 VChart/VTable 私有配置。 */
export const businessDashboardWidgetDescriptors: readonly BusinessDashboardWidgetDescriptor[] = [
  {
    type: 'chart',
    label: '统计表',
    description: '用维度与指标观察趋势和分布',
    defaultTitle: '未命名统计表',
    defaultLayout: { w: 6, h: 4 },
    defaultSettings: {
      encoding: { dimensions: [], metrics: [] },
      display: {
        variant: 'bar',
        orientation: 'vertical',
        stack: 'none',
        legend: { visible: true, position: 'bottom' },
        labels: { visible: true },
      },
    },
  },
  {
    type: 'table',
    label: '明细表',
    description: '按字段列展示业务记录明细',
    defaultTitle: '未命名明细表',
    defaultLayout: { w: 12, h: 5 },
    defaultSettings: {
      columns: [],
      sorts: [],
      pagination: { pageSize: 20 },
      display: {
        density: 'default',
        striped: true,
        bordered: false,
        showHeader: true,
        emptyText: '暂无数据',
      },
    },
  },
];
