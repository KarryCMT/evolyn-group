import type { FormRecordFilterField } from '~/composables/useFormRecordDataSource';
import { SYSTEM_RECORD_FIELDS } from '~/composables/useFormRecordDataSource';
import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { nextTick } from 'vue';
import FormRecordFilterPanel from '../FormRecordFilterPanel.vue';

const memberApi = vi.hoisted(() => ({ listMembers: vi.fn() }));

vi.mock('~/api/member', () => memberApi);

const fields: FormRecordFilterField[] = [
  { field: 'name', label: '名称', type: 'text' },
  { field: 'amount', label: '金额', type: 'number' },
  { field: SYSTEM_RECORD_FIELDS.submittedBy, label: '提交人', type: 'enum', group: 'system' },
  { field: SYSTEM_RECORD_FIELDS.submittedAt, label: '提交时间', type: 'datetime', group: 'system' },
];

describe('FormRecordFilterPanel', () => {
  it('emits a typed Query DSL condition and closes after applying it', async () => {
    const wrapper = mount(FormRecordFilterPanel, { props: { fields, modelValue: undefined } });
    const selects = wrapper.findAll('select');
    await selects[0].setValue('amount');
    await selects[1].setValue('gt');
    await wrapper.find('input').setValue('42');
    await wrapper.get('.form-record-filter__apply').trigger('click');

    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([
      {
        type: 'condition',
        field: 'amount',
        operator: 'gt',
        value: 42,
      },
    ]);
    expect(wrapper.emitted('close')).toHaveLength(1);
  });

  it('groups the field dropdown into form fields and system fields', () => {
    const wrapper = mount(FormRecordFilterPanel, { props: { fields, modelValue: undefined } });
    const groups = wrapper.findAll('optgroup');
    expect(groups.map((group) => group.attributes('label'))).toEqual(['表单字段', '系统字段']);
    expect(groups[1].text()).toContain('提交人');
    expect(groups[1].text()).toContain('提交时间');
  });

  it('renders a member picker for the submitter system field and emits member ids', async () => {
    memberApi.listMembers.mockResolvedValue({ items: [{ id: 5, name: '张三' }], total: 1 });
    const wrapper = mount(FormRecordFilterPanel, { props: { fields, modelValue: undefined } });
    const selects = wrapper.findAll('select');
    await selects[0].setValue(SYSTEM_RECORD_FIELDS.submittedBy);

    expect(memberApi.listMembers).toHaveBeenCalled();
    const select = wrapper.findComponent({ name: 'ElSelect' });
    expect(select.exists()).toBe(true);
    // 模拟成员选择（绕过 ElSelect 内部交互，直改 model-value 绑定）；
    // 等待重渲染解除应用按钮的 disabled 后再提交
    select.vm.$emit('update:modelValue', 5);
    await nextTick();
    await wrapper.get('.form-record-filter__apply').trigger('click');

    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([
      {
        type: 'condition',
        field: SYSTEM_RECORD_FIELDS.submittedBy,
        operator: 'eq',
        value: 5,
      },
    ]);
  });

  it('clears the current condition', async () => {
    const wrapper = mount(FormRecordFilterPanel, {
      props: {
        fields,
        modelValue: { type: 'condition', field: 'name', operator: 'contains', value: '灵衍' },
      },
    });
    await wrapper.get('button').trigger('click');
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([undefined]);
  });
});
