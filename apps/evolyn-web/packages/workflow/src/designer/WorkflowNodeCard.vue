<script setup lang="ts">
import { RiPlayFill, RiStopFill, RiUser3Fill } from '@remixicon/vue';
import { computed } from 'vue';
import type { WorkflowNodeType } from '../schema';

defineOptions({ name: 'WorkflowNodeCard' });

interface LogicFlowNodeProperties {
  workflowType?: WorkflowNodeType;
  label?: string;
  selected?: boolean;
}

interface LogicFlowVueNode {
  properties?: LogicFlowNodeProperties;
}

const props = defineProps<{
  node: LogicFlowVueNode;
}>();

const nodeType = computed<WorkflowNodeType>(
  () => props.node.properties?.workflowType ?? 'approval',
);
const label = computed(() => props.node.properties?.label ?? '未命名节点');
const nodeClasses = computed(() => [
  'workflow-node-card',
  `workflow-node-card--${nodeType.value}`,
  { 'workflow-node-card--selected': props.node.properties?.selected },
]);
</script>

<template>
  <div :class="nodeClasses">
    <span class="workflow-node-card__icon" aria-hidden="true">
      <RiPlayFill v-if="nodeType === 'start'" />
      <RiUser3Fill v-else-if="nodeType === 'approval'" />
      <RiStopFill v-else />
    </span>
    <span class="workflow-node-card__label">{{ label }}</span>
  </div>
</template>

<style scoped lang="scss">
.workflow-node-card {
  display: flex;
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  padding: 8px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 600;
  background: var(--el-bg-color);
  border: 2px solid transparent;
  border-radius: 10px;
  cursor: pointer;
  transition:
    background-color var(--el-transition-duration),
    border-color var(--el-transition-duration),
    color var(--el-transition-duration);

  &:hover {
    border-color: var(--el-color-primary-light-5);
    background: var(--el-color-primary-light-9);
  }

  &--selected,
  &--selected:hover {
    border-color: var(--el-color-primary);
    background: var(--el-bg-color);
  }

  &--start,
  &--end {
    border-radius: 999px;
  }

  &--end {
    color: var(--el-text-color-regular);
  }

  &__icon {
    display: inline-flex;
    width: 20px;
    height: 20px;
    align-items: center;
    justify-content: center;
    color: var(--el-color-white);
    background: var(--el-color-success);
    border-radius: 50%;

    svg {
      width: 12px;
      height: 12px;
    }
  }

  &--approval &__icon {
    width: 20px;
    height: 20px;
    color: var(--el-color-primary);
    background: transparent;

    svg {
      width: 20px;
      height: 20px;
    }
  }

  &--end &__icon {
    color: var(--el-color-white);
    background: var(--el-text-color-secondary);
  }

  &__label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
</style>
