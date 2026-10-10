import { shallowMount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import DashboardAutoRefreshCard from '../DashboardAutoRefreshCard.vue';
import DashboardScheduledReminderCard from '../DashboardScheduledReminderCard.vue';

const elementStubs = {
  ElSwitch: {
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template:
      '<button class="switch-stub" type="button" @click="$emit(\'update:modelValue\', !modelValue)" />',
  },
  ElSelect: { template: '<div class="select-stub"><slot /></div>' },
  ElOption: { props: ['label'], template: '<span>{{ label }}</span>' },
  ElDatePicker: true,
  ElInput: true,
  ElCheckbox: true,
  ElTooltip: { template: '<span><slot /></span>' },
  ElButton: { template: '<button><slot /></button>' },
  MemberPickerDialog: true,
};

describe('dashboard extension cards', () => {
  it('reveals the refresh interval after automatic refresh is enabled', async () => {
    const wrapper = shallowMount(DashboardAutoRefreshCard, {
      global: { stubs: elementStubs },
    });

    expect(wrapper.find('[aria-label="刷新间隔"]').exists()).toBe(false);
    await wrapper.get('.switch-stub').trigger('click');
    expect(wrapper.find('[aria-label="刷新间隔"]').exists()).toBe(true);
    expect(wrapper.text()).toContain('15分钟');
  });

  it('reveals the complete reminder form after scheduled reminders are enabled', async () => {
    const wrapper = shallowMount(DashboardScheduledReminderCard, {
      global: { stubs: elementStubs },
    });

    expect(wrapper.find('.dashboard-reminder-card__settings').exists()).toBe(false);
    await wrapper.get('.switch-stub').trigger('click');
    expect(wrapper.find('.dashboard-reminder-card__settings').exists()).toBe(true);
    expect(wrapper.text()).toContain('选择成员或部门');
    expect(wrapper.text()).toContain('提醒方式');
  });
});
