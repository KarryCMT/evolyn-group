import type { DataContext, DataQuery, DataRecord, DataSource } from '@evolyn.do/data';
import type { DataColumn } from '@evolyn.do/data-workspace';
import type { QueryDocument, QueryFieldType } from '@evolyn.do/query';
import type { Component, ComputedRef, ShallowRef } from 'vue';
import type { FormRuntimeBootstrap } from '~/types';
import { normalizeQuery, validateQuery } from '@evolyn.do/query';
import { RiTimeFill, RiUser3Fill } from '@remixicon/vue';
import { computed, markRaw, readonly, shallowRef, watch } from 'vue';
import { getFormRuntime, listFormRecords } from '~/api/form';
import { widgetIconOfType } from '~/components/form/widgetIcons';

export type FormRecordDataStatus = 'loading' | 'ready' | 'error';

/**
 * 记录系统字段命名空间键（与后端 record_system_fields.go 镜像）：
 * 提交人/提交时间/更新时间是记录行物理属性，不属于发布快照字段矩阵。
 * `sys.` 前缀保证永不与 `_widget_` 字段键冲突；同键同时用于表格列取值
 * （提交人为展示名快照）与 Query DSL 筛选（提交人为成员 ID）。
 */
export const SYSTEM_RECORD_FIELDS = {
  workflowInstanceNo: 'sys.workflowInstanceNo',
  submittedBy: 'sys.submittedBy',
  submittedAt: 'sys.submittedAt',
  updatedAt: 'sys.updatedAt',
} as const;

/** 选项类字段的候选项（单选/下拉/多选/下拉多选），供筛选值控件渲染下拉。 */
export interface FormRecordFilterOption {
  label: string;
  value: string;
}

/**
 * datetime 字段的存储格式（与后端 normalizedValueSQL 的格式正则一一对应）：
 * 时间值在服务端按该格式的字符串比较，筛选值控件必须产出同格式字符串。
 */
export type FormRecordDateFieldFormat = 'date' | 'datetime' | 'month' | 'time';

export interface FormRecordFilterField {
  field: string;
  label: string;
  type: QueryFieldType;
  /** system 标记进入筛选面板「系统字段」分组；缺省为表单字段。 */
  group?: 'system';
  /** 筛选面板行内展示的字段类型图标（与列设置同源，markRaw 传入）。 */
  icon?: Component;
  /** 选项类字段候选；缺省的 enum 字段（成员/部门等 ID 值）退化为文本输入。 */
  options?: readonly FormRecordFilterOption[];
  /** datetime 字段存储格式，决定值控件形态与 value-format；缺省按 datetime。 */
  format?: FormRecordDateFieldFormat;
}

/**
 * 时间类列（表单日期时间字段与提交/更新系统列）的最小宽度：值是 19 字符的
 * 秒级时间串（YYYY-MM-DD HH:mm:ss），144px 减去单元格左右内边距后放不下会
 * 触发省略号截断。
 */
const FORM_COLUMN_MIN_WIDTH = 144;
const DATETIME_COLUMN_MIN_WIDTH = 168;

interface UseFormRecordDataSourceOptions {
  appCode: ComputedRef<string>;
  formCode: ComputedRef<string>;
  query: ShallowRef<DataQuery>;
}

/**
 * Form data page's domain adapter. The DataSource owns HTTP adaptation only;
 * route state, permission checks and query compilation remain on the server.
 */
export function useFormRecordDataSource(options: UseFormRecordDataSourceOptions) {
  const records = shallowRef<DataRecord[]>([]);
  const total = shallowRef(0);
  const status = shallowRef<FormRecordDataStatus>('loading');
  const errorMessage = shallowRef('');
  const runtime = shallowRef<FormRuntimeBootstrap | null>(null);
  let requestVersion = 0;

  // 单号是只读系统列；记录含流程绑定时展示，不作为用户表单控件保存。
  const columns = computed<DataColumn[]>(() => {
    const base = columnsFromRuntime(runtime.value);
    if (records.value.some((record) => record[SYSTEM_RECORD_FIELDS.workflowInstanceNo])) {
      base.unshift({ field: SYSTEM_RECORD_FIELDS.workflowInstanceNo, title: '流程单号', minWidth: 210 });
    }
    return base;
  });
  const filterFields = computed<FormRecordFilterField[]>(() =>
    filterFieldsFromRuntime(runtime.value),
  );
  const context = computed<DataContext>(() => ({
    resource: `forms/${options.formCode.value}/records`,
    metadata: { appCode: options.appCode.value, formCode: options.formCode.value },
  }));

  const source: DataSource = {
    async load(_context, query) {
      const document = queryDocument(query);
      const response = await listFormRecords(options.formCode.value, document);
      return {
        // 系统字段以 sys.* 键进入行记录（提交人取展示名快照），与表单字段
        // 值同层供列取数；筛选语义里的提交人值（成员 ID）仅在 Query DSL 中出现。
        records: response.items.map((item) => ({
          id: item.id,
          ...(item.workflowInstanceNo ? { [SYSTEM_RECORD_FIELDS.workflowInstanceNo]: item.workflowInstanceNo } : {}),
          [SYSTEM_RECORD_FIELDS.submittedBy]: item.submittedByName,
          [SYSTEM_RECORD_FIELDS.submittedAt]: item.submittedAt,
          [SYSTEM_RECORD_FIELDS.updatedAt]: item.updatedAt,
          ...item.values,
        })),
        total: response.total,
      };
    },
  };

  async function load(): Promise<void> {
    const code = options.formCode.value;
    const appCode = options.appCode.value;
    const version = ++requestVersion;
    if (!code.startsWith('form_') || !appCode) {
      records.value = [];
      total.value = 0;
      status.value = 'error';
      errorMessage.value = '表单上下文无效，无法加载数据';
      return;
    }
    status.value = 'loading';
    errorMessage.value = '';
    try {
      // Runtime is the immutable published snapshot used for visible columns;
      // records API independently enforces the same row/field permissions.
      const [bootstrap, page] = await Promise.all([
        runtime.value?.formCode === code
          ? Promise.resolve(runtime.value)
          : getFormRuntime(appCode, code),
        source.load(context.value, options.query.value),
      ]);
      if (version !== requestVersion) return;
      runtime.value = bootstrap;
      records.value = [...page.records];
      total.value = page.total;
      status.value = 'ready';
    } catch {
      if (version !== requestVersion) return;
      records.value = [];
      total.value = 0;
      status.value = 'error';
      errorMessage.value = '表单数据加载失败，请稍后重试';
    }
  }

  watch(
    [options.appCode, options.formCode, options.query],
    () => {
      void load();
    },
    { immediate: true },
  );

  return {
    columns,
    filterFields,
    records: readonly(records),
    total: readonly(total),
    status: readonly(status),
    errorMessage: readonly(errorMessage),
    reload: load,
    source,
  };
}

function queryDocument(query: DataQuery): QueryDocument & { keyword?: string } {
  const validation = validateQuery({
    filter: query.filter,
    sorts: query.sorts,
    paging: { page: query.page, pageSize: query.pageSize },
    projection: query.projection,
    groupBy: query.groupBy,
    aggregates: query.aggregates,
  });
  if (!validation.document) {
    throw new Error(validation.diagnostics[0]?.message ?? '筛选条件无效');
  }
  return {
    ...normalizeQuery(validation.document),
    ...(query.keyword ? { keyword: query.keyword } : {}),
  };
}

function columnsFromRuntime(runtime: FormRuntimeBootstrap | null): DataColumn[] {
  if (!runtime) return [];
  const items = runtime.content.content?.items ?? [];
  const permissions = runtime.permissions?.viewFields;
  const formColumns = items.flatMap((item) => {
    const widget = item.widget;
    const field = widget?.widgetName;
    if (!field || widget.type === 'separator' || widget.type === 'button') return [];
    if (permissions && !permissions[field]?.visible) return [];
    return [
      {
        field,
        title: item.label || field,
        minWidth: widget.type === 'datetime' ? DATETIME_COLUMN_MIN_WIDTH : FORM_COLUMN_MIN_WIDTH,
        // 列设置面板行内的字段类型图标，与设计器素材面板共用映射
        icon: markRaw(widgetIconOfType(widget.type)),
      },
    ];
  });
  // 系统列固定追加在表单字段之后：不受字段矩阵裁剪（行级可见即可见），
  // 由数据工作台「列设置」统一勾选显隐。
  return [
    ...formColumns,
    {
      field: SYSTEM_RECORD_FIELDS.submittedBy,
      title: '提交人',
      minWidth: 120,
      icon: markRaw(RiUser3Fill),
    },
    {
      field: SYSTEM_RECORD_FIELDS.submittedAt,
      title: '提交时间',
      minWidth: DATETIME_COLUMN_MIN_WIDTH,
      icon: markRaw(RiTimeFill),
    },
    {
      field: SYSTEM_RECORD_FIELDS.updatedAt,
      title: '更新时间',
      minWidth: DATETIME_COLUMN_MIN_WIDTH,
      icon: markRaw(RiTimeFill),
    },
  ];
}

function filterFieldsFromRuntime(runtime: FormRuntimeBootstrap | null): FormRecordFilterField[] {
  if (!runtime) return [];
  const permissions = runtime.permissions?.viewFields;
  const formFields = runtime.content.content.items.flatMap((item) => {
    const field = item.widget.widgetName;
    if (item.widget.type === 'separator' || item.widget.type === 'button') return [];
    if (permissions && !permissions[field]?.visible) return [];
    const type = queryFieldTypeOf(item.widget.type);
    if (!type) return [];
    return [
      {
        field,
        label: item.label || field,
        type,
        icon: markRaw(widgetIconOfType(item.widget.type)),
        ...widgetFilterExtras(item.widget),
      },
    ];
  });
  // 系统字段进入筛选面板「系统字段」分组；类型对齐后端操作符矩阵
  //（提交人=enum，值=成员 ID；时间=datetime，秒级或日期值）。
  return [
    ...formFields,
    {
      field: SYSTEM_RECORD_FIELDS.submittedBy,
      label: '提交人',
      type: 'enum',
      group: 'system',
      icon: markRaw(RiUser3Fill),
    },
    {
      field: SYSTEM_RECORD_FIELDS.submittedAt,
      label: '提交时间',
      type: 'datetime',
      group: 'system',
      icon: markRaw(RiTimeFill),
    },
    {
      field: SYSTEM_RECORD_FIELDS.updatedAt,
      label: '更新时间',
      type: 'datetime',
      group: 'system',
      icon: markRaw(RiTimeFill),
    },
  ];
}

/**
 * 提取选项类/时间类字段的筛选值元数据：选项下拉候选与存储格式。
 * 参数用结构化形状收窄 widget 联合类型（options/format 为可选成员）。
 */
function widgetFilterExtras(widget: {
  type: string;
  options?: readonly { label: string; value: string }[];
  format?: string;
}): { options?: FormRecordFilterOption[]; format?: FormRecordDateFieldFormat } {
  const extras: { options?: FormRecordFilterOption[]; format?: FormRecordDateFieldFormat } = {};
  if (Array.isArray(widget.options)) {
    extras.options = widget.options.map((option) => ({ label: option.label, value: option.value }));
  }
  if (widget.type === 'datetime' && isDateFieldFormat(widget.format)) {
    extras.format = widget.format;
  }
  return extras;
}

function isDateFieldFormat(value: unknown): value is FormRecordDateFieldFormat {
  return value === 'date' || value === 'datetime' || value === 'month' || value === 'time';
}

function queryFieldTypeOf(widgetType: string): QueryFieldType | null {
  switch (widgetType) {
    case 'text':
    case 'textarea':
      return 'text';
    case 'number':
      return 'number';
    case 'datetime':
      return 'datetime';
    case 'radiogroup':
    case 'combo':
    case 'user':
    case 'dept':
    case 'checkboxgroup':
    case 'combocheck':
    case 'usergroup':
    case 'deptgroup':
      return 'enum';
    default:
      return null;
  }
}
