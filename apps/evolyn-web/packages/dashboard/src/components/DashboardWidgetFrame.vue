<script setup lang="ts">
defineOptions({ name: 'DashboardWidgetFrame' });

defineProps<{
  title: string;
}>();

defineSlots<{
  default(): unknown;
  actions?(): unknown;
}>();
</script>

<template>
  <section class="dashboard-widget">
    <header class="dashboard-widget__header">
      <span class="dashboard-widget__drag-handle" aria-label="拖动卡片">⠿</span>
      <strong class="dashboard-widget__title">{{ title }}</strong>
      <div v-if="$slots.actions" class="dashboard-widget__actions">
        <slot name="actions" />
      </div>
    </header>
    <div class="dashboard-widget__body">
      <slot />
    </div>
  </section>
</template>

<style scoped lang="scss">
.dashboard-widget {
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  padding: 16px;
  overflow: hidden;
  background: var(--el-bg-color);
  border: 0;
  border-radius: var(--el-border-radius-base);
  box-shadow: var(--el-box-shadow-lighter);

  &__header {
    display: flex;
    align-items: center;
    min-height: var(--el-component-size-small);
  }

  &__drag-handle {
    width: 0;
    overflow: hidden;
    color: var(--el-text-color-secondary);
    cursor: grab;
    transition: width var(--el-transition-duration-fast);
  }

  &__title {
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-base);
  }

  &__actions {
    margin-left: auto;
  }

  &__body {
    height: calc(100% - var(--el-component-size-small));
  }
}
</style>
