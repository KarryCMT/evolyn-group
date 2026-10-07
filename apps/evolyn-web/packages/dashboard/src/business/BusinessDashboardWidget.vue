<script setup lang="ts">
import type { BusinessDashboardWidget, BusinessDashboardWidgetRuntime } from './types.js';
import { EvolynChart, EvolynTable } from '@evolyn.do/ui';
import { computed } from 'vue';
import { buildBusinessChartSpec, buildBusinessTableAdapter } from './adapters.js';

defineOptions({ name: 'BusinessDashboardWidget' });
const props = defineProps<{
  widget: BusinessDashboardWidget;
  runtime?: BusinessDashboardWidgetRuntime;
}>();
const emit = defineEmits<{
  retry: [widgetId: string];
  pageChange: [widgetId: string, page: number];
}>();

const chartSpec = computed(() =>
  props.widget.type === 'chart' && props.runtime?.result
    ? buildBusinessChartSpec(props.widget, props.runtime.result)
    : null,
);
const tableAdapter = computed(() =>
  props.widget.type === 'table' && props.runtime?.result
    ? buildBusinessTableAdapter(props.widget, props.runtime.result)
    : null,
);
</script>

<template>
  <article class="business-widget">
    <header class="business-widget__header">
      <span class="business-widget__eyebrow">{{
        widget.type === 'chart' ? 'CHART' : 'TABLE'
      }}</span>
      <strong class="business-widget__title">{{ widget.title || '未命名组件' }}</strong>
    </header>
    <div v-if="!widget.datasetId" class="business-widget__state">
      <strong>尚未绑定 Dataset</strong><span>从属性面板选择数据集并配置字段。</span>
    </div>
    <div
      v-else-if="runtime?.status === 'loading'"
      v-loading="true"
      class="business-widget__state"
    />
    <div
      v-else-if="runtime?.status === 'error'"
      class="business-widget__state business-widget__state--error"
    >
      <strong>数据加载失败</strong><span>{{ runtime.message || '请稍后重试' }}</span>
      <button type="button" @click="emit('retry', widget.id)">重试</button>
    </div>
    <div
      v-else-if="runtime?.status === 'success' && runtime.result?.rows.length === 0"
      class="business-widget__state"
    >
      <strong>{{
        widget.type === 'chart' ? '暂无可绘制的数据' : widget.settings.display.emptyText
      }}</strong>
      <span>调整筛选条件或确认数据源中已有记录。</span>
    </div>
    <div v-else-if="chartSpec" class="business-widget__content">
      <EvolynChart :spec="chartSpec" width="100%" height="100%" />
    </div>
    <div
      v-else-if="tableAdapter && runtime?.result"
      class="business-widget__content business-widget__content--table"
    >
      <EvolynTable
        :columns="tableAdapter.columns"
        :records="runtime.result.rows"
        :options="tableAdapter.options"
        :empty-text="widget.type === 'table' ? widget.settings.display.emptyText : '暂无数据'"
        width="100%"
        height="calc(100% - 34px)"
      />
      <footer class="business-widget__pager">
        <button
          type="button"
          :disabled="runtime.result.page <= 1"
          @click="emit('pageChange', widget.id, runtime.result.page - 1)"
        >
          上一页
        </button>
        <span>第 {{ runtime.result.page }} 页 · 共 {{ runtime.result.total }} 条</span>
        <button
          type="button"
          :disabled="runtime.result.page * runtime.result.pageSize >= runtime.result.total"
          @click="emit('pageChange', widget.id, runtime.result.page + 1)"
        >
          下一页
        </button>
      </footer>
    </div>
    <div v-else class="business-widget__state">
      <strong>等待预览数据</strong><span>保存草稿后在预览页查看真实结果。</span>
    </div>
  </article>
</template>

<style scoped>
.business-widget {
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  padding: 18px;
  overflow: hidden;
  color: #172033;
  background: #fff;
  border: 1px solid rgba(23, 32, 51, 0.08);
  border-radius: 14px;
  box-shadow: 0 10px 30px rgba(33, 48, 77, 0.07);
}
.business-widget__header {
  display: flex;
  align-items: baseline;
  gap: 10px;
}
.business-widget__eyebrow {
  color: #0f8f84;
  font:
    700 10px/1 ui-monospace,
    monospace;
  letter-spacing: 0.12em;
}
.business-widget__title {
  overflow: hidden;
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.business-widget__content,
.business-widget__state {
  position: relative;
  display: flex;
  height: calc(100% - 34px);
  margin-top: 16px;
  overflow: hidden;
  color: #8b95a7;
  background: #f5f7fa;
  border-radius: 8px;
}
.business-widget__state {
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 6px;
  padding: 14px;
  text-align: center;
}
.business-widget__state strong {
  color: #536075;
  font-size: 12px;
}
.business-widget__state span {
  font-size: 11px;
}
.business-widget__state button {
  padding: 5px 11px;
  color: #fff;
  cursor: pointer;
  background: #0f8f84;
  border: 0;
  border-radius: 6px;
}
.business-widget__state--error strong {
  color: #b54734;
}
.business-widget__content {
  display: block;
  background: #fff;
}
.business-widget__content--table {
  display: flex;
  flex-direction: column;
}
.business-widget__pager {
  display: flex;
  height: 34px;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  color: #68758a;
  font-size: 11px;
}
.business-widget__pager button {
  padding: 3px 7px;
  color: #0f766e;
  cursor: pointer;
  background: transparent;
  border: 1px solid #d7e0e6;
  border-radius: 5px;
}
.business-widget__pager button:disabled {
  color: #aab2bf;
  cursor: not-allowed;
}
</style>
