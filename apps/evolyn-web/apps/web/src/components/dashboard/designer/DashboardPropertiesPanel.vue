<script setup lang="ts">
import type {
  BusinessDashboardDataset,
  BusinessDashboardIssue,
  BusinessDashboardWidget,
  BusinessDashboardWidgetPatch,
} from '@evolyn.do/dashboard';
import type { DashboardFormFieldCatalog } from '~/types';
import { computed, shallowRef, watch } from 'vue';

const props = defineProps<{
  widget: BusinessDashboardWidget | null;
  issues: BusinessDashboardIssue[];
  focusedIssuePath: string;
  datasets: BusinessDashboardDataset[];
  catalogs: Record<string, DashboardFormFieldCatalog>;
}>();
const emit = defineEmits<{
  update: [id: string, patch: BusinessDashboardWidgetPatch];
  remove: [id: string];
  focusIssue: [issue: BusinessDashboardIssue];
  requestCatalog: [formCode: string];
}>();

const title = shallowRef('');
const selectedIssue = computed(() =>
  props.issues.find((issue) => issue.path === props.focusedIssuePath),
);
const selectedDataset = computed(
  () => props.datasets.find((item) => item.id === props.widget?.datasetId) ?? null,
);
const selectedCatalog = computed(() => {
  const formCode = selectedDataset.value?.source.formCode;
  return formCode ? (props.catalogs[formCode] ?? null) : null;
});
const sortableTableFields = computed(() => {
  const widget = props.widget;
  if (!widget || widget.type !== 'table') return [];
  const columns = new Set(widget.settings.columns.map((column) => column.field.fieldId));
  return (selectedCatalog.value?.fields ?? []).filter(
    (field) => field.sortable && columns.has(field.fieldId),
  );
});

watch(
  () => props.widget,
  (widget) => {
    title.value = widget?.title ?? '';
  },
  { immediate: true },
);

function commitTitle() {
  const value = title.value.trim();
  if (!props.widget || !value || value === props.widget.title) return;
  emit('update', props.widget.id, { title: value });
}

function selectDataset(datasetId: string) {
  if (!props.widget) return;
  const settings =
    props.widget.type === 'chart'
      ? { ...props.widget.settings, encoding: { dimensions: [], metrics: [] } }
      : { ...props.widget.settings, columns: [], sorts: [] };
  emit('update', props.widget.id, { datasetId, settings });
  const dataset = props.datasets.find((item) => item.id === datasetId);
  if (dataset) emit('requestCatalog', dataset.source.formCode);
}

function patchSettings(patch: Record<string, unknown>) {
  if (!props.widget) return;
  emit('update', props.widget.id, {
    settings: { ...props.widget.settings, ...patch } as BusinessDashboardWidget['settings'],
  });
}

function patchChartEncoding(key: 'dimensions' | 'metrics', values: string[]) {
  if (!props.widget || props.widget.type !== 'chart') return;
  const encoding = {
    ...props.widget.settings.encoding,
    [key]:
      key === 'dimensions'
        ? values.map((fieldId) => ({ field: { fieldId } }))
        : values.map((aggregateAlias) => ({ aggregateAlias })),
  };
  patchSettings({ encoding });
}

function patchChartDisplay(key: string, value: unknown) {
  if (!props.widget || props.widget.type !== 'chart') return;
  patchSettings({ display: { ...props.widget.settings.display, [key]: value } });
}

function patchTableColumns(values: string[]) {
  if (!props.widget || props.widget.type !== 'table') return;
  const byField = new Map(props.widget.settings.columns.map((item) => [item.field.fieldId, item]));
  patchSettings({
    columns: values.map(
      (fieldId, index) =>
        byField.get(fieldId) ?? {
          id: `column_${index + 1}_${fieldId}`,
          field: { fieldId },
          align: 'left',
          format: 'auto',
        },
    ),
  });
}

function patchTableSort(fieldId: string | undefined) {
  if (!props.widget || props.widget.type !== 'table') return;
  patchSettings({
    sorts: fieldId
      ? [{ field: { fieldId }, direction: props.widget.settings.sorts[0]?.direction ?? 'asc' }]
      : [],
  });
}

function patchTableSortDirection(direction: 'asc' | 'desc') {
  if (!props.widget || props.widget.type !== 'table') return;
  const field = props.widget.settings.sorts[0]?.field;
  if (field) patchSettings({ sorts: [{ field, direction }] });
}
</script>

<template>
  <aside class="properties-panel" aria-label="组件属性">
    <header class="properties-panel__header">
      <span>INSPECTOR</span>
      <strong>属性面板</strong>
    </header>
    <div v-if="widget" class="properties-panel__body">
      <div class="properties-panel__type">
        <span>{{ widget.type === 'chart' ? '统计图' : '明细表' }}</span>
        <code>{{ widget.id }}</code>
      </div>
      <el-form label-position="top">
        <el-form-item
          label="组件标题"
          :class="{ 'is-issue': selectedIssue?.path.endsWith('.title') }"
        >
          <el-input v-model="title" maxlength="60" show-word-limit @change="commitTitle" />
        </el-form-item>
        <el-form-item label="Dataset">
          <el-select
            :model-value="widget.datasetId"
            placeholder="选择数据集"
            @change="selectDataset"
          >
            <el-option
              v-for="dataset in datasets"
              :key="dataset.id"
              :label="dataset.name"
              :value="dataset.id"
            />
          </el-select>
        </el-form-item>
        <template v-if="selectedDataset && selectedCatalog && widget.type === 'chart'">
          <el-form-item label="维度字段">
            <el-select
              multiple
              :model-value="widget.settings.encoding.dimensions.map((item) => item.field.fieldId)"
              @change="patchChartEncoding('dimensions', $event)"
            >
              <el-option
                v-for="field in selectedCatalog.fields.filter((item) => item.groupable)"
                :key="field.fieldId"
                :label="field.label"
                :value="field.fieldId"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="聚合指标">
            <el-select
              multiple
              :model-value="widget.settings.encoding.metrics.map((item) => item.aggregateAlias)"
              @change="patchChartEncoding('metrics', $event)"
            >
              <el-option
                v-for="metric in selectedDataset.query.aggregates ?? []"
                :key="metric.alias"
                :label="metric.alias"
                :value="metric.alias"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="图表类型">
            <el-segmented
              :model-value="widget.settings.display.variant"
              :options="[
                { label: '柱状', value: 'bar' },
                { label: '折线', value: 'line' },
                { label: '饼图', value: 'pie' },
              ]"
              @change="patchChartDisplay('variant', $event)"
            />
          </el-form-item>
        </template>
        <template v-if="selectedDataset && selectedCatalog && widget.type === 'table'">
          <el-form-item label="展示列">
            <el-select
              multiple
              :model-value="widget.settings.columns.map((item) => item.field.fieldId)"
              @change="patchTableColumns"
            >
              <el-option
                v-for="field in selectedCatalog.fields.filter((item) => item.projectable)"
                :key="field.fieldId"
                :label="field.label"
                :value="field.fieldId"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="表格密度">
            <el-select
              :model-value="widget.settings.display.density"
              @change="patchSettings({ display: { ...widget.settings.display, density: $event } })"
            >
              <el-option label="紧凑" value="compact" /><el-option
                label="默认"
                value="default"
              /><el-option label="宽松" value="comfortable" />
            </el-select>
          </el-form-item>
          <el-form-item label="每页数量">
            <el-input-number
              :model-value="widget.settings.pagination.pageSize"
              :min="1"
              :max="100"
              @change="patchSettings({ pagination: { pageSize: $event ?? 20 } })"
            />
          </el-form-item>
          <el-form-item label="默认排序">
            <el-select
              clearable
              :model-value="widget.settings.sorts[0]?.field.fieldId"
              @change="patchTableSort"
            >
              <el-option
                v-for="field in sortableTableFields"
                :key="field.fieldId"
                :label="field.label"
                :value="field.fieldId"
              />
            </el-select>
          </el-form-item>
          <el-form-item v-if="widget.settings.sorts[0]" label="排序方向">
            <el-segmented
              :model-value="widget.settings.sorts[0].direction"
              :options="[
                { label: '升序', value: 'asc' },
                { label: '降序', value: 'desc' },
              ]"
              @change="patchTableSortDirection"
            />
          </el-form-item>
          <el-form-item label="展示选项">
            <div class="properties-panel__switches">
              <el-switch
                :model-value="widget.settings.display.striped"
                active-text="斑马纹"
                @change="
                  patchSettings({ display: { ...widget.settings.display, striped: $event } })
                "
              />
              <el-switch
                :model-value="widget.settings.display.bordered"
                active-text="边框"
                @change="
                  patchSettings({ display: { ...widget.settings.display, bordered: $event } })
                "
              />
              <el-switch
                :model-value="widget.settings.display.showHeader"
                active-text="表头"
                @change="
                  patchSettings({ display: { ...widget.settings.display, showHeader: $event } })
                "
              />
            </div>
          </el-form-item>
        </template>
        <div class="properties-panel__layout">
          <span v-for="key in ['x', 'y', 'w', 'h'] as const" :key="key">
            <small>{{ key.toUpperCase() }}</small><strong>{{ widget.layout[key] }}</strong>
          </span>
        </div>
        <p class="properties-panel__hint">
          拖动画布组件改变位置，使用边缘手柄调整尺寸。
        </p>
      </el-form>
      <el-button
        class="properties-panel__delete"
        text
        type="danger"
        @click="emit('remove', widget.id)"
      >
        删除组件
      </el-button>
    </div>
    <div v-else class="properties-panel__empty">
      <span>SELECT A WIDGET</span>
      <strong>选择画布中的组件</strong>
      <p>这里会显示标题、布局和后续的数据配置。</p>
    </div>
    <div v-if="issues.length" class="properties-panel__issues">
      <strong>需要处理的问题</strong>
      <button
        v-for="issue in issues"
        :key="`${issue.path}-${issue.code}`"
        type="button"
        @click="emit('focusIssue', issue)"
      >
        <code>{{ issue.path }}</code><span>{{ issue.message }}</span>
      </button>
    </div>
  </aside>
</template>

<style scoped>
/* Vue 的 :deep() 用于保证抽屉中的 Element Plus 控件占满可用宽度。 */
/* stylelint-disable selector-pseudo-class-no-unknown */
.properties-panel {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  min-height: 0;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
}

.properties-panel__header {
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 22px 52px 17px 20px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.properties-panel__header span,
.properties-panel__empty > span {
  font:
    800 9px/1 ui-monospace,
    monospace;
  color: var(--el-color-primary);
  letter-spacing: 0.15em;
}

.properties-panel__header strong {
  font-size: 16px;
}

.properties-panel__body {
  padding: 20px;
  overflow: auto;
}

.properties-panel__type {
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 14px;
  margin-bottom: 20px;
  background: var(--el-fill-color-light);
  border-radius: 11px;
}

.properties-panel__type span {
  font-size: 13px;
  font-weight: 700;
}

.properties-panel__type code {
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 10px;
  color: var(--el-text-color-secondary);
}

.properties-panel__layout {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 6px;
}

.properties-panel__switches {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
}

.properties-panel__layout span {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 9px;
  text-align: center;
  background: var(--el-fill-color-light);
  border-radius: 8px;
}

.properties-panel__layout small {
  font:
    700 9px/1 ui-monospace,
    monospace;
  color: var(--el-text-color-placeholder);
}

.properties-panel__layout strong {
  font-size: 13px;
}

.properties-panel__hint {
  font-size: 11px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

.properties-panel__delete {
  margin-top: 12px;
}

.properties-panel__empty {
  display: flex;
  flex-direction: column;
  gap: 9px;
  padding: 34px 24px;
}

.properties-panel__empty strong {
  font-size: 15px;
}

.properties-panel__empty p {
  margin: 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

.properties-panel__issues {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 230px;
  padding: 16px 18px;
  margin-top: auto;
  overflow: auto;
  background: var(--el-color-danger-light-9);
  border-top: 1px solid var(--el-color-danger-light-7);
}

.properties-panel__issues > strong {
  font-size: 12px;
  color: var(--el-color-danger);
}

.properties-panel__issues button {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 9px;
  color: var(--el-color-danger-dark-2);
  text-align: left;
  cursor: pointer;
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-color-danger-light-7);
  border-radius: 8px;
}

.properties-panel__issues code {
  font-size: 9px;
}

.properties-panel__issues span {
  font-size: 11px;
}

.is-issue :deep(.el-input__wrapper) {
  box-shadow: 0 0 0 1px var(--el-color-danger) inset;
}
</style>
