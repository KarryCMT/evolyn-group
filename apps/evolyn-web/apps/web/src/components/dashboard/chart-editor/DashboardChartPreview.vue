<script setup lang="ts">
import type {
  BusinessDashboardChartWidget,
  BusinessDashboardDataset,
  BusinessDashboardDatasetResult,
} from '@evolyn.do/dashboard';
import type { DashboardDataSourceField } from '~/types';
import { buildBusinessChartSpec } from '@evolyn.do/dashboard';
import { EvolynChart } from '@evolyn.do/ui';
import { RiBarChartBoxLine, RiSortAsc, RiSortDesc } from '@remixicon/vue';
import { computed, shallowRef } from 'vue';

const props = defineProps<{
  widget: BusinessDashboardChartWidget;
  dataset: BusinessDashboardDataset;
  fields: DashboardDataSourceField[];
  theme: 'light' | 'dark';
}>();

const ready = computed(
  () =>
    props.widget.settings.encoding.dimensions.length > 0 &&
    props.widget.settings.encoding.metrics.length > 0,
);
const sortDirection = shallowRef<'default' | 'ascending' | 'descending'>('default');
const sampleSeries = [
  { label: '测试', value: 1 },
  { label: '产品经理', value: 1 },
  { label: '技术实施', value: 1 },
  { label: '前端工程师', value: 1 },
  { label: '销售培训师', value: 2 },
  { label: '销售员', value: 4 },
  { label: 'HR', value: 1 },
  { label: 'UI设计师', value: 1 },
] as const;

const sampleRows = computed(() => {
  const dimensionIDs = props.widget.settings.encoding.dimensions.map((item) => item.field.fieldId);
  const metrics = props.widget.settings.encoding.metrics;
  const primaryDimension = props.fields.find((field) => field.fieldId === dimensionIDs[0]);

  // 两个维度时生成与目标稿一致的多系列数据，便于设计期直接判断颜色、图例和分组效果。
  const rows = dimensionIDs[1]
    ? sampleSeries.map((series) => {
        const row: Record<string, unknown> = {
          [dimensionIDs[0]!]: primaryDimension?.type === 'member' ? '李同学' : '示例数据',
          [dimensionIDs[1]!]: series.label,
        };
        metrics.forEach((metric, index) => {
          row[metric.aggregateAlias] = series.value + index;
        });
        return row;
      })
    : [
        {
          [dimensionIDs[0]!]: primaryDimension?.type === 'member' ? '李同学' : '示例数据',
          ...Object.fromEntries(metrics.map((metric, index) => [metric.aggregateAlias, 4 + index])),
        },
      ];

  if (sortDirection.value === 'default') return rows;
  const firstMetric = metrics[0]?.aggregateAlias;
  if (!firstMetric) return rows;
  const direction = sortDirection.value === 'ascending' ? 1 : -1;
  return [...rows].sort(
    (left, right) => (Number(left[firstMetric]) - Number(right[firstMetric])) * direction,
  );
});

const previewResult = computed<BusinessDashboardDatasetResult>(() => {
  const dimensions = props.widget.settings.encoding.dimensions
    .map((item) => item.field.fieldId)
    .map((fieldID) => props.fields.find((field) => field.fieldId === fieldID))
    .filter((field): field is DashboardDataSourceField => Boolean(field));
  const metrics = props.widget.settings.encoding.metrics;
  return {
    datasetId: props.dataset.id,
    columns: [
      ...dimensions.map((field) => ({
        key: field.fieldId,
        label: field.label,
        type: field.type,
      })),
      ...metrics.map((metric) => ({
        key: metric.aggregateAlias,
        label: metric.label ?? metric.aggregateAlias,
        type: 'number',
      })),
    ],
    rows: ready.value ? sampleRows.value : [],
    total: ready.value ? sampleRows.value.length : 0,
    page: 1,
    pageSize: 20,
  };
});
const spec = computed(() =>
  ready.value
    ? { ...buildBusinessChartSpec(props.widget, previewResult.value), animation: false }
    : null,
);

function toggleSort() {
  sortDirection.value =
    sortDirection.value === 'default'
      ? 'descending'
      : sortDirection.value === 'descending'
        ? 'ascending'
        : 'default';
}
</script>

<template>
  <section class="chart-preview" aria-label="图表预览">
    <header class="chart-preview__header">
      <strong>{{ widget.title || '未命名统计表' }}</strong>
      <button
        v-if="ready"
        type="button"
        :aria-label="sortDirection === 'ascending' ? '切换为默认排序' : '切换图表排序'"
        :title="sortDirection === 'default' ? '图表排序' : `当前为${sortDirection === 'ascending' ? '升序' : '降序'}`"
        @click="toggleSort"
      >
        <RiSortAsc v-if="sortDirection !== 'ascending'" aria-hidden="true" />
        <RiSortDesc v-else aria-hidden="true" />
      </button>
    </header>
    <div v-if="spec" class="chart-preview__chart">
      <EvolynChart :spec="spec" :theme="theme" width="100%" height="100%" />
    </div>
    <div v-else class="chart-preview__empty">
      <span class="chart-preview__empty-icon"><RiBarChartBoxLine aria-hidden="true" /></span>
      <p>
        拖拽左侧字段到上方
        <strong>维度</strong>、<strong>指标</strong>栏来添加数据
      </p>
    </div>
  </section>
</template>

<style scoped>
.chart-preview {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  margin: 0 16px 8px;
  overflow: hidden;
  background: var(--el-bg-color);
}

.chart-preview__header {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: space-between;
  height: 56px;
  padding: 0 30px;
}

.chart-preview__header strong {
  font-size: 15px;
  font-weight: 500;
}

.chart-preview__header button {
  display: grid;
  width: 32px;
  height: 32px;
  padding: 0;
  color: #3478f6;
  cursor: pointer;
  background: #fff;
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 2px;
  box-shadow: 0 2px 8px rgb(53 73 95 / 9%);
  place-items: center;
}

.chart-preview__header button:hover,
.chart-preview__header button:focus-visible {
  color: #1f64dd;
  outline: 1px solid #3478f6;
}

.chart-preview__header button svg {
  width: 18px;
}

.chart-preview__chart {
  flex: 1;
  min-height: 0;
  padding: 0 34px 20px;
}

.chart-preview__empty {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 26px;
  align-items: center;
  justify-content: center;
  min-height: 0;
  color: var(--el-text-color-secondary);
}

.chart-preview__empty-icon {
  display: grid;
  width: 82px;
  height: 82px;
  color: var(--el-color-primary-light-3);
  background: var(--el-color-primary-light-9);
  border-radius: 12px;
  box-shadow: 0 12px 20px rgb(115 148 177 / 14%);
  place-items: center;
}

.chart-preview__empty-icon svg {
  width: 52px;
  height: 52px;
}

.chart-preview__empty p {
  margin: 0;
  font-size: 14px;
}

.chart-preview__empty strong {
  font-weight: 500;
  color: var(--el-color-primary);
}
</style>
