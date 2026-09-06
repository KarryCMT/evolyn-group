import type { DataQuery } from '@evolyn.do/data';
import type { FormRuntimeBootstrap } from '~/types';
import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { computed, defineComponent, shallowRef } from 'vue';
import { SYSTEM_RECORD_FIELDS, useFormRecordDataSource } from '../useFormRecordDataSource';

const api = vi.hoisted(() => ({
  getFormRuntime: vi.fn(),
  listFormRecords: vi.fn(),
}));

vi.mock('~/api/form', () => api);

function bootstrap(): FormRuntimeBootstrap {
  return {
    formCode: 'form_a',
    name: '项目',
    publishedVersion: 1,
    schemaRevision: '1',
    protocolVersion: 6,
    content: {
      content: {
        type: 'form',
        layout: 'normal',
        items: [
          {
            label: '项目名称',
            description: '',
            labelHidden: false,
            lineWidth: 12,
            widget: {
              type: 'text',
              widgetName: 'name',
              enable: true,
              visible: true,
              allowBlank: true,
            },
          },
        ],
        layout_fields: [],
        field_layout: ['name'],
        fieldShowRules: [],
        submitRule: 2,
        widget_submit_rules: {},
      },
    },
  };
}

function mountSource(query = shallowRef<DataQuery>({ keyword: '', page: 1, pageSize: 20 })) {
  const appCode = computed(() => 'app_a');
  const formCode = computed(() => 'form_a');
  const Harness = defineComponent({
    setup() {
      return useFormRecordDataSource({ appCode, formCode, query });
    },
    template: '<div />',
  });
  return { wrapper: mount(Harness), query };
}

describe('useFormRecordDataSource', () => {
  it('projects the returned workflow number as a separate read-only system column', async () => {
    api.getFormRuntime.mockResolvedValue(bootstrap());
    api.listFormRecords.mockResolvedValue({items: [{id: 7, values: {name: '申请'}, workflowInstanceNo: 'WF-20260905-000001'}], total: 1, page: 1, pageSize: 20});
    const {wrapper} = mountSource();
    await flushPromises();
    expect(wrapper.vm.records[0][SYSTEM_RECORD_FIELDS.workflowInstanceNo]).toBe('WF-20260905-000001');
    expect(wrapper.vm.columns[0]).toMatchObject({field: SYSTEM_RECORD_FIELDS.workflowInstanceNo, title: '流程单号'});
    wrapper.unmount();
  });
  it('uses DataSource to send server-side paging and keyword search', async () => {
    api.getFormRuntime.mockResolvedValue(bootstrap());
    api.listFormRecords.mockResolvedValue({
      items: [
        {
          id: 7,
          values: { name: '灵衍云' },
          submittedByMemberId: 3,
          submittedByName: '张三',
          submittedAt: '2026-09-04 10:00:00',
          updatedAt: '2026-09-04 11:00:00',
        },
      ],
      total: 41,
      page: 2,
      pageSize: 50,
    });
    const { wrapper, query } = mountSource(shallowRef({ keyword: '灵衍', page: 2, pageSize: 50 }));
    await flushPromises();

    expect(api.listFormRecords).toHaveBeenCalledWith(
      'form_a',
      expect.objectContaining({
        keyword: '灵衍',
        paging: { page: 2, pageSize: 50 },
      }),
    );
    expect(wrapper.vm.total).toBe(41);
    // 系统字段以 sys.* 键与表单字段值同层进入行记录
    expect(wrapper.vm.records).toEqual([
      {
        id: 7,
        name: '灵衍云',
        [SYSTEM_RECORD_FIELDS.submittedBy]: '张三',
        [SYSTEM_RECORD_FIELDS.submittedAt]: '2026-09-04 10:00:00',
        [SYSTEM_RECORD_FIELDS.updatedAt]: '2026-09-04 11:00:00',
      },
    ]);

    api.listFormRecords.mockResolvedValue({ items: [], total: 41, page: 3, pageSize: 50 });
    query.value = { ...query.value, page: 3 };
    await flushPromises();
    expect(api.listFormRecords).toHaveBeenLastCalledWith(
      'form_a',
      expect.objectContaining({ paging: { page: 3, pageSize: 50 } }),
    );
  });

  it('appends system columns and system filter fields after form fields', async () => {
    api.getFormRuntime.mockResolvedValue(bootstrap());
    api.listFormRecords.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20 });
    const { wrapper } = mountSource();
    await flushPromises();

    // 列：表单字段在前，系统列固定追加（列设置面板按列顺序平铺展示）
    expect(wrapper.vm.columns).toHaveLength(4);
    expect(wrapper.vm.columns[0]).toMatchObject({ field: 'name', title: '项目名称' });
    expect(wrapper.vm.columns.slice(1)).toMatchObject([
      { field: SYSTEM_RECORD_FIELDS.submittedBy, title: '提交人' },
      { field: SYSTEM_RECORD_FIELDS.submittedAt, title: '提交时间' },
      { field: SYSTEM_RECORD_FIELDS.updatedAt, title: '更新时间' },
    ]);
    // 筛选字段：系统字段进入独立分组（提交人=enum、时间=datetime）
    expect(wrapper.vm.filterFields.slice(-3)).toMatchObject([
      { field: SYSTEM_RECORD_FIELDS.submittedBy, label: '提交人', type: 'enum', group: 'system' },
      {
        field: SYSTEM_RECORD_FIELDS.submittedAt,
        label: '提交时间',
        type: 'datetime',
        group: 'system',
      },
      {
        field: SYSTEM_RECORD_FIELDS.updatedAt,
        label: '更新时间',
        type: 'datetime',
        group: 'system',
      },
    ]);
  });

  it('clears stale records and exposes a recoverable error state', async () => {
    api.getFormRuntime.mockResolvedValue(bootstrap());
    api.listFormRecords.mockRejectedValue(new Error('network'));
    const { wrapper } = mountSource();
    await flushPromises();

    expect(wrapper.vm.status).toBe('error');
    expect(wrapper.vm.records).toEqual([]);
    expect(wrapper.vm.total).toBe(0);
    expect(wrapper.vm.errorMessage).toContain('加载失败');
  });
});
