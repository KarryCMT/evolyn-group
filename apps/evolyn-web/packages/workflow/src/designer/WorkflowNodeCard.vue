<script setup lang="ts">
import {
  RiBookmarkFill,
  RiGitBranchFill,
  RiPlayFill,
  RiPuzzle2Fill,
  RiServerFill,
  RiStopFill,
  RiUser3Fill,
} from '@remixicon/vue';
import { computed } from 'vue';
import type { WorkflowNodeType } from '../schema';

defineOptions({ name: 'WorkflowNodeCard' });

interface LogicFlowNodeProperties {
  nodeKey?: string;
  workflowType?: WorkflowNodeType;
  label?: string;
  selected?: boolean;
  error?: boolean;
  detailed?: boolean;
  subtitle?: string;
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
const subtitle = computed(() => props.node.properties?.subtitle ?? '');
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
    case 'subflow':
      return RiGitBranchFill;
    case 'plugin':
      return RiPuzzle2Fill;
    case 'service':
      return RiServerFill;
    case 'end':
      return RiStopFill;
    default:
      return RiUser3Fill;
  }
});

function requestAdd(direction: 'top' | 'right' | 'bottom' | 'left', event: MouseEvent) {
  const nodeKey = props.node.properties?.nodeKey;
  if (!nodeKey) return;
  (event.currentTarget as HTMLElement).dispatchEvent(
    new CustomEvent('workflow-node-add', {
      bubbles: true,
      composed: true,
      detail: { nodeKey, direction, clientX: event.clientX, clientY: event.clientY },
    }),
  );
}
</script>

<template>
  <div :class="nodeClasses">
    <template v-if="node.properties?.selected && !['end'].includes(nodeType)">
      <button
        v-for="direction in (['top', 'right', 'bottom', 'left'] as const)"
        :key="direction"
        type="button"
        class="workflow-node-card__add"
        :class="`workflow-node-card__add--${direction}`"
        aria-label="从当前节点添加后续节点"
        @mousedown.stop
        @click.stop="requestAdd(direction, $event)"
      >
        ＋
      </button>
    </template>
    <span class="workflow-node-card__icon" aria-hidden="true">
      <component :is="typeIcon" />
    </span>
    <span class="workflow-node-card__content">
      <span class="workflow-node-card__label">{{ label }}</span>
      <span v-if="node.properties?.detailed && subtitle" class="workflow-node-card__subtitle">
        {{ subtitle }}
      </span>
    </span>
  </div>
</template>

<style scoped lang="scss">
.workflow-node-card {
  position: relative;
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
  &--subflow &__icon,
  &--plugin &__icon,
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

  &__content {
    display: flex;
    min-width: 0;
    flex: 1;
    flex-direction: column;
    justify-content: center;
    gap: 5px;
  }

  &__subtitle {
    overflow: hidden;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    font-weight: 400;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__add {
    position: absolute;
    z-index: 2;
    display: flex;
    width: 18px;
    height: 18px;
    padding: 0;
    align-items: center;
    justify-content: center;
    color: var(--el-color-white);
    background: var(--el-color-primary-light-3);
    border: 2px solid var(--el-bg-color);
    border-radius: 50%;
    cursor: pointer;
    font-size: 13px;
    line-height: 1;

    &--top { top: -11px; left: 50%; transform: translateX(-50%); }
    &--right { top: 50%; right: -11px; transform: translateY(-50%); }
    &--bottom { bottom: -11px; left: 50%; transform: translateX(-50%); }
    &--left { top: 50%; left: -11px; transform: translateY(-50%); }
  }
}
</style>
