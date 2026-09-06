import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import type { FormItem, FormJsonValue, SubformWidget } from '../../schema/types';
import SubformField from '../widgets/base/SubformField.vue';

function child(widget: Record<string, unknown>, label: string): FormItem {
  return {
    widget: {
      enable: true,
      visible: true,
      allowBlank: true,
      ...widget,
    } as FormItem['widget'],
    label,
    description: '',
    labelHidden: false,
    lineWidth: 12,
  };
}

function subform(items: FormItem[], quickFill = false): FormItem<SubformWidget> {
  return {
    widget: {
      type: 'subform',
      widgetName: '_widget_subform',
      enable: true,
      visible: true,
      allowBlank: true,
      items,
      subformCreate: true,
      subformInsert: true,
      subformEdit: true,
      subformDelete: true,
      quickFill,
      pcStickyColumn: { enable: true, limit: 1 },
      mobileStickyColumn: { enable: false, limit: 1 },
      mobileViewStyle: 'vertical',
      mobileSummaryFieldCount: 3,
    },
    label: '明细',
    description: '',
    labelHidden: false,
    lineWidth: 12,
  };
}

function mountSubform(item: FormItem<SubformWidget>, modelValue: FormJsonValue[] = []) {
  return mount(SubformField, {
    props: {
      item,
      modelValue,
      disabled: false,
      readonly: false,
      errors: [],
    },
  });
}

describe('SubformField', () => {
  it('没有明细行时仍渲染子字段列头，预览可识别表格结构', () => {
    const wrapper = mountSubform(
      subform([
        child({ type: 'text', widgetName: '_widget_name' }, '名称'),
        child({ type: 'number', widgetName: '_widget_quantity' }, '数量'),
      ]),
    );

    expect(wrapper.find('.evf-subform__head').text()).toContain('名称');
    expect(wrapper.find('.evf-subform__head').text()).toContain('数量');
    expect(wrapper.find('.evf-subform__empty-cell').text()).toBe('暂无明细行');
    expect(wrapper.find('.evf-subform__empty-cell').attributes('colspan')).toBe('3');
  });

  it('新增首行按子字段生成类型化空值', async () => {
    const wrapper = mountSubform(
      subform([
        child({ type: 'text', widgetName: '_widget_name' }, '名称'),
        child({ type: 'combocheck', widgetName: '_widget_tags', options: [] }, '标签'),
      ]),
    );

    await wrapper.find('.evf-subform__add').trigger('click');

    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([
      [{ _widget_name: null, _widget_tags: [] }],
    ]);
  });

  it('快速填报复制上一行的值，但不共享行对象', async () => {
    const wrapper = mountSubform(
      subform([child({ type: 'text', widgetName: '_widget_name' }, '名称')], true),
      [{ _widget_name: '上一行' }],
    );

    await wrapper.find('.evf-subform__add').trigger('click');

    const rows = wrapper.emitted('update:modelValue')?.[0]?.[0] as Array<Record<string, string>>;
    expect(rows).toEqual([{ _widget_name: '上一行' }, { _widget_name: '上一行' }]);
    expect(rows[0]).not.toBe(rows[1]);
  });

  it('未配置子字段时显示受控提示，且不允许新增空白行', () => {
    const wrapper = mountSubform(subform([]));

    expect(wrapper.find('.evf-subform__empty').text()).toBe('暂未配置子字段');
    expect(wrapper.find('.evf-subform__add').exists()).toBe(false);
  });
});
