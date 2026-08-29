import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import type { FormRuntimeActionDefinition } from '../actions/types';
import FormRuntimeActionBar from '../surface/FormRuntimeActionBar.vue';

const actions: FormRuntimeActionDefinition[] = [
  { key: 'draft', label: '保存草稿', behavior: 'save-draft', order: 10 },
  { key: 'print', label: '打印', behavior: 'custom', order: 20 },
  { key: 'transfer', label: '转交', behavior: 'custom', order: 30 },
  { key: 'submit', label: '提交', behavior: 'submit', intent: 'primary', order: 100 },
];

describe('FormRuntimeActionBar', () => {
  it('桌面端保留主要动作，并将超额低频动作折叠到更多菜单', () => {
    const wrapper = mount(FormRuntimeActionBar, {
      props: { actions, formDomId: 'form-a', layout: 'desktop' },
    });
    const desktop = wrapper.find('.evf-runtime-action-bar__layout--desktop');
    expect(desktop.find('[data-action-key="submit"]').exists()).toBe(true);
    expect(desktop.findAll('[data-action-key]').length).toBe(3);
    expect(desktop.find('.evf-runtime-action-bar__more').exists()).toBe(true);
  });

  it('点击动作只向上发出类型化事件', async () => {
    const wrapper = mount(FormRuntimeActionBar, {
      props: { actions: actions.slice(0, 2), formDomId: 'form-a' },
    });
    await wrapper
      .find('.evf-runtime-action-bar__layout--desktop [data-action-key="draft"]')
      .trigger('click');
    expect(wrapper.emitted('action')?.[0]?.[0]).toMatchObject({ key: 'draft' });
  });

  it('移动端只保留两个直接动作，其余动作进入更多菜单', () => {
    const wrapper = mount(FormRuntimeActionBar, {
      props: { actions, formDomId: 'form-a', layout: 'mobile' },
    });
    const mobile = wrapper.find('.evf-runtime-action-bar__layout--mobile');
    expect(
      mobile.findAll('[data-action-key]').map((button) => button.attributes('data-action-key')),
    ).toEqual(['draft', 'submit']);
    expect(mobile.find('.evf-runtime-action-bar__mobile-more').exists()).toBe(true);
  });

  it('过滤隐藏动作，并阻止禁用或加载中的动作触发', async () => {
    const wrapper = mount(FormRuntimeActionBar, {
      props: {
        formDomId: 'form-a',
        actions: [
          { key: 'hidden', label: '隐藏', behavior: 'custom', visible: false },
          { key: 'disabled', label: '禁用', behavior: 'custom', disabled: true },
          { key: 'loading', label: '加载', behavior: 'custom', loading: true },
        ],
      },
    });
    expect(wrapper.find('[data-action-key="hidden"]').exists()).toBe(false);
    await wrapper.find('[data-action-key="disabled"]').trigger('click');
    await wrapper.find('[data-action-key="loading"]').trigger('click');
    expect(wrapper.emitted('action')).toBeUndefined();
  });
});
