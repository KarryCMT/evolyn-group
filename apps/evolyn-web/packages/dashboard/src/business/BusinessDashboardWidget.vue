<script setup lang="ts">
import type { BusinessDashboardWidget, BusinessDashboardWidgetRuntime } from './types.js';
import { EvolynChart, EvolynTable } from '@evolyn.do/ui';
import { computed } from 'vue';
import { buildBusinessChartSpec, buildBusinessTableAdapter } from './adapters.js';

defineOptions({ name: 'BusinessDashboardWidget' });
const props = withDefaults(
  defineProps<{
    widget: BusinessDashboardWidget;
    runtime?: BusinessDashboardWidgetRuntime;
    theme?: 'light' | 'dark';
  }>(),
  { runtime: undefined, theme: 'light' },
);
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
    ? buildBusinessTableAdapter(props.widget, props.runtime.result, props.theme)
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
      <EvolynChart :spec="chartSpec" :theme="theme" width="100%" height="100%" />
    </div>
    <div
      v-else-if="tableAdapter && runtime?.result"
      class="business-widget__content business-widget__content--table"
    >
      <EvolynTable
        :columns="tableAdapter.columns"
        :records="runtime.result.rows"
        :options="tableAdapter.options"
        :theme="theme"
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
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  padding: 12px;
  overflow: hidden;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 2px;
}

.business-widget__header {
  display: flex;
  flex: 0 0 auto;
  gap: 10px;
  align-items: baseline;
  min-width: 0;
}

.business-widget__eyebrow {
  display: none;
}

.business-widget__title {
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
}

.business-widget__content,
.business-widget__state {
  position: relative;
  box-sizing: border-box;
  display: flex;
  flex: 1 1 auto;
  width: 100%;
  min-width: 0;
  min-height: 0;
  margin-top: 8px;
  overflow: hidden;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
  border-radius: 0;
}

.business-widget__state {
  flex-direction: column;
  gap: 6px;
  align-items: center;
  justify-content: center;
  padding: 14px;
  text-align: center;
}

.business-widget__state strong {
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.business-widget__state span {
  font-size: 11px;
}

.business-widget__state button {
  padding: 5px 11px;
  color: var(--el-color-white);
  cursor: pointer;
  background: var(--el-color-primary);
  border: 0;
  border-radius: 6px;
}

.business-widget__state--error strong {
  color: var(--el-color-danger);
}

.business-widget__content {
  display: block;
  background: var(--el-bg-color);
}

.business-widget__content--table {
  display: flex;
  flex-direction: column;
}

.business-widget__pager {
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: flex-end;
  height: 34px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

.business-widget__pager button {
  padding: 3px 7px;
  color: var(--el-color-primary);
  cursor: pointer;
  background: transparent;
  border: 1px solid var(--el-border-color);
  border-radius: 5px;
}

.business-widget__pager button:disabled {
  color: var(--el-text-color-disabled);
  cursor: not-allowed;
}
</style>
