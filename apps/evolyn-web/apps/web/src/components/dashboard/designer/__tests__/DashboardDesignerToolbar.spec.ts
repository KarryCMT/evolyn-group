import { shallowMount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import DashboardDesignerToolbar from '../DashboardDesignerToolbar.vue';

describe('dashboard designer toolbar', () => {
  it('replaces the draft status area with an accessible help action', async () => {
    const wrapper = shallowMount(DashboardDesignerToolbar, {
      props: {
        name: '员工信息分析',
        dirty: false,
        saveStatus: 'idle',
        renaming: false,
      },
      global: {
        stubs: {
          ElButton: { template: '<button><slot /></button>' },
          ElTooltip: { template: '<span><slot /></span>' },
          WorkspaceTitleEditor: true,
        },
      },
    });

    expect(wrapper.find('.designer-toolbar__meta').exists()).toBe(false);
    expect(wrapper.find('[aria-label="帮助"]').exists()).toBe(true);

    await wrapper.get('[aria-label="帮助"]').trigger('click');
    expect(wrapper.emitted('help')).toHaveLength(1);
  });

  it('opens the extension workspace from the primary navigation', async () => {
    const wrapper = shallowMount(DashboardDesignerToolbar, {
      props: {
        name: '员工信息分析',
        dirty: false,
        saveStatus: 'idle',
      },
      global: {
        stubs: {
          ElButton: { template: '<button><slot /></button>' },
          ElTooltip: { template: '<span><slot /></span>' },
          WorkspaceTitleEditor: true,
        },
      },
    });

    await wrapper
      .getComponent({ name: 'DashboardWorkspaceNavigation' })
      .vm.$emit('navigate', 'extensions');
    expect(wrapper.emitted('navigate')).toEqual([['extensions']]);
  });
});
