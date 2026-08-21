import type { TYPES } from '@visactor/vtable';

/**
 * Element Plus CSS 变量的兜底值：变量缺失时（宿主应用未引入 EP 样式的场景）
 * 按 EP 默认亮色主题渲染，保证组件独立可用。
 */
const EP_VAR_FALLBACKS = {
  primary: '#409eff',
  primaryLight9: '#ecf5ff',
  textPrimary: '#303133',
  textRegular: '#606266',
  textSecondary: '#909399',
  borderLighter: '#ebeef5',
  fillLight: '#f5f7fa',
  bgColor: '#ffffff',
  fontFamily:
    "'Helvetica Neue', Helvetica, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', Arial, sans-serif",
} as const;

/** 读取 documentElement 上的 CSS 变量；缺失或 SSR 环境返回兜底值 */
function cssVar(name: string, fallback: string): string {
  if (typeof window === 'undefined') return fallback;
  const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  return value || fallback;
}

/**
 * 构建跟随 Element Plus 主题的 VTable 主题。
 *
 * VTable 是 canvas 渲染，无法直接消费 --el-* CSS 变量，因此在运行时读取
 * documentElement 上级联后的实际值再映射为 ITableThemeDefine。暗色模式由
 * 应用切换 html.dark 后变量值自动翻转，mode 入参仅作为「重算触发器」——
 * EvolynTable 的 theme prop 变化会驱动本函数重新执行并 updateOption。
 */
export function createElementTheme(mode: 'light' | 'dark'): TYPES.ITableThemeDefine {
  // 依赖收集：读取 mode 让 computed 感知 prop 变化，真实取值跟随级联变量
  void mode;

  const primary = cssVar('--el-color-primary', EP_VAR_FALLBACKS.primary);
  const primaryLight9 = cssVar('--el-color-primary-light-9', EP_VAR_FALLBACKS.primaryLight9);
  const textPrimary = cssVar('--el-text-color-primary', EP_VAR_FALLBACKS.textPrimary);
  const textRegular = cssVar('--el-text-color-regular', EP_VAR_FALLBACKS.textRegular);
  const textSecondary = cssVar('--el-text-color-secondary', EP_VAR_FALLBACKS.textSecondary);
  const borderLighter = cssVar('--el-border-color-lighter', EP_VAR_FALLBACKS.borderLighter);
  const fillLight = cssVar('--el-fill-color-light', EP_VAR_FALLBACKS.fillLight);
  const bgColor = cssVar('--el-bg-color', EP_VAR_FALLBACKS.bgColor);
  const fontFamily = cssVar('--el-font-family', EP_VAR_FALLBACKS.fontFamily);

  return {
    // 表格绘制范围外的画布底色与页面背景保持一致，避免暗色模式下露白
    underlayBackgroundColor: bgColor,
    // 表头对齐 el-table：浅灰底 + 主文字色 + 半粗字重
    headerStyle: {
      color: textPrimary,
      bgColor: fillLight,
      borderColor: borderLighter,
      fontSize: 14,
      fontWeight: 600,
      fontFamily,
      padding: [10, 12, 10, 12],
    },
    bodyStyle: {
      color: textRegular,
      bgColor,
      borderColor: borderLighter,
      fontSize: 14,
      fontFamily,
      padding: [9, 12, 9, 12],
      // 悬停高亮对齐 el-table 的 --el-table-row-hover-bg-color（取 fill-color-light）
      hover: { cellBgColor: fillLight },
    },
    // 外框与单元格共用 lighter 边框线，形成简道云式的浅网格
    frameStyle: {
      borderColor: borderLighter,
      borderLineWidth: 1,
      cornerRadius: 0,
    },
    // 选中态对齐 el-table：主色描边 + primary-light-9 行底色
    selectionStyle: {
      cellBorderColor: primary,
      cellBorderLineWidth: 1,
      cellBgColor: primaryLight9,
      inlineRowBgColor: primaryLight9,
    },
    // 滚动条用弱化文字色，视觉接近 EP 滚动条
    scrollStyle: {
      scrollSliderColor: textSecondary,
      scrollRailColor: borderLighter,
      visible: 'scrolling',
    },
    // 表头功能性图标（排序/冻结）统一次级文字色
    functionalIconsStyle: {
      sort_color: textSecondary,
      frozen_color: textSecondary,
      collapse_color: textSecondary,
      expand_color: textSecondary,
      dragReorder_color: textSecondary,
    },
  };
}
