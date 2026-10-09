import type {
  BusinessDashboardChartWidget,
  BusinessDashboardDatasetResult,
  BusinessDashboardTableWidget,
} from '@evolyn.do/dashboard';
import {
  buildBusinessChartSpec,
  buildBusinessTableAdapter,
  BusinessDashboardCanvas,
  businessDashboardWidgetDescriptors,
  BusinessDashboardWidgetView,
  createEmptyBusinessDashboardDocument,
  DashboardDesignCanvas,
  useBusinessDashboardEditor,
} from '@evolyn.do/dashboard';
import { shallowMount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { shallowRef } from 'vue';
import { resolveDashboardWidgetActionPlacement } from '../../../../../../packages/dashboard/src/designer/actionPlacement';

describe('business dashboard canvas', () => {
  it('renders a genuine empty state and removes it after adding a component', async () => {
    const document = shallowRef(createEmptyBusinessDashboardDocument());
    const editor = useBusinessDashboardEditor({
      document,
      createID: () => 'widget_chart',
    });
    const wrapper = shallowMount(BusinessDashboardCanvas, {
      props: { document: document.value, selectedWidgetId: null },
    });

    expect(wrapper.find('.business-canvas__empty').exists()).toBe(true);
    expect(wrapper.text()).toContain('空画布不会自动生成示例图表');

    editor.addWidget(businessDashboardWidgetDescriptors[0]);
    await wrapper.setProps({ document: document.value, selectedWidgetId: 'widget_chart' });
    expect(wrapper.find('.business-canvas__empty').exists()).toBe(false);
  });

  it('projects path issues as a component-level warning count', () => {
    const wrapper = shallowMount(BusinessDashboardCanvas, {
      props: {
        document: createEmptyBusinessDashboardDocument(),
        selectedWidgetId: null,
        issueWidgetIds: ['widget_a', 'widget_b'],
      },
    });

    expect(wrapper.find('.business-canvas__issue-count').text()).toBe('2 个组件需要处理');
  });

  it('uses the persisted desktop row height for the design grid', () => {
    const document = createEmptyBusinessDashboardDocument();
    document.settings.desktop.rowHeight = 96;
    const wrapper = shallowMount(BusinessDashboardCanvas, {
      props: { document, selectedWidgetId: null },
    });

    expect(wrapper.findComponent({ name: 'DashboardDesignCanvas' }).props('rowHeight')).toBe(96);
  });

  it('keeps the resize handle available while the canvas is in move mode', () => {
    const wrapper = shallowMount(DashboardDesignCanvas, {
      props: {
        modelValue: { version: 1, widgets: [] },
        widgetRegistry: {},
        interactionMode: 'move',
      },
    });
    const options = wrapper.findComponent({ name: 'EvolynGrid' }).props('options');

    expect(options).toMatchObject({
      disableDrag: false,
      disableResize: false,
      resizable: { handles: 'e,se', autoHide: false },
    });
  });

  it('moves the component action bar outside after the widget leaves the first row', () => {
    expect(resolveDashboardWidgetActionPlacement(0)).toBe('inside');
    expect(resolveDashboardWidgetActionPlacement(2)).toBe('outside');
  });

  it('projects an external palette drop into a business widget placement event', async () => {
    const wrapper = shallowMount(BusinessDashboardCanvas, {
      props: {
        document: createEmptyBusinessDashboardDocument(),
        selectedWidgetId: null,
      },
    });
    const canvas = wrapper.findComponent({ name: 'DashboardDesignCanvas' });

    canvas.vm.$emit('update:modelValue', {
      version: 1,
      widgets: [
        {
          id: 'palette-chart',
          type: 'chart',
          title: '未命名统计图',
          x: 3,
          y: 2,
          w: 6,
          h: 4,
          config: {},
        },
      ],
    });
    await wrapper.vm.$nextTick();

    expect(wrapper.emitted('drop')).toEqual([['chart', { x: 3, y: 2, w: 6, h: 4 }]]);
    expect(wrapper.emitted('update-layouts')).toEqual([[[]]]);
  });

  it('adapts platform chart and table semantics without coercing decimal strings', () => {
    const result: BusinessDashboardDatasetResult = {
      datasetId: 'dataset_sales',
      columns: [
        { key: 'field_region', label: '区域', type: 'text' },
        { key: 'total_amount', label: '销售额', type: 'decimal' },
      ],
      rows: [{ field_region: '华东', total_amount: '9007199254740993.123456' }],
      total: 1,
      page: 1,
      pageSize: 20,
    };
    const chart: BusinessDashboardChartWidget = {
      id: 'widget_chart',
      type: 'chart',
      datasetId: 'dataset_sales',
      layout: { x: 0, y: 0, w: 6, h: 4 },
      settings: {
        encoding: {
          dimensions: [{ field: { fieldId: 'field_region' } }],
          metrics: [{ aggregateAlias: 'total_amount' }],
        },
        display: {
          variant: 'bar',
          orientation: 'vertical',
          stack: 'none',
          legend: { visible: true, position: 'top' },
          labels: { visible: false },
        },
      },
    };
    const table: BusinessDashboardTableWidget = {
      id: 'widget_table',
      type: 'table',
      datasetId: 'dataset_sales',
      layout: { x: 0, y: 4, w: 12, h: 5 },
      settings: {
        columns: [
          {
            id: 'column_amount',
            field: { fieldId: 'total_amount' },
            align: 'right',
            format: 'decimal',
          },
        ],
        sorts: [{ field: { fieldId: 'total_amount' }, direction: 'desc' }],
        pagination: { pageSize: 20 },
        display: {
          density: 'default',
          striped: true,
          bordered: false,
          showHeader: true,
          emptyText: '暂无数据',
        },
      },
    };

    const chartSpec = buildBusinessChartSpec(chart, result) as unknown as Record<string, unknown>;
    const tableAdapter = buildBusinessTableAdapter(table, result);
    const darkTableAdapter = buildBusinessTableAdapter(table, result, 'dark');
    const darkCellStyle = darkTableAdapter.columns[0].style as unknown as (args: {
      row: number;
    }) => Record<string, unknown>;
    expect(chartSpec.xField).toEqual(['field_region']);
    expect(chartSpec.yField).toEqual(['total_amount']);
    expect((chartSpec.data as Array<{ values: unknown[] }>)[0].values[0]).toEqual(result.rows[0]);
    expect(tableAdapter.columns[0].format?.(result.rows[0])).toBe('9007199254740993.123456');
    expect(darkCellStyle({ row: 0 })).toMatchObject({
      bgColor: '#202225',
      borderColor: '#414243',
    });
  });

  it('keeps a failed component mounted and emits a scoped retry', async () => {
    const widget: BusinessDashboardChartWidget = {
      id: 'widget_retry',
      type: 'chart',
      title: '经营趋势',
      datasetId: 'dataset_retry',
      layout: { x: 0, y: 0, w: 6, h: 4 },
      settings: businessDashboardWidgetDescriptors[0]
        .defaultSettings as BusinessDashboardChartWidget['settings'],
    };
    const wrapper = shallowMount(BusinessDashboardWidgetView, {
      props: { widget, runtime: { status: 'error', message: '查询超时' } },
    });

    expect(wrapper.text()).toContain('查询超时');
    await wrapper.get('button').trigger('click');
    expect(wrapper.emitted('retry')).toEqual([['widget_retry']]);
  });

  it('passes dark mode to the chart renderer', () => {
    const widget: BusinessDashboardChartWidget = {
      id: 'widget_dark_chart',
      type: 'chart',
      title: '暗黑图表',
      datasetId: 'dataset_dark',
      layout: { x: 0, y: 0, w: 6, h: 4 },
      settings: businessDashboardWidgetDescriptors[0]
        .defaultSettings as BusinessDashboardChartWidget['settings'],
    };
    const wrapper = shallowMount(BusinessDashboardWidgetView, {
      props: {
        widget,
        theme: 'dark',
        runtime: {
          status: 'success',
          result: {
            datasetId: 'dataset_dark',
            columns: [],
            rows: [{}],
            total: 1,
            page: 1,
            pageSize: 20,
          },
        },
      },
    });

    expect(wrapper.findComponent({ name: 'EvolynChart' }).props('theme')).toBe('dark');
  });

  it('requests the next table page without changing the saved Dataset', async () => {
    const widget: BusinessDashboardTableWidget = {
      id: 'widget_table_page',
      type: 'table',
      datasetId: 'dataset_table',
      layout: { x: 0, y: 0, w: 12, h: 5 },
      settings: businessDashboardWidgetDescriptors[1]
        .defaultSettings as BusinessDashboardTableWidget['settings'],
    };
    const wrapper = shallowMount(BusinessDashboardWidgetView, {
      props: {
        widget,
        runtime: {
          status: 'success',
          result: {
            datasetId: 'dataset_table',
            columns: [],
            rows: [{}],
            total: 41,
            page: 1,
            pageSize: 20,
          },
        },
      },
    });

    await wrapper.findAll('.business-widget__pager button')[1]!.trigger('click');
    expect(wrapper.emitted('pageChange')).toEqual([['widget_table_page', 2]]);
  });
});
