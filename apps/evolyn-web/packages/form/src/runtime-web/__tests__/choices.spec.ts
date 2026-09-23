import { flushPromises, mount } from '@vue/test-utils';
import { ElOption } from 'element-plus';
import { shallowRef } from 'vue';
import { describe, expect, it, vi } from 'vitest';
import { createWidgetItem } from '../../schema/dictionary';
import { FormRendererContextKey } from '../../runtime/store/injection';
import WebBasicField from '../widgets/WebBasicField.vue';
import WebSubformCellEditor from '../widgets/WebSubformCellEditor.vue';

describe('单选组与复选组布局', () => {
  it.each(['radiogroup', 'checkboxgroup'] as const)(
    '%s 默认在 Web 运行时横向展示，并支持显式切换为纵向',
    (type) => {
      const item = createWidgetItem(type);
      const props = { item, modelValue: null, disabled: false, readonly: false, errors: [] };
      const horizontal = mount(WebBasicField, { props });
      expect(horizontal.find('.evf-web-basic-field__choices--vertical').exists()).toBe(false);

      if (item.widget.type === 'radiogroup' || item.widget.type === 'checkboxgroup') {
        item.widget.layout = 'vertical';
      }
      const vertical = mount(WebBasicField, { props });
      expect(vertical.find('.evf-web-basic-field__choices--vertical').exists()).toBe(true);
    },
  );

  it('子表单单元格也默认横向展示选择组', () => {
    const field = createWidgetItem('checkboxgroup');
    const wrapper = mount(WebSubformCellEditor, {
      props: { field, modelValue: [], disabled: false, readonly: false },
    });

    expect(wrapper.find('.evf-web-subform-cell__choice-group--vertical').exists()).toBe(false);
  });

  it('Web 关联下拉通过运行时查询选项，不回退到静态选项', async () => {
    const item = createWidgetItem('combo');
    if (item.widget.type !== 'combo') throw new Error('测试夹具应为下拉框');
    item.widget.optionSource = {
      mode: 'related',
      related: {
        source: { type: 'form', appId: 1, sourceId: 'form_source', fieldId: '_widget_name' },
        sort: { fieldId: '_widget_name', direction: 'asc' },
        filter: { logic: 'and', conditions: [] },
      },
    };
    const queryRelatedOptions = vi.fn().mockResolvedValue([{ label: '流程单 12', value: '12' }]);
    const runtime = shallowRef({
      state: { values: {} },
      queryRelatedOptions,
    });
    const noop = () => {};
    const wrapper = mount(WebBasicField, {
      props: { item, modelValue: null, disabled: false, readonly: false, errors: [] },
      global: {
        provide: {
          [FormRendererContextKey as symbol]: {
            runtime,
            registry: {},
            registerFieldFocus: noop,
            unregisterFieldFocus: noop,
            registerFieldReveal: noop,
            unregisterFieldReveal: noop,
            focusField: () => false,
            reportUnsupportedField: noop,
          },
        },
      },
    });

    await flushPromises();

    expect(queryRelatedOptions).toHaveBeenCalledWith(item.widget.widgetName, expect.any(AbortSignal));
    expect(wrapper.findAllComponents(ElOption).map((option) => option.props('label'))).toEqual([
      '流程单 12',
    ]);
    expect(wrapper.text()).not.toContain('选项1');
  });
});
