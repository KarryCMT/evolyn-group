<template>
  <el-drawer
    :model-value="modelValue"
    title="字段显隐规则"
    size="680px"
    append-to-body
    destroy-on-close
    class="form-field-show-rules"
    @update:model-value="$emit('update:model-value', $event)"
  >
    <template v-if="mode === 'list'" #default>
      <div class="form-field-show-rules__toolbar">
        <p class="form-field-show-rules__hint">
          条件成立时显示所选字段，否则隐藏；隐藏字段保留已填内容，仅不进入提交载荷。
        </p>
        <el-button type="primary" :disabled="rules.length >= 200" @click="startCreate">
          <el-icon><RiAddFill /></el-icon>
          添加显隐规则
        </el-button>
      </div>

      <p v-if="rules.length === 0" class="form-field-show-rules__empty">暂无显隐规则</p>

      <Draggable
        v-else
        :list="localRules"
        item-key="id"
        handle=".form-field-show-rules__drag"
        :animation="150"
        @end="emitReorder"
      >
        <template #item="{ element }">
          <div class="form-field-show-rules__row">
            <el-icon class="form-field-show-rules__drag"><RiDragMoveFill /></el-icon>
            <div class="form-field-show-rules__summary">
              <p class="form-field-show-rules__summary-line">当{{ conditionSummary(element) }}时</p>
              <p class="form-field-show-rules__summary-line">显示{{ targetSummary(element) }}</p>
            </div>
            <div class="form-field-show-rules__actions">
              <el-tooltip content="编辑" placement="top">
                <el-button
                  text
                  :icon="RiEditFill"
                  aria-label="编辑规则"
                  @click="startEdit(element)"
                />
              </el-tooltip>
              <el-tooltip content="复制" placement="top">
                <el-button
                  text
                  :icon="RiFileCopyFill"
                  aria-label="复制规则"
                  @click="$emit('duplicate-rule', element.id)"
                />
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button
                  text
                  type="danger"
                  :icon="RiDeleteBin6Fill"
                  aria-label="删除规则"
                  @click="confirmRemove(element)"
                />
              </el-tooltip>
            </div>
          </div>
        </template>
      </Draggable>
    </template>

    <template v-else #default>
      <div v-if="editing" class="form-field-show-rules__editor">
        <el-form label-position="top" size="default" @submit.prevent>
          <el-form-item label="条件组合方式">
            <el-radio-group v-model="editing.filter.rel">
              <el-radio value="and">满足所有条件</el-radio>
              <el-radio value="or">满足任一条件</el-radio>
            </el-radio-group>
          </el-form-item>
        </el-form>

        <div
          v-for="(condition, index) in editing.filter.cond"
          :key="index"
          class="form-field-show-rules__condition"
        >
          <div class="form-field-show-rules__condition-main">
            <el-select
              :model-value="condition.field"
              filterable
              placeholder="选择条件字段"
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
              placeholder="比较值"
              @update:model-value="setSingleText(condition, String($event ?? ''))"
            />
            <el-input-number
              v-else-if="valueKind(condition) === 'number'"
              :model-value="singleNumberValue(condition)"
              :controls="false"
              placeholder="比较值"
              @update:model-value="setSingleNumber(condition, $event)"
            />
            <el-date-picker
              v-else-if="valueKind(condition) === 'datetime'"
              :model-value="singleTextValue(condition) || undefined"
              :type="datePickerType(condition)"
              :value-format="dateValueFormat(condition)"
              placeholder="选择日期时间"
              @update:model-value="setSingleText(condition, String($event ?? ''))"
            />
            <el-time-picker
              v-else-if="valueKind(condition) === 'time'"
              :model-value="singleTextValue(condition) || undefined"
              value-format="HH:mm"
              placeholder="选择时间"
              @update:model-value="setSingleText(condition, String($event ?? ''))"
            />
            <el-select
              v-else-if="valueKind(condition) === 'option-single'"
              :model-value="singleTextValue(condition)"
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

        <el-button
          text
          type="primary"
          :icon="RiAddFill"
          :disabled="editing.filter.cond.length >= 20"
          @click="addCondition"
        >
          添加条件
        </el-button>

        <el-form label-position="top" size="default" @submit.prevent>
          <el-form-item class="form-field-show-rules__targets" label="显示以下字段">
            <el-select
              v-model="editing.fields"
              multiple
              filterable
              placeholder="选择条件成立后显示的字段"
            >
              <el-option
                v-for="option in targetOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
                :disabled="option.disabled"
              >
                <span>{{ option.label }}</span>
                <span v-if="option.note" class="form-field-show-rules__option-note">
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

        <p class="form-field-show-rules__preview">
          当{{ conditionSummary(editing) }}时，显示{{ targetSummary(editing) }}
        </p>
      </div>
    </template>

    <template #footer>
      <template v-if="mode === 'edit'">
        <el-button @click="cancelEdit">取消</el-button>
        <el-button type="primary" @click="confirmSave">保存</el-button>
      </template>
      <template v-else>
        <el-button @click="$emit('update:model-value', false)">关闭</el-button>
      </template>
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue';
import Draggable from 'vuedraggable';
import {
  ElAlert,
  ElButton,
  ElCheckbox,
  ElDatePicker,
  ElDrawer,
  ElForm,
  ElFormItem,
  ElIcon,
  ElInput,
  ElInputNumber,
  ElMessageBox,
  ElOption,
  ElRadio,
  ElRadioGroup,
  ElSelect,
  ElTimePicker,
  ElTooltip,
} from 'element-plus';
import {
  RiAddFill,
  RiDeleteBin6Fill,
  RiDragMoveFill,
  RiEditFill,
  RiFileCopyFill,
} from '@remixicon/vue';
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
  FormWidgetType,
} from '../schema/types';

/**
 * 字段显隐规则管理抽屉（v5 设计方案 §5）：列表支持新增/编辑/复制/删除与
 * 拖拽重排（重排只改善阅读与审计 diff，不参与运行结果）；编辑器按字段
 * 类型 × 方法渲染值控件，保存前以共享校验器整体复核并把环、引用与类型
 * 问题定位到具体路径。成员/部门值本期录入稳定 ID，目录搜索选择器随后续
 * 适配器批次接入。
 */
const props = defineProps<{
  modelValue: boolean;
  rules: FieldShowRule[];
  document: FormSchemaDocument;
  items: FormItem[];
}>();

const emit = defineEmits<{
  'update:model-value': [value: boolean];
  'save-rule': [rule: FieldShowRule];
  'remove-rule': [ruleId: string];
  'duplicate-rule': [ruleId: string];
  'reorder-rules': [ruleIds: string[]];
}>();

const mode = shallowRef<'list' | 'edit'>('list');
const editing = ref<FieldShowRule | null>(null);
const editingRuleId = shallowRef('');
const issues = shallowRef<FormSchemaIssue[]>([]);
// 列表拖拽在本地副本上排序，结束后以 id 序列上抛（规则仍由宿主单一事实源维护）。
const localRules = ref<FieldShowRule[]>([]);

watch(
  () => props.rules,
  (rules) => {
    localRules.value = JSON.parse(JSON.stringify(rules)) as FieldShowRule[];
  },
  { immediate: true, deep: true },
);

watch(
  () => props.modelValue,
  (open) => {
    if (!open) {
      mode.value = 'list';
      editing.value = null;
      editingRuleId.value = '';
      issues.value = [];
    }
  },
);

// ---- 字段选择器选项 ----

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

// ---- 条件值控件分派 ----

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
  | 'none'
  | 'text'
  | 'number'
  | 'datetime'
  | 'time'
  | 'option-single'
  | 'option-multi'
  | 'id-single'
  | 'id-multi';

/** 值控件分派：字段类型 × 方法共同决定；切换字段/方法即重置值。 */
function valueKind(condition: FieldShowCondition): ValueKind {
  if (isEmptyMethod(condition.method)) return 'none';
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
      // usergroup / deptgroup。
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

// ---- 协议 value 数组的 get/set 投影 ----

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
  // 切换字段后立即重置方法与值，防止静默保留不兼容旧条件（设计方案 §5.2）。
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

function addCondition(): void {
  const first = conditionFieldOptions.value[0];
  if (!editing.value || !first) return;
  editing.value.filter.cond.push({
    field: first.value,
    type: typeOfField(first.value) ?? 'text',
    method: 'isEmpty',
  });
}

function removeCondition(index: number): void {
  editing.value?.filter.cond.splice(index, 1);
}

function typeOfField(field: string): FormWidgetType | undefined {
  return props.items.find((entry) => entry.widget.widgetName === field)?.widget.type;
}

// ---- 编辑模式切换与保存 ----

function startCreate(): void {
  const first = conditionFieldOptions.value[0];
  editing.value = {
    id: generateFieldShowRuleId(),
    filter: {
      rel: 'and',
      cond: first
        ? [{ field: first.value, type: typeOfField(first.value) ?? 'text', method: 'isEmpty' }]
        : [],
    },
    fields: [],
  };
  editingRuleId.value = '';
  issues.value = [];
  mode.value = 'edit';
}

function startEdit(rule: FieldShowRule): void {
  editing.value = JSON.parse(JSON.stringify(rule)) as FieldShowRule;
  editingRuleId.value = rule.id;
  issues.value = [];
  mode.value = 'edit';
}

function cancelEdit(): void {
  mode.value = 'list';
  editing.value = null;
  editingRuleId.value = '';
  issues.value = [];
}

/** 保存前以共享校验器整体复核；仅展示 fieldShowRules 路径的问题。 */
function confirmSave(): void {
  if (!editing.value) return;
  const candidate = JSON.parse(JSON.stringify(props.document)) as FormSchemaDocument;
  const others = props.rules.filter((rule) => rule.id !== editingRuleId.value);
  candidate.content.fieldShowRules = [...others, editing.value];
  const result = validateFormSchema(candidate);
  issues.value = result.issues.filter((issue) => issue.path.startsWith('content.fieldShowRules'));
  if (issues.value.length > 0) return;
  emit('save-rule', JSON.parse(JSON.stringify(editing.value)) as FieldShowRule);
  cancelEdit();
}

async function confirmRemove(rule: FieldShowRule): Promise<void> {
  try {
    await ElMessageBox.confirm(
      `删除后「${targetSummary(rule)}」将不再受该规则控制，确定删除？`,
      '删除显隐规则',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' },
    );
  } catch {
    return;
  }
  emit('remove-rule', rule.id);
}

function emitReorder(): void {
  emit(
    'reorder-rules',
    localRules.value.map((rule) => rule.id),
  );
}

// ---- 自然语言摘要 ----

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
</script>

<style lang="scss">
.form-field-show-rules {
  .el-drawer__header {
    height: 56px;
    margin-bottom: 0;
    padding: 0 var(--el-space-lg);
  }

  .el-drawer__title {
    font-size: 18px;
    line-height: 26px;
  }

  .el-drawer__close-btn {
    width: 32px;
    height: 32px;

    .el-icon {
      font-size: 22px;
    }
  }

  &__toolbar {
    display: flex;
    gap: var(--el-space-md);
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: var(--el-space-lg);
  }

  &__hint {
    flex: 1;
    margin: 0;
    font-size: var(--el-font-size-extra-small);
    line-height: 1.6;
    color: var(--el-text-color-secondary);
  }

  &__empty {
    padding: var(--el-space-2xl) 0;
    font-size: var(--el-font-size-small);
    color: var(--el-text-color-secondary);
    text-align: center;
  }

  &__row {
    display: flex;
    gap: var(--el-space-sm);
    align-items: center;
    padding: var(--el-space-md);
    margin-bottom: var(--el-space-sm);
    background-color: var(--el-fill-color-extra-lighter);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-base);

    &:hover {
      border-color: var(--el-color-primary-light-5);
    }
  }

  &__drag {
    flex-shrink: 0;
    color: var(--el-text-color-secondary);
    cursor: grab;
  }

  &__summary {
    flex: 1;
    min-width: 0;
  }

  &__summary-line {
    margin: 0;
    font-size: var(--el-font-size-small);
    line-height: 22px;
    color: var(--el-text-color-regular);
    overflow-wrap: anywhere;
  }

  &__actions {
    display: flex;
    flex-shrink: 0;
    gap: var(--el-space-xs);
  }

  &__condition {
    display: flex;
    gap: var(--el-space-sm);
    align-items: center;
    padding: var(--el-space-md);
    margin-bottom: var(--el-space-sm);
    background-color: var(--el-fill-color-extra-lighter);
    border-radius: var(--el-border-radius-base);
  }

  &__condition-main {
    display: grid;
    flex: 1;
    grid-template-columns: repeat(2, minmax(0, 1fr));
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

  &__targets {
    margin-top: var(--el-space-lg);

    .el-select {
      width: 100%;
    }
  }

  &__option-note {
    float: right;
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-secondary);
  }

  &__preview {
    padding: var(--el-space-md);
    margin: var(--el-space-lg) 0 0;
    font-size: var(--el-font-size-small);
    line-height: 1.6;
    color: var(--el-text-color-primary);
    background-color: var(--el-fill-color-lighter);
    border-radius: var(--el-border-radius-base);
  }
}
</style>
