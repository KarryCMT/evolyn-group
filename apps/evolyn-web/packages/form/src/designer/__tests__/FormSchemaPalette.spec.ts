import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import FormSchemaPalette, { type FormSchemaPaletteGroup } from '../FormSchemaPalette.vue';

const icon = {};

const groups: FormSchemaPaletteGroup[] = [
  {
    key: 'basic',
    title: '基础字段',
    enabled: true,
    entries: [
      {
        type: 'number',
        label: '数字',
        icon,
        shortcuts: [
          { type: 'decimal', label: '高精度小数', icon },
          { type: 'money', label: '金额', icon },
          { type: 'percent', label: '百分比', icon },
        ],
      },
    ],
  },
];

describe('FormSchemaPalette 数值入口', () => {
  it('主入口创建普通数字，二级快捷菜单创建对应的真实字段类型', async () => {
    const wrapper = mount(FormSchemaPalette, { props: { groups } });

    await wrapper.find('.form-schema-palette__item').trigger('click');
    expect(wrapper.emitted('add-field')?.[0]?.[0]).toMatchObject({ type: 'number' });

    wrapper.findComponent({ name: 'ElDropdown' }).vm.$emit('command', 'money');
    expect(wrapper.emitted('add-field')?.[1]?.[0]).toMatchObject({ type: 'money', label: '金额' });
  });
});
