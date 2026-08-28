import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import EvolynIconPicker from '../EvolynIconPicker.vue';

describe('EvolynIconPicker', () => {
  it('选择系统图标时同步 v-model 并触发 change', async () => {
    const wrapper = mount(EvolynIconPicker, { attachTo: document.body });
    await wrapper.findAll('.evolyn-icon-picker__icon-option')[0]!.trigger('click');

    expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toMatchObject({
      type: 'remix',
      background: '#f7be54,#eda426',
    });
    expect(wrapper.emitted('change')).toHaveLength(1);
  });

  it('可以切换至自定义图标页签', async () => {
    const wrapper = mount(EvolynIconPicker);
    await wrapper.findAll('.evolyn-icon-picker__tab')[1]!.trigger('click');

    expect(wrapper.find('.evolyn-icon-picker__upload-empty').exists()).toBe(true);
  });

  it('仅展示模式不渲染任何选择交互，并使用默认图标兜底', () => {
    const wrapper = mount(EvolynIconPicker, { props: { displayOnly: true, size: 40 } });

    expect(wrapper.find('.evolyn-icon-picker__display').attributes('style')).toContain(
      'width: 40px',
    );
    expect(wrapper.find('.evolyn-icon-picker__tabs').exists()).toBe(false);
    expect(wrapper.find('.evolyn-icon-picker__display svg').exists()).toBe(true);
  });
});
