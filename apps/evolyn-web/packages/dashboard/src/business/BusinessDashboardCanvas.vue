<script setup lang="ts">
import type { DashboardSchema } from '../schema/types.js';
import { computed, markRaw } from 'vue';
import DashboardDesignCanvas from '../designer/DashboardDesignCanvas.vue';
import BusinessDashboardWidgetView from './BusinessDashboardWidget.vue';
import type {
  BusinessDashboardDocument,
  BusinessDashboardWidget,
  BusinessDashboardWidgetType,
} from './types.js';

const props = withDefaults(
  defineProps<{
    document: BusinessDashboardDocument;
    selectedWidgetId: string | null;
    issueWidgetIds?: string[];
    preview?: 'desktop' | 'mobile';
    interactionMode?: 'move' | 'resize';
    theme?: 'light' | 'dark';
    emptyIllustration?: string;
  }>(),
  {
    issueWidgetIds: () => [],
    preview: 'desktop',
    interactionMode: 'move',
    theme: 'light',
    emptyIllustration: '',
  },
);
const emit = defineEmits<{
  'update-layouts': [value: Array<{ id: string; layout: BusinessDashboardWidget['layout'] }>];
  remove: [id: string];
  select: [id: string];
  edit: [id: string];
  duplicate: [id: string];
  drop: [type: BusinessDashboardWidgetType, layout: BusinessDashboardWidget['layout']];
  learn: [];
}>();

const widgetRegistry = {
  chart: markRaw(BusinessDashboardWidgetView),
  table: markRaw(BusinessDashboardWidgetView),
};
const schema = computed<DashboardSchema<'chart' | 'table'>>(() => ({
  version: 1,
  widgets: props.document.widgets.map((widget) => ({
    id: widget.id,
    type: widget.type,
    title: widget.title ?? '',
    ...widget.layout,
    config: { businessWidget: widget },
  })),
}));

function updateSchema(value: DashboardSchema<'chart' | 'table'>) {
  const existingIDs = new Set(props.document.widgets.map((widget) => widget.id));
  for (const widget of value.widgets) {
    if (!existingIDs.has(widget.id)) {
      emit('drop', widget.type, { x: widget.x, y: widget.y, w: widget.w, h: widget.h });
    }
  }
  emit(
    'update-layouts',
    value.widgets
      .filter((widget) => existingIDs.has(widget.id))
      .map((widget) => ({
        id: widget.id,
        layout: { x: widget.x, y: widget.y, w: widget.w, h: widget.h },
      })),
  );
}

function componentProps(content: { config?: Record<string, unknown> }) {
  return { widget: content.config?.businessWidget, theme: props.theme };
}
</script>

<template>
  <section class="business-canvas" @click.self="emit('select', '')">
    <DashboardDesignCanvas
      :model-value="schema"
      :widget-registry="widgetRegistry"
      :get-component-props="componentProps"
      :selected-widget-id="selectedWidgetId"
      :preview="preview"
      :interaction-mode="interactionMode"
      :row-height="document.settings.desktop.rowHeight"
      @update:model-value="updateSchema"
      @remove="emit('remove', $event)"
      @select="emit('select', $event)"
      @edit="emit('edit', $event)"
      @duplicate="emit('duplicate', $event)"
    />
    <div v-if="document.widgets.length === 0" class="business-canvas__empty">
      <img
        v-if="emptyIllustration"
        class="business-canvas__empty-image"
        :src="emptyIllustration"
        alt=""
      >
      <p>从左侧拖拽或点击添加图表/组件</p>
      <button type="button" @click="emit('learn')">了解仪表盘和组件</button>
    </div>
    <div v-if="issueWidgetIds?.length" class="business-canvas__issue-count">
      {{ issueWidgetIds.length }} 个组件需要处理
    </div>
  </section>
</template>

<style scoped>
.business-canvas {
  position: relative;
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  background: var(--el-bg-color-page);
}

.business-canvas__empty {
  position: absolute;
  top: 50%;
  left: 50%;
  display: flex;
  flex-direction: column;
  align-items: center;
  width: min(420px, calc(100% - 64px));
  color: var(--el-text-color-primary);
  text-align: center;
  transform: translate(-50%, -50%);
}

.business-canvas__empty-image {
  width: 156px;
  height: 118px;
  object-fit: contain;
}

.business-canvas__empty p {
  margin: 18px 0 8px;
  font-size: 14px;
  color: var(--el-text-color-secondary);
}

.business-canvas__empty button {
  padding: 0;
  font: inherit;
  font-size: 14px;
  color: var(--el-color-primary);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-bottom: 1px solid currentcolor;
}

.business-canvas__empty button:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: 4px;
}

.business-canvas__issue-count {
  position: absolute;
  right: 18px;
  bottom: 18px;
  padding: 8px 12px;
  font-size: 12px;
  color: var(--el-color-danger);
  background: var(--el-color-danger-light-9);
  border: 1px solid var(--el-color-danger-light-7);
  border-radius: 999px;
}
</style>
