import type { FormRecordFilterField } from '~/composables/useFormRecordDataSource';
import type { QueryExpression } from '@evolyn.do/query';
import { SYSTEM_RECORD_FIELDS } from '~/composables/useFormRecordDataSource';
import type { VueWrapper } from '@vue/test-utils';
import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, nextTick } from 'vue';
import FormRecordFilterPanel from '../FormRecordFilterPanel.vue';

const memberApi = vi.hoisted(() => ({ listMembers: vi.fn() }));

vi.mock('~/api/member', () => memberApi);

const fields: FormRecordFilterField[] = [
  { field: 'name', label: '名称', type: 'text' },
  { field: 'amount', label: '金额', type: 'number' },
  { field: SYSTEM_RECORD_FIELDS.submittedBy, label: '提交人', type: 'enum', group: 'system' },
  { field: SYSTEM_RECORD_FIELDS.submittedAt, label: '提交时间', type: 'datetime', group: 'system' },
];

// Element Plus 的下拉层由 Popper/Teleport 管理。这里用保留 v-model 事件和
// slot 结构的轻量桩，只隔离第三方浮层实现，仍通过组件公开事件驱动业务状态。
const ElPopoverStub = defineComponent({
  name: 'ElPopover',
  props: { visible: Boolean },
  emits: ['update:visible'],
  template: `
    <div class="el-popover-stub">
      <div class="el-popover-reference-stub" @click="$emit('update:visible', true)">
        <slot name="reference" />
      </div>
      <slot />
    </div>
  `,
});

const ElSelectStub = defineComponent({
  name: 'ElSelect',
  inheritAttrs: false,
  props: { modelValue: { type: null, default: undefined } },
  emits: ['update:modelValue'],
  template: `
    <select
      v-bind="$attrs"
      :value="modelValue"
      @change="$emit('update:modelValue', $event.target.value)"
    ><slot /></select>
  `,
});

const ElOptionStub = defineComponent({
  name: 'ElOption',
  props: { label: String, value: { type: null, default: undefined } },
  template: '<option :value="value">{{ label }}<slot /></option>',
});

const ElOptionGroupStub = defineComponent({
  name: 'ElOptionGroup',
  props: { label: String },
  template: '<optgroup :label="label"><slot /></optgroup>',
});

const ElInputStub = defineComponent({
  name: 'ElInput',
  inheritAttrs: false,
  props: { modelValue: { type: null, default: undefined } },
  emits: ['update:modelValue'],
  template: `
    <input
      v-bind="$attrs"
      :value="modelValue"
      @input="$emit('update:modelValue', $event.target.value)"
    />
  `,
});

const ElButtonStub = defineComponent({
  name: 'ElButton',
  inheritAttrs: false,
  props: { disabled: Boolean },
  emits: ['click'],
  template: `
    <button v-bind="$attrs" :disabled="disabled" @click="$emit('click')"><slot /></button>
  `,
});

function mountPanel(modelValue: QueryExpression | undefined = undefined) {
  return mount(FormRecordFilterPanel, {
    props: { fields, modelValue },
    global: {
      stubs: {
        ElPopover: ElPopoverStub,
        ElSelect: ElSelectStub,
        ElOption: ElOptionStub,
        ElOptionGroup: ElOptionGroupStub,
        ElInput: ElInputStub,
        ElButton: ElButtonStub,
      },
    },
  });
}

function footerButton(wrapper: VueWrapper, label: string) {
  const button = wrapper
    .findAll('.form-record-filter__footer button')
    .find((item) => item.text() === label);
  if (!button) throw new Error(`footer button not found: ${label}`);
  return button;
}

describe('FormRecordFilterPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    memberApi.listMembers.mockResolvedValue({ items: [], total: 0 });
  });

  it('emits a typed Query DSL condition and closes after applying it', async () => {
    const wrapper = mountPanel();
    await wrapper.get('.form-record-filter-trigger').trigger('click');
    await wrapper.get('.form-record-filter__field').setValue('amount');
    await wrapper.get('.form-record-filter__operator').setValue('gt');
    await wrapper.get('.form-record-filter__value input').setValue('42');
    await footerButton(wrapper, '筛选').trigger('click');

    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([
      {
        type: 'condition',
        field: 'amount',
        operator: 'gt',
        value: 42,
      },
    ]);
    expect(wrapper.findComponent(ElPopoverStub).props('visible')).toBe(false);
  });

  it('groups the field dropdown into form fields and system fields', async () => {
    const wrapper = mountPanel();
    await wrapper.get('.form-record-filter-trigger').trigger('click');
    const groups = wrapper.findAll('optgroup');
    expect(groups.map((group) => group.attributes('label'))).toEqual(['表单字段', '系统字段']);
    expect(groups[1].text()).toContain('提交人');
    expect(groups[1].text()).toContain('提交时间');
  });

  it('renders a member picker for the submitter system field and emits member ids', async () => {
    memberApi.listMembers.mockResolvedValue({ items: [{ id: 5, name: '张三' }], total: 1 });
    const wrapper = mountPanel();
    await wrapper.get('.form-record-filter-trigger').trigger('click');
    await wrapper.get('.form-record-filter__field').setValue(SYSTEM_RECORD_FIELDS.submittedBy);
    await nextTick();

    expect(memberApi.listMembers).toHaveBeenCalled();
    const memberSelect = wrapper.get('.form-record-filter__value').findComponent(ElSelectStub);
    expect(memberSelect.exists()).toBe(true);
    // 原生 select 会把值收窄为字符串；直接发出 Element Plus 实际提供的数字 ID。
    memberSelect.vm.$emit('update:modelValue', 5);
    await nextTick();
    await footerButton(wrapper, '筛选').trigger('click');

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
    const wrapper = mountPanel({
      type: 'condition',
      field: 'name',
      operator: 'contains',
      value: '灵衍',
    });
    await wrapper.get('.form-record-filter-trigger').trigger('click');
    await footerButton(wrapper, '清空').trigger('click');
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([undefined]);
  });
});
