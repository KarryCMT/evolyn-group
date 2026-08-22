<script setup lang="ts" generic="TType extends string">
import { RiDeleteBinFill } from '@remixicon/vue';
import type { Component } from 'vue';
import type { DashboardWidgetContent } from '../schema';
import DashboardWidgetHost from '../renderer/DashboardWidgetHost.vue';

defineOptions({ name: 'DashboardDesignWidgetHost' });

const props = defineProps<{
  widget: DashboardWidgetContent<TType>;
  widgetRegistry: Partial<Record<TType, Component>>;
  getComponentProps?: (widget: DashboardWidgetContent<TType>) => Record<string, unknown>;
}>();
const emit = defineEmits<{
  remove: [id: string];
}>();
</script>

<template>
  <div class="dashboard-design-widget">
    <DashboardWidgetHost
      :widget="widget"
      :widget-registry="widgetRegistry"
      :get-component-props="getComponentProps"
    />
    <div class="dashboard-design-widget__actions">
      <button
        class="dashboard-design-widget__delete"
        type="button"
        aria-label="删除卡片"
        @click.stop="emit('remove', widget.id)"
      >
        <RiDeleteBinFill />
      </button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.dashboard-design-widget {
  position: relative;
  width: 100%;
  height: 100%;

  &__actions {
    position: absolute;
    top: 12px;
    right: 12px;
    z-index: 2;
    display: flex;
    pointer-events: none;
    opacity: 0;
    transition: opacity var(--el-transition-duration-fast);
  }

  &:hover &__actions,
  &:focus-within &__actions {
    pointer-events: auto;
    opacity: 1;
  }

  &__delete {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    padding: 0;
    cursor: pointer;
    color: var(--el-text-color-secondary);
    background: transparent;
    border: 0;
    border-radius: var(--el-border-radius-small);

    &:hover {
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
