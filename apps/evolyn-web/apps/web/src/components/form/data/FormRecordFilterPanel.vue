<script setup lang="ts">
import type {
  QueryCondition,
  QueryConjunction,
  QueryExpression,
  QueryOperator,
  QueryValue,
} from '@evolyn.do/query';
import type {
  FormRecordDateFieldFormat,
  FormRecordFilterField,
} from '~/composables/useFormRecordDataSource';
import { QUERY_OPERATORS_BY_FIELD_TYPE } from '@evolyn.do/query';
import { RiAddFill, RiCloseFill, RiFilter3Fill } from '@remixicon/vue';
import {
  ElButton,
  ElDatePicker,
  ElInput,
  ElMessage,
  ElOption,
  ElOptionGroup,
  ElPopover,
  ElSelect,
  ElTimePicker,
} from 'element-plus';
import { computed, ref, shallowRef, watch } from 'vue';
import { listMembers, type MemberListItemDto } from '~/api/member';

defineOptions({ name: 'FormRecordFilterPanel' });

const model = defineModel<QueryExpression | undefined>({ required: true });
const props = defineProps<{ fields: readonly FormRecordFilterField[] }>();

/** 条件行数上限：对齐后端 Query group 子条件上限（record_list_query.go）。 */
const MAX_FILTER_ROWS = 50;
/** 提交人系统字段：值=成员 ID，渲染成员远程搜索选择器。 */
const MEMBER_FIELD = 'sys.submittedBy';

/** 操作符中文文案：仅展示层字典，语义仍以 @evolyn.do/query 协议为准。 */
const OPERATOR_LABELS: Record<QueryOperator, string> = {
  eq: '等于',
  neq: '不等于',
  contains: '包含',
  notContains: '不包含',
  startsWith: '开头为',
  endsWith: '结尾为',
  gt: '大于',
  gte: '大于等于',
  lt: '小于',
  lte: '小于等于',
  in: '属于',
  notIn: '不属于',
  between: '介于',
  isNull: '为空',
  isNotNull: '不为空',
};

/** datetime 字段存储格式 → 出网 value-format（服务端按该格式做字符串比较）。 */
const DATE_VALUE_FORMAT: Record<FormRecordDateFieldFormat, string> = {
  date: 'YYYY-MM-DD',
  datetime: 'YYYY-MM-DD HH:mm:ss',
  month: 'YYYY-MM',
  time: 'HH:mm',
};
/** 日期时间选择器的默认时刻：单值取零点，区间覆盖整天（00:00:00–23:59:59）。 */
const SINGLE_DEFAULT_TIME = new Date(2000, 0, 1, 0, 0, 0);
const RANGE_DEFAULT_TIMES: [Date, Date] = [
  new Date(2000, 0, 1, 0, 0, 0),
  new Date(2000, 0, 1, 23, 59, 59),
];

/**
 * 日期选择器 type：time 格式在模板中已由 ElTimePicker 分支前置承接，
 * 此处只剩 date/datetime/month 三种格式（及其 range 组合）。
 */
function datePickerTypeOf(
  view: FilterRowView,
): 'date' | 'datetime' | 'month' | 'daterange' | 'datetimerange' | 'monthrange' {
  return (view.row.operator === 'between' ? `${view.format}range` : view.format) as ReturnType<
    typeof datePickerTypeOf
  >;
}

/**
 * 单条条件的值编辑态。形态由字段类别决定：文本/数字/布尔/时间为字符串
 * （区间为二元组），选项/成员为数组（单值操作符提交时取首项）。
 */
type RowValue = string | number[] | string[] | [string, string] | null;

interface FilterRow {
  key: number;
  field: string;
  operator: QueryOperator;
  value: RowValue;
}

type FieldKind = 'text' | 'number' | 'boolean' | 'datetime' | 'options' | 'member';

const open = shallowRef(false);
const conjunction = shallowRef<QueryConjunction>('and');
// 行对象在行内被各类值控件就地写入，使用深响应保证模板即时更新
const rows = ref<FilterRow[]>([]);
let rowKeySeed = 0;

const formFields = computed(() => props.fields.filter((item) => item.group !== 'system'));
const systemFields = computed(() => props.fields.filter((item) => item.group === 'system'));
const fieldMap = computed(() => new Map(props.fields.map((item) => [item.field, item])));
const hasFilter = computed(() => model.value !== undefined);
const hasMemberField = computed(() => props.fields.some((item) => item.field === MEMBER_FIELD));

/** 模板行视图：预解析字段元信息，避免模板内重复查表与函数调用。 */
interface FilterRowView {
  row: FilterRow;
  field?: FormRecordFilterField;
  kind: FieldKind | null;
  operators: readonly QueryOperator[];
  format: FormRecordDateFieldFormat;
  requiresValue: boolean;
  isSet: boolean;
}

const rowViews = computed<FilterRowView[]>(() =>
  rows.value.map((row) => {
    const field = fieldMap.value.get(row.field);
    return {
      row,
      field,
      kind: field ? kindOf(field) : null,
      operators: operatorsOf(field),
      format: field?.format ?? 'datetime',
      requiresValue: !isNoValueOperator(row.operator),
      isSet: isSetOperator(row.operator),
    };
  }),
);

const completeCount = computed(() => rows.value.filter((row) => rowComplete(row)).length);

function kindOf(field: FormRecordFilterField): FieldKind {
  if (field.field === MEMBER_FIELD) return 'member';
  if (field.type === 'number') return 'number';
  if (field.type === 'boolean') return 'boolean';
  if (field.type === 'datetime') return 'datetime';
  if (field.type === 'enum' && field.options?.length) return 'options';
  return 'text';
}

function operatorsOf(field: FormRecordFilterField | undefined): readonly QueryOperator[] {
  return field ? (QUERY_OPERATORS_BY_FIELD_TYPE[field.type] ?? []) : [];
}

function isSetOperator(operator: QueryOperator): boolean {
  return operator === 'in' || operator === 'notIn';
}

function isNoValueOperator(operator: QueryOperator): boolean {
  return operator === 'isNull' || operator === 'isNotNull';
}

/** 各字段类别/操作符组合下的空值形态；切换字段或值形态变化时用于重置。 */
function valueBlankOf(field: FormRecordFilterField | undefined, operator: QueryOperator): RowValue {
  if (!field) return '';
  switch (kindOf(field)) {
    case 'options':
    case 'member':
      return [];
    case 'datetime':
      return operator === 'between' ? null : '';
    default:
      return '';
  }
}

function blankRow(): FilterRow {
  const first = props.fields[0];
  const operator = first ? (operatorsOf(first)[0] ?? 'eq') : 'eq';
  return {
    key: ++rowKeySeed,
    field: first?.field ?? '',
    operator,
    value: valueBlankOf(first, operator),
  };
}

/** 行完整性 = 字段有效 + 操作符合法 + 需要值时已填。不完整的行提交时被丢弃。 */
function rowComplete(row: FilterRow): boolean {
  const field = fieldMap.value.get(row.field);
  if (!field || !operatorsOf(field).includes(row.operator)) return false;
  if (isNoValueOperator(row.operator)) return true;
  const value = row.value;
  if (Array.isArray(value)) return value.length > 0 && value.every((item) => item !== '');
  return typeof value === 'string' && value.trim() !== '';
}

/* ---- 编辑态 ↔ Query DSL 双向换算 ---- */

/** 打开面板时从已生效的筛选重建编辑态；嵌套 group 展平为单层条件列表。 */
function resetEditor(): void {
  const expression = model.value;
  conjunction.value =
    expression?.type === 'group' && expression.conjunction === 'or' ? 'or' : 'and';
  const conditions: QueryCondition[] = [];
  collectConditions(expression, conditions);
  const rebuilt = conditions
    .map((condition) => {
      const field = fieldMap.value.get(condition.field);
      return field && operatorsOf(field).includes(condition.operator)
        ? conditionToRow(condition, field)
        : null;
    })
    .filter((row): row is FilterRow => row !== null);
  rows.value = rebuilt.length ? rebuilt : [blankRow()];
}

function collectConditions(expression: QueryExpression | undefined, sink: QueryCondition[]): void {
  if (!expression) return;
  if (expression.type === 'condition') {
    sink.push(expression);
    return;
  }
  for (const child of expression.children) collectConditions(child, sink);
}

function conditionToRow(condition: QueryCondition, field: FormRecordFilterField): FilterRow {
  const kind = kindOf(field);
  const operator = condition.operator;
  let value: RowValue = valueBlankOf(field, operator);
  const raw = condition.value;
  if (raw !== undefined) {
    switch (kind) {
      case 'member':
      case 'options': {
        const list = (Array.isArray(raw) ? raw : [raw]).filter(
          (item) => item !== null && item !== '',
        );
        value =
          kind === 'member'
            ? list.map(Number).filter((item) => Number.isFinite(item))
            : list.map(String);
        break;
      }
      case 'datetime': {
        if (operator === 'between') {
          const pair = Array.isArray(raw) ? raw : [];
          value =
            pair.length >= 2 &&
            pair[0] !== null &&
            pair[1] !== null &&
            pair[0] !== '' &&
            pair[1] !== ''
              ? [String(pair[0]), String(pair[1])]
              : null;
        } else {
          value = String(Array.isArray(raw) ? (raw[0] ?? '') : raw);
        }
        break;
      }
      default:
        value = Array.isArray(raw) ? raw.map((item) => String(item)).join(', ') : String(raw ?? '');
    }
  }
  return { key: ++rowKeySeed, field: condition.field, operator, value };
}

/* ---- 行编辑动作 ---- */

function addRow(): void {
  if (rows.value.length >= MAX_FILTER_ROWS) return;
  rows.value = [...rows.value, blankRow()];
}

function removeRow(key: number): void {
  rows.value = rows.value.filter((row) => row.key !== key);
}

/** 字段切换：操作符回落到新字段首个合法操作符，值按新形态重置。 */
function onFieldChange(row: FilterRow, field: string): void {
  const meta = fieldMap.value.get(field);
  const operator = meta ? (operatorsOf(meta)[0] ?? 'eq') : 'eq';
  row.field = field;
  row.operator = operator;
  row.value = valueBlankOf(meta, operator);
}

/** 操作符切换：值形态变化时重置（时间单值↔区间、有值↔无值），其余保留已输入值。 */
function onOperatorChange(row: FilterRow, operator: QueryOperator): void {
  const meta = fieldMap.value.get(row.field);
  const kind = meta ? kindOf(meta) : null;
  const shapeChanged =
    (kind === 'datetime' && (row.operator === 'between') !== (operator === 'between')) ||
    isNoValueOperator(operator) !== isNoValueOperator(row.operator);
  row.operator = operator;
  if (shapeChanged) row.value = valueBlankOf(meta, operator);
}

/* ---- 值控件辅助 ---- */

function singleOf(value: RowValue): string | number | undefined {
  return Array.isArray(value) ? value[0] : undefined;
}

function textOf(value: RowValue): string {
  return typeof value === 'string' ? value : '';
}

function rangeOf(value: RowValue): [string, string] | null {
  if (!Array.isArray(value) || value.length < 2) return null;
  const [lo, hi] = [String(value[0] ?? ''), String(value[1] ?? '')];
  return lo && hi ? [lo, hi] : null;
}

function normalizeRange(next: unknown): [string, string] | null {
  if (!Array.isArray(next) || next.length < 2) return null;
  const [lo, hi] = [String(next[0] ?? ''), String(next[1] ?? '')];
  return lo && hi ? [lo, hi] : null;
}

function onTextInput(row: FilterRow, next: string): void {
  row.value = next ?? '';
}

/** 成员选择输入收敛：集合操作符绑定整个数组，单值操作符提交时取首项。 */
function onMemberInput(row: FilterRow, next: unknown): void {
  if (Array.isArray(next)) {
    row.value = next.map(Number).filter((item) => Number.isFinite(item));
  } else if (typeof next === 'number' && Number.isFinite(next)) {
    row.value = [next];
  } else {
    row.value = [];
  }
}

function onOptionInput(row: FilterRow, next: unknown): void {
  if (Array.isArray(next)) {
    row.value = next.map(String);
  } else if (typeof next === 'string' && next) {
    row.value = [next];
  } else {
    row.value = [];
  }
}

function onDateInput(row: FilterRow, next: unknown): void {
  row.value = typeof next === 'string' && next ? next : '';
}

function onRangeInput(row: FilterRow, next: unknown): void {
  row.value = normalizeRange(next);
}

function onBooleanInput(row: FilterRow, next: unknown): void {
  row.value = next === 'true' || next === 'false' ? next : '';
}

function valuePlaceholderOf(view: FilterRowView): string {
  if (view.kind === 'number') {
    if (view.row.operator === 'between') return '两个数值用英文逗号分隔';
    if (view.isSet) return '多个数值用英文逗号分隔';
    return '输入数值';
  }
  if (view.kind === 'text' && view.isSet) return '多个值用英文逗号分隔';
  return '请输入筛选值';
}

/* ---- 数字解析（文本输入 → 数字/数字数组） ---- */

function parseNumberValue(
  operator: QueryOperator,
  text: string,
): { ok: boolean; value?: QueryValue } {
  const parseScalar = (chunk: string): number | null => {
    const trimmed = chunk.trim();
    if (!trimmed) return null;
    const parsed = Number(trimmed);
    return Number.isFinite(parsed) ? parsed : null;
  };
  if (operator === 'in' || operator === 'notIn' || operator === 'between') {
    const numbers = text
      .split(',')
      .map((chunk) => parseScalar(chunk))
      .filter((item): item is number => item !== null);
    if (numbers.length === 0 || text.split(',').some((chunk) => !chunk.trim())) {
      return { ok: false };
    }
    if (operator === 'between' && numbers.length !== 2) return { ok: false };
    return { ok: true, value: numbers };
  }
  const single = parseScalar(text);
  return single === null ? { ok: false } : { ok: true, value: single };
}

/* ---- 提交/清空 ---- */

function rowToCondition(row: FilterRow): QueryCondition {
  const field = fieldMap.value.get(row.field)!;
  const kind = kindOf(field);
  let value: QueryValue | undefined;
  if (!isNoValueOperator(row.operator)) {
    switch (kind) {
      case 'member':
      case 'options': {
        const list = row.value as number[] | string[];
        value = row.operator === 'in' || row.operator === 'notIn' ? list.slice() : list[0];
        break;
      }
      case 'number':
        value = parseNumberValue(row.operator, String(row.value)).value;
        break;
      case 'boolean':
        value = row.value === 'true';
        break;
      case 'datetime':
        value =
          row.operator === 'between'
            ? (rangeOf(row.value)?.slice() as [string, string] | undefined)
            : textOf(row.value);
        break;
      default: {
        const text = String(row.value).trim();
        value =
          row.operator === 'in' || row.operator === 'notIn'
            ? text
                .split(',')
                .map((chunk) => chunk.trim())
                .filter(Boolean)
            : text;
      }
    }
  }
  return {
    type: 'condition',
    field: row.field,
    operator: row.operator,
    ...(value !== undefined ? { value } : {}),
  };
}

function apply(): void {
  const complete = rows.value.filter((row) => rowComplete(row));
  // 数字行先在本地校验格式，避免整单提交后才被服务端拒绝
  for (const row of complete) {
    const field = fieldMap.value.get(row.field)!;
    if (kindOf(field) === 'number' && !isNoValueOperator(row.operator)) {
      if (!parseNumberValue(row.operator, String(row.value)).ok) {
        ElMessage.warning(
          row.operator === 'between'
            ? `「${field.label}」需填两个数值（英文逗号分隔）`
            : `「${field.label}」的筛选值需为数字`,
        );
        return;
      }
    }
  }
  if (!complete.length) return;
  const children = complete.map(rowToCondition);
  // 单条件直接下发 condition；多条件按「所有/任一」组合为单层 group
  model.value =
    children.length === 1
      ? children[0]
      : { type: 'group', conjunction: conjunction.value, children };
  open.value = false;
}

function clear(): void {
  model.value = undefined;
  rows.value = [];
  open.value = false;
}

/* ---- 成员远程搜索（条件行共享候选缓存） ---- */

const memberOptions = shallowRef<MemberListItemDto[]>([]);
const memberLoading = shallowRef(false);

async function searchMembers(keyword?: string): Promise<void> {
  memberLoading.value = true;
  try {
    const page = await listMembers(keyword ? { keyword, pageSize: 50 } : { pageSize: 50 });
    memberOptions.value = page.items;
  } catch {
    memberOptions.value = [];
  } finally {
    memberLoading.value = false;
  }
}

watch(open, (value) => {
  if (!value) return;
  // 每次打开都从已生效的筛选重建编辑态，避免上次未提交的草稿残留
  resetEditor();
  if (hasMemberField.value && memberOptions.value.length === 0) void searchMembers();
});

// 字段目录变化（如切换表单）时同步重建：失效字段自动剔除为空行
watch(
  () => props.fields,
  () => resetEditor(),
  { immediate: true },
);
</script>

<template>
  <ElPopover
    v-model:visible="open"
    trigger="click"
    placement="bottom-end"
    :offset="6"
    :show-arrow="false"
    popper-class="form-record-filter__popper"
  >
    <template #reference>
      <button
        type="button"
        class="form-record-filter-trigger"
        :class="{
          'form-record-filter-trigger--active': open || hasFilter,
        }"
        :disabled="!fields.length"
        aria-label="筛选数据"
        title="筛选"
      >
        <RiFilter3Fill />
        <span>筛选</span>
        <!-- 已生效筛选的常显标记：与工具栏图标按钮的激活反馈同语言 -->
        <i v-if="hasFilter" class="form-record-filter-trigger__dot" aria-hidden="true"></i>
      </button>
    </template>

    <div class="form-record-filter" role="group" aria-label="筛选数据">
      <div class="form-record-filter__header">
        <span>筛选出符合以下</span>
        <ElSelect
          class="form-record-filter__conjunction"
          :model-value="conjunction"
          :teleported="false"
          aria-label="条件组合方式"
          @update:model-value="conjunction = $event === 'or' ? 'or' : 'and'"
        >
          <ElOption label="所有" value="and" />
          <ElOption label="任一" value="or" />
        </ElSelect>
        <span>条件的数据</span>
      </div>

      <div class="form-record-filter__toolbar">
        <button
          type="button"
          class="form-record-filter__add"
          :disabled="rows.length >= MAX_FILTER_ROWS"
          @click="addRow"
        >
          <RiAddFill />
          <span>添加过滤条件</span>
        </button>
        <button
          type="button"
          class="form-record-filter__remove-all"
          :disabled="!rows.length"
          @click="rows = []"
        >
          删除全部
        </button>
      </div>

      <div v-if="rows.length" class="form-record-filter__rows">
        <div v-for="view in rowViews" :key="view.row.key" class="form-record-filter__row">
          <ElSelect
            class="form-record-filter__field"
            :model-value="view.row.field"
            filterable
            placeholder="选择字段"
            :teleported="false"
            @update:model-value="onFieldChange(view.row, String($event ?? ''))"
          >
            <ElOptionGroup v-if="formFields.length" label="表单字段">
              <ElOption
                v-for="item in formFields"
                :key="item.field"
                :label="item.label"
                :value="item.field"
              >
                <span class="form-record-filter__option">
                  <component :is="item.icon" v-if="item.icon" />
                  <span class="form-record-filter__option-label">{{ item.label }}</span>
                </span>
              </ElOption>
            </ElOptionGroup>
            <ElOptionGroup v-if="systemFields.length" label="系统字段">
              <ElOption
                v-for="item in systemFields"
                :key="item.field"
                :label="item.label"
                :value="item.field"
              >
                <span class="form-record-filter__option">
                  <component :is="item.icon" v-if="item.icon" />
                  <span class="form-record-filter__option-label">{{ item.label }}</span>
                </span>
              </ElOption>
            </ElOptionGroup>
          </ElSelect>

          <ElSelect
            class="form-record-filter__operator"
            :model-value="view.row.operator"
            placeholder="条件"
            :teleported="false"
            @update:model-value="onOperatorChange(view.row, $event as QueryOperator)"
          >
            <ElOption
              v-for="item in view.operators"
              :key="item"
              :label="OPERATOR_LABELS[item]"
              :value="item"
            />
          </ElSelect>

          <div class="form-record-filter__value">
            <!-- 提交人：成员远程搜索（值=成员 ID） -->
            <ElSelect
              v-if="view.kind === 'member' && view.requiresValue"
              :model-value="view.isSet ? view.row.value : singleOf(view.row.value)"
              :multiple="view.isSet"
              filterable
              remote
              clearable
              :remote-method="searchMembers"
              :loading="memberLoading"
              :teleported="false"
              placeholder="搜索并选择成员"
              @update:model-value="onMemberInput(view.row, $event)"
            >
              <ElOption
                v-for="member in memberOptions"
                :key="member.id"
                :label="member.name"
                :value="member.id"
              />
            </ElSelect>

            <!-- 选项类字段：候选来自发布快照 -->
            <ElSelect
              v-else-if="view.kind === 'options' && view.requiresValue"
              :model-value="view.isSet ? view.row.value : singleOf(view.row.value)"
              :multiple="view.isSet"
              filterable
              clearable
              :teleported="false"
              placeholder="选择选项"
              @update:model-value="onOptionInput(view.row, $event)"
            >
              <ElOption
                v-for="option in view.field?.options ?? []"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </ElSelect>

            <!-- time 格式时间字段：时间选择器（单值/区间） -->
            <ElTimePicker
              v-else-if="view.kind === 'datetime' && view.format === 'time' && view.requiresValue"
              :model-value="
                view.row.operator === 'between' ? rangeOf(view.row.value) : textOf(view.row.value)
              "
              :is-range="view.row.operator === 'between'"
              value-format="HH:mm"
              :teleported="false"
              range-separator="至"
              start-placeholder="开始时间"
              end-placeholder="结束时间"
              placeholder="选择时间"
              @update:model-value="
                view.row.operator === 'between'
                  ? onRangeInput(view.row, $event)
                  : onDateInput(view.row, $event)
              "
            />

            <!-- 其余时间格式：日期(时间)选择器，type 随字段存储格式 -->
            <ElDatePicker
              v-else-if="view.kind === 'datetime' && view.requiresValue"
              :model-value="
                view.row.operator === 'between' ? rangeOf(view.row.value) : textOf(view.row.value)
              "
              :type="datePickerTypeOf(view)"
              :value-format="DATE_VALUE_FORMAT[view.format]"
              :default-time="
                view.format === 'datetime'
                  ? view.row.operator === 'between'
                    ? RANGE_DEFAULT_TIMES
                    : SINGLE_DEFAULT_TIME
                  : undefined
              "
              :teleported="false"
              range-separator="至"
              start-placeholder="开始日期"
              end-placeholder="结束日期"
              placeholder="选择日期"
              @update:model-value="
                view.row.operator === 'between'
                  ? onRangeInput(view.row, $event)
                  : onDateInput(view.row, $event)
              "
            />

            <!-- 布尔字段（当前无真实控件映射，协议完整性的兜底形态） -->
            <ElSelect
              v-else-if="view.kind === 'boolean' && view.requiresValue"
              :model-value="textOf(view.row.value)"
              :teleported="false"
              placeholder="选择值"
              @update:model-value="onBooleanInput(view.row, $event)"
            >
              <ElOption label="是" value="true" />
              <ElOption label="否" value="false" />
            </ElSelect>

            <!-- 文本/数字：集合与区间按英文逗号分隔，提交时解析 -->
            <ElInput
              v-else-if="view.requiresValue"
              :model-value="textOf(view.row.value)"
              :placeholder="valuePlaceholderOf(view)"
              clearable
              @update:model-value="onTextInput(view.row, $event)"
              @keydown.enter.prevent="apply"
            />

            <span v-else class="form-record-filter__no-value">无需填值</span>
          </div>

          <button
            type="button"
            class="form-record-filter__row-remove"
            :aria-label="`删除条件：${view.field?.label ?? view.row.field}`"
            @click="removeRow(view.row.key)"
          >
            <RiCloseFill />
          </button>
        </div>
      </div>

      <p v-else class="form-record-filter__empty">暂无过滤条件，点击「添加过滤条件」创建</p>

      <footer class="form-record-filter__footer">
        <ElButton type="primary" :disabled="!completeCount" @click="apply">筛选</ElButton>
        <ElButton :disabled="!hasFilter && !rows.length" @click="clear">清空</ElButton>
      </footer>
    </div>
  </ElPopover>
</template>

<style scoped lang="scss">
.form-record-filter-trigger {
  // 与工具栏动作按钮同语言的触发入口：图标 + 文案；已生效筛选时常显激活态
  flex-shrink: 0;
  min-height: 34px;
  padding: 0 10px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
  color: var(--el-text-color-regular);
  background: transparent;
  border: 0;
  border-radius: var(--el-border-radius-base);
  cursor: pointer;
  font-size: 14px;
  font-weight: 550;
  transition:
    color 0.18s ease,
    background-color 0.18s ease;

  svg {
    width: 17px;
    height: 17px;
  }

  &:hover:not(:disabled),
  &--active {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }

  &:disabled {
    color: var(--el-text-color-disabled);
    cursor: not-allowed;
  }

  &__dot {
    width: 6px;
    height: 6px;
    border-radius: var(--el-border-radius-circle);
    background: var(--el-color-primary);
  }
}
</style>

<style lang="scss">
/* 传送至 body 的 popper 内容以唯一块类限定，避免影响其他弹层 */
.form-record-filter__popper.el-popover.el-popper {
  min-width: min(560px, calc(100vw - 40px));
  max-width: min(720px, calc(100vw - 40px));
}

.form-record-filter__popper {
  .form-record-filter {
    display: grid;
    gap: 10px;
  }

  .form-record-filter__header {
    display: flex;
    align-items: center;
    gap: 2px;
    color: var(--el-text-color-regular);
    font-size: 14px;
  }

  // 「所有/任一」以主色无边框形态内联在标题句中
  .form-record-filter__conjunction.el-select {
    width: 74px;

    .el-select__wrapper,
    .el-select__wrapper.is-hovering,
    .el-select__wrapper.is-focused {
      min-height: 26px;
      padding: 0 4px 0 6px;
      font-weight: 600;
      color: var(--el-color-primary);
      background: transparent;
      box-shadow: none;
    }
  }

  .form-record-filter__toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .form-record-filter__add,
  .form-record-filter__remove-all {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 3px 6px;
    background: transparent;
    border: 0;
    border-radius: var(--el-border-radius-base);
    font-size: 13px;
    font-weight: 550;
    cursor: pointer;
    transition: background-color 0.18s ease;

    svg {
      width: 14px;
      height: 14px;
    }

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }

  .form-record-filter__add {
    color: var(--el-color-primary);

    &:hover:not(:disabled) {
      background: var(--el-color-primary-light-9);
    }
  }

  .form-record-filter__remove-all {
    color: var(--el-color-danger);

    &:hover:not(:disabled) {
      background: var(--el-color-danger-light-9);
    }
  }

  .form-record-filter__rows {
    display: grid;
    gap: 8px;
  }

  .form-record-filter__row {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
  }

  .form-record-filter__field {
    flex: 0 0 auto;
    width: 176px;
  }

  .form-record-filter__operator {
    flex: 0 0 auto;
    width: 118px;
  }

  .form-record-filter__value {
    flex: 1 1 150px;
    min-width: 150px;

    // 值控件统一占满剩余宽度：日期(区间)编辑器宽度走 EP 的编辑器宽度变量
    > .el-select,
    > .el-input,
    > .el-date-editor {
      --el-date-editor-width: 100%;
      width: 100%;
    }
  }

  .form-record-filter__option {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    min-width: 0;

    svg {
      flex-shrink: 0;
      width: 14px;
      height: 14px;
      color: var(--el-text-color-secondary);
    }
  }

  .form-record-filter__option-label {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .form-record-filter__no-value {
    display: inline-flex;
    align-items: center;
    height: 32px;
    color: var(--el-text-color-placeholder);
    font-size: 13px;
  }

  .form-record-filter__row-remove {
    flex-shrink: 0;
    width: 26px;
    height: 26px;
    padding: 0;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-secondary);
    background: transparent;
    border: 0;
    border-radius: var(--el-border-radius-base);
    cursor: pointer;
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    svg {
      width: 16px;
      height: 16px;
    }

    &:hover {
      color: var(--el-color-danger);
      background: var(--el-color-danger-light-9);
    }
  }

  .form-record-filter__empty {
    margin: 4px 0;
    padding: 10px 0;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    text-align: center;
    background: var(--el-fill-color-lighter);
    border-radius: var(--el-border-radius-base);
  }

  .form-record-filter__footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding-top: 10px;
    border-top: 1px solid var(--el-border-color-lighter);
  }
}

@media (max-width: 620px) {
  .form-record-filter__popper {
    .form-record-filter__row {
      flex-wrap: wrap;
    }

    .form-record-filter__field {
      width: 100%;
    }

    .form-record-filter__operator {
      width: 132px;
    }

    .form-record-filter__value {
      flex: 1 1 100%;
    }
  }
}
</style>
