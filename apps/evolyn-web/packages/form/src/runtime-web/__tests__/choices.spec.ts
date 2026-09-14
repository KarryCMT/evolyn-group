import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { createWidgetItem } from '../../schema/dictionary';
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
});
