import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';

import FormDataPage from '../data.vue';

const state = vi.hoisted(() => ({ operations: ['batch_print'] as string[] }));

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { appCode: 'app_demo', formCode: 'form_users' }, query: {} }),
}));

vi.mock('@evolyn.do/data-workspace', async () => {
  const { defineComponent, h, shallowRef } = await import('vue');
  return {
    useDataWorkspace: () => ({ query: shallowRef({ page: 1, pageSize: 20 }), updateQuery: vi.fn() }),
    DataWorkspace: defineComponent({
      name: 'DataWorkspace',
      props: ['actions', 'selectionResetVersion'],
      emits: ['action', 'selection-change', 'update-query', 'cell-click'],
      setup(props, { emit }) {
        return () => h('div', { class: 'workspace' }, [
          h('button', { class: 'select', onClick: () => emit('selection-change', [11, 12]) }, 'select'),
          h('button', { class: 'download', onClick: () => emit('action', 'download-labels') }, 'download'),
          h('button', { class: 'clear', onClick: () => emit('action', 'clear-selection') }, 'clear'),
          h('pre', { class: 'actions' }, JSON.stringify(props.actions)),
          h('span', { class: 'reset-version' }, String(props.selectionResetVersion)),
        ]);
      },
    }),
  };
});

vi.mock('~/composables/useFormRecordDataSource', async () => {
  const { computed, shallowRef } = await import('vue');
  return {
    SYSTEM_RECORD_FIELDS: { recordId: 'sys.recordId' },
    memberReferencesOf: () => [],
    useFormRecordDataSource: () => ({
      columns: shallowRef([]), filterFields: shallowRef([]), tableRecords: shallowRef([{ id: 11 }, { id: 12 }]),
      total: shallowRef(2), status: shallowRef('ready'), errorMessage: shallowRef(''), reload: vi.fn(),
      runtime: computed(() => ({ permissions: { operations: state.operations } })),
    }),
  };
});

vi.mock('~/api/form', () => ({ deleteFormRecords: vi.fn() }));

const childStubs = {
  FormRecordCreateDialog: { template: '<div />' },
  FormRecordFilterPanel: { template: '<div />' },
  FormRecordMemberCardPopover: { template: '<div />' },
  FormRecordLabelDownloadDialog: {
    name: 'FormRecordLabelDownloadDialog',
    props: ['modelValue', 'formCode', 'recordIds'],
    template: '<div class="label-dialog" :data-open="modelValue" :data-records="recordIds.join(\',\')" />',
  },
};

describe('form data label download entry', () => {
  it('仅把当前勾选记录交给二维码标签弹窗，并可清空选择', async () => {
    state.operations = ['batch_print'];
    const wrapper = mount(FormDataPage, { global: { stubs: childStubs } });
    await wrapper.get('.select').trigger('click');
    expect(wrapper.get('.actions').text()).toContain('下载二维码标签');
    await wrapper.get('.download').trigger('click');
    await flushPromises();
    expect(wrapper.get('.label-dialog').attributes('data-open')).toBe('true');
    expect(wrapper.get('.label-dialog').attributes('data-records')).toBe('11,12');

    await wrapper.get('.clear').trigger('click');
    expect(wrapper.get('.actions').text()).not.toContain('下载二维码标签');
    expect(wrapper.get('.reset-version').text()).toBe('1');
  });

  it('无 batch_print 权限时禁用下载动作', async () => {
    state.operations = [];
    const wrapper = mount(FormDataPage, { global: { stubs: childStubs } });
    await wrapper.get('.select').trigger('click');
    const actions = JSON.parse(wrapper.get('.actions').text()) as Array<{ children?: Array<{ key: string; disabled?: boolean }> }>;
    const download = actions.flatMap((action) => action.children ?? []).find((action) => action.key === 'download-labels');
    expect(download?.disabled).toBe(true);
  });
});
