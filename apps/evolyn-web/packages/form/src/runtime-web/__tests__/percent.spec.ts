import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { createWidgetItem } from '../../schema/dictionary';
import DecimalField from '../../runtime/widgets/base/DecimalField.vue';
import WebBasicField from '../widgets/WebBasicField.vue';
import WebSubformCellEditor from '../widgets/WebSubformCellEditor.vue';

describe('WebBasicField 百分比', () => {
  it('以百分数展示比例值，并在输入时回写精确比例', async () => {
    const item = createWidgetItem('percent');
    const wrapper = mount(WebBasicField, {
      props: {
        item,
        modelValue: '0.15',
        disabled: false,
        readonly: false,
        errors: [],
      },
    });

    const input = wrapper.find('input');
    expect((input.element as HTMLInputElement).value).toBe('15');
    expect(wrapper.text()).toContain('%');

    await input.setValue('12.5');
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['0.125']);
  });

  it('移动端和子表单编辑器遵循相同的显示与回写协议', async () => {
    const item = createWidgetItem('percent');
    const mobile = mount(DecimalField, {
      props: { item, modelValue: '0.15', disabled: false, readonly: false, errors: [] },
    });
    expect((mobile.find('input').element as HTMLInputElement).value).toBe('15');
    await mobile.find('input').setValue('20');
    expect(mobile.emitted('update:modelValue')?.[0]).toEqual(['0.2']);

    const subform = mount(WebSubformCellEditor, {
      props: { field: item, modelValue: '0.15', disabled: false, readonly: false },
    });
    expect((subform.find('input').element as HTMLInputElement).value).toBe('15');
    await subform.find('input').setValue('20');
    expect(subform.emitted('update:modelValue')?.[0]).toEqual(['0.2']);
  });

  it('历史百分比未声明比例协议时保持原始存储语义，不被放大一百倍', () => {
    const item = createWidgetItem('percent');
    if (item.widget.type !== 'percent') throw new Error('expected percent widget');
    delete item.widget.percentValueMode;
    const wrapper = mount(WebBasicField, {
      props: { item, modelValue: '15', disabled: false, readonly: false, errors: [] },
    });

    expect((wrapper.find('input').element as HTMLInputElement).value).toBe('15');
  });
});
