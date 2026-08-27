import type { ITheme } from '@visactor/vchart';

const ELEMENT_FALLBACKS = {
  primary: '#409eff',
  success: '#67c23a',
  warning: '#e6a23c',
  danger: '#f56c6c',
  info: '#909399',
  text: '#303133',
  regular: '#606266',
  border: '#ebeef5',
  background: '#ffffff',
  fontFamily:
    "'Helvetica Neue', Helvetica, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', Arial, sans-serif",
} as const;

/** SSR 或宿主未引入 Element Plus 样式时，使用视觉接近的稳定兜底值。 */
function cssVar(name: string, fallback: string) {
  if (typeof window === 'undefined') return fallback;
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback;
}

/** 将 Element Plus 的实际级联色映射为 VChart 主题，支持 html.dark 运行时切换。 */
export function createElementChartTheme(mode: 'light' | 'dark'): ITheme {
  const primary = cssVar('--el-color-primary', ELEMENT_FALLBACKS.primary);
  const success = cssVar('--el-color-success', ELEMENT_FALLBACKS.success);
  const warning = cssVar('--el-color-warning', ELEMENT_FALLBACKS.warning);
  const danger = cssVar('--el-color-danger', ELEMENT_FALLBACKS.danger);
  const info = cssVar('--el-color-info', ELEMENT_FALLBACKS.info);
  const text = cssVar('--el-text-color-primary', ELEMENT_FALLBACKS.text);
  const regular = cssVar('--el-text-color-regular', ELEMENT_FALLBACKS.regular);
  const border = cssVar('--el-border-color-lighter', ELEMENT_FALLBACKS.border);
  const background = cssVar('--el-bg-color', ELEMENT_FALLBACKS.background);
  const fontFamily = cssVar('--el-font-family', ELEMENT_FALLBACKS.fontFamily);

  return {
    type: mode,
    background,
    fontFamily,
    colorScheme: { default: [primary, success, warning, danger, info] },
    // 部分组件主题为 VChart 的可扩展对象，不同图表类型可再由业务 Spec 覆盖。
    component: {
      axis: {
        label: { style: { fill: regular } },
        domainLine: { style: { stroke: border } },
        tick: { style: { stroke: border } },
        grid: { style: { stroke: border, lineDash: [3, 3] } },
      },
      legend: { label: { style: { fill: regular } } },
      tooltip: {
        panel: { fill: background, stroke: border },
        titleLabel: { style: { fill: text } },
        contentLabel: { style: { fill: regular } },
      },
    },
  } as ITheme;
}
