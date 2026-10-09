import { mount } from '@vue/test-utils';
import { ElInput } from 'element-plus';
import { describe, expect, it } from 'vitest';
import WorkspaceTitleEditor from '../WorkspaceTitleEditor.vue';

describe('workspace title editor', () => {
  it('switches to input mode and submits a normalized name', async () => {
    const wrapper = mount(WorkspaceTitleEditor, {
      props: { name: '未命名仪表盘', resourceLabel: '仪表盘' },
      global: { components: { ElInput } },
    });

    await wrapper.get('button[aria-label="修改仪表盘名称：未命名仪表盘"]').trigger('click');
    const input = wrapper.get('input[aria-label="仪表盘名称"]');
    await input.setValue('  经营驾驶舱  ');
    await input.trigger('keydown', { key: 'Enter' });

    const submitted = wrapper.emitted('submit')?.[0];
    expect(submitted?.[0]).toBe('经营驾驶舱');
    expect(submitted?.[1]).toBeTypeOf('function');
    (submitted?.[1] as () => void)();
    await wrapper.vm.$nextTick();
    expect(wrapper.find('input').exists()).toBe(false);
  });

  it('cancels editing without submitting when Escape is pressed', async () => {
    const wrapper = mount(WorkspaceTitleEditor, {
      props: { name: '经营驾驶舱', resourceLabel: '仪表盘' },
      global: { components: { ElInput } },
    });

    await wrapper.get('button').trigger('click');
    await wrapper.get('input').trigger('keydown', { key: 'Escape' });

    expect(wrapper.emitted('submit')).toBeUndefined();
    expect(wrapper.get('button').text()).toContain('经营驾驶舱');
  });
});
