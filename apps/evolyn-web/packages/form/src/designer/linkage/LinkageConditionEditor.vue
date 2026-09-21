<script setup lang="ts">
import type { DataLinkageCondition, DataLinkageOperator, FormJsonValue } from '../../schema/types';
import {
  DATA_LINKAGE_OPERATOR_LABELS,
  type LinkageSourceField,
  isLinkageFieldCompatible,
} from '../../schema/linkage';
import { RiDeleteBin6Line } from '@remixicon/vue';
import { ElButton, ElInput, ElOption, ElSelect } from 'element-plus';
import { computed } from 'vue';

const props = defineProps<{
  modelValue: DataLinkageCondition[];
  sourceFields: LinkageSourceField[];
  currentFields: LinkageSourceField[];
  loading?: boolean;
}>();

const emit = defineEmits<{ 'update:modelValue': [conditions: DataLinkageCondition[]] }>();

const rows = computed(() => props.modelValue);

function patchRow(index: number, patch: Partial<DataLinkageCondition>): void {
  const next = props.modelValue.map((condition) => ({
    ...condition,
    value: condition.value ? { ...condition.value } : undefined,
  }));
  next[index] = { ...next[index]!, ...patch };
  emit('update:modelValue', next);
}

function setSourceField(index: number, fieldId: string): void {
  const field = props.sourceFields.find((entry) => entry.id === fieldId);
  const current = props.modelValue[index]!;
  const operator = field?.operators.includes(current.operator) ? current.operator : 'eq';
  patchRow(index, { sourceFieldId: fieldId, operator });
}

function setValueType(index: number, type: 'field' | 'constant'): void {
  patchRow(index, {
    value: type === 'field' ? { type: 'field', fieldId: '' } : { type: 'constant', value: '' },
  });
}

function setFieldValue(index: number, fieldId: string): void {
  patchRow(index, { value: { type: 'field', fieldId } });
}

function setConstantValue(index: number, value: string): void {
  patchRow(index, { value: { type: 'constant', value: value as FormJsonValue } });
}

function removeRow(index: number): void {
  emit(
    'update:modelValue',
    props.modelValue.filter((_, rowIndex) => rowIndex !== index),
  );
}

function operatorsOf(condition: DataLinkageCondition): DataLinkageOperator[] {
  return (
    props.sourceFields.find((field) => field.id === condition.sourceFieldId)?.operators ?? ['eq']
  );
}

function currentFieldsOf(condition: DataLinkageCondition): LinkageSourceField[] {
  const source = props.sourceFields.find((field) => field.id === condition.sourceFieldId);
  if (!source) return props.currentFields;
  return props.currentFields.filter((field) => isLinkageFieldCompatible(source.type, field.type));
}

function requiresValue(operator: DataLinkageOperator): boolean {
  return operator !== 'empty' && operator !== 'not_empty';
}
</script>

<template>
  <div class="linkage-conditions">
    <div v-for="(condition, index) in rows" :key="condition.id" class="linkage-conditions__row">
      <el-select
        :model-value="condition.sourceFieldId"
        filterable
        :loading="loading"
        placeholder="请选择字段"
        aria-label="联动表单字段"
        @update:model-value="setSourceField(index, String($event))"
      >
        <el-option
          v-for="field in sourceFields"
          :key="field.id"
          :label="field.name"
          :value="field.id"
        />
      </el-select>

      <el-select
        :model-value="condition.operator"
        class="linkage-conditions__operator"
        aria-label="过滤操作符"
        @update:model-value="patchRow(index, { operator: $event as DataLinkageOperator })"
      >
        <el-option
          v-for="operator in operatorsOf(condition)"
          :key="operator"
          :label="DATA_LINKAGE_OPERATOR_LABELS[operator]"
          :value="operator"
        />
      </el-select>

      <template v-if="requiresValue(condition.operator)">
        <el-select
          :model-value="condition.value?.type ?? 'field'"
          class="linkage-conditions__value-type"
          aria-label="条件值来源"
          @update:model-value="setValueType(index, $event as 'field' | 'constant')"
        >
          <el-option label="当前表单字段" value="field" />
          <el-option label="自定义" value="constant" />
        </el-select>
        <el-select
          v-if="condition.value?.type === 'field'"
          :model-value="condition.value.fieldId"
          filterable
          :loading="loading"
          placeholder="请选择当前表单字段"
          aria-label="当前表单字段"
          @update:model-value="setFieldValue(index, String($event))"
        >
          <el-option
            v-for="field in currentFieldsOf(condition)"
            :key="field.id"
            :label="field.name"
            :value="field.id"
          />
        </el-select>
        <el-input
          v-else
          :model-value="String(condition.value?.value ?? '')"
          placeholder="请输入自定义值"
          aria-label="自定义条件值"
          @update:model-value="setConstantValue(index, String($event))"
        />
      </template>
      <span v-else class="linkage-conditions__empty-value">无需条件值</span>

      <el-button text circle aria-label="删除过滤条件" @click="removeRow(index)">
        <RiDeleteBin6Line />
      </el-button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.linkage-conditions {
  display: flex;
  flex-direction: column;
  gap: var(--el-space-sm);

  &__row {
    display: grid;
    grid-template-columns: minmax(150px, 1fr) 120px 150px minmax(180px, 1.2fr) 36px;
    gap: var(--el-space-sm);
    align-items: center;
  }

  &__empty-value {
    grid-column: span 2;
    color: var(--el-text-color-placeholder);
  }
}

@media (max-width: 900px) {
  .linkage-conditions__row {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
