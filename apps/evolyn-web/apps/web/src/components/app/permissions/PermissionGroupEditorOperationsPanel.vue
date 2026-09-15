<script setup lang="ts">
import type { PermissionOperation } from './permission.types';
import { computed } from 'vue';

defineOptions({ name: 'PermissionGroupEditorOperationsPanel' });

const props = defineProps<{
  workflow: boolean;
}>();

const operations = defineModel<PermissionOperation[]>({ required: true });

interface OperationOption {
  value: PermissionOperation;
  label: string;
}

const standardOperations: OperationOption[] = [
  { value: 'view', label: '查看' },
  { value: 'add', label: '添加' },
  { value: 'copy', label: '复制' },
  { value: 'edit', label: '编辑' },
  { value: 'delete', label: '删除' },
  { value: 'batch_print', label: '批量打印' },
  { value: 'batch_modify', label: '批量修改' },
  { value: 'import', label: '导入' },
  { value: 'export', label: '导出' },
];

const workflowOperations: OperationOption[] = [
  { value: 'workflow_owner_transfer', label: '调整流程负责人' },
  { value: 'workflow_terminate', label: '结束流程' },
  { value: 'workflow_activate', label: '激活流程' },
];

const availableOperations = computed(() =>
  props.workflow ? [...standardOperations, ...workflowOperations] : standardOperations,
);
const allSelected = computed(() =>
  availableOperations.value.every((operation) => operations.value.includes(operation.value)),
);
const indeterminate = computed(() => operations.value.length > 0 && !allSelected.value);

function toggleAll(checked: boolean | string | number) {
  operations.value = checked ? availableOperations.value.map((operation) => operation.value) : [];
}
</script>

<template>
  <section class="permission-group-editor-operations-panel" aria-label="操作权限">
    <p class="permission-group-editor-operations-panel__intro">可对流程和数据进行哪些操作</p>
    <el-checkbox
      class="permission-group-editor-operations-panel__check-all"
      :model-value="allSelected"
      :indeterminate="indeterminate"
      @update:model-value="toggleAll"
    >
      全选
    </el-checkbox>
    <el-checkbox-group v-model="operations" class="permission-group-editor-operations-panel__list">
      <el-checkbox
        v-for="operation in availableOperations"
        :key="operation.value"
        :value="operation.value"
      >
        {{ operation.label }}
      </el-checkbox>
    </el-checkbox-group>
  </section>
</template>

<style scoped lang="scss">
.permission-group-editor-operations-panel {
  &__intro {
    margin: 0 0 var(--el-space-xl);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
    line-height: 24px;
  }

  &__check-all {
    margin-bottom: var(--el-space-lg);
  }

  &__list {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: var(--el-space-lg) var(--el-space-2xl);
  }
}
</style>
