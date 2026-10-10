<script setup lang="ts">
import type {
  BusinessDashboardChartSettings,
  BusinessDashboardChartWidget,
} from '@evolyn.do/dashboard';
import type { QueryAggregate, QueryAggregateOperator } from '@evolyn.do/query';
import type { DashboardDataSourceField, DashboardFormDataSource } from '~/types';
import { RiArrowLeftLine } from '@remixicon/vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { computed, shallowRef, watch } from 'vue';
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router';
import DashboardChartConfigPanel from '~/components/dashboard/chart-editor/DashboardChartConfigPanel.vue';
import DashboardChartDataSidebar from '~/components/dashboard/chart-editor/DashboardChartDataSidebar.vue';
import DashboardChartFieldShelves from '~/components/dashboard/chart-editor/DashboardChartFieldShelves.vue';
import DashboardChartPreview from '~/components/dashboard/chart-editor/DashboardChartPreview.vue';
import DashboardDataSourceDialog from '~/components/dashboard/designer/DashboardDataSourceDialog.vue';
import { useDashboardDesignWorkspaceContext } from '~/components/dashboard/designer/dashboardDesignWorkspace';

defineOptions({ name: 'DashboardChartEditorPage' });

const route = useRoute();
const router = useRouter();
const workspace = useDashboardDesignWorkspaceContext();
const sourceDialogOpen = shallowRef(false);
let allowRouteLeave = false;

const draft = computed(() => workspace.chartEditor.draft.value);
const catalog = computed(() => {
  const formCode = draft.value?.source.code;
  return formCode ? (workspace.dataCatalog.catalogs.value[formCode] ?? null) : null;
});
const fields = computed(() => catalog.value?.fields ?? []);
const fieldsByID = computed(() => new Map(fields.value.map((field) => [field.fieldId, field])));
const dimensions = computed(() =>
  (draft.value?.widget.settings.encoding.dimensions ?? []).map((item) => ({
    id: item.field.fieldId,
    label: item.label ?? fieldsByID.value.get(item.field.fieldId)?.label ?? item.field.fieldId,
  })),
);
const metrics = computed(() => {
  const aggregates = new Map(
    (draft.value?.dataset.query.aggregates ?? []).map((item) => [item.alias, item]),
  );
  return (draft.value?.widget.settings.encoding.metrics ?? []).map((metric) => {
    const aggregate = aggregates.get(metric.aggregateAlias);
    const field = aggregate ? fieldsByID.value.get(aggregate.field) : undefined;
    return {
      id: metric.aggregateAlias,
      label:
        metric.label ??
        (field && aggregate ? `${field.label}(${aggregateLabel(aggregate.operator)})` : metric.aggregateAlias),
    };
  });
});

watch(
  () => String(route.params.widgetId ?? ''),
  async (widgetID) => {
    if (!widgetID || draft.value?.widget.id === widgetID) return;
    const widget = workspace.designer.editingDocument.value.widgets.find(
      (item): item is BusinessDashboardChartWidget => item.id === widgetID && item.type === 'chart',
    );
    const dataset = workspace.designer.editingDocument.value.datasets.find(
      (item) => item.id === widget?.datasetId,
    );
    const source = workspace.dataCatalog.sources.value.find(
      (item) => item.code === dataset?.source.formCode,
    );
    if (!widget || !dataset || !source) {
      ElMessage.warning('未找到可编辑的统计表，已返回仪表盘');
      allowRouteLeave = true;
      await navigateToCanvas();
      return;
    }
    await workspace.dataCatalog.ensureCatalog(source.code);
    workspace.chartEditor.beginEdit({ widget, dataset, source });
  },
  { immediate: true },
);

onBeforeRouteLeave(async () => {
  if (allowRouteLeave || !workspace.chartEditor.isDirty.value) {
    workspace.chartEditor.discard();
    return true;
  }
  try {
    await ElMessageBox.confirm('当前统计表配置尚未保存，离开后将丢失修改。', '未保存的配置', {
      confirmButtonText: '放弃并离开',
      cancelButtonText: '继续编辑',
      type: 'warning',
    });
    workspace.chartEditor.discard();
    return true;
  } catch {
    return false;
  }
});

function addField(field: DashboardDataSourceField, target: 'dimension' | 'metric' | 'filter') {
  if (target === 'filter') {
    ElMessage.info('过滤条件编辑将在后续迭代开放');
    return;
  }
  if (target === 'dimension') addDimension(field);
  else addMetric(field);
}

function addDimension(field: DashboardDataSourceField) {
  if (!field.groupable || !draft.value) {
    ElMessage.warning('该字段不能作为维度');
    return;
  }
  if (draft.value.widget.settings.encoding.dimensions.some((item) => item.field.fieldId === field.fieldId)) return;
  workspace.chartEditor.updateDataset((dataset) => ({
    ...dataset,
    query: {
      ...dataset.query,
      groupBy: [...(dataset.query.groupBy ?? []), field.fieldId],
    },
  }));
  workspace.chartEditor.updateWidget((widget) => ({
    ...widget,
    settings: {
      ...widget.settings,
      encoding: {
        ...widget.settings.encoding,
        dimensions: [...widget.settings.encoding.dimensions, { field: { fieldId: field.fieldId } }],
      },
    },
  }));
}

function addMetric(field: DashboardDataSourceField) {
  if (!field.aggregates.length || !draft.value) {
    ElMessage.warning('该字段不能作为指标');
    return;
  }
  const operator: QueryAggregateOperator = field.aggregates.includes('sum') ? 'sum' : field.aggregates[0]!;
  const alias = metricAlias(field.fieldId, operator);
  if (draft.value.widget.settings.encoding.metrics.some((item) => item.aggregateAlias === alias)) return;
  const aggregate: QueryAggregate = { field: field.fieldId, operator, alias };
  workspace.chartEditor.updateDataset((dataset) => ({
    ...dataset,
    query: {
      ...dataset.query,
      aggregates: [...(dataset.query.aggregates ?? []), aggregate],
    },
  }));
  workspace.chartEditor.updateWidget((widget) => ({
    ...widget,
    settings: {
      ...widget.settings,
      encoding: {
        ...widget.settings.encoding,
        metrics: [
          ...widget.settings.encoding.metrics,
          { aggregateAlias: alias, label: `${field.label}(${aggregateLabel(operator)})` },
        ],
      },
    },
  }));
}

function removeDimension(fieldID: string) {
  workspace.chartEditor.updateDataset((dataset) => ({
    ...dataset,
    query: {
      ...dataset.query,
      groupBy: (dataset.query.groupBy ?? []).filter((item) => item !== fieldID),
    },
  }));
  workspace.chartEditor.updateWidget((widget) => ({
    ...widget,
    settings: {
      ...widget.settings,
      encoding: {
        ...widget.settings.encoding,
        dimensions: widget.settings.encoding.dimensions.filter(
          (item) => item.field.fieldId !== fieldID,
        ),
      },
    },
  }));
}

function removeMetric(alias: string) {
  workspace.chartEditor.updateDataset((dataset) => ({
    ...dataset,
    query: {
      ...dataset.query,
      aggregates: (dataset.query.aggregates ?? []).filter((item) => item.alias !== alias),
    },
  }));
  workspace.chartEditor.updateWidget((widget) => ({
    ...widget,
    settings: {
      ...widget.settings,
      encoding: {
        ...widget.settings.encoding,
        metrics: widget.settings.encoding.metrics.filter((item) => item.aggregateAlias !== alias),
      },
    },
  }));
}

function updateSettings(settings: BusinessDashboardChartSettings) {
  workspace.chartEditor.updateWidget((widget) => ({ ...widget, settings }));
}

function dropField(fieldID: string, target: 'dimension' | 'metric' | 'filter') {
  const field = fieldsByID.value.get(fieldID);
  if (field) addField(field, target);
}

function showNewFormHint() {
  ElMessage.info('可从应用工作区新建表单');
}

async function changeSource(source: DashboardFormDataSource) {
  const loaded = await workspace.dataCatalog.ensureCatalog(source.code);
  if (!loaded) {
    ElMessage.error(workspace.dataCatalog.errorMessage.value || '字段目录加载失败');
    return;
  }
  workspace.chartEditor.replaceSource(source);
  sourceDialogOpen.value = false;
}

async function save() {
  const current = draft.value;
  if (!current) return;
  if (!current.widget.settings.encoding.dimensions.length) {
    ElMessage.warning('请至少添加一个维度');
    return;
  }
  if (!current.widget.settings.encoding.metrics.length) {
    ElMessage.warning('请至少添加一个指标');
    return;
  }
  workspace.chartEditor.commit();
  ElMessage.success('统计表配置已保存到当前仪表盘草稿');
  allowRouteLeave = true;
  await navigateToCanvas();
}

async function back() {
  if (!workspace.chartEditor.isDirty.value) {
    workspace.chartEditor.discard();
    allowRouteLeave = true;
    await navigateToCanvas();
    return;
  }
  try {
    await ElMessageBox.confirm('返回后将放弃当前统计表配置。', '返回仪表盘', {
      confirmButtonText: '放弃并返回',
      cancelButtonText: '继续编辑',
      type: 'warning',
    });
  } catch {
    return;
  }
  workspace.chartEditor.discard();
  allowRouteLeave = true;
  await navigateToCanvas();
}

function navigateToCanvas() {
  return router.push({
    name: 'dashboard-design',
    params: {
      appCode: workspace.appCode.value,
      dashboardCode: workspace.dashboardCode.value,
    },
  });
}

function metricAlias(fieldID: string, operator: QueryAggregateOperator) {
  return `metric_${fieldID.replaceAll(/\W/g, '_')}_${operator}`;
}

function aggregateLabel(operator: QueryAggregateOperator) {
  return { count: '计数', sum: '求和', avg: '平均值', min: '最小值', max: '最大值' }[operator];
}
</script>

<template>
  <main v-if="draft && catalog" class="chart-editor-page">
    <header class="chart-editor-page__header">
      <button type="button" aria-label="返回仪表盘" @click="back">
        <RiArrowLeftLine aria-hidden="true" />
      </button>
      <h1>{{ draft.widget.title || '未命名统计表' }}</h1>
      <div class="chart-editor-page__actions">
        <button type="button" @click="workspace.showHelp">
          如何创建统计表？
        </button>
        <el-button type="primary" @click="save">
          保存
        </el-button>
      </div>
    </header>
    <section class="chart-editor-page__workspace">
      <DashboardChartDataSidebar
        :source="draft.source"
        :fields="fields"
        @change-source="sourceDialogOpen = true"
        @add-field="addField"
      />
      <section class="chart-editor-page__center">
        <DashboardChartFieldShelves
          :dimensions="dimensions"
          :metrics="metrics"
          @drop-field="dropField"
          @remove-dimension="removeDimension"
          @remove-metric="removeMetric"
        />
        <DashboardChartPreview
          :widget="draft.widget"
          :dataset="draft.dataset"
          :fields="fields"
          theme="light"
        />
      </section>
      <DashboardChartConfigPanel :settings="draft.widget.settings" @update="updateSettings" />
    </section>
    <DashboardDataSourceDialog
      v-model="sourceDialogOpen"
      :sources="workspace.dataCatalog.sources.value"
      :loading="workspace.dataCatalog.loading.value"
      :error-message="workspace.dataCatalog.errorMessage.value"
      @confirm="changeSource"
      @retry="workspace.dataCatalog.load(workspace.dashboardCode.value)"
      @new-form="showNewFormHint"
    />
  </main>
  <section v-else v-loading="true" class="chart-editor-page__loading" />
</template>

<style scoped>
.chart-editor-page {
  /* 统计表配置属于高密度生产工具，固定使用浅色工作台以保持图表与配置项的辨识度。 */
  --el-color-primary: #08aaa3;
  --el-color-primary-light-3: #52c4bf;
  --el-color-primary-light-5: #88d8d4;
  --el-color-primary-light-7: #bce9e6;
  --el-color-primary-light-8: #d7f2f0;
  --el-color-primary-light-9: #edf9f8;
  --el-color-primary-dark-2: #078d87;
  --el-bg-color: #fff;
  --el-bg-color-page: #f5f7f9;
  --el-bg-color-overlay: #fff;
  --el-fill-color: #f0f3f5;
  --el-fill-color-light: #f4f6f8;
  --el-fill-color-lighter: #f7f9fa;
  --el-fill-color-blank: #fff;
  --el-text-color-primary: #263241;
  --el-text-color-regular: #4c5968;
  --el-text-color-secondary: #7d8793;
  --el-text-color-placeholder: #a2aab4;
  --el-border-color: #d8dee5;
  --el-border-color-light: #e3e8ed;
  --el-border-color-lighter: #edf0f3;
  --el-border-color-extra-light: #f4f6f8;
  --el-disabled-bg-color: #f3f5f6;
  --el-disabled-text-color: #a4abb3;
  --el-disabled-border-color: #e1e5e9;
  --el-box-shadow-lighter: 0 2px 9px rgb(35 52 70 / 10%);

  display: flex;
  flex-direction: column;
  width: 100%;
  min-width: 1160px;
  height: 100vh;
  overflow: hidden;
  color: var(--el-text-color-primary);
  color-scheme: light;
  background: var(--el-bg-color-page);
}

.chart-editor-page__header {
  display: grid;
  flex: none;
  grid-template-columns: 28px minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
  min-height: 54px;
  padding: 0 20px 0 12px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.chart-editor-page__header > button {
  display: grid;
  width: 28px;
  height: 28px;
  padding: 0;
  color: var(--el-text-color-regular);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 4px;
  place-items: center;
}

.chart-editor-page__header > button:hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.chart-editor-page__header > button svg {
  width: 22px;
}

.chart-editor-page__header h1 {
  margin: 0;
  overflow: hidden;
  font-size: 17px;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chart-editor-page__actions {
  display: flex;
  gap: 30px;
  align-items: center;
}

.chart-editor-page__actions > button {
  padding: 0 0 3px;
  font: inherit;
  color: var(--el-color-primary);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-bottom: 1px solid currentcolor;
}

.chart-editor-page__actions :deep(.el-button) {
  min-width: 80px;
  height: 32px;
  padding: 0 22px;
  color: #fff;
  background: var(--el-color-primary);
  border-color: var(--el-color-primary);
  border-radius: 6px;
}

.chart-editor-page__workspace {
  display: flex;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.chart-editor-page__center {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 520px;
  min-height: 0;
  background: var(--el-bg-color-page);
}

.chart-editor-page__loading {
  min-height: 100vh;
  background: var(--el-bg-color-page);
}

@media (width <= 1240px) {
  .chart-editor-page :deep(.chart-data-sidebar) {
    flex-basis: 220px;
  }

  .chart-editor-page :deep(.chart-config-panel) {
    flex-basis: 280px;
  }
}
</style>
