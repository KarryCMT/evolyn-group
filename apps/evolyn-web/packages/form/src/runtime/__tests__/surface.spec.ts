import { ElMessageBox } from 'element-plus';
import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import type { FormItem, FormSchemaDocument } from '../../schema/types';
import FormRuntimeSurface from '../surface/FormRuntimeSurface.vue';

function schema(): FormSchemaDocument {
  return {
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
  };
}

describe('FormRuntimeSurface', () => {
  it('滚动内容区与固定操作区是同级节点', () => {
    const wrapper = mount(FormRuntimeSurface, {
      props: {
        schema: schema(),
        actions: [{ key: 'submit', label: '提交', behavior: 'submit' }],
      },
    });
    const root = wrapper.find('.evf-runtime-surface');
    expect(root.find('.evf-runtime-surface__scrollbar').exists()).toBe(true);
    expect(root.element.children[0]?.classList.contains('evf-runtime-surface__scrollbar')).toBe(
      true,
    );
    expect(root.element.children[1]?.classList.contains('evf-runtime-action-bar')).toBe(true);
  });

  it('Web 表面通过 Web Registry 渲染 Element Plus 基础字段', () => {
    const wrapper = mount(FormRuntimeSurface, {
      props: { schema: schema() },
    });

    expect(wrapper.find('.el-input').exists()).toBe(true);
  });

  it('提交动作调用渲染器同一校验/提交链', async () => {
    const submit = vi.fn(async () => ({ accepted: true }));
    const wrapper = mount(FormRuntimeSurface, {
      props: {
        schema: schema(),
        adapter: { submit },
        actions: [{ key: 'submit', label: '提交', behavior: 'submit', intent: 'primary' }],
      },
    });
    await wrapper.find('input').setValue('张三');
    await wrapper
      .find('.evf-runtime-action-bar__layout--desktop [data-action-key="submit"]')
      .trigger('click');
    await flushPromises();

    expect(submit).toHaveBeenCalledTimes(1);
    expect(wrapper.emitted('submitSuccess')).toHaveLength(1);
  });

  it('custom 动作不进入 Store，确认后原样交给宿主', async () => {
    const confirm = vi
      .spyOn(ElMessageBox, 'confirm')
      .mockResolvedValue({ action: 'confirm' } as never);
    const action = {
      key: 'print',
      label: '打印',
      behavior: 'custom' as const,
      confirmText: '确认打印？',
    };
    const wrapper = mount(FormRuntimeSurface, {
      props: { schema: schema(), actions: [action] },
    });
    const button = wrapper.find(
      '.evf-runtime-action-bar__layout--desktop [data-action-key="print"]',
    );
    await button.trigger('click');
    await flushPromises();
    expect(confirm).toHaveBeenCalledWith(
      '确认打印？',
      '请确认',
      expect.objectContaining({ confirmButtonText: '确定' }),
    );
    expect(wrapper.emitted('action')?.[0]?.[0]).toEqual(action);
  });

  it('服务端非字段错误展示在固定操作区摘要', async () => {
    const wrapper = mount(FormRuntimeSurface, {
      props: {
        schema: schema(),
        adapter: { submit: async () => ({ accepted: false, message: '表单版本已更新' }) },
        actions: [{ key: 'submit', label: '提交', behavior: 'submit' }],
      },
    });
    await wrapper.find('input').setValue('张三');
    await wrapper
      .find('.evf-runtime-action-bar__layout--desktop [data-action-key="submit"]')
      .trigger('click');
    await flushPromises();

    expect(wrapper.find('.evf-runtime-action-bar__issues').text()).toContain('表单版本已更新');
  });
});
