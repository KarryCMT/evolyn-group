<script setup lang="ts">
import type { PermissionDataScope, PermissionField } from './permission.types';
import { computed } from 'vue';

defineOptions({ name: 'PermissionGroupEditorDataPanel' });

const props = defineProps<{
  fields: readonly PermissionField[];
}>();
const dataScope = defineModel<PermissionDataScope>('dataScope', { required: true });

type ConditionOperator =
  | 'eq'
  | 'ne'
  | 'contains'
  | 'gt'
  | 'gte'
  | 'lt'
  | 'lte'
  | 'in'
  | 'not_in'
  | 'empty'
  | 'not_empty';

const operatorsByType: Record<string, ConditionOperator[]> = {
  text: ['eq', 'ne', 'contains', 'empty', 'not_empty'],
  textarea: ['eq', 'ne', 'contains', 'empty', 'not_empty'],
  number: ['eq', 'ne', 'gt', 'gte', 'lt', 'lte', 'empty', 'not_empty'],
  decimal: ['eq', 'ne', 'gt', 'gte', 'lt', 'lte', 'empty', 'not_empty'],
  money: ['eq', 'ne', 'gt', 'gte', 'lt', 'lte', 'empty', 'not_empty'],
  percent: ['eq', 'ne', 'gt', 'gte', 'lt', 'lte', 'empty', 'not_empty'],
  datetime: ['eq', 'ne', 'gt', 'gte', 'lt', 'lte', 'empty', 'not_empty'],
  radiogroup: ['eq', 'ne', 'in', 'not_in', 'empty', 'not_empty'],
  combo: ['eq', 'ne', 'in', 'not_in', 'empty', 'not_empty'],
  user: ['eq', 'ne', 'in', 'not_in', 'empty', 'not_empty'],
  dept: ['eq', 'ne', 'in', 'not_in', 'empty', 'not_empty'],
  checkboxgroup: ['contains', 'in', 'not_in', 'empty', 'not_empty'],
  combocheck: ['contains', 'in', 'not_in', 'empty', 'not_empty'],
  usergroup: ['contains', 'in', 'not_in', 'empty', 'not_empty'],
  deptgroup: ['contains', 'in', 'not_in', 'empty', 'not_empty'],
};

const conditionFields = computed(() =>
  props.fields.filter((field) => operatorsByType[field.type]?.length),
);
const match = computed({
  get: () => dataScope.value.match,
  set: (value: PermissionDataScope['match']) => {
    dataScope.value = { ...dataScope.value, match: value };
  },
});

function operatorsFor(fieldName: string): ConditionOperator[] {
  const type = props.fields.find((field) => field.field === fieldName)?.type;
  return type ? (operatorsByType[type] ?? []) : [];
}

function updateCondition(index: number, patch: Partial<PermissionDataScope['conditions'][number]>) {
  dataScope.value = {
    ...dataScope.value,
    conditions: dataScope.value.conditions.map((condition, current) =>
      current === index ? { ...condition, ...patch } : condition,
    ),
  };
}

function addCondition() {
  const field = conditionFields.value[0];
  if (!field) return;
  const operator = operatorsFor(field.field)[0];
  dataScope.value = {
    ...dataScope.value,
    conditions: [...dataScope.value.conditions, { field: field.field, operator, value: [] }],
  };
}

function removeCondition(index: number) {
  dataScope.value = {
    ...dataScope.value,
    conditions: dataScope.value.conditions.filter((_condition, current) => current !== index),
  };
}

function updateField(index: number, field: string) {
  updateCondition(index, { field, operator: operatorsFor(field)[0], value: [] });
}

function updateValues(index: number, text: string) {
  const condition = dataScope.value.conditions[index];
  if (!condition || isValueless(condition.operator)) return;
  const type = props.fields.find((field) => field.field === condition.field)?.type;
  const values = text
    .split(',')
    .map((value) => value.trim())
    .filter(Boolean)
    .map((value) => (isNumberType(type) ? Number(value) : value));
  updateCondition(index, { value: values });
}

function isValueless(operator: string) {
  return operator === 'empty' || operator === 'not_empty';
}

function isNumberType(type: string | undefined) {
  return type === 'number' || type === 'decimal' || type === 'money' || type === 'percent';
}

function operatorLabel(operator: ConditionOperator) {
  return {
    eq: '等于',
    ne: '不等于',
    contains: '包含',
    gt: '大于',
    gte: '大于等于',
    lt: '小于',
    lte: '小于等于',
    in: '属于',
    not_in: '不属于',
    empty: '为空',
    not_empty: '不为空',
  }[operator];
}
</script>

<template>
  <section class="permission-group-editor-data-panel" aria-label="数据权限">
    <p class="permission-group-editor-data-panel__intro">可以管理哪些数据</p>
    <div class="permission-group-editor-data-panel__rule">
      <span>筛选出符合以下</span>
      <el-select v-model="match" aria-label="条件组合方式">
        <el-option label="所有" value="all" />
        <el-option label="任一" value="any" />
      </el-select>
      <span>条件的数据</span>
    </div>
    <div class="permission-group-editor-data-panel__conditions">
      <div
        v-for="(condition, index) in dataScope.conditions"
        :key="`${condition.field}-${index}`"
        class="permission-group-editor-data-panel__condition"
      >
        <el-select
          :model-value="condition.field"
          @update:model-value="updateField(index, String($event))"
        >
          <el-option
            v-for="field in conditionFields"
            :key="field.field"
            :label="field.label"
            :value="field.field"
          />
        </el-select>
        <el-select
          :model-value="condition.operator"
          @update:model-value="updateCondition(index, { operator: String($event), value: [] })"
        >
          <el-option
            v-for="operator in operatorsFor(condition.field)"
            :key="operator"
            :label="operatorLabel(operator)"
            :value="operator"
          />
        </el-select>
        <el-input
          v-if="!isValueless(condition.operator)"
          :model-value="condition.value.join(', ')"
          placeholder="多个值用逗号分隔"
          @update:model-value="updateValues(index, String($event))"
        />
        <span v-else class="permission-group-editor-data-panel__empty-value">无需比较值</span>
        <el-button text type="danger" @click="removeCondition(index)"> 删除 </el-button>
      </div>
    </div>
    <el-button :disabled="!conditionFields.length" @click="addCondition"> 添加条件 </el-button>
    <p class="permission-group-editor-data-panel__hint">
      不添加条件时，成员可管理全部数据；多个值使用英文逗号分隔。
    </p>
  </section>
</template>

<style scoped lang="scss">
.permission-group-editor-data-panel {
  &__intro {
    margin: 0 0 var(--el-space-xl);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
    line-height: 24px;
  }

  &__rule {
    display: flex;
    align-items: center;
    gap: var(--el-space-md);
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
    line-height: 32px;
  }

  &__rule :deep(.el-select) {
    width: 108px;
  }

  &__hint {
    margin: var(--el-space-lg) 0 0;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
    line-height: 20px;
  }

  &__conditions {
    display: flex;
    margin: var(--el-space-lg) 0;
    flex-direction: column;
    gap: var(--el-space-md);
  }

  &__condition {
    display: grid;
    grid-template-columns: minmax(120px, 1fr) minmax(104px, 0.8fr) minmax(160px, 1.2fr) auto;
    align-items: center;
    gap: var(--el-space-md);
  }

  &__empty-value {
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
  }
}
</style>
