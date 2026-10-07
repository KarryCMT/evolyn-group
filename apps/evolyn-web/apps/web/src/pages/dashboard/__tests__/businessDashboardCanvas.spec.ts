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
  useBusinessDashboardEditor,
} from '@evolyn.do/dashboard';
import { shallowMount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { shallowRef } from 'vue';

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
    expect(chartSpec.xField).toEqual(['field_region']);
    expect(chartSpec.yField).toEqual(['total_amount']);
    expect((chartSpec.data as Array<{ values: unknown[] }>)[0].values[0]).toEqual(result.rows[0]);
    expect(tableAdapter.columns[0].format?.(result.rows[0])).toBe('9007199254740993.123456');
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
