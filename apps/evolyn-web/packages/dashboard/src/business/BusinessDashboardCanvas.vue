<script setup lang="ts">
import type { DashboardSchema } from '../schema/types.js';
import { computed, markRaw } from 'vue';
import DashboardDesignCanvas from '../designer/DashboardDesignCanvas.vue';
import BusinessDashboardWidgetView from './BusinessDashboardWidget.vue';
import type { BusinessDashboardDocument, BusinessDashboardWidget } from './types.js';

const props = defineProps<{
  document: BusinessDashboardDocument;
  selectedWidgetId: string | null;
  issueWidgetIds?: string[];
}>();
const emit = defineEmits<{
  'update-layouts': [value: Array<{ id: string; layout: BusinessDashboardWidget['layout'] }>];
  remove: [id: string];
  select: [id: string];
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
  emit(
    'update-layouts',
    value.widgets.map((widget) => ({
      id: widget.id,
      layout: { x: widget.x, y: widget.y, w: widget.w, h: widget.h },
    })),
  );
}

function componentProps(content: { config?: Record<string, unknown> }) {
  return { widget: content.config?.businessWidget };
}
</script>

<template>
  <section class="business-canvas" @click.self="emit('select', '')">
    <DashboardDesignCanvas
      :model-value="schema"
      :widget-registry="widgetRegistry"
      :get-component-props="componentProps"
      :selected-widget-id="selectedWidgetId"
      @update:model-value="updateSchema"
      @remove="emit('remove', $event)"
      @select="emit('select', $event)"
    />
    <div v-if="document.widgets.length === 0" class="business-canvas__empty">
      <span class="business-canvas__empty-index">01</span>
      <strong>从左侧加入第一个组件</strong>
      <p>空画布不会自动生成示例图表。你的每个组件都来自明确的设计选择。</p>
    </div>
    <div v-if="issueWidgetIds?.length" class="business-canvas__issue-count">
      {{ issueWidgetIds.length }} 个组件需要处理
    </div>
  </section>
</template>

<style scoped>
.business-canvas {
  position: relative;
  flex: 1;
  min-width: 0;
  min-height: 0;
  background-image: radial-gradient(circle at 1px 1px, rgba(32, 45, 69, 0.12) 1px, transparent 0);
  background-size: 22px 22px;
}
.business-canvas__empty {
  position: absolute;
  top: 50%;
  left: 50%;
  width: min(420px, calc(100% - 64px));
  padding: 34px;
  color: #263248;
  background: rgba(255, 255, 255, 0.92);
  border: 1px solid rgba(38, 50, 72, 0.12);
  border-radius: 18px;
  box-shadow: 0 20px 60px rgba(24, 38, 61, 0.1);
  transform: translate(-50%, -50%);
  backdrop-filter: blur(12px);
}
.business-canvas__empty-index {
  display: block;
  margin-bottom: 18px;
  color: #0f8f84;
  font:
    800 12px/1 ui-monospace,
    monospace;
  letter-spacing: 0.16em;
}
.business-canvas__empty strong {
  font-size: 22px;
  letter-spacing: -0.02em;
}
.business-canvas__empty p {
  max-width: 340px;
  margin: 10px 0 0;
  color: #748096;
  font-size: 13px;
  line-height: 1.7;
}
.business-canvas__issue-count {
  position: absolute;
  right: 18px;
  bottom: 18px;
  padding: 8px 12px;
  color: #a54029;
  background: #fff2ed;
  border: 1px solid #f0c5b8;
  border-radius: 999px;
  font-size: 12px;
}
</style>
