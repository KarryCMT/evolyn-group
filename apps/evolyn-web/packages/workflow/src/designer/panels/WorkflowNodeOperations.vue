<script setup lang="ts">
import { ElSwitch } from 'element-plus';
import { computed } from 'vue';
import type { WorkflowNodeOperations } from '../../schema';

defineOptions({ name: 'WorkflowNodeOperations' });

const props = defineProps<{ operations: WorkflowNodeOperations | undefined }>();
const emit = defineEmits<{ update: [operations: WorkflowNodeOperations] }>();

const defaults: WorkflowNodeOperations = {
  submit: true,
  saveDraft: true,
  temporarySave: false,
  submitAndPrint: false,
  endProcess: false,
};
const value = computed(() => ({ ...defaults, ...(props.operations ?? {}) }));

function patch(key: keyof WorkflowNodeOperations, enabled: string | number | boolean) {
  emit('update', { ...value.value, [key]: Boolean(enabled) });
}

const rows: Array<{ key: keyof WorkflowNodeOperations; label: string }> = [
  { key: 'submit', label: '提交' },
  { key: 'saveDraft', label: '保存草稿' },
  { key: 'temporarySave', label: '暂存' },
  { key: 'submitAndPrint', label: '提交并打印' },
  { key: 'endProcess', label: '结束流程' },
];
</script>

<template>
  <section class="workflow-node-operations" aria-label="节点操作">
    <h3>节点操作</h3>
    <div class="workflow-node-operations__card">
      <div v-for="row in rows" :key="row.key" class="workflow-node-operations__row">
        <span>{{ row.label }}</span>
        <ElSwitch :model-value="value[row.key]" @change="(enabled) => patch(row.key, enabled)" />
      </div>
    </div>
  </section>
</template>

<style scoped lang="scss">
.workflow-node-operations {
  h3 { margin: 4px 0 14px; font-size: 16px; }

  &__card {
    padding: 0 16px;
    border: 1px solid var(--el-border-color);
    border-radius: 10px;
  }

  &__row {
    display: flex;
    min-height: 52px;
    align-items: center;
    justify-content: space-between;
    color: var(--el-text-color-primary);
    border-bottom: 1px solid var(--el-border-color-lighter);
    font-size: 14px;
    &:last-child { border-bottom: 0; }
  }
}
</style>
