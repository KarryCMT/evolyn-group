import { type IVChart, VChart } from '@visactor/vchart';
import type { Ref } from 'vue';
import type { EvolynChartOptions, EvolynChartSpec } from './EvolynChart.types';
import { createElementChartTheme } from './theme';

interface UseEvolynChartOptions {
  container: Readonly<Ref<HTMLElement | null>>;
  spec: () => EvolynChartSpec;
  theme: () => 'light' | 'dark';
  autoFit: () => boolean;
  options: () => EvolynChartOptions | undefined;
  onReady: (chart: IVChart) => void;
  onError: (error: Error) => void;
}

/** 管理 VChart 实例生命周期；Vue 组件只负责响应式更新与渲染容器。 */
export function useEvolynChart(options: UseEvolynChartOptions) {
  let chart: IVChart | undefined;

  function reportError(error: unknown) {
    options.onError(error instanceof Error ? error : new Error(String(error)));
  }

  function create() {
    const container = options.container.value;
    if (!container) return;

    try {
      chart = new VChart(options.spec(), {
        ...options.options(),
        dom: container,
        autoFit: options.autoFit(),
        theme: createElementChartTheme(options.theme()),
      });
      chart.renderSync();
      options.onReady(chart);
    } catch (error) {
      chart = undefined;
      reportError(error);
    }
  }

  function updateSpec() {
    if (!chart) {
      create();
      return;
    }
    chart.updateSpec(options.spec()).catch(reportError);
  }

  /** 主题为初始化选项，切换时重建实例以保证画布的全量颜色均同步。 */
  function rebuild() {
    release();
    create();
  }

  function release() {
    chart?.release();
    chart = undefined;
  }

  function getChart() {
    return chart;
  }

  return { create, getChart, rebuild, release, updateSpec };
}
