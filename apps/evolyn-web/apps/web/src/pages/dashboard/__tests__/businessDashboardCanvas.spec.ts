import {
  BusinessDashboardCanvas,
  businessDashboardWidgetDescriptors,
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
});
