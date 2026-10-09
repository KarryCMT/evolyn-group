<script
  setup
  lang="ts"
  generic="TType extends string, TPreset extends DashboardWidgetPreset<TType>"
>
import { ElScrollbar } from 'element-plus';
import { computed, onMounted } from 'vue';
import type { DashboardWidgetContent, DashboardWidgetPreset } from '../schema';
import { setupDashboardWidgetDragSources } from './drag.js';

const props = withDefaults(
  defineProps<{
    presets: TPreset[];
    disabledPresetKeys?: string[];
    widgetComponent?: string;
    getWidgetProps?: (widget: DashboardWidgetContent<TType>) => Record<string, unknown>;
  }>(),
  {
    disabledPresetKeys: () => [],
    widgetComponent: 'DashboardDesignWidgetHost',
    getWidgetProps: undefined,
  },
);
const emit = defineEmits<{
  add: [preset: TPreset];
}>();

const disabledPresetKeySet = computed(() => new Set(props.disabledPresetKeys));

function isDisabled(preset: DashboardWidgetPreset<TType>) {
  return disabledPresetKeySet.value.has(preset.key);
}

function addPreset(preset: TPreset) {
  if (isDisabled(preset)) return;
  emit('add', preset);
}

onMounted(() =>
  setupDashboardWidgetDragSources({
    presets: props.presets,
    widgetComponent: props.widgetComponent,
    getWidgetProps: props.getWidgetProps,
  }),
);
</script>

<template>
  <aside class="dashboard-widget-palette">
    <strong class="dashboard-widget-palette__title">
      <slot name="title">页面组件</slot>
    </strong>
    <ElScrollbar class="dashboard-widget-palette__scrollbar">
      <div class="dashboard-widget-palette__list">
        <div
          v-for="preset in presets"
          :key="preset.key"
          class="dashboard-widget-palette__item dashboard-widget-palette__drag-source"
          :class="{ 'dashboard-widget-palette__item--disabled': isDisabled(preset) }"
          :data-widget-key="preset.key"
          role="button"
          :aria-disabled="isDisabled(preset)"
          :tabindex="isDisabled(preset) ? -1 : 0"
          @click="addPreset(preset)"
          @keydown.enter="addPreset(preset)"
          @keydown.space.prevent="addPreset(preset)"
        >
          <slot name="item" :preset="preset" :disabled="isDisabled(preset)">
            <span>{{ preset.title }}</span>
          </slot>
        </div>
      </div>
    </ElScrollbar>
  </aside>
</template>

<style scoped lang="scss">
.dashboard-widget-palette {
  box-sizing: border-box;
  display: flex;
  flex: 0 0 168px;
  flex-direction: column;
  min-height: 0;
  padding: 14px 12px;
  overflow: hidden;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color-lighter);

  &__title {
    display: block;
    margin-bottom: 8px;
    font-size: var(--el-font-size-base);
  }

  &__scrollbar {
    flex: 1;
    min-height: 0;
  }

  &__list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  &__item {
    box-sizing: border-box;
    display: flex;
    gap: 8px;
    align-items: center;
    width: 100%;
    height: var(--el-component-size);
    padding: 0 15px;
    margin: 0;
    color: var(--el-text-color-regular);
    cursor: grab;
    background: var(--el-fill-color-light);
    border-radius: var(--el-border-radius-base);

    &:hover,
    &:focus-visible {
      color: var(--el-color-primary);
      outline: none;
      background: var(--el-color-primary-light-9);
    }

    &:active {
      cursor: grabbing;
    }

    &--disabled {
      color: var(--el-text-color-disabled);
      pointer-events: none;
      cursor: not-allowed;
      background: var(--el-fill-color-lighter);
    }
  }
}
</style>
