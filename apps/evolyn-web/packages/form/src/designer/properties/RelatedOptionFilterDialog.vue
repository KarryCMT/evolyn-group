<script setup lang="ts">
import type { DataLinkageCondition, FormRelatedOptionSource } from '../../schema/types';
import { type LinkageSourceField, createDataLinkageConditionId } from '../../schema/linkage';
import { ElButton, ElDialog, ElMessage, ElOption, ElSelect } from 'element-plus';
import { shallowReactive, watch } from 'vue';
import LinkageConditionEditor from '../linkage/LinkageConditionEditor.vue';

const props = defineProps<{ modelValue: boolean; filter: FormRelatedOptionSource['filter']; sourceFields: LinkageSourceField[]; currentFields: LinkageSourceField[] }>();
const emit = defineEmits<{ 'update:modelValue': [visible: boolean]; confirm: [filter: FormRelatedOptionSource['filter']] }>();
const draft = shallowReactive<FormRelatedOptionSource['filter']>({ logic: 'and', conditions: [] });
watch(() => props.modelValue, (visible) => { if (visible) Object.assign(draft, cloneFilter(props.filter)); });
/** 显式复制 JSON 协议对象，避免 structuredClone 接收 Vue Proxy。 */
function cloneFilter(filter: FormRelatedOptionSource['filter']): FormRelatedOptionSource['filter'] {
  return {
    logic: filter.logic,
    conditions: filter.conditions.map((condition) => ({
      ...condition,
      value: condition.value ? { ...condition.value } : undefined,
    })),
  };
}
function addCondition(): void {
  const condition: DataLinkageCondition = { id: createDataLinkageConditionId(), sourceFieldId: '', operator: 'eq', value: { type: 'field', fieldId: '' } };
  draft.conditions = [...draft.conditions, condition];
}
function confirm(): void {
  if (draft.conditions.some((condition) => !condition.sourceFieldId || (!['empty', 'not_empty'].includes(condition.operator) && (!condition.value || (condition.value.type === 'field' && !condition.value.fieldId))))) {
    ElMessage.warning('请完善过滤条件'); return;
  }
  emit('confirm', cloneFilter(draft)); emit('update:modelValue', false);
}
</script>

<template>
  <el-dialog :model-value="modelValue" class="related-option-filter-dialog" title="添加过滤条件" width="min(1120px, 92vw)" top="7vh" append-to-body destroy-on-close :close-on-click-modal="false" @update:model-value="emit('update:modelValue', $event)">
    <div class="related-option-filter-dialog__body">
      <p class="related-option-filter-dialog__hint">添加过滤条件来限定选项内容</p>
      <div class="related-option-filter-dialog__logic"><span>符合以下</span><el-select v-model="draft.logic" aria-label="条件关系"><el-option label="所有" value="and" /><el-option label="任一" value="or" /></el-select><span>条件的数据</span></div>
      <el-button type="primary" link class="related-option-filter-dialog__add" @click="addCondition">＋ 添加过滤条件</el-button>
      <LinkageConditionEditor v-model="draft.conditions" :source-fields="sourceFields" :current-fields="currentFields" />
    </div>
    <template #footer><el-button @click="emit('update:modelValue', false)">取消</el-button><el-button type="primary" @click="confirm">确定</el-button></template>
  </el-dialog>
</template>

<style lang="scss">
.related-option-filter-dialog {
  min-height: 560px;
  border-radius: var(--el-border-radius-large);
  .el-dialog__header { padding: 24px 32px 20px; border-bottom: 1px solid var(--el-border-color-lighter); }
  .el-dialog__body { min-height: 380px; padding: 32px; }
  .el-dialog__footer { padding: 20px 32px; border-top: 1px solid var(--el-border-color-lighter); }
  &__hint { margin: 0 0 28px; color: var(--el-text-color-secondary); }
  &__logic { display: flex; gap: 10px; align-items: center; margin-bottom: 12px; }
  &__logic .el-select { width: 88px; }
  &__add { margin-bottom: 18px; }
}
</style>
