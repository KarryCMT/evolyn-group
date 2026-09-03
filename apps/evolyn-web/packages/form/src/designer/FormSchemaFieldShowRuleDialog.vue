<template>
  <el-dialog
    :model-value="modelValue"
    title="字段显隐规则"
    width="min(94vw, 1512px)"
    top="2vh"
    append-to-body
    destroy-on-close
    class="form-field-show-rule-dialog"
    aria-label="字段显隐规则编辑"
    @update:model-value="onDialogVisibleChange"
  >
    <div v-if="editing" class="form-field-show-rule-dialog__editor">
      <div class="form-field-show-rule-dialog__logic-combiner">
        <span>满足以下</span>
        <el-select v-model="editing.filter.rel" aria-label="条件组合方式">
          <el-option label="所有" value="and" />
          <el-option label="任一" value="or" />
        </el-select>
        <span>条件时</span>
      </div>

      <el-button
        class="form-field-show-rule-dialog__add-condition"
        text
        type="primary"
        :icon="RiAddFill"
        :disabled="editing.filter.cond.length >= 20"
        @click="addCondition"
      >
        添加条件
      </el-button>

      <div
        v-for="(condition, index) in editing.filter.cond"
        :key="index"
        class="form-field-show-rule-dialog__condition"
      >
        <div class="form-field-show-rule-dialog__condition-main">
          <el-select
            :model-value="condition.field"
            filterable
            placeholder="请选择字段"
            aria-label="条件字段"
            @update:model-value="onConditionFieldChange(index, $event as string)"
          >
            <el-option
              v-for="option in conditionFieldOptions"
              :key="option.value"
              :label="`${option.label}（${option.typeLabel}）`"
              :value="option.value"
            />
          </el-select>
          <el-select
            :model-value="condition.method"
            :disabled="!condition.field"
            placeholder="比较方法"
            aria-label="比较方法"
            @update:model-value="onConditionMethodChange(index, $event as FieldShowMethod)"
          >
            <el-option
              v-for="method in methodsOf(condition)"
              :key="method"
              :label="FIELD_SHOW_METHOD_LABELS[method]"
              :value="method"
            />
          </el-select>

          <el-input
            v-if="valueKind(condition) === 'text'"
            :model-value="singleTextValue(condition)"
            :disabled="!condition.field"
            placeholder="比较值"
            @update:model-value="setSingleText(condition, String($event ?? ''))"
          />
          <el-input-number
            v-else-if="valueKind(condition) === 'number'"
            :model-value="singleNumberValue(condition)"
            :disabled="!condition.field"
            :controls="false"
            placeholder="比较值"
            @update:model-value="setSingleNumber(condition, $event)"
          />
          <el-date-picker
            v-else-if="valueKind(condition) === 'datetime'"
            :model-value="singleTextValue(condition) || undefined"
            :disabled="!condition.field"
            :type="datePickerType(condition)"
            :value-format="dateValueFormat(condition)"
            placeholder="选择日期时间"
            @update:model-value="setSingleText(condition, String($event ?? ''))"
          />
          <el-time-picker
            v-else-if="valueKind(condition) === 'time'"
            :model-value="singleTextValue(condition) || undefined"
            :disabled="!condition.field"
            value-format="HH:mm"
            placeholder="选择时间"
            @update:model-value="setSingleText(condition, String($event ?? ''))"
          />
          <el-select
            v-else-if="valueKind(condition) === 'option-single'"
            :model-value="singleTextValue(condition)"
            :disabled="!condition.field"
            placeholder="选择比较值"
            clearable
            @update:model-value="setSingleText(condition, $event as string)"
          >
            <el-option
              v-for="option in optionsOf(condition)"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </el-select>
          <el-select
            v-else-if="valueKind(condition) === 'option-multi'"
            :model-value="multiValueOf(condition)"
            :disabled="!condition.field"
            multiple
            filterable
            placeholder="选择比较值"
            @update:model-value="setMultiValue(condition, $event as string[])"
          >
            <el-option
              v-for="option in optionsOf(condition)"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </el-select>
          <el-select
            v-else-if="valueKind(condition) === 'id-single'"
            :model-value="singleTextValue(condition)"
            :disabled="!condition.field"
            filterable
            allow-create
            default-first-option
            clearable
            placeholder="输入成员/部门 ID 后回车"
            @update:model-value="setSingleText(condition, $event as string)"
          >
            <el-option
              v-if="singleTextValue(condition)"
              :label="singleTextValue(condition)"
              :value="singleTextValue(condition)"
            />
          </el-select>
          <el-select
            v-else-if="valueKind(condition) === 'id-multi'"
            :model-value="multiValueOf(condition)"
            :disabled="!condition.field"
            multiple
            filterable
            allow-create
            default-first-option
            placeholder="输入成员/部门 ID 后回车"
            @update:model-value="setMultiValue(condition, $event as string[])"
          >
            <el-option
              v-for="entry in multiValueOf(condition)"
              :key="entry"
              :label="entry"
              :value="entry"
            />
          </el-select>

          <el-checkbox
            v-if="currentMemberAllowed(condition)"
            :model-value="condition.includeCurrentMember === true"
            label="或为当前成员"
            @update:model-value="condition.includeCurrentMember = $event as boolean"
          />
        </div>
        <el-button
          text
          type="danger"
          :icon="RiDeleteBin6Fill"
          aria-label="删除条件"
          :disabled="editing.filter.cond.length <= 1"
          @click="removeCondition(index)"
        />
      </div>

      <el-form label-position="top" size="default" @submit.prevent>
        <el-form-item class="form-field-show-rule-dialog__targets" label="显示以下字段">
          <el-select
            v-model="editing.fields"
            multiple
            filterable
            :disabled="!hasConfiguredConditions"
            :placeholder="hasConfiguredConditions ? '选择条件成立后显示的字段' : '请先添加条件'"
          >
            <el-option
              v-for="option in targetOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
              :disabled="option.disabled"
            >
              <span>{{ option.label }}</span>
              <span v-if="option.note" class="form-field-show-rule-dialog__option-note">
                {{ option.note }}
              </span>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>

      <el-alert
        v-for="issue in issues"
        :key="issue.path + issue.message"
        type="error"
        :title="issue.message"
        :description="issue.path"
        :closable="false"
        show-icon
      />

      <p class="form-field-show-rule-dialog__preview">
        当{{ conditionSummary(editing) }}时，显示{{ targetSummary(editing) }}
      </p>
    </div>

    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" @click="confirmSave">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { RiAddFill, RiDeleteBin6Fill } from '@remixicon/vue';
import { computed, ref, shallowRef, watch } from 'vue';
import {
  ElAlert,
  ElButton,
  ElCheckbox,
  ElDatePicker,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElOption,
  ElSelect,
  ElTimePicker,
} from 'element-plus';
import {
  FIELD_SHOW_CONDITION_METHODS,
  FIELD_SHOW_CURRENT_MEMBER_TYPES,
  FIELD_SHOW_EMPTY_METHODS,
  FIELD_SHOW_METHOD_LABELS,
  generateFieldShowRuleId,
  widgetTypeLabel,
} from '../schema/dictionary';
import { readWidgetOptions } from '../schema/codec';
import { validateFormSchema, type FormSchemaIssue } from '../schema/validate';
import type {
  FieldShowCondition,
  FieldShowMethod,
  FieldShowRule,
  FormItem,
  FormSchemaDocument,
  FormWidgetOption,
} from '../schema/types';

/**
 * 字段显隐规则编辑弹窗：独立维护规则草稿与校验状态，只通过 save/update:model-value
 * 与规则列表抽屉通信，避免编辑态与列表态在同一组件内耦合。
 */
const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    rule?: FieldShowRule | null;
    rules?: FieldShowRule[];
    document: FormSchemaDocument;
    items?: FormItem[];
  }>(),
  {
    rule: null,
    rules: () => [],
    items: () => [],
  },
);

const emit = defineEmits<{
  'update:model-value': [value: boolean];
  save: [rule: FieldShowRule];
}>();

const editing = ref<FieldShowRule | null>(null);
const editingRuleId = shallowRef('');
const issues = shallowRef<FormSchemaIssue[]>([]);

watch(
  () => [props.modelValue, props.rule] as const,
  ([open, rule]) => {
    if (!open) {
      resetDraft();
      return;
    }
    editing.value = rule ? cloneRule(rule) : createDraftRule();
    editingRuleId.value = rule?.id ?? '';
    issues.value = [];
  },
  { immediate: true },
);

const conditionFieldOptions = computed(() =>
  props.items
    .filter(
      (item) => item.widget.type in FIELD_SHOW_CONDITION_METHODS && item.widget.visible !== false,
    )
    .map((item) => ({
      value: item.widget.widgetName,
      label: item.label,
      typeLabel: widgetTypeLabel(item.widget.type),
    })),
);

const targetOptions = computed(() => {
  const occupiedBy = new Map<string, string>();
  for (const rule of props.rules) {
    if (rule.id === editingRuleId.value) continue;
    for (const field of rule.fields) occupiedBy.set(field, rule.id);
  }
  return props.items
    .filter((item) => item.widget.type !== 'separator' && item.widget.type !== 'button')
    .map((item) => {
      const occupied = occupiedBy.get(item.widget.widgetName);
      const staticallyHidden = item.widget.visible === false;
      return {
        value: item.widget.widgetName,
        label: item.label,
        disabled: Boolean(occupied) || staticallyHidden,
        note: occupied ? '已被其他规则使用' : staticallyHidden ? '静态隐藏字段不能作为目标' : '',
      };
    });
});

const hasConfiguredConditions = computed(() => {
  const conditions = editing.value?.filter.cond ?? [];
  return (
    conditions.length > 0 &&
    conditions.every(
      (condition) =>
        Boolean(condition.field) &&
        (isEmptyMethod(condition.method) || (condition.value?.length ?? 0) > 0),
    )
  );
});

function methodsOf(condition: FieldShowCondition): readonly FieldShowMethod[] {
  return FIELD_SHOW_CONDITION_METHODS[condition.type] ?? [];
}

function isEmptyMethod(method: string): boolean {
  return FIELD_SHOW_EMPTY_METHODS.has(method);
}

function currentMemberAllowed(condition: FieldShowCondition): boolean {
  return FIELD_SHOW_CURRENT_MEMBER_TYPES.has(condition.type);
}

type ValueKind =
  | 'text'
  | 'number'
  | 'datetime'
  | 'time'
  | 'option-single'
  | 'option-multi'
  | 'id-single'
  | 'id-multi';

function valueKind(condition: FieldShowCondition): ValueKind {
  switch (condition.type) {
    case 'text':
    case 'textarea':
      return 'text';
    case 'number':
      return 'number';
    case 'datetime':
      return datetimeFormatOf(condition) === 'time' ? 'time' : 'datetime';
    case 'radiogroup':
    case 'combo':
      return condition.method === 'in' || condition.method === 'notIn'
        ? 'option-multi'
        : 'option-single';
    case 'user':
    case 'dept':
      return condition.method === 'in' || condition.method === 'notIn' ? 'id-multi' : 'id-single';
    case 'checkboxgroup':
    case 'combocheck':
      return 'option-multi';
    default:
      return 'id-multi';
  }
}

function optionsOf(condition: FieldShowCondition): FormWidgetOption[] {
  const item = props.items.find((entry) => entry.widget.widgetName === condition.field);
  return item ? readWidgetOptions(item.widget) : [];
}

function datetimeFormatOf(condition: FieldShowCondition): 'date' | 'datetime' | 'month' | 'time' {
  const item = props.items.find((entry) => entry.widget.widgetName === condition.field);
  const format = (item?.widget as { format?: unknown } | undefined)?.format;
  return format === 'date' || format === 'datetime' || format === 'month' || format === 'time'
    ? format
    : 'datetime';
}

function datePickerType(condition: FieldShowCondition): 'date' | 'datetime' | 'month' {
  const format = datetimeFormatOf(condition);
  return format === 'datetime' ? 'datetime' : format === 'month' ? 'month' : 'date';
}

function dateValueFormat(condition: FieldShowCondition): string {
  switch (datetimeFormatOf(condition)) {
    case 'datetime':
      return 'YYYY-MM-DD HH:mm:ss';
    case 'month':
      return 'YYYY-MM';
    case 'time':
      return 'HH:mm';
    default:
      return 'YYYY-MM-DD';
  }
}

function singleTextValue(condition: FieldShowCondition): string {
  return typeof condition.value?.[0] === 'string' ? condition.value[0] : '';
}

function singleNumberValue(condition: FieldShowCondition): number | undefined {
  return typeof condition.value?.[0] === 'number' ? condition.value[0] : undefined;
}

function setSingleText(condition: FieldShowCondition, value: string): void {
  condition.value = value ? [value] : [];
}

function setSingleNumber(condition: FieldShowCondition, value: number | null | undefined): void {
  condition.value = typeof value === 'number' && Number.isFinite(value) ? [value] : [];
}

function multiValueOf(condition: FieldShowCondition): string[] {
  return (condition.value ?? []).filter((entry): entry is string => typeof entry === 'string');
}

function setMultiValue(condition: FieldShowCondition, next: string[]): void {
  condition.value = [...next];
}

function onConditionFieldChange(index: number, field: string): void {
  const condition = editing.value?.filter.cond[index];
  const item = props.items.find((entry) => entry.widget.widgetName === field);
  if (!condition || !item) return;
  condition.field = field;
  condition.type = item.widget.type;
  condition.method = (FIELD_SHOW_CONDITION_METHODS[item.widget.type] ?? [])[0] ?? 'isEmpty';
  condition.value = isEmptyMethod(condition.method) ? undefined : [];
  condition.includeCurrentMember = undefined;
}

function onConditionMethodChange(index: number, method: FieldShowMethod): void {
  const condition = editing.value?.filter.cond[index];
  if (!condition) return;
  condition.method = method;
  condition.value = isEmptyMethod(method) ? undefined : [];
}

function createEmptyCondition(): FieldShowCondition {
  return { field: '', type: 'text', method: 'eq', value: [] };
}

function createDraftRule(): FieldShowRule {
  return {
    id: generateFieldShowRuleId(),
    filter: { rel: 'and', cond: [createEmptyCondition()] },
    fields: [],
  };
}

function cloneRule(rule: FieldShowRule): FieldShowRule {
  return JSON.parse(JSON.stringify(rule)) as FieldShowRule;
}

function addCondition(): void {
  editing.value?.filter.cond.push(createEmptyCondition());
}

function removeCondition(index: number): void {
  editing.value?.filter.cond.splice(index, 1);
}

function confirmSave(): void {
  if (!editing.value) return;
  const candidate = JSON.parse(JSON.stringify(props.document)) as FormSchemaDocument;
  const others = props.rules.filter((rule) => rule.id !== editingRuleId.value);
  candidate.content.fieldShowRules = [...others, editing.value];
  const result = validateFormSchema(candidate);
  issues.value = result.issues.filter((issue) => issue.path.startsWith('content.fieldShowRules'));
  if (issues.value.length > 0) return;
  emit('save', cloneRule(editing.value));
  close();
}

function labelOf(field: string): string {
  return props.items.find((entry) => entry.widget.widgetName === field)?.label ?? field;
}

function valueText(condition: FieldShowCondition): string {
  const values = condition.value ?? [];
  if (values.length === 0) return '（未设置）';
  return values.map((entry) => (typeof entry === 'string' ? entry : String(entry))).join('、');
}

function conditionSummary(rule: FieldShowRule): string {
  const joiner = rule.filter.rel === 'and' ? '且' : '或';
  const parts = rule.filter.cond.map((condition) => {
    const base = `［${labelOf(condition.field)}］${FIELD_SHOW_METHOD_LABELS[condition.method] ?? condition.method}`;
    if (isEmptyMethod(condition.method)) return base;
    const suffix = condition.includeCurrentMember ? '（或当前成员）' : '';
    return `${base}［${valueText(condition)}］${suffix}`;
  });
  return parts.length > 0 ? parts.join(joiner) : '（未配置条件）';
}

function targetSummary(rule: FieldShowRule): string {
  if (rule.fields.length === 0) return '（未选择字段）';
  return rule.fields.map((field) => `［${labelOf(field)}］`).join('、');
}

function onDialogVisibleChange(open: boolean): void {
  if (!open) close();
}

function close(): void {
  emit('update:model-value', false);
}

function resetDraft(): void {
  editing.value = null;
  editingRuleId.value = '';
  issues.value = [];
}
</script>

<style lang="scss">
.form-field-show-rule-dialog.el-dialog {
  display: flex;
  width: min(94vw, 1512px) !important;
  height: calc(100vh - 52px);
  min-height: 520px;
  max-height: none;
  flex-direction: column;
  margin: 26px auto;
  overflow: hidden;
  border-radius: 16px;

  .el-dialog__header {
    display: flex;
    flex-shrink: 0;
    align-items: center;
    min-height: 100px;
    padding: 0 40px;
    margin: 0;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .el-dialog__title {
    font-size: 28px;
    font-weight: 600;
    line-height: 1.35;
    color: var(--el-text-color-primary);
  }

  .el-dialog__headerbtn {
    top: 0;
    right: 32px;
    display: grid;
    width: 48px;
    height: 100px;
    place-items: center;

    .el-dialog__close {
      font-size: 28px;
    }
  }

  .el-dialog__body {
    display: flex;
    min-height: 0;
    flex: 1;
    padding: 48px 40px;
    overflow-y: auto;
  }

  .el-dialog__footer {
    display: flex;
    flex-shrink: 0;
    gap: var(--el-space-md);
    align-items: center;
    justify-content: flex-end;
    min-height: 112px;
    padding: 0 40px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .form-field-show-rule-dialog__editor {
    width: 100%;
  }

  .form-field-show-rule-dialog__logic-combiner {
    display: flex;
    gap: var(--el-space-sm);
    align-items: center;
    margin-bottom: var(--el-space-md);
    font-size: var(--el-font-size-base);
    color: var(--el-text-color-primary);

    .el-select {
      width: 108px;
    }
  }

  .form-field-show-rule-dialog__add-condition.el-button {
    margin: 0 0 var(--el-space-lg);
    padding-left: 0;
    font-size: var(--el-font-size-base);
  }

  .form-field-show-rule-dialog__condition {
    display: flex;
    gap: var(--el-space-sm);
    align-items: center;
    margin-bottom: var(--el-space-md);
  }

  .form-field-show-rule-dialog__condition-main {
    display: grid;
    flex: 1;
    grid-template-columns: minmax(220px, 320px) minmax(140px, 200px) minmax(0, 1fr);
    gap: var(--el-space-sm);

    .el-date-editor,
    .el-select,
    .el-input-number {
      width: 100%;
    }

    .el-checkbox {
      grid-column: 1 / -1;
      margin-right: 0;
    }
  }

  .form-field-show-rule-dialog__targets {
    margin-top: var(--el-space-lg);

    .el-select {
      width: 100%;
    }
  }

  .form-field-show-rule-dialog__option-note {
    float: right;
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-secondary);
  }

  .form-field-show-rule-dialog__preview {
    padding: var(--el-space-md);
    margin: var(--el-space-lg) 0 0;
    font-size: var(--el-font-size-small);
    line-height: 1.6;
    color: var(--el-text-color-primary);
    background-color: var(--el-fill-color-lighter);
    border-radius: var(--el-border-radius-base);
  }
}

@media (max-width: 767px) {
  .form-field-show-rule-dialog.el-dialog {
    width: calc(100vw - 24px) !important;
    height: calc(100vh - 24px);
    margin: 12px auto;

    .el-dialog__header {
      min-height: 64px;
      padding: 0 20px;
    }

    .el-dialog__title {
      font-size: 20px;
    }

    .el-dialog__headerbtn {
      right: 8px;
      height: 64px;
    }

    .el-dialog__body {
      padding: 24px 20px;
    }

    .el-dialog__footer {
      min-height: 76px;
      padding: 0 20px;
    }

    .form-field-show-rule-dialog__condition-main {
      grid-template-columns: 1fr;
    }
  }
}
</style>
