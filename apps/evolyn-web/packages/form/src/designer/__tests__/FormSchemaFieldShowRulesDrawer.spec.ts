import { mount } from '@vue/test-utils';
import { afterEach, describe, expect, it } from 'vitest';
import { nextTick } from 'vue';
import type { FormSchemaDocument } from '../../schema/types';
import FormSchemaFieldShowRulesDrawer from '../FormSchemaFieldShowRulesDrawer.vue';

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

afterEach(() => {
  document.body.innerHTML = '';
});

describe('FormSchemaFieldShowRulesDrawer', () => {
  it('无规则时展示插图引导与新增规则主操作', async () => {
    const wrapper = mount(FormSchemaFieldShowRulesDrawer, {
      attachTo: document.body,
      props: {
        modelValue: true,
        rules: [],
        document: schemaDocument,
        items: [],
      },
      global: {
        stubs: {
          Teleport: true,
          ElDialog: {
            props: ['modelValue'],
            template:
              '<div v-if="modelValue" class="form-field-show-rule-dialog"><slot /><footer><slot name="footer" /></footer></div>',
          },
          ElSelect: { template: '<div class="el-select"><slot /></div>' },
          ElOption: true,
        },
      },
    });
    await nextTick();

    const emptyState = document.body.querySelector('.form-field-show-rules__empty-state');
    const illustration = document.body.querySelector('.form-field-show-rules__empty-illustration');
    expect(emptyState).not.toBeNull();
    expect(illustration?.getAttribute('alt')).toBe('');
    expect(illustration?.getAttribute('src')).toMatch(/^data:image\/png;base64,/);
    expect(document.body.textContent).toContain('让字段在特定条件成立时显示，反之隐藏。');
    expect(document.body.querySelector('.form-field-show-rules__empty-action')).not.toBeNull();

    await wrapper.get('.form-field-show-rules__empty-action').trigger('click');
    await nextTick();
    expect(document.body.querySelector('.form-field-show-rule-dialog')).not.toBeNull();
    expect(document.body.textContent).toContain('满足以下');
    wrapper.unmount();
  });
});
