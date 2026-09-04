<script setup lang="ts">
import { ElAlert, ElInput } from 'element-plus';
import { computed } from 'vue';
import type { WorkflowDocument, WorkflowEdge } from '../../schema';

/**
 * 连线属性面板：非条件出边只读展示走向；条件分支出边提供
 * 表达式编辑与「默认分支」切换（与条件节点面板同一编辑通道）。
 */
defineOptions({ name: 'WorkflowEdgePanel' });

const props = defineProps<{
  edge: WorkflowEdge;
  document: WorkflowDocument;
}>();

const emit = defineEmits<{
  updateCondition: [expression: string | null];
  removeEdge: [];
}>();

const sourceNode = computed(
  () => props.document.nodes.find((node) => node.key === props.edge.source) ?? null,
);
const targetNode = computed(
  () => props.document.nodes.find((node) => node.key === props.edge.target) ?? null,
);
const isConditionEdge = computed(() => sourceNode.value?.type === 'condition');
const isDefault = computed(() => props.edge.condition == null);
</script>

<template>
  <div class="workflow-edge-panel">
    <div class="workflow-edge-panel__route">
      <span>{{ sourceNode?.name ?? edge.source }}</span>
      <span class="workflow-edge-panel__arrow" aria-hidden="true">→</span>
      <span>{{ targetNode?.name ?? edge.target }}</span>
    </div>

    <template v-if="isConditionEdge">
      <ElAlert
        class="workflow-edge-panel__tip"
        :type="isDefault ? 'success' : 'info'"
        :closable="false"
        show-icon
        :title="
          isDefault
            ? '默认分支：前面的条件全部未命中时走这条连线'
            : '条件分支：表达式命中时走这条连线'
        "
      />
      <ElInput
        v-if="!isDefault"
        :model-value="edge.condition?.expression ?? ''"
        placeholder="例如：form.amount > 10000"
        @update:model-value="(value) => emit('updateCondition', value)"
      />
      <ElInput
        v-else
        class="workflow-edge-panel__readonly"
        model-value="（无条件，兜底分支）"
        disabled
      />
    </template>
    <ElAlert
      v-else
      class="workflow-edge-panel__tip"
      type="info"
      :closable="false"
      show-icon
      title="普通连线不支持配置条件；分支条件请在条件分支节点的出边上配置"
    />
  </div>
</template>

<style scoped lang="scss">
.workflow-edge-panel {
  display: flex;
  padding: 0 var(--el-space-md);
  flex-direction: column;
  gap: var(--el-space-sm);

  &__route {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--el-space-sm);
    color: var(--el-text-color-primary);
    font-size: 14px;
    font-weight: 600;
  }

  &__arrow {
    color: var(--el-text-color-secondary);
  }

  &__tip {
    border-radius: var(--el-border-radius-base);
  }

  &__readonly {
    :deep(.el-input__inner) {
      color: var(--el-text-color-secondary);
    }
  }
}
</style>
