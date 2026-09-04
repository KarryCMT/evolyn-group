<script setup lang="ts">
import {
  RiBookmarkFill,
  RiGitBranchFill,
  RiServerFill,
  RiStopFill,
  RiUser3Fill,
} from '@remixicon/vue';
import { ElTooltip } from 'element-plus';
import type { Component } from 'vue';
import type { WorkflowNodeType } from '../schema';

/** 素材面板：可新增的节点类型（start 全图唯一、parallel 暂不开放编排） */
defineOptions({ name: 'WorkflowPalette' });

const emit = defineEmits<{
  addNode: [type: WorkflowNodeType];
}>();

const PALETTE_ITEMS: Array<{ type: WorkflowNodeType; label: string; icon: Component }> = [
  { type: 'approval', label: '审批节点', icon: RiUser3Fill },
  { type: 'condition', label: '条件分支', icon: RiGitBranchFill },
  { type: 'cc', label: '抄送节点', icon: RiBookmarkFill },
  { type: 'service', label: '服务调用', icon: RiServerFill },
  { type: 'end', label: '结束节点', icon: RiStopFill },
];
</script>

<template>
  <nav class="workflow-palette" aria-label="节点素材">
    <ElTooltip
      v-for="item in PALETTE_ITEMS"
      :key="item.type"
      :content="item.label"
      placement="right"
    >
      <button
        class="workflow-palette__item"
        type="button"
        :aria-label="`添加${item.label}`"
        @click="emit('addNode', item.type)"
      >
        <component :is="item.icon" />
        <span>{{ item.label.slice(0, 2) }}</span>
      </button>
    </ElTooltip>
  </nav>
</template>

<style scoped lang="scss">
.workflow-palette {
  display: flex;
  width: 72px;
  padding: var(--el-space-sm) 0;
  flex-direction: column;
  align-items: center;
  gap: var(--el-space-sm);
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color-lighter);

  &__item {
    display: flex;
    width: 52px;
    height: 52px;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 2px;
    color: var(--el-text-color-regular);
    font-size: 11px;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-base);
    cursor: pointer;
    transition:
      color var(--el-transition-duration),
      border-color var(--el-transition-duration),
      background-color var(--el-transition-duration);

    svg {
      width: 18px;
      height: 18px;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
      border-color: var(--el-color-primary-light-5);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 1px;
    }
  }
}
</style>
