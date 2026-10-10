import type { EvolynChartSpec, EvolynTableColumn, EvolynTableOptions } from '@evolyn.do/ui';
import type {
  BusinessDashboardChartWidget,
  BusinessDashboardDatasetResult,
  BusinessDashboardTableWidget,
} from './types.js';

// VTable 使用 Canvas 绘制单元格，无法直接解析组件 CSS 变量，因此在适配层按主题提供实色。
const TABLE_THEME_COLORS = {
  light: { striped: '#f8fafb', border: '#e4e9ed' },
  dark: { striped: '#202225', border: '#414243' },
} as const;

/** 平台图表语义到 VChart 的唯一适配点；持久化文档中禁止出现原生 spec。 */
export function buildBusinessChartSpec(
  widget: BusinessDashboardChartWidget,
  result: BusinessDashboardDatasetResult,
): EvolynChartSpec {
  const dimensions = widget.settings.encoding.dimensions.map((item) => item.field.fieldId);
  const metrics = widget.settings.encoding.metrics.map((item) => item.aggregateAlias);
  const columnLabels = new Map(result.columns.map((item) => [item.key, item.label]));
  const metricLabels = new Map(
    widget.settings.encoding.metrics.map((item) => [
      item.aggregateAlias,
      item.label || columnLabels.get(item.aggregateAlias) || item.aggregateAlias,
    ]),
  );
  const variant = widget.settings.display.variant;
  const primaryCategoryCount = dimensions[0]
    ? new Set(result.rows.map((row) => String(row[dimensions[0]!]))).size
    : 0;
  const singleCategoryBarWidth = Math.max(
    24,
    Math.min(74, Math.floor(600 / Math.max(result.rows.length, 1))),
  );
  const common = {
    data: [{ id: 'dataset', values: result.rows }],
    // 设计器与运行态共用稳定的多系列色板，避免主题主色覆盖全部业务序列。
    color: ['#59a7df', '#70d28c', '#f2c774', '#f58c7e', '#75cbc7', '#9494ad', '#738fd9', '#efa15e'],
    legends: {
      visible: widget.settings.display.legend.visible,
      orient: widget.settings.display.legend.position,
      // VChart 默认会展示内部系列 ID；在适配层映射回业务指标名，避免泄露实现细节。
      item: {
        label: {
          formatMethod: (text: string | number, _item: unknown, index: number) =>
            dimensions.length > 1
              ? String(text)
              : metricLabels.get(String(text)) ??
                metricLabels.get(metrics[index] ?? '') ??
                String(text),
        },
      },
    },
    label: { visible: widget.settings.display.labels.visible },
  };
  if (variant === 'pie') {
    return {
      ...common,
      type: 'pie',
      categoryField: dimensions[0],
      valueField: metrics[0],
    } as EvolynChartSpec;
  }
  return {
    ...common,
    type: variant,
    // 单个主分类即使拆成多个系列也应放宽柱体；多分类仍交由 VChart 自适应避免重叠。
    ...(variant === 'bar' && primaryCategoryCount === 1
      ? {
          barWidth: singleCategoryBarWidth,
          // 单分类图缩小类目轴两侧留白，使多系列柱体在宽画布中保持目标稿的视觉占比。
          axes:
            widget.settings.display.orientation === 'vertical'
              ? [{ orient: 'bottom', bandPadding: 0.08 }, { orient: 'left' }]
              : [{ orient: 'left', bandPadding: 0.08 }, { orient: 'bottom' }],
        }
      : {}),
    xField: widget.settings.display.orientation === 'vertical' ? dimensions : metrics,
    yField: widget.settings.display.orientation === 'vertical' ? metrics : dimensions,
    seriesField: dimensions.length > 1 ? dimensions[1] : undefined,
    stack: widget.settings.display.stack === 'normal',
  } as EvolynChartSpec;
}

/** 平台明细表列/展示语义到 VTable 的适配点。 */
export function buildBusinessTableAdapter(
  widget: BusinessDashboardTableWidget,
  result: BusinessDashboardDatasetResult,
  theme: 'light' | 'dark' = 'light',
): { columns: EvolynTableColumn[]; options: EvolynTableOptions } {
  const colors = TABLE_THEME_COLORS[theme];
  const metadata = new Map(result.columns.map((item) => [item.key, item]));
  const rowHeight = { compact: 32, default: 40, comfortable: 48 }[widget.settings.display.density];
  const columns: EvolynTableColumn[] = widget.settings.columns.map((item) => ({
    field: item.field.fieldId,
    title: item.label || metadata.get(item.field.fieldId)?.label || item.field.fieldId,
    width: item.width,
    align: item.align,
    sortable: widget.settings.sorts.some((sort) => sort.field.fieldId === item.field.fieldId),
    format: (record) => formatTableValue(record[item.field.fieldId], item.format),
    style: (args: { row: number }) => ({
      ...(widget.settings.display.striped && args.row % 2 === 0
        ? { bgColor: colors.striped }
        : {}),
      borderColor: colors.border,
      borderLineWidth: widget.settings.display.bordered ? 1 : 0,
    }),
  }));
  return {
    columns,
    options: {
      showHeader: widget.settings.display.showHeader,
      defaultHeaderRowHeight: rowHeight,
      defaultRowHeight: rowHeight,
      hover: { highlightMode: 'row' },
    },
  };
}

function formatTableValue(
  value: unknown,
  format: BusinessDashboardTableWidget['settings']['columns'][number]['format'],
) {
  if (value === null || value === undefined) return '';
  if (format === 'percent') return `${value}%`;
  if (format === 'money') return `¥${value}`;
  return String(value);
}
