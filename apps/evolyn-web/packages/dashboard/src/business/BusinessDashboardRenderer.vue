<script setup lang="ts">
import { computed, markRaw } from 'vue';
import DashboardRenderer from '../renderer/DashboardRenderer.vue';
import type { DashboardSchema } from '../schema/types.js';
import BusinessDashboardWidgetView from './BusinessDashboardWidget.vue';
import type { BusinessDashboardDocument, BusinessDashboardWidgetRuntime } from './types.js';

const props = withDefaults(
  defineProps<{
    document: BusinessDashboardDocument;
    runtimes?: Record<string, BusinessDashboardWidgetRuntime>;
    theme?: 'light' | 'dark';
  }>(),
  { runtimes: () => ({}), theme: 'light' },
);
const emit = defineEmits<{
  retry: [widgetId: string];
  pageChange: [widgetId: string, page: number];
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

function componentProps(content: { config?: Record<string, unknown> }) {
  const widget = content.config?.businessWidget as { id?: string } | undefined;
  return {
    widget,
    runtime: widget?.id ? props.runtimes?.[widget.id] : undefined,
    theme: props.theme,
    onRetry: (widgetId: string) => emit('retry', widgetId),
    onPageChange: (widgetId: string, page: number) => emit('pageChange', widgetId, page),
  };
}
</script>

<template>
  <div class="business-renderer">
    <DashboardRenderer
      v-if="document.widgets.length"
      :schema="schema"
      :widget-registry="widgetRegistry"
      :get-component-props="componentProps"
      :options="{ cellHeight: document.settings.desktop.rowHeight }"
    />
    <div v-else class="business-renderer__empty">
      <span>EMPTY DASHBOARD</span>
      <strong>这个仪表盘还是空的</strong>
      <p>返回设计页添加组件后，再来预览布局效果。</p>
    </div>
  </div>
</template>

<style scoped>
.business-renderer {
  display: flex;
  min-height: 100%;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color-page);
}

.business-renderer__empty {
  width: min(520px, calc(100% - 48px));
  padding: 48px;
  margin: auto;
  text-align: center;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 20px;
  box-shadow: var(--el-box-shadow-dark);
}

.business-renderer__empty span {
  display: block;
  margin-bottom: 18px;
  font:
    800 11px/1 ui-monospace,
    monospace;
  color: var(--el-color-primary);
  letter-spacing: 0.16em;
}

.business-renderer__empty strong {
  display: block;
  font-size: 24px;
}

.business-renderer__empty p {
  margin: 10px 0 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
</style>
