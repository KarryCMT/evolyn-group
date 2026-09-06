import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { nextTick } from 'vue';
import type { FormItem, FormJsonValue, SubformWidget } from '../../schema/types';
import WebSubformField from '../widgets/WebSubformField.vue';
import WebSubformTable from '../widgets/WebSubformTable.vue';

function child(widget: Record<string, unknown>, label: string): FormItem {
  return {
    widget: { enable: true, visible: true, allowBlank: true, ...widget } as FormItem['widget'],
    label,
    description: '',
    labelHidden: false,
    lineWidth: 12,
  };
}

function subform(items: FormItem[]): FormItem<SubformWidget> {
  return {
    widget: {
      type: 'subform',
      widgetName: '_widget_lines',
      enable: true,
      visible: true,
      allowBlank: true,
      items,
      subformCreate: true,
      subformInsert: true,
      subformEdit: true,
      subformDelete: true,
      quickFill: true,
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

function mountSubform(
  item: FormItem<SubformWidget>,
  modelValue: FormJsonValue[] = [],
  errors: readonly string[] = [],
) {
  return mount(WebSubformField, {
    props: { item, modelValue, disabled: false, readonly: false, errors },
  });
}

describe('WebSubformField', () => {
  it('以 Element Plus 表格将操作列固定在最左侧，并保留悬停操作入口', async () => {
    const wrapper = mountSubform(
      subform([child({ type: 'text', widgetName: '_widget_name' }, '名称')]),
      [{ _widget_name: '第一行' }],
    );
    await nextTick();
    await nextTick();

    expect(wrapper.find('.el-table').exists()).toBe(true);
    const headers = wrapper.findAll('.el-table__header th').map((cell) => cell.text().trim());
    expect(headers[0]).toBe('操作');
    expect(headers).not.toContain('序号');
    expect(wrapper.find('[data-action="fullscreen"]').exists()).toBe(true);
    expect(wrapper.find('[data-action="batch"]').exists()).toBe(true);
  });

  it('添加空白行、快速填报和更多行命令都保持行对象独立', async () => {
    const wrapper = mountSubform(
      subform([child({ type: 'text', widgetName: '_widget_name' }, '名称')]),
      [{ _widget_name: '上一行' }],
    );

    await wrapper.find('[data-action="add"]').trigger('click');
    expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toEqual([
      { _widget_name: '上一行' },
      { _widget_name: null },
    ]);

    await wrapper.find('[data-action="quick-fill"]').trigger('click');
    const quickFilled = wrapper.emitted('update:modelValue')?.[1]?.[0] as Array<
      Record<string, FormJsonValue>
    >;
    expect(quickFilled).toEqual([{ _widget_name: '上一行' }, { _widget_name: '上一行' }]);
    expect(quickFilled[0]).not.toBe(quickFilled[1]);

    wrapper.findComponent(WebSubformTable).vm.$emit('rowCommand', 'copy-next', 0);
    await nextTick();
    const copied = wrapper.emitted('update:modelValue')?.[2]?.[0] as Array<
      Record<string, FormJsonValue>
    >;
    expect(copied).toEqual([{ _widget_name: '上一行' }, { _widget_name: '上一行' }]);
    expect(copied[0]).not.toBe(copied[1]);
  });

  it('批量删除受最小行数保护，子字段校验错误定位到单元格', async () => {
    const item = subform([
      child({ type: 'text', widgetName: '_widget_name', allowBlank: false }, '名称'),
    ]);
    item.widget.minRowCount = 1;
    const wrapper = mountSubform(item, [{ _widget_name: null }], ['明细第 1 行：请输入名称']);
    await nextTick();
    await nextTick();

    expect(wrapper.find('.is-error').exists()).toBe(true);
    expect(wrapper.find('.evf-web-subform__validation').text()).toContain('明细第 1 行');
    expect(wrapper.find('.evf-web-subform-table__field-error').text()).toContain('请输入名称');

    const table = wrapper.findComponent(WebSubformTable);
    table.vm.$emit('toggleBatchMode');
    table.vm.$emit('changeSelection', [0]);
    await nextTick();

    const remove = wrapper.find('[data-action="delete-selected"]');
    expect(remove.attributes('disabled')).toBeDefined();
  });
});
