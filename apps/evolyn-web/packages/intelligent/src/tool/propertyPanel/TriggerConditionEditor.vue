<script setup lang="ts">
import { RiAddLine, RiDeleteBin6Line } from '@remixicon/vue';
import { computed } from 'vue';
import type { IntelligentFormFieldOption } from '../../mock/nodeTemplates';
import type {
  FormTriggerAction,
  FormTriggerCondition,
  FormTriggerConditionMode,
  FormTriggerConditionOperator,
} from '../../schema';
import TriggerConditionValue from './TriggerConditionValue.vue';
import TriggerFieldSelect from './TriggerFieldSelect.vue';
import TriggerOptionSelect from './TriggerOptionSelect.vue';

defineOptions({ name: 'TriggerConditionEditor' });

const props = defineProps<{
  actions: readonly FormTriggerAction[];
  fields: readonly IntelligentFormFieldOption[];
}>();

const conditions = defineModel<FormTriggerCondition[]>('conditions', { default: () => [] });
const mode = defineModel<FormTriggerConditionMode>('mode', { default: 'all' });
const modeOptions = [
  { value: 'all', label: '所有' },
  { value: 'any', label: '任一' },
] as const;
const operatorOptions = [
  { value: 'equals', label: '等于' },
  { value: 'not-equals', label: '不等于' },
  { value: 'equals-any', label: '等于任意一个' },
  { value: 'not-equals-any', label: '不等于任意一个' },
  { value: 'is-empty', label: '为空' },
  { value: 'is-not-empty', label: '不为空' },
] as const;
const actionSentence = computed(() => {
  const labels = props.actions.map((action) => {
    if (action.type === 'create') return '新增';
    if (action.type === 'update') return '修改后';
    return '被删除';
  });
  return labels.length > 0 ? `${labels.join(' 或 ')} 的数据满足` : '表单数据满足';
});

function createId(): string {
  return `trigger_condition_${globalThis.crypto?.randomUUID?.() ?? Math.random().toString(36).slice(2)}`;
}

function addCondition(): void {
  conditions.value = [
    ...conditions.value,
    {
      id: createId(),
      fieldId: props.fields[0]?.value ?? '',
      operator: 'equals-any',
      values: [],
    },
  ];
}

function removeCondition(conditionId: string): void {
  conditions.value = conditions.value.filter((condition) => condition.id !== conditionId);
}

function updateCondition(conditionId: string, patch: Partial<FormTriggerCondition>): void {
  conditions.value = conditions.value.map((condition) =>
    condition.id === conditionId
      ? {
          ...condition,
          ...patch,
          values: patch.values ? [...patch.values] : condition.values,
        }
      : condition,
  );
}

function updateField(conditionId: string, fieldId: string): void {
  updateCondition(conditionId, { fieldId, values: [] });
}

function updateOperator(conditionId: string, value: string): void {
  const operator = value as FormTriggerConditionOperator;
  // 一元运算符不接受筛选值，切换时同步清理旧值，避免保存不可见脏数据。
  updateCondition(conditionId, {
    operator,
    values: operator === 'is-empty' || operator === 'is-not-empty' ? [] : undefined,
  });
}

function updateValues(conditionId: string, values: string[]): void {
  updateCondition(conditionId, { values });
}

function selectedField(fieldId: string): IntelligentFormFieldOption | undefined {
  return props.fields.find((field) => field.value === fieldId);
}

function isValueDisabled(operator: FormTriggerConditionOperator): boolean {
  return operator === 'is-empty' || operator === 'is-not-empty';
}
</script>

<template>
  <section class="trigger-condition-editor">
    <h3>触发条件</h3>
    <div class="trigger-condition-editor__summary">
      <span>{{ actionSentence }}</span>
      <TriggerOptionSelect v-model="mode" :options="modeOptions" control-label="条件匹配方式" />
      <span>条件时，触发后续动作</span>
    </div>
    <button type="button" class="trigger-condition-editor__add" @click="addCondition">
      <RiAddLine />添加条件
    </button>

    <div
      v-for="condition in conditions"
      :key="condition.id"
      class="trigger-condition-editor__row"
      :class="{ 'is-unary': isValueDisabled(condition.operator) }"
    >
      <TriggerFieldSelect
        :model-value="condition.fieldId"
        :options="fields"
        control-label="选择条件字段"
        @update:model-value="updateField(condition.id, $event)"
      />
      <TriggerOptionSelect
        :model-value="condition.operator"
        :options="operatorOptions"
        control-label="选择条件运算符"
        @update:model-value="updateOperator(condition.id, $event)"
      />
      <TriggerConditionValue
        v-if="!isValueDisabled(condition.operator)"
        :model-value="condition.values"
        :field="selectedField(condition.fieldId)"
        @update:model-value="updateValues(condition.id, $event)"
      />
      <button
        type="button"
        class="trigger-condition-editor__remove"
        aria-label="删除触发条件"
        @click="removeCondition(condition.id)"
      >
        <RiDeleteBin6Line />
      </button>
    </div>
  </section>
</template>

<style scoped lang="scss">
.trigger-condition-editor {
  padding: 28px 32px 36px;
  border-top: 12px solid #f5f6f8;

  h3 { margin: 0 0 22px; font-size: 17px; font-weight: 650; }

  &__summary {
    display: flex;
    min-height: 42px;
    align-items: center;
    flex-wrap: wrap;
    gap: 10px;
    color: #788395;

    :deep(.trigger-option-select) { width: 90px; }
    :deep(.trigger-option-select__button) { height: 34px; padding: 0 7px; border-color: transparent; background: #f3f5f7; }
  }

  &__add {
    display: inline-flex;
    margin-top: 14px;
    padding: 3px 0;
    align-items: center;
    gap: 5px;
    color: #00a99d;
    background: transparent;
    border: 0;
    cursor: pointer;
    font: inherit;
    font-size: 16px;

    svg { width: 22px; height: 22px; }
  }

  &__row {
    display: grid;
    margin-top: 12px;
    align-items: center;
    grid-template-columns: minmax(170px, 1fr) minmax(145px, 0.75fr) minmax(220px, 1.45fr) 30px;
    gap: 10px;

    &.is-unary { grid-template-columns: minmax(170px, 1fr) minmax(145px, 0.75fr) 30px; }
  }

  &__remove {
    display: inline-flex;
    width: 30px;
    height: 36px;
    padding: 0;
    align-items: center;
    justify-content: center;
    color: #697487;
    background: transparent;
    border: 0;
    cursor: pointer;

    svg { width: 18px; height: 18px; }
  }
}

@media (max-width: 980px) {
  .trigger-condition-editor {
    padding: 22px 20px;

    &__row,
    &__row.is-unary { grid-template-columns: minmax(0, 1fr) 30px; }
    &__row > :not(.trigger-condition-editor__remove) { grid-column: 1; }
    &__remove { grid-column: 2; grid-row: 1; }
  }
}
</style>
