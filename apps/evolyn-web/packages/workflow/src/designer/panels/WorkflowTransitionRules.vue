<script setup lang="ts">
import { ElOption, ElSelect } from 'element-plus';
import type { WorkflowSubmitCondition } from '../../schema';

defineOptions({ name: 'WorkflowTransitionRules' });

defineProps<{ submitCondition: WorkflowSubmitCondition | undefined }>();
const emit = defineEmits<{ update: [condition: WorkflowSubmitCondition] }>();
</script>

<template>
  <section class="workflow-transition-rules" aria-label="流转规则">
    <h3>节点提交条件</h3>
    <ElSelect
      :model-value="submitCondition ?? 'all'"
      @update:model-value="(value) => emit('update', value as WorkflowSubmitCondition)"
    >
      <ElOption value="all" label="所有数据均可提交" />
      <ElOption value="valid" label="满足条件的数据才可提交" />
    </ElSelect>
    <p>选择“满足条件”后，表单字段校验通过时才能提交当前审批节点。</p>
  </section>
</template>

<style scoped lang="scss">
.workflow-transition-rules {
  h3 { margin: 4px 0 14px; font-size: 16px; }
  :deep(.el-select) { width: 100%; }
  p { margin: 12px 0 0; color: var(--el-text-color-secondary); font-size: 13px; line-height: 1.7; }
}
</style>
