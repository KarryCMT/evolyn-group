import type { BusinessDashboardWidgetDescriptor } from './types.js';

/** 阶段二组件目录；仅提供结构与空态，真实数据绑定和渲染在阶段三接入。 */
export const businessDashboardWidgetDescriptors: readonly BusinessDashboardWidgetDescriptor[] = [
  {
    type: 'chart',
    label: '统计图',
    description: '用维度与指标观察趋势和分布',
    defaultTitle: '未命名统计图',
    defaultLayout: { w: 6, h: 4 },
    defaultSettings: {},
  },
  {
    type: 'table',
    label: '明细表',
    description: '按字段列展示业务记录明细',
    defaultTitle: '未命名明细表',
    defaultLayout: { w: 12, h: 5 },
    defaultSettings: {},
  },
];
