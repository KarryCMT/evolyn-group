<script setup lang="ts" generic="TType extends string">
import { RiDeleteBin6Line, RiEditLine, RiFileCopyLine } from '@remixicon/vue';
import type { Component } from 'vue';
import type { DashboardWidgetContent } from '../schema';
import DashboardWidgetHost from '../renderer/DashboardWidgetHost.vue';
import type { DashboardWidgetActionPlacement } from './actionPlacement';

defineOptions({ name: 'DashboardDesignWidgetHost' });

withDefaults(
  defineProps<{
    widget: DashboardWidgetContent<TType>;
    widgetRegistry: Partial<Record<TType, Component>>;
    getComponentProps?: (widget: DashboardWidgetContent<TType>) => Record<string, unknown>;
    selected?: boolean;
    actionPlacement?: DashboardWidgetActionPlacement;
  }>(),
  {
    getComponentProps: undefined,
    selected: false,
    actionPlacement: 'inside',
  },
);
const emit = defineEmits<{
  remove: [id: string];
  select: [id: string];
  edit: [id: string];
  duplicate: [id: string];
}>();
</script>

<template>
  <div
    class="dashboard-design-widget"
    :class="{ 'dashboard-design-widget--selected': selected }"
    role="button"
    tabindex="0"
    @click="emit('select', widget.id)"
    @keydown.enter="emit('select', widget.id)"
    @keydown.space.prevent="emit('select', widget.id)"
  >
    <div class="dashboard-design-widget__content">
      <DashboardWidgetHost
        :widget="widget"
        :widget-registry="widgetRegistry"
        :get-component-props="getComponentProps"
      />
    </div>
    <div
      class="dashboard-design-widget__actions"
      :class="`dashboard-design-widget__actions--${actionPlacement}`"
    >
      <button
        class="dashboard-design-widget__action"
        type="button"
        aria-label="编辑组件"
        title="编辑组件"
        @click.stop="emit('edit', widget.id)"
      >
        <RiEditLine />
      </button>
      <button
        class="dashboard-design-widget__action"
        type="button"
        aria-label="复制组件"
        title="复制组件"
        @click.stop="emit('duplicate', widget.id)"
      >
        <RiFileCopyLine />
      </button>
      <button
        class="dashboard-design-widget__action dashboard-design-widget__action--danger"
        type="button"
        aria-label="删除组件"
        title="删除组件"
        @click.stop="emit('remove', widget.id)"
      >
        <RiDeleteBin6Line />
      </button>
    </div>
  </div>
</template>

<style scoped lang="scss">
/* Vue 的 :deep() 用于控制 Remix Icon 动态组件生成的 SVG。 */
/* stylelint-disable selector-pseudo-class-no-unknown */
.dashboard-design-widget {
  position: relative;
  width: 100%;
  height: 100%;

  &__content {
    width: 100%;
    height: 100%;
    overflow: hidden;
  }

  &__actions {
    position: absolute;
    top: 0;
    right: 0;
    z-index: 3;
    display: flex;
    gap: 1px;
    padding: 2px;
    pointer-events: none;
    background: var(--el-bg-color-overlay);
    border: 1px solid var(--el-border-color);
    border-radius: 3px;
    box-shadow: var(--el-box-shadow-light);
    opacity: 0;
    transform: translateY(calc(-100% - 6px));
    transition:
      opacity var(--el-transition-duration-fast),
      transform var(--el-transition-duration-fast);
  }

  &__actions--inside {
    top: 6px;
    right: 6px;
    transform: none;
  }

  &--selected {
    outline: 2px solid var(--el-color-primary-light-3);
    outline-offset: -2px;
    border-radius: 2px;
  }

  &--selected &__actions,
  &:hover &__actions,
  &:focus-within &__actions {
    pointer-events: auto;
    opacity: 1;
  }

  &__action {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    padding: 0;
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: 2px;

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &--danger:hover {
      color: var(--el-color-danger);
      background: var(--el-color-danger-light-9);
    }

    :deep(svg) {
      width: 18px;
      height: 18px;
    }
  }
}
</style>
