import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { nextTick } from 'vue';
import { ElSegmented, ElSelect } from 'element-plus';
import type { FormItem, FormMultitabLayout } from '../../schema/types';
import FormSchemaPropertyPanel from '../FormSchemaPropertyPanel.vue';

const layout: FormMultitabLayout = {
  name: '_layout_tabs',
  type: 'multitab',
  tabStyle: 'style2',
  container: [
    { name: '_tab_first', type: 'tab', title: '标签页1', field_layout: [] },
    { name: '_tab_second', type: 'tab', title: '标签页2', field_layout: [] },
  ],
};

const field: FormItem = {
  widget: {
    type: 'text',
    widgetName: '_widget_name',
    enable: true,
    visible: true,
    allowBlank: true,
  },
  label: '姓名',
  description: '',
  labelHidden: false,
  lineWidth: 12,
};

describe('FormSchemaPropertyPanel', () => {
  it('选中标签页布局后在右侧展示并上抛标签配置动作', async () => {
    const wrapper = mount(FormSchemaPropertyPanel, { props: { layout } });
    expect(wrapper.text()).toContain('多标签显示');
    expect(wrapper.text()).toContain('添加标签页');

    await wrapper.find('input[aria-label="标签页标题"]').setValue('基础信息');
    expect(wrapper.emitted('rename-tab')?.[0]).toEqual(['_tab_first', '基础信息']);
  });

  it('字段属性通过副本事件提交，不直接修改父级字段对象', async () => {
    const wrapper = mount(FormSchemaPropertyPanel, { props: { item: field } });
    await wrapper.find('input[placeholder="请输入字段名称"]').setValue('联系人');
    await nextTick();

    expect(field.label).toBe('姓名');
    expect(wrapper.emitted('update-item')?.at(-1)?.[0]).toMatchObject({ label: '联系人' });
  });

  it('表单属性上抛布局切换，字段宽度按产品映射展示', async () => {
    const wrapper = mount(FormSchemaPropertyPanel, { props: { formLayout: 'grid-2' } });
    wrapper.findComponent(ElSegmented).vm.$emit('update:modelValue', 'form');
    await nextTick();

    const layoutSelect = wrapper.findComponent(ElSelect);
    layoutSelect.vm.$emit('update:modelValue', 'grid-4');
    expect(wrapper.emitted('update-form-layout')?.[0]).toEqual(['grid-4']);

    await wrapper.setProps({ item: field });
    wrapper.findComponent(ElSegmented).vm.$emit('update:modelValue', 'field');
    await nextTick();
    expect(wrapper.text()).toContain('1/4');
    expect(wrapper.text()).toContain('2/3');
    expect(wrapper.text()).toContain('整行');
  });
});
