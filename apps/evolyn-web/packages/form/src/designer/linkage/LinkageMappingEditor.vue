<script setup lang="ts">
import type { DataLinkageMapping } from '../../schema/types';
import { type LinkageSourceField, isLinkageFieldCompatible } from '../../schema/linkage';
import { RiDeleteBin6Line } from '@remixicon/vue';
import { ElButton, ElOption, ElSelect } from 'element-plus';

const props = defineProps<{
  modelValue: DataLinkageMapping[];
  sourceFields: LinkageSourceField[];
  currentFields: LinkageSourceField[];
  loading?: boolean;
}>();

const emit = defineEmits<{ 'update:modelValue': [mappings: DataLinkageMapping[]] }>();

function patchRow(index: number, patch: Partial<DataLinkageMapping>): void {
  const next = props.modelValue.map((mapping) => ({ ...mapping }));
  next[index] = { ...next[index]!, ...patch };
  emit('update:modelValue', next);
}

function removeRow(index: number): void {
  emit(
    'update:modelValue',
    props.modelValue.filter((_, rowIndex) => rowIndex !== index),
  );
}

function setSourceField(index: number, sourceFieldId: string): void {
  const mapping = props.modelValue[index]!;
  const source = props.sourceFields.find((field) => field.id === sourceFieldId);
  const target = props.currentFields.find((field) => field.id === mapping.targetFieldId);
  patchRow(index, {
    sourceFieldId,
    // 切换来源字段后，已不兼容的目标不能残留到发布阶段才暴露错误。
    targetFieldId:
      source && target && !isLinkageFieldCompatible(source.type, target.type)
        ? ''
        : mapping.targetFieldId,
  });
}

function targetFieldsOf(mapping: DataLinkageMapping): LinkageSourceField[] {
  const source = props.sourceFields.find((field) => field.id === mapping.sourceFieldId);
  if (!source) return props.currentFields;
  return props.currentFields.filter((field) => isLinkageFieldCompatible(source.type, field.type));
}
</script>

<template>
  <div class="linkage-mappings">
    <div v-for="(mapping, index) in modelValue" :key="index" class="linkage-mappings__row">
      <el-select
        :model-value="mapping.targetFieldId"
        filterable
        :loading="loading"
        placeholder="当前表单字段"
        aria-label="联动目标字段"
        @update:model-value="patchRow(index, { targetFieldId: String($event) })"
      >
        <el-option
          v-for="field in targetFieldsOf(mapping)"
          :key="field.id"
          :label="field.name"
          :value="field.id"
        />
      </el-select>
      <span class="linkage-mappings__label">联动显示</span>
      <el-select
        :model-value="mapping.sourceFieldId"
        filterable
        :loading="loading"
        placeholder="联动表单字段"
        aria-label="联动来源字段"
        @update:model-value="setSourceField(index, String($event))"
      >
        <el-option
          v-for="field in sourceFields"
          :key="field.id"
          :label="field.name"
          :value="field.id"
        />
      </el-select>
      <span>的值</span>
      <el-button text circle aria-label="删除联动映射" @click="removeRow(index)">
        <RiDeleteBin6Line />
      </el-button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.linkage-mappings {
  display: flex;
  flex-direction: column;
  gap: var(--el-space-sm);

  &__row {
    display: grid;
    grid-template-columns: minmax(180px, 1fr) auto minmax(180px, 1fr) auto 36px;
    gap: var(--el-space-md);
    align-items: center;
  }

  &__label {
    color: var(--el-text-color-primary);
  }
}
</style>
