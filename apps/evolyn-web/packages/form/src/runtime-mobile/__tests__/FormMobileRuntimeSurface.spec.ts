import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import type { FormItem, FormSchemaDocument } from '../../schema/types';
import FormMobileRuntimeSurface from '../FormMobileRuntimeSurface.vue';

const schema = (): FormSchemaDocument => ({
  content: {
    type: 'form',
    layout: 'normal',
    items: [
      {
        widget: {
          type: 'text',
          widgetName: '_widget_name',
          enable: true,
          visible: true,
          allowBlank: false,
        },
        label: '姓名',
        description: '',
        labelHidden: false,
        lineWidth: 12,
      } as FormItem,
    ],
    layout_fields: [],
    field_layout: ['_widget_name'],
    fieldShowRules: [],
  },
});

describe('FormMobileRuntimeSurface', () => {
  it('使用原生字段与独立移动操作栏，不装配 Element Plus Web 外壳', () => {
    const wrapper = mount(FormMobileRuntimeSurface, {
      props: {
        schema: schema(),
        actions: [{ key: 'submit', label: '提交', behavior: 'submit', intent: 'primary' }],
      },
    });

    expect(wrapper.find('.evf-mobile-runtime-surface').exists()).toBe(true);
    expect(wrapper.find('.evf-mobile-action-bar').exists()).toBe(true);
    expect(wrapper.find('.el-scrollbar').exists()).toBe(false);
    expect(wrapper.find('input').exists()).toBe(true);
  });

  it('移动操作栏复用 Core 的校验与提交状态机', async () => {
    const submit = vi.fn(async () => ({ accepted: true }));
    const wrapper = mount(FormMobileRuntimeSurface, {
      props: {
        schema: schema(),
        adapter: { submit },
        actions: [{ key: 'submit', label: '提交', behavior: 'submit', intent: 'primary' }],
      },
    });

    await wrapper.find('input').setValue('张三');
    await wrapper.find('[data-action-key="submit"]').trigger('click');
    await flushPromises();

    expect(submit).toHaveBeenCalledTimes(1);
    expect(wrapper.emitted('submitSuccess')).toHaveLength(1);
  });
});
