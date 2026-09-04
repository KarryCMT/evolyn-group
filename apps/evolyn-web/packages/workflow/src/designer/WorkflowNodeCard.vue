<script setup lang="ts">
import {
  RiBookmarkFill,
  RiGitBranchFill,
  RiPlayFill,
  RiServerFill,
  RiStopFill,
  RiUser3Fill,
} from '@remixicon/vue';
import { computed } from 'vue';
import type { WorkflowNodeType } from '../schema';

defineOptions({ name: 'WorkflowNodeCard' });

interface LogicFlowNodeProperties {
  workflowType?: WorkflowNodeType;
  label?: string;
  selected?: boolean;
  error?: boolean;
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
  {
    'workflow-node-card--selected': props.node.properties?.selected,
    'workflow-node-card--error': props.node.properties?.error,
  },
]);

/** 节点类型图标：起止用状态色胶囊，业务节点用主题色线性图标 */
const typeIcon = computed(() => {
  switch (nodeType.value) {
    case 'start':
      return RiPlayFill;
    case 'condition':
      return RiGitBranchFill;
    case 'cc':
      return RiBookmarkFill;
    case 'service':
      return RiServerFill;
    case 'end':
      return RiStopFill;
    default:
      return RiUser3Fill;
  }
});
</script>

<template>
  <div :class="nodeClasses">
    <span class="workflow-node-card__icon" aria-hidden="true">
      <component :is="typeIcon" />
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

  // 校验错误态：红色描边优先级最高，与错误面板/边高亮同语义
  &--error,
  &--error:hover {
    border-color: var(--el-color-danger);
    background: var(--el-color-danger-light-9);
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
    flex-shrink: 0;
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

  &--approval &__icon,
  &--condition &__icon,
  &--cc &__icon,
  &--service &__icon {
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

  &--error &__icon {
    color: var(--el-color-danger);
    background: transparent;
  }

  &__label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
</style>
