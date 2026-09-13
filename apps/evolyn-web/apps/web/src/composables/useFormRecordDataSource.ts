import type { DataContext, DataQuery, DataRecord, DataSource } from '@evolyn.do/data';
import type { DataColumn } from '@evolyn.do/data-workspace';
import type { QueryDocument, QueryFieldType } from '@evolyn.do/query';
import type { Component, ComputedRef, ShallowRef } from 'vue';
import type { FormRecordMemberReference, FormRuntimeBootstrap } from '~/types';
import { normalizeQuery, validateQuery } from '@evolyn.do/query';
import { RiFileList2Fill, RiFileChartFill, RiTimeFill, RiUser3Fill } from '@remixicon/vue';
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
  workflowStatus: 'sys.workflowStatus',
  workflowUpdatedAt: 'sys.workflowUpdatedAt',
  submittedBy: 'sys.submittedBy',
  submittedAt: 'sys.submittedAt',
  updatedBy: 'sys.updatedBy',
  updatedAt: 'sys.updatedAt',
} as const;

/** 选项类字段的候选项（单选/下拉/多选/下拉多选），供筛选值控件渲染下拉。 */
export interface FormRecordFilterOption {
  label: string;
  value: string;
}

/**
 * 流程状态选项（与后端 wf_instance 状态枚举 + NONE 冻结一致；投影字段，
 * 事实源在流程实例）。用于筛选面板 enum 候选与列展示翻译。
 */
export const WORKFLOW_STATUS_OPTIONS = [
  { label: '进行中', value: 'RUNNING' },
  { label: '已完成', value: 'COMPLETED' },
  { label: '已驳回', value: 'REJECTED' },
  { label: '已撤回/终止', value: 'CANCELLED' },
] as const;

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

/**
 * 数据管理把一条含子表单的记录投影成多行时的私有分组键。它只服务于表格
 * 的纵向单元格合并，绝不会成为可配置列或传给后端的查询字段。
 */
const EXPANDED_RECORD_GROUP_FIELD = '__evolyn_form_record_group';
const SUBFORM_FIELD_PREFIX = '__evolyn_subform';
/** 表格行私有成员展示投影键：不属于字段值，禁止透传到筛选、导出或提交。 */
export const MEMBER_REFERENCE_PRESENTATIONS_FIELD = '__evolyn_member_references';
/** 展示名称覆盖原值后保留的成员引用，用于点击成员卡片与子表单行精确匹配。 */
const MEMBER_REFERENCE_RAW_VALUES_FIELD = '__evolyn_member_reference_raw_values';

export type FormRecordMemberPresentations = Record<string, FormRecordMemberReference[]>;
type FormRecordMemberRawValues = Record<string, unknown>;

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
      base.unshift({
        field: SYSTEM_RECORD_FIELDS.workflowInstanceNo,
        title: '流程单号',
        minWidth: 210,
        mergeCell: mergeExpandedRecordRows,
      });
    }
    return base;
  });
  // 子表单不是以 JSON 文本塞进一个单元格，而是展开为明细行：父表字段借由
  // mergeCell 纵向合并，子表字段按行对齐，使分组表头与数据结构一一对应。
  const tableRecords = computed<DataRecord[]>(() =>
    formatMoneyRecordValues(expandSubformRecords(records.value, runtime.value), runtime.value),
  );
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
        records: response.items.map((item) => {
          const memberReferences = item.memberReferences ?? {};
          const rawMemberValues = Object.fromEntries(
            Object.keys(memberReferences)
              .filter((field) => !field.startsWith(`${SUBFORM_FIELD_PREFIX}:`))
              .map((field) => [field, item.values[field]]),
          );
          return {
            id: item.id,
            ...(item.workflowInstanceNo
              ? { [SYSTEM_RECORD_FIELDS.workflowInstanceNo]: item.workflowInstanceNo }
              : {}),
            ...(item.workflowStatus && item.workflowStatus !== 'NONE'
              ? {
                  [SYSTEM_RECORD_FIELDS.workflowStatus]: item.workflowStatus,
                  ...(item.workflowUpdatedAt
                    ? { [SYSTEM_RECORD_FIELDS.workflowUpdatedAt]: item.workflowUpdatedAt }
                    : {}),
                }
              : {}),
            [SYSTEM_RECORD_FIELDS.submittedBy]: item.submittedByName,
            [SYSTEM_RECORD_FIELDS.submittedAt]: item.submittedAt,
            [SYSTEM_RECORD_FIELDS.updatedBy]: item.updatedByName,
            [SYSTEM_RECORD_FIELDS.updatedAt]: item.updatedAt,
            [MEMBER_REFERENCE_PRESENTATIONS_FIELD]: memberReferences,
            [MEMBER_REFERENCE_RAW_VALUES_FIELD]: rawMemberValues,
            ...memberDisplayValues(item.values, memberReferences),
          };
        }),
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
    tableRecords: readonly(tableRecords),
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
  const subforms = new Map(
    subformDefinitionsFromRuntime(runtime).map((definition) => [definition.field, definition]),
  );
  const formColumns = items.flatMap<DataColumn>((item): DataColumn | readonly DataColumn[] => {
    const widget = item.widget;
    const field = widget?.widgetName;
    if (!field || widget.type === 'separator' || widget.type === 'button') return [];
    if (permissions && !permissions[field]?.visible) return [];
    if (widget.type === 'subform') {
      const subform = subforms.get(field);
      if (!subform) return [];
      return [
        {
          title: subform.title,
          columns: subform.children.map((child) => ({
            field: subformFieldOf(subform.field, child.field),
            title: child.title,
            minWidth: child.type === 'datetime' ? DATETIME_COLUMN_MIN_WIDTH : FORM_COLUMN_MIN_WIDTH,
            icon: markRaw(widgetIconOfType(child.type)),
            ...(isMemberWidgetType(child.type)
              ? {
                  cellType: 'link' as const,
                }
              : {}),
          })),
        } as unknown as DataColumn,
      ];
    }
    return [
      {
        field,
        title: item.label || field,
        minWidth: widget.type === 'datetime' ? DATETIME_COLUMN_MIN_WIDTH : FORM_COLUMN_MIN_WIDTH,
        mergeCell: mergeExpandedRecordRows,
        // 列设置面板行内的字段类型图标，与设计器素材面板共用映射
        icon: markRaw(widgetIconOfType(widget.type)),
        ...(isMemberWidgetType(widget.type) ? { cellType: 'link' as const } : {}),
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
      mergeCell: mergeExpandedRecordRows,
      icon: markRaw(RiUser3Fill),
    },
    {
      field: SYSTEM_RECORD_FIELDS.submittedAt,
      title: '提交时间',
      minWidth: DATETIME_COLUMN_MIN_WIDTH,
      mergeCell: mergeExpandedRecordRows,
      icon: markRaw(RiTimeFill),
    },
    {
      field: SYSTEM_RECORD_FIELDS.updatedBy,
      title: '更新人',
      minWidth: 120,
      mergeCell: mergeExpandedRecordRows,
      icon: markRaw(RiUser3Fill),
    },
    {
      field: SYSTEM_RECORD_FIELDS.updatedAt,
      title: '更新时间',
      minWidth: DATETIME_COLUMN_MIN_WIDTH,
      mergeCell: mergeExpandedRecordRows,
      icon: markRaw(RiTimeFill),
    },
  ];
}

function isMemberWidgetType(type: string): boolean {
  return type === 'user' || type === 'usergroup';
}

/** 名称仅来自服务端本页批量投影；未解析的历史值保留受控回退，绝不触发单元格请求。 */
export function memberReferenceLabel(record: DataRecord, field: string): string {
  const names = memberReferencesOf(record, field)
    .map((entry) => entry.name)
    .filter(Boolean);
  if (names.length > 0) return names.join('、');
  const raw = record[field];
  if (Array.isArray(raw)) return raw.length > 0 ? '未知成员' : '';
  return raw ? '未知成员' : '';
}

export function memberReferencesOf(record: DataRecord, field: string): FormRecordMemberReference[] {
  const presentations = record[MEMBER_REFERENCE_PRESENTATIONS_FIELD];
  if (!isMemberPresentations(presentations)) return [];
  const entries = presentations[field] ?? [];
  const rawReferences = memberReferenceValues(memberReferenceRawValue(record, field));
  if (rawReferences.length === 0) return entries;
  // 子表单列会按行展开；只保留当前展开行原始引用命中的成员，避免把其它
  // 明细行的成员名称/卡片混入本单元格。
  return entries.filter((entry) => rawReferences.includes(entry.reference));
}

function memberReferenceRawValue(record: DataRecord, field: string): unknown {
  const rawValues = record[MEMBER_REFERENCE_RAW_VALUES_FIELD];
  if (isMemberRawValues(rawValues) && field in rawValues) return rawValues[field];
  return record[field];
}

function isMemberPresentations(value: unknown): value is FormRecordMemberPresentations {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isMemberRawValues(value: unknown): value is FormRecordMemberRawValues {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function memberReferenceValues(value: unknown): string[] {
  if (typeof value === 'string') return value.trim() ? [value.trim()] : [];
  if (!Array.isArray(value)) return [];
  return value
    .filter((entry): entry is string => typeof entry === 'string' && Boolean(entry.trim()))
    .map((entry) => entry.trim());
}

function memberDisplayValues(
  values: Record<string, unknown>,
  references: FormRecordMemberPresentations,
): Record<string, unknown> {
  const displayed = { ...values };
  for (const [field, members] of Object.entries(references)) {
    if (!field.startsWith(`${SUBFORM_FIELD_PREFIX}:`) && members.length > 0) {
      displayed[field] = members
        .map((member) => member.name)
        .filter(Boolean)
        .join('、');
    }
  }
  return displayed;
}

interface SubformDefinition {
  field: string;
  title: string;
  children: Array<{
    field: string;
    title: string;
    type: string;
    widget?: MoneyDisplayWidget;
  }>;
}

/** 发布 Schema 的金额展示所需最小视图；不让数据工作台依赖完整表单运行时。 */
interface MoneyDisplayWidget {
  currencyCode?: unknown;
  scale?: number | null;
}

/**
 * 子表单的字段名在其自身作用域内唯一。数据表的扁平取值键因此由「子表单键 +
 * 子字段键」组成；两者均受 schema 字段名规则约束，不会与普通字段冲突。
 */
function subformFieldOf(subformField: string, childField: string): string {
  return `${SUBFORM_FIELD_PREFIX}:${subformField}:${childField}`;
}

/**
 * 从发布快照提取真正可展示的子表单明细字段。布局和动作控件不属于记录值，
 * 没有明细字段的子表单也不渲染空的合并表头。
 */
function subformDefinitionsFromRuntime(runtime: FormRuntimeBootstrap | null): SubformDefinition[] {
  if (!runtime) return [];
  const permissions = runtime.permissions?.viewFields;
  return (runtime.content.content?.items ?? []).flatMap((item) => {
    const widget = item.widget;
    const field = widget?.widgetName;
    if (!field || widget.type !== 'subform') return [];
    if (permissions && !permissions[field]?.visible) return [];
    const children = widget.items.flatMap((child) => {
      const childWidget = child.widget;
      const childField = childWidget?.widgetName;
      if (!childField || childWidget.type === 'separator' || childWidget.type === 'button')
        return [];
      return [
        {
          field: childField,
          title: child.label || childField,
          type: childWidget.type,
          ...(childWidget.type === 'money'
            ? { widget: childWidget as unknown as MoneyDisplayWidget }
            : {}),
        },
      ];
    });
    return children.length === 0 ? [] : [{ field, title: item.label || field, children }];
  });
}

/**
 * 记录查询仍保留原始 decimal string 供筛选、成员引用和后续操作使用；只在工作台
 * 行副本上格式化金额。顶层与子表单明细都根据其发布快照中的币种独立呈现。
 */
function formatMoneyRecordValues(
  records: readonly DataRecord[],
  runtime: FormRuntimeBootstrap | null,
): DataRecord[] {
  if (!runtime) return [...records];
  const moneyFields = new Map<string, MoneyDisplayWidget>();
  for (const item of runtime.content.content.items) {
    if (item.widget.type === 'money') {
      moneyFields.set(item.widget.widgetName, item.widget as unknown as MoneyDisplayWidget);
    }
  }
  for (const subform of subformDefinitionsFromRuntime(runtime)) {
    for (const child of subform.children) {
      if (child.widget) moneyFields.set(subformFieldOf(subform.field, child.field), child.widget);
    }
  }
  if (moneyFields.size === 0) return [...records];
  return records.map((record) => {
    const displayed: DataRecord = { ...record };
    for (const [field, widget] of moneyFields) {
      if (typeof record[field] === 'string')
        displayed[field] = formatRecordMoney(record[field], widget);
    }
    return displayed;
  });
}

/**
 * Web 应用通过已发布包的类型声明消费 Schema，不能依赖未构建的包内运行时代码。
 * 此处仅消费已终审的 canonical decimal string 做展示，保留原始记录在数据源缓存中。
 */
function formatRecordMoney(value: string, widget: MoneyDisplayWidget): string {
  if (!/^-?\d+(\.\d+)?$/.test(value)) return value;
  const currency = typeof widget.currencyCode === 'string' ? widget.currencyCode : 'CNY';
  const symbol = RECORD_MONEY_SYMBOLS[currency] ?? '¥';
  const scale =
    typeof widget.scale === 'number' && widget.scale >= 0 && widget.scale <= 18 ? widget.scale : 2;
  const negative = value.startsWith('-');
  const [integer = '0', fraction = ''] = (negative ? value.slice(1) : value).split('.');
  // 值在提交时已按字段 scale 终审；展示只补齐尾零，不执行二次舍入或汇率转换。
  const fixedFraction = scale === 0 ? '' : `${fraction}${'0'.repeat(scale)}`.slice(0, scale);
  const grouped = integer.replace(/\B(?=(\d{3})+(?!\d))/g, ',');
  return `${negative ? '-' : ''}${symbol}${grouped}${scale === 0 ? '' : `.${fixedFraction}`}`;
}

const RECORD_MONEY_SYMBOLS: Readonly<Record<string, string>> = {
  CNY: '¥',
  USD: '$',
  EUR: '€',
  GBP: '£',
  JPY: '￥',
  HKD: 'HK$',
  KRW: '₩',
  SGD: 'S$',
  AUD: 'A$',
  CAD: 'CA$',
  CHF: 'CHF',
  AED: 'AED',
};

/**
 * 多个子表单以同一个父记录为行组按索引并排展开，行数取可见子表单明细的最大
 * 值。这样既能展示两个及以上子表单，也避免笛卡尔积导致一条记录被错误放大。
 */
function expandSubformRecords(
  records: readonly DataRecord[],
  runtime: FormRuntimeBootstrap | null,
): DataRecord[] {
  const subforms = subformDefinitionsFromRuntime(runtime);
  if (subforms.length === 0) return records.map((record) => ({ ...record }));

  return records.flatMap((record) => {
    const rowsBySubform = subforms.map((subform) => {
      const value = record[subform.field];
      return Array.isArray(value) ? value.filter(isRecordValue) : [];
    });
    const rowCount = Math.max(1, ...rowsBySubform.map((rows) => rows.length));
    const group = String(record.id);

    return Array.from({ length: rowCount }, (_, rowIndex) => {
      const expanded: DataRecord = { ...record, [EXPANDED_RECORD_GROUP_FIELD]: group };
      subforms.forEach((subform, subformIndex) => {
        const childRecord = rowsBySubform[subformIndex][rowIndex];
        for (const child of subform.children) {
          const field = subformFieldOf(subform.field, child.field);
          expanded[field] = childRecord?.[child.field];
          if (!isMemberWidgetType(child.type)) continue;
          const members = memberReferencesOf(expanded, field);
          if (members.length === 0) continue;
          const rawValues = isMemberRawValues(expanded[MEMBER_REFERENCE_RAW_VALUES_FIELD])
            ? expanded[MEMBER_REFERENCE_RAW_VALUES_FIELD]
            : {};
          expanded[MEMBER_REFERENCE_RAW_VALUES_FIELD] = { ...rawValues, [field]: expanded[field] };
          expanded[field] = members
            .map((member) => member.name)
            .filter(Boolean)
            .join('、');
        }
      });
      return expanded;
    });
  });
}

function isRecordValue(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/**
 * VTable 会将同列相邻单元格逐个交给 mergeCell 判断。只有展开自同一父记录的
 * 明细行才允许合并，避免两条普通记录的相同值被误合并。
 */
function mergeExpandedRecordRows(
  _sourceValue: unknown,
  _targetValue: unknown,
  context: {
    source: { col: number; row: number };
    target: { col: number; row: number };
    table: { getRecordByCell(col: number, row: number): unknown };
  },
): boolean {
  const source = context.table.getRecordByCell(context.source.col, context.source.row);
  const target = context.table.getRecordByCell(context.target.col, context.target.row);
  if (!isRecordValue(source) || !isRecordValue(target)) return false;
  const sourceGroup = source[EXPANDED_RECORD_GROUP_FIELD];
  return sourceGroup !== undefined && sourceGroup === target[EXPANDED_RECORD_GROUP_FIELD];
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
      field: SYSTEM_RECORD_FIELDS.workflowInstanceNo,
      label: '流程单号',
      type: 'text',
      group: 'system',
      icon: markRaw(RiFileList2Fill),
    },
    {
      field: SYSTEM_RECORD_FIELDS.workflowStatus,
      label: '流程状态',
      type: 'enum',
      group: 'system',
      options: WORKFLOW_STATUS_OPTIONS,
      icon: markRaw(RiFileChartFill),
    },
    {
      field: SYSTEM_RECORD_FIELDS.workflowUpdatedAt,
      label: '流程更新时间',
      type: 'datetime',
      group: 'system',
      icon: markRaw(RiTimeFill),
    },
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
      field: SYSTEM_RECORD_FIELDS.updatedBy,
      label: '更新人',
      type: 'enum',
      group: 'system',
      icon: markRaw(RiUser3Fill),
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
    // 数值字段族（decimal/money/percent）与 number 共用数值筛选语义：
    // 条件值暂按 number 协议出网（后端 ::numeric 比较），decimal-string
    // 查询值协议随数值运行时 Phase 7 落地。
    case 'decimal':
    case 'money':
    case 'percent':
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
