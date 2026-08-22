<script
  setup
  lang="ts"
  generic="TType extends string, TPreset extends DashboardWidgetPreset<TType>"
>
import { ElScrollbar } from 'element-plus';
import { GridStack as GridStackCore } from 'gridstack';
import { computed, nextTick, onMounted } from 'vue';
import type { DashboardWidgetContent, DashboardWidgetPreset } from '../schema';

const props = withDefaults(
  defineProps<{
    presets: TPreset[];
    disabledPresetKeys?: string[];
    widgetComponent: string;
    getWidgetProps?: (widget: DashboardWidgetContent<TType>) => Record<string, unknown>;
  }>(),
  { disabledPresetKeys: () => [] },
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

/**
 * 与 DashboardDesignCanvas 的默认 dragSourceSelector 对齐。
 * 业务应用只传预设与 Widget Host，GridStack 的拖放协议在包内维护。
 */
async function setupDragSources() {
  await nextTick();
  const widgets = props.presets.map((preset) => {
    const widget = toWidgetContent(preset);
    return {
      id: widget.id,
      x: 0,
      y: 0,
      w: preset.w,
      h: preset.h,
      minW: preset.minW,
      minH: preset.minH,
      maxW: preset.maxW,
      maxH: preset.maxH,
      component: props.widgetComponent,
      props: props.getWidgetProps?.(widget) ?? { widget },
    };
  });

  GridStackCore.setupDragIn(
    '.dashboard-widget-palette__drag-source',
    { appendTo: 'body', helper: 'clone' },
    widgets,
  );
}

function toWidgetContent(preset: DashboardWidgetPreset<TType>): DashboardWidgetContent<TType> {
  return {
    id: `palette-${preset.key}`,
    type: preset.type,
    title: preset.title,
    config: preset.config,
  };
}

onMounted(setupDragSources);
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
  flex: 0 0 168px;
  box-sizing: border-box;
  display: flex;
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
    align-items: center;
    gap: 8px;
    width: 100%;
    height: var(--el-component-size);
    margin: 0;
    padding: 0 15px;
    cursor: grab;
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-light);
    border-radius: var(--el-border-radius-base);

    &:hover,
    &:focus-visible {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
      outline: none;
    }

    &:active {
      cursor: grabbing;
    }
    &--disabled {
      pointer-events: none;
      cursor: not-allowed;
      color: var(--el-text-color-disabled);
      background: var(--el-fill-color-lighter);
    }
  }
}
</style>
