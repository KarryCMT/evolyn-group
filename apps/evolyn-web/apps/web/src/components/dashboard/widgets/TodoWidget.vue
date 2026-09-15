<script setup lang="ts">
import type { Component } from 'vue';
import type { DashboardWidgetContent } from '~/types/dashboard';
import { DashboardWidgetFrame } from '@evolyn.do/dashboard';
import {
  RiCalendarCheckFill,
  RiHandHeartFill,
  RiLoginBoxFill,
  RiNotification3Fill,
  RiPlayCircleFill,
  RiSendPlaneFill,
} from '@remixicon/vue';

defineOptions({ name: 'TodoWidget' });
defineProps<{ widget: DashboardWidgetContent }>();

interface WorkflowEntry {
  label: string;
  icon: Component;
}

/** 与流程中心的常用入口保持一致，工作台中可快速识别并进入对应范围。 */
const workflowEntries: WorkflowEntry[] = [
  { label: '我的待办', icon: RiNotification3Fill },
  { label: '我发起的', icon: RiPlayCircleFill },
  { label: '我处理的', icon: RiCalendarCheckFill },
  { label: '抄送我的', icon: RiSendPlaneFill },
];

/** 操作类入口与个人流程列表分组展示，减少连续文字带来的视觉拥挤。 */
const actionEntries: WorkflowEntry[] = [
  { label: '发起流程', icon: RiLoginBoxFill },
  { label: '待办委托', icon: RiHandHeartFill },
];
</script>

<template>
  <DashboardWidgetFrame :title="widget.title">
    <nav class="todo-widget" aria-label="流程中心快捷入口">
      <div class="todo-widget__group">
        <button
          v-for="entry in workflowEntries"
          :key="entry.label"
          class="todo-widget__entry"
          type="button"
        >
          <component :is="entry.icon" class="todo-widget__entry-icon" aria-hidden="true" />
          <span>{{ entry.label }}</span>
        </button>
      </div>

      <div class="todo-widget__divider" aria-hidden="true" />

      <div class="todo-widget__group todo-widget__group--actions">
        <button
          v-for="entry in actionEntries"
          :key="entry.label"
          class="todo-widget__entry"
          type="button"
        >
          <component :is="entry.icon" class="todo-widget__entry-icon" aria-hidden="true" />
          <span>{{ entry.label }}</span>
        </button>
      </div>
    </nav>
  </DashboardWidgetFrame>
</template>

<style scoped lang="scss">
.todo-widget {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: var(--el-space-xs) var(--el-space-md) 0;
  color: var(--el-text-color-primary);

  &__group {
    display: grid;
    gap: var(--el-space-sm);
  }

  &__group--actions {
    padding-top: var(--el-space-xs);
  }

  &__entry {
    display: inline-flex;
    box-sizing: border-box;
    align-items: center;
    width: 100%;
    min-height: 30px;
    padding: 0 var(--el-space-lg);
    color: inherit;
    font: inherit;
    font-size: var(--el-font-size-base);
    font-weight: 600;
    line-height: 1.5;
    text-align: left;
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: var(--el-border-radius-small);
    transition:
      background-color var(--el-transition-duration-fast),
      box-shadow var(--el-transition-duration-fast);

    &:hover {
      background: var(--el-fill-color-light);
    }

    &:focus-visible {
      outline: none;
      background: var(--el-fill-color-light);
      box-shadow: 0 0 0 2px var(--el-color-primary-light-7);
    }
  }

  &__entry-icon {
    flex: 0 0 20px;
    width: 20px;
    height: 20px;
    margin-right: var(--el-space-lg);
    color: var(--el-text-color-regular);
  }

  &__divider {
    height: 1px;
    margin: var(--el-space-lg) 0 var(--el-space-sm);
    background: var(--el-border-color-lighter);
  }
}
</style>
