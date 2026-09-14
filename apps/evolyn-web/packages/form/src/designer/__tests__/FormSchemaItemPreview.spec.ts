import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { createWidgetItem } from '../../schema/dictionary';
import FormSchemaItemPreview from '../FormSchemaItemPreview.vue';

describe('FormSchemaItemPreview', () => {
  it.each([
    ['radiogroup', '.el-radio-group', 'input[type="radio"]'],
    ['checkboxgroup', '.el-checkbox-group', 'input[type="checkbox"]'],
  ] as const)('%s 在设计画布中使用对应的选项组组件', (type, groupSelector, inputSelector) => {
    const wrapper = mount(FormSchemaItemPreview, {
      props: { item: createWidgetItem(type) },
    });

    expect(wrapper.find(groupSelector).exists()).toBe(true);
    expect(wrapper.findAll(inputSelector)).toHaveLength(2);
    expect(wrapper.text()).toContain('选项1');
    expect(wrapper.text()).toContain('选项2');
    expect(wrapper.find('.el-select').exists()).toBe(false);
    expect(wrapper.find(groupSelector).classes()).not.toContain(
      'form-schema-item-preview__choices--vertical',
    );
  });

  it('部门多选在设计画布中呈现与运行时一致的选择区', () => {
    const wrapper = mount(FormSchemaItemPreview, {
      props: { item: createWidgetItem('deptgroup') },
    });

    const control = wrapper.get('.form-schema-item-preview__member');
    expect(control.attributes('disabled')).toBeDefined();
    expect(control.classes()).toContain('form-schema-item-preview__member--department-multiple');
    expect(control.text()).toContain('选择部门');
    expect(wrapper.text()).not.toContain('随后续版本开放');
  });
});
