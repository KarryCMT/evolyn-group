import { mount } from '@vue/test-utils';
import { defineComponent, reactive } from 'vue';
import { describe, expect, it } from 'vitest';
import { ElForm, ElFormItem } from 'element-plus';
import FormSchemaFrontendEventEditor from '../FormSchemaFrontendEventEditor.vue';
import type { FormEvent, FormEventFieldOption } from '../frontend-events';

const PopoverStub = defineComponent({
  props: { visible: Boolean },
  emits: ['update:visible'],
  template: `
    <div>
      <div class="popover-reference" @click="$emit('update:visible', true)">
        <slot name="reference" />
      </div>
      <div v-if="visible" class="popover-content"><slot /></div>
    </div>
  `,
});

function createDraft(): FormEvent {
  return reactive({
    id: 'event-1',
    name: '自动补全',
    description: '',
    enabled: true,
    trigger_type: 'widget',
    trigger: '',
    request_type: 0,
    request: {
      method: 'get',
      url: 'https://api.lingyanyun.com',
      header: [],
      body: [],
      format: 'json',
    },
    action: [],
    request_rely: [],
    action_rely: [],
    subform_fill_rule: 'merge',
  });
}

describe('FormSchemaFrontendEventEditor', () => {
  it('uses Element Plus form items for the event metadata step', () => {
    const draft = createDraft();
    const wrapper = mount(FormSchemaFrontendEventEditor, {
      props: {
        draft,
        step: 1,
        fields: [],
        requestFields: [],
        nameError: '请输入事件名称',
      },
    });

    expect(wrapper.getComponent(ElForm).props('model')).toBe(draft);
    const items = wrapper.findAllComponents(ElFormItem);
    expect(items).toHaveLength(2);
    expect(items.map((item) => item.props('prop'))).toEqual(['name', 'description']);
    expect(items[0]?.props()).toMatchObject({ required: true, error: '请输入事件名称' });
  });

  it('closes the trigger field picker after selecting a field', async () => {
    const field: FormEventFieldOption = {
      key: '_widget_name',
      label: '单行文本',
      type: 'text',
    };
    const wrapper = mount(FormSchemaFrontendEventEditor, {
      props: {
        draft: createDraft(),
        step: 2,
        fields: [field],
        requestFields: [field],
        nameError: '',
      },
      global: {
        stubs: {
          ElPopover: PopoverStub,
          ElSelect: { template: '<div><slot /></div>' },
          ElOption: true,
          ElRadioGroup: { template: '<div><slot /></div>' },
          ElRadio: { template: '<span><slot /></span>' },
        },
      },
    });

    await wrapper.get('.form-event-dialog__field-select').trigger('click');
    expect(wrapper.find('.event-field-picker').exists()).toBe(true);

    await wrapper.get('.event-field-picker__option').trigger('click');

    expect(wrapper.find('.event-field-picker').exists()).toBe(false);
    expect(wrapper.emitted('selectTrigger')?.[0]).toEqual([field]);
  });
});
