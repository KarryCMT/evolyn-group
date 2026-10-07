<script setup lang="ts">
import { computed, markRaw } from 'vue';
import DashboardRenderer from '../renderer/DashboardRenderer.vue';
import type { DashboardSchema } from '../schema/types.js';
import BusinessDashboardWidgetView from './BusinessDashboardWidget.vue';
import type { BusinessDashboardDocument } from './types.js';

const props = defineProps<{ document: BusinessDashboardDocument }>();
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
  return { widget: content.config?.businessWidget };
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
  color: #172033;
  background: #eef2f5;
}
.business-renderer__empty {
  width: min(520px, calc(100% - 48px));
  margin: auto;
  padding: 48px;
  text-align: center;
  background: #fff;
  border: 1px solid rgba(23, 32, 51, 0.08);
  border-radius: 20px;
  box-shadow: 0 24px 70px rgba(30, 44, 68, 0.09);
}
.business-renderer__empty span {
  display: block;
  margin-bottom: 18px;
  color: #0f8f84;
  font:
    800 11px/1 ui-monospace,
    monospace;
  letter-spacing: 0.16em;
}
.business-renderer__empty strong {
  display: block;
  font-size: 24px;
}
.business-renderer__empty p {
  margin: 10px 0 0;
  color: #7b8699;
  font-size: 13px;
}
</style>
