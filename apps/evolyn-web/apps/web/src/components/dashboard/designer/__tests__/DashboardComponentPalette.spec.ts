import { businessDashboardWidgetDescriptors } from '@evolyn.do/dashboard';
import { shallowMount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import DashboardComponentPalette from '../DashboardComponentPalette.vue';

vi.mock('@evolyn.do/dashboard', async () => {
  const actual = await vi.importActual<typeof import('@evolyn.do/dashboard')>(
    '@evolyn.do/dashboard',
  );
  return { ...actual, setupDashboardWidgetDragSources: vi.fn() };
});

describe('dashboard component palette', () => {
  it('renders the complete grouped catalog while only enabling implemented widgets', async () => {
    const wrapper = shallowMount(DashboardComponentPalette, {
      props: { descriptors: businessDashboardWidgetDescriptors },
    });

    expect(wrapper.findAll('.component-palette__group-title').map((item) => item.text())).toEqual([
      '图表',
      '组件',
      '工具',
    ]);
    expect(wrapper.findAll('.component-palette__item').map((item) => item.text())).toEqual([
      '统计表',
      '明细表',
      '数据管理表',
      '点地图',
      '日历',
      '甘特图',
      '数据列表',
      '流程分析表',
      '图片组件',
      '文本组件',
      '实时时间',
      '快捷入口',
      '嵌入页面',
      '布局容器',
      '筛选组件',
      '快捷筛选',
      '筛选按钮',
    ]);
    expect(wrapper.findAll('.dashboard-widget-palette__drag-source')).toHaveLength(2);
    expect(wrapper.findAll('.component-palette__item:disabled')).toHaveLength(15);
    expect(wrapper.get('[aria-label="点地图暂未开放"]').classes()).toContain('is-muted');

    await wrapper.get('[aria-label="添加统计表"]').trigger('click');
    await wrapper.get('[aria-label="日历暂未开放"]').trigger('click');

    expect(wrapper.emitted('add')).toEqual([[businessDashboardWidgetDescriptors[0]]]);
  });
});
