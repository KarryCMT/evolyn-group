import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => {
  const chart = {
    release: vi.fn(),
    renderSync: vi.fn(),
    updateSpec: vi.fn().mockResolvedValue(undefined),
  };
  // VChart 是构造器，使用普通 function mock 才能被 new 正确调用。
  function VChartMock() {
    return chart;
  }
  return { chart, VChartMock };
});

vi.mock('@visactor/vchart', () => ({ VChart: mocks.VChartMock }));

import EvolynChart from '../EvolynChart.vue';

describe('EvolynChart', () => {
  beforeEach(() => {
    mocks.chart.release.mockClear();
    mocks.chart.renderSync.mockClear();
    mocks.chart.updateSpec.mockClear();
  });

  it('创建图表并在卸载时释放实例', () => {
    const wrapper = mount(EvolynChart, { props: { spec: { type: 'line', data: [] } } });

    expect(mocks.chart.renderSync).toHaveBeenCalledTimes(1);
    wrapper.unmount();
    expect(mocks.chart.release).toHaveBeenCalledTimes(1);
  });

  it('替换 Spec 时增量更新已有实例', async () => {
    const wrapper = mount(EvolynChart, { props: { spec: { type: 'line', data: [] } } });
    await wrapper.setProps({ spec: { type: 'bar', data: [] } });

    expect(mocks.chart.updateSpec).toHaveBeenCalledWith({ type: 'bar', data: [] });
  });
});
