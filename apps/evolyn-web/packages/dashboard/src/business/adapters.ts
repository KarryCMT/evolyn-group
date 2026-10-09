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
  const variant = widget.settings.display.variant;
  const common = {
    data: [{ id: 'dataset', values: result.rows }],
    legends: {
      visible: widget.settings.display.legend.visible,
      orient: widget.settings.display.legend.position,
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
