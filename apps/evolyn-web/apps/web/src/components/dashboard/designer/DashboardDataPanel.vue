<script setup lang="ts">
import type { BusinessDashboardDataset, BusinessDashboardDatasetPatch } from '@evolyn.do/dashboard';
import type {
  QueryAggregate,
  QueryAggregateOperator,
  QueryCondition,
  QueryDocument,
  QueryOperator,
  QuerySort,
} from '@evolyn.do/query';
import type {
  DashboardDataSourceField,
  DashboardFormDataSource,
  DashboardFormFieldCatalog,
} from '~/types';
import { QUERY_OPERATORS_BY_FIELD_TYPE } from '@evolyn.do/query';
import { computed } from 'vue';

const props = defineProps<{
  sources: DashboardFormDataSource[];
  datasets: BusinessDashboardDataset[];
  catalogs: Record<string, DashboardFormFieldCatalog>;
  selectedDatasetId: string | null;
  loading: boolean;
  errorMessage: string;
}>();
const emit = defineEmits<{
  addDataset: [source: DashboardFormDataSource];
  selectDataset: [id: string | null];
  updateDataset: [id: string, patch: BusinessDashboardDatasetPatch];
  removeDataset: [id: string];
  requestCatalog: [formCode: string];
}>();

const selectedDataset = computed(
  () => props.datasets.find((item) => item.id === props.selectedDatasetId) ?? null,
);
const catalog = computed(() => {
  const formCode = selectedDataset.value?.source.formCode;
  return formCode ? (props.catalogs[formCode] ?? null) : null;
});
const fields = computed(() => catalog.value?.fields ?? []);

function selectDataset(id: string) {
  emit('selectDataset', id);
  const dataset = props.datasets.find((item) => item.id === id);
  if (dataset) emit('requestCatalog', dataset.source.formCode);
}

function patchQuery(patch: Partial<QueryDocument>) {
  if (!selectedDataset.value) return;
  emit('updateDataset', selectedDataset.value.id, {
    query: { ...selectedDataset.value.query, ...patch },
  });
}

function fieldOptions(predicate: (field: DashboardDataSourceField) => boolean) {
  return fields.value.filter(predicate);
}

function addAggregate() {
  const field = fieldOptions((item) => item.aggregates.length > 0)[0];
  if (!selectedDataset.value || !field) return;
  const aggregates = [...(selectedDataset.value.query.aggregates ?? [])];
  aggregates.push({
    field: field.fieldId,
    operator: field.aggregates[0],
    alias: `metric_${aggregates.length + 1}`,
  });
  patchQuery({ aggregates });
}

function updateAggregate(index: number, patch: Partial<QueryAggregate>) {
  const aggregates = [...(selectedDataset.value?.query.aggregates ?? [])];
  aggregates[index] = { ...aggregates[index]!, ...patch };
  patchQuery({ aggregates });
}

function changeAggregateField(index: number, fieldId: string) {
  const operator = aggregateOperators(fieldId)[0];
  if (operator) updateAggregate(index, { field: fieldId, operator });
}

function removeAggregate(index: number) {
  patchQuery({
    aggregates: (selectedDataset.value?.query.aggregates ?? []).filter((_, i) => i !== index),
  });
}

function addSort() {
  const field = fieldOptions((item) => item.sortable)[0];
  if (!field) return;
  patchQuery({
    sorts: [
      ...(selectedDataset.value?.query.sorts ?? []),
      { field: field.fieldId, direction: 'asc' },
    ],
  });
}

function updateSort(index: number, patch: Partial<QuerySort>) {
  const sorts = [...(selectedDataset.value?.query.sorts ?? [])];
  sorts[index] = { ...sorts[index]!, ...patch };
  patchQuery({ sorts });
}

function removeSort(index: number) {
  patchQuery({ sorts: (selectedDataset.value?.query.sorts ?? []).filter((_, i) => i !== index) });
}

function addFilter() {
  const field = fieldOptions((item) => item.filterable)[0];
  if (!field) return;
  const condition: QueryCondition = {
    type: 'condition',
    field: field.fieldId,
    operator: QUERY_OPERATORS_BY_FIELD_TYPE[field.type][0]!,
    value: '',
  };
  const current = selectedDataset.value?.query.filter;
  patchQuery({
    filter: current
      ? {
          type: 'group',
          conjunction: 'and',
          children:
            current.type === 'group' ? [...current.children, condition] : [current, condition],
        }
      : condition,
  });
}

function conditions(): QueryCondition[] {
  const filter = selectedDataset.value?.query.filter;
  if (!filter) return [];
  if (filter.type === 'condition') return [filter];
  return filter.children.filter((item): item is QueryCondition => item.type === 'condition');
}

function updateCondition(index: number, patch: Partial<QueryCondition>) {
  const current = conditions();
  current[index] = { ...current[index]!, ...patch };
  patchQuery({
    filter:
      current.length === 1 ? current[0] : { type: 'group', conjunction: 'and', children: current },
  });
}

function changeConditionField(index: number, fieldId: string) {
  const operator = operatorsFor(fieldId)[0];
  if (operator) updateCondition(index, { field: fieldId, operator, value: '' });
}

function changeConditionOperator(index: number, operator: QueryOperator) {
  updateCondition(index, {
    operator,
    value: operator === 'isNull' || operator === 'isNotNull' ? undefined : '',
  });
}

function conditionValueText(condition: QueryCondition) {
  return Array.isArray(condition.value)
    ? condition.value.join(', ')
    : String(condition.value ?? '');
}

function changeConditionValue(index: number, condition: QueryCondition, raw: string) {
  const value =
    condition.operator === 'in' ||
    condition.operator === 'notIn' ||
    condition.operator === 'between'
      ? raw
          .split(',')
          .map((item) => item.trim())
          .filter(Boolean)
      : raw;
  updateCondition(index, { value });
}

function removeCondition(index: number) {
  const next = conditions().filter((_, i) => i !== index);
  patchQuery({
    filter:
      next.length === 0
        ? undefined
        : next.length === 1
          ? next[0]
          : { type: 'group', conjunction: 'and', children: next },
  });
}

function fieldByID(fieldId: string) {
  return fields.value.find((item) => item.fieldId === fieldId);
}

function operatorsFor(fieldId: string): readonly QueryOperator[] {
  const field = fieldByID(fieldId);
  return field ? QUERY_OPERATORS_BY_FIELD_TYPE[field.type] : [];
}

function aggregateOperators(fieldId: string): readonly QueryAggregateOperator[] {
  return fieldByID(fieldId)?.aggregates ?? [];
}
</script>

<template>
  <aside class="data-panel" aria-label="数据面板">
    <header><span>DATASETS</span><strong>数据面板</strong></header>
    <div v-loading="loading" class="data-panel__scroll">
      <el-alert v-if="errorMessage" :title="errorMessage" type="error" :closable="false" />
      <section>
        <label>已发布表单</label>
        <button
          v-for="source in sources"
          :key="source.code"
          class="data-panel__source"
          type="button"
          @click="emit('addDataset', source)"
        >
          <span><strong>{{ source.name }}</strong><small>v{{ source.publishedVersion }}</small></span>
          <b>+</b>
        </button>
        <p v-if="!loading && !sources.length" class="data-panel__empty">
          暂无可查看的已发布表单
        </p>
      </section>
      <section v-if="datasets.length">
        <label>Dataset</label>
        <el-select
          :model-value="selectedDatasetId"
          placeholder="选择 Dataset"
          @change="selectDataset"
        >
          <el-option v-for="item in datasets" :key="item.id" :label="item.name" :value="item.id" />
        </el-select>
      </section>
      <template v-if="selectedDataset && catalog">
        <section>
          <label>Dataset 名称</label>
          <el-input
            :model-value="selectedDataset.name"
            maxlength="80"
            @change="emit('updateDataset', selectedDataset.id, { name: String($event).trim() })"
          />
        </section>
        <section>
          <label>投影字段</label>
          <el-select
            multiple
            collapse-tags
            :model-value="selectedDataset.query.projection ?? []"
            @change="patchQuery({ projection: $event })"
          >
            <el-option
              v-for="field in fieldOptions((item) => item.projectable)"
              :key="field.fieldId"
              :label="field.label"
              :value="field.fieldId"
            />
          </el-select>
        </section>
        <section>
          <label>分组字段</label>
          <el-select
            multiple
            collapse-tags
            :model-value="selectedDataset.query.groupBy ?? []"
            @change="patchQuery({ groupBy: $event })"
          >
            <el-option
              v-for="field in fieldOptions((item) => item.groupable)"
              :key="field.fieldId"
              :label="field.label"
              :value="field.fieldId"
            />
          </el-select>
        </section>
        <section>
          <div class="data-panel__section-title">
            <label>聚合指标</label><el-button text size="small" @click="addAggregate">
              添加
            </el-button>
          </div>
          <div
            v-for="(aggregate, index) in selectedDataset.query.aggregates ?? []"
            :key="`${aggregate.alias}-${index}`"
            class="data-panel__row"
          >
            <el-select :model-value="aggregate.field" @change="changeAggregateField(index, $event)">
              <el-option
                v-for="field in fieldOptions((item) => item.aggregates.length > 0)"
                :key="field.fieldId"
                :label="field.label"
                :value="field.fieldId"
              />
            </el-select>
            <el-select
              :model-value="aggregate.operator"
              @change="updateAggregate(index, { operator: $event })"
            >
              <el-option
                v-for="operator in aggregateOperators(aggregate.field)"
                :key="operator"
                :label="operator"
                :value="operator"
              />
            </el-select>
            <el-input
              :model-value="aggregate.alias"
              @change="updateAggregate(index, { alias: $event })"
            />
            <el-button text type="danger" @click="removeAggregate(index)">
              删除
            </el-button>
          </div>
        </section>
        <section>
          <div class="data-panel__section-title">
            <label>筛选条件</label><el-button text size="small" @click="addFilter">
              添加
            </el-button>
          </div>
          <div v-for="(condition, index) in conditions()" :key="index" class="data-panel__row">
            <el-select :model-value="condition.field" @change="changeConditionField(index, $event)">
              <el-option
                v-for="field in fieldOptions((item) => item.filterable)"
                :key="field.fieldId"
                :label="field.label"
                :value="field.fieldId"
              />
            </el-select>
            <el-select
              :model-value="condition.operator"
              @change="changeConditionOperator(index, $event)"
            >
              <el-option
                v-for="operator in operatorsFor(condition.field)"
                :key="operator"
                :label="operator"
                :value="operator"
              />
            </el-select>
            <el-input
              v-if="condition.operator !== 'isNull' && condition.operator !== 'isNotNull'"
              :model-value="conditionValueText(condition)"
              :placeholder="condition.operator === 'between' ? '起始值, 结束值' : '条件值'"
              @change="changeConditionValue(index, condition, $event)"
            />
            <el-button text type="danger" @click="removeCondition(index)">
              删除
            </el-button>
          </div>
        </section>
        <section>
          <div class="data-panel__section-title">
            <label>排序</label><el-button text size="small" @click="addSort">
              添加
            </el-button>
          </div>
          <div
            v-for="(sort, index) in selectedDataset.query.sorts"
            :key="index"
            class="data-panel__row data-panel__row--sort"
          >
            <el-select :model-value="sort.field" @change="updateSort(index, { field: $event })">
              <el-option
                v-for="field in fieldOptions((item) => item.sortable)"
                :key="field.fieldId"
                :label="field.label"
                :value="field.fieldId"
              />
              <el-option
                v-for="aggregate in selectedDataset.query.aggregates ?? []"
                :key="`aggregate-${aggregate.alias}`"
                :label="`指标 · ${aggregate.alias}`"
                :value="aggregate.alias"
              />
            </el-select>
            <el-select
              :model-value="sort.direction"
              @change="updateSort(index, { direction: $event })"
            >
              <el-option label="升序" value="asc" /><el-option label="降序" value="desc" />
            </el-select>
            <el-button text type="danger" @click="removeSort(index)">
              删除
            </el-button>
          </div>
        </section>
        <section>
          <label>分页大小</label>
          <el-input-number
            :model-value="selectedDataset.query.paging.pageSize"
            :min="1"
            :max="100"
            @change="patchQuery({ paging: { page: 1, pageSize: $event ?? 20 } })"
          />
        </section>
        <el-button text type="danger" @click="emit('removeDataset', selectedDataset.id)">
          删除 Dataset
        </el-button>
      </template>
    </div>
  </aside>
</template>

<style scoped>
/* Vue 的 :deep() 用于覆盖 Element Plus 内部控件宽度。 */
/* stylelint-disable selector-pseudo-class-no-unknown */
.data-panel {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  min-height: 0;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
}

.data-panel header {
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 22px 52px 16px 20px;
}

.data-panel header span {
  font:
    800 9px/1 ui-monospace,
    monospace;
  color: var(--el-color-primary);
  letter-spacing: 0.15em;
}

.data-panel header strong {
  font-size: 15px;
}

.data-panel__scroll {
  padding: 0 14px 20px;
  overflow: auto;
}

.data-panel section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 0;
  border-top: 1px solid var(--el-border-color-lighter);
}

.data-panel label {
  font-size: 11px;
  font-weight: 700;
  color: var(--el-text-color-regular);
}

.data-panel__source {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 9px 10px;
  color: inherit;
  text-align: left;
  cursor: pointer;
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color);
  border-radius: 9px;
}

.data-panel__source span {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.data-panel__source strong {
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 12px;
  white-space: nowrap;
}

.data-panel__source small,
.data-panel__empty {
  font-size: 10px;
  color: var(--el-text-color-secondary);
}

.data-panel__source b {
  font-size: 18px;
  color: var(--el-color-primary);
}

.data-panel__section-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.data-panel__row {
  display: grid;
  grid-template-columns: 1.25fr 0.8fr 1fr auto;
  gap: 5px;
}

.data-panel__row--sort {
  grid-template-columns: 1.6fr 1fr auto;
}

.data-panel :deep(.el-select),
.data-panel :deep(.el-input-number) {
  width: 100%;
}
</style>
