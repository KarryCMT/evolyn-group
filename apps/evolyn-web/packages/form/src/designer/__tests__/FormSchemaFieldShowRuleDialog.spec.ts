import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import type { FormSchemaDocument } from '../../schema/types';
import FormSchemaFieldShowRuleDialog from '../FormSchemaFieldShowRuleDialog.vue';

const schemaDocument: FormSchemaDocument = {
  content: {
    type: 'form',
    layout: 'normal',
    items: [],
    layout_fields: [],
    field_layout: [],
    fieldShowRules: [],
    submitRule: 2,
    widget_submit_rules: {},
  },
};

describe('FormSchemaFieldShowRuleDialog', () => {
  it('独立维护新增草稿，并在取消时仅通知关闭弹窗', async () => {
    const wrapper = mount(FormSchemaFieldShowRuleDialog, {
      props: {
        modelValue: true,
        document: schemaDocument,
      },
      global: {
        stubs: {
          Teleport: true,
          ElDialog: {
            props: ['modelValue'],
            template:
              '<div v-if="modelValue"><slot /><footer><slot name="footer" /></footer></div>',
          },
          ElSelect: { template: '<div class="el-select"><slot /></div>' },
          ElOption: true,
        },
      },
    });

    expect(wrapper.text()).toContain('满足以下');
    const cancelButton = wrapper.findAll('button').find((button) => button.text() === '取消');
    expect(cancelButton).toBeDefined();

    await cancelButton!.trigger('click');
    expect(wrapper.emitted('update:model-value')).toEqual([[false]]);
    expect(wrapper.emitted('save')).toBeUndefined();
  });
});
