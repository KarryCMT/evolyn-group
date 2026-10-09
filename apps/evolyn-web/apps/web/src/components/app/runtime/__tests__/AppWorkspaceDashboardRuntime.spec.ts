import type { BusinessDashboardDatasetResult } from '@evolyn.do/dashboard';
import type { AppWorkspaceAsset } from '../../workspace/appWorkspace.types';
import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';
import { isDark } from '~/composables/dark';
import AppWorkspaceDashboardRuntime from '../AppWorkspaceDashboardRuntime.vue';

const api = vi.hoisted(() => ({
  getDashboardRuntime: vi.fn(),
  queryDashboardWidget: vi.fn(),
}));

vi.mock('~/api/dashboard', () => api);

const RendererStub = defineComponent({
  name: 'BusinessDashboardRenderer',
  props: {
    document: Object,
    runtimes: Object,
    theme: String,
  },
  emits: ['retry', 'pageChange'],
  template: '<div class="renderer-stub" />',
});

function asset(targetCode: string): AppWorkspaceAsset {
  return {
    code: `menu_${targetCode}`,
    label: targetCode,
    icon: defineComponent({ render: () => null }),
    iconKey: 'chart',
    type: 'dashboard',
    targetCode,
    formType: null,
    favorited: false,
    capabilities: {
      view: true,
      favorite: true,
      actions: {
        edit: true,
        rename: true,
        switchType: false,
        referenceView: false,
        copyInApp: false,
        copyCrossApp: false,
        move: true,
        hide: false,
        delete: true,
      },
    },
  };
}

function dashboardBootstrap(code: string) {
  return {
    code,
    name: code,
    protocolVersion: 1,
    version: 3,
    document: {
      version: 1,
      settings: { desktop: { columns: 12, rowHeight: 80 } },
      datasets: [
        {
          id: 'dataset_1',
          name: '数据集',
          source: { type: 'form', formCode: 'form_demo' },
          query: {
            version: 1,
            sorts: [],
            paging: { page: 1, pageSize: 20 },
            projection: [],
            aggregates: [],
            groupBy: [],
          },
        },
      ],
      widgets: [
        {
          id: 'widget_1',
          type: 'chart',
          title: '统计图',
          datasetId: 'dataset_1',
          layout: { x: 0, y: 0, w: 6, h: 4 },
          settings: {
            encoding: { dimensions: [], metrics: [] },
            display: {
              variant: 'bar',
              orientation: 'vertical',
              stack: 'none',
              legend: { visible: true, position: 'bottom' },
              labels: { visible: false },
            },
          },
        },
      ],
      filters: [],
      interactions: [],
      publishScope: { type: 'all' },
    },
  };
}

function result(datasetId: string): BusinessDashboardDatasetResult {
  return {
    datasetId,
    columns: [{ key: 'count', label: '数量', type: 'number' }],
    rows: [{ count: 2 }],
    total: 1,
    page: 1,
    pageSize: 20,
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((next) => {
    resolve = next;
  });
  return { promise, resolve };
}

function mountRuntime(targetCode = 'dashboard_a') {
  return mount(AppWorkspaceDashboardRuntime, {
    props: { asset: asset(targetCode) },
    global: {
      directives: { loading: () => undefined },
      stubs: {
        BusinessDashboardRenderer: RendererStub,
        ElButton: true,
        ElResult: true,
      },
    },
  });
}

describe('app workspace dashboard runtime', () => {
  afterEach(() => {
    isDark.value = false;
  });

  it('加载已保存文档，并并行查询有数据集的组件', async () => {
    api.getDashboardRuntime.mockResolvedValue(dashboardBootstrap('dashboard_a'));
    api.queryDashboardWidget.mockResolvedValue(result('dataset_1'));

    const wrapper = mountRuntime();
    await flushPromises();

    expect(api.getDashboardRuntime).toHaveBeenCalledWith('dashboard_a', expect.any(AbortSignal));
    expect(api.queryDashboardWidget).toHaveBeenCalledWith(
      'dashboard_a',
      'widget_1',
      3,
      1,
      expect.any(AbortSignal),
    );
    expect(wrapper.findComponent(RendererStub).props('runtimes')).toEqual({
      widget_1: { status: 'success', result: result('dataset_1') },
    });
  });

  it('切换仪表盘时取消旧请求，迟到定义不会覆盖当前运行时', async () => {
    const first = deferred<ReturnType<typeof dashboardBootstrap>>();
    const second = deferred<ReturnType<typeof dashboardBootstrap>>();
    const signals: AbortSignal[] = [];
    api.getDashboardRuntime.mockImplementation((_code: string, signal: AbortSignal) => {
      signals.push(signal);
      return signals.length === 1 ? first.promise : second.promise;
    });
    api.queryDashboardWidget.mockResolvedValue(result('dataset_1'));

    const wrapper = mountRuntime();
    await flushPromises();
    await wrapper.setProps({ asset: asset('dashboard_b') });
    await flushPromises();

    expect(signals[0]?.aborted).toBe(true);
    second.resolve(dashboardBootstrap('dashboard_b'));
    await flushPromises();
    first.resolve(dashboardBootstrap('dashboard_a'));
    await flushPromises();

    expect(api.queryDashboardWidget).toHaveBeenCalledWith(
      'dashboard_b',
      'widget_1',
      3,
      1,
      expect.any(AbortSignal),
    );
    expect(api.queryDashboardWidget).not.toHaveBeenCalledWith(
      'dashboard_a',
      expect.anything(),
      expect.anything(),
      expect.anything(),
      expect.anything(),
    );
  });

  it('表格翻页事件复用当前修订重新查询组件', async () => {
    api.getDashboardRuntime.mockResolvedValue(dashboardBootstrap('dashboard_a'));
    api.queryDashboardWidget.mockResolvedValue(result('dataset_1'));
    const wrapper = mountRuntime();
    await flushPromises();

    wrapper.findComponent(RendererStub).vm.$emit('pageChange', 'widget_1', 2);
    await flushPromises();

    expect(api.queryDashboardWidget).toHaveBeenLastCalledWith(
      'dashboard_a',
      'widget_1',
      3,
      2,
      expect.any(AbortSignal),
    );
  });

  it('将全局暗黑模式传递给仪表盘图表和表格运行时', async () => {
    isDark.value = true;
    api.getDashboardRuntime.mockResolvedValue(dashboardBootstrap('dashboard_a'));
    api.queryDashboardWidget.mockResolvedValue(result('dataset_1'));

    const wrapper = mountRuntime();
    await flushPromises();

    expect(wrapper.findComponent(RendererStub).props('theme')).toBe('dark');
  });
});
