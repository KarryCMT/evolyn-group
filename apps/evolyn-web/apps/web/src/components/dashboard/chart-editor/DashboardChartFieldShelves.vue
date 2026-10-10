<script setup lang="ts">
import { RiArrowDownSLine, RiCloseLine, RiFilter3Line } from '@remixicon/vue';

defineProps<{
  dimensions: Array<{ id: string; label: string }>;
  metrics: Array<{ id: string; label: string }>;
}>();
const emit = defineEmits<{
  dropField: [fieldId: string, target: 'dimension' | 'metric' | 'filter'];
  removeDimension: [fieldId: string];
  removeMetric: [alias: string];
}>();

function acceptDrop(event: DragEvent, target: 'dimension' | 'metric' | 'filter') {
  const fieldId = event.dataTransfer?.getData('application/x-dashboard-field');
  if (fieldId) emit('dropField', fieldId, target);
}
</script>

<template>
  <section class="chart-field-shelves" aria-label="图表字段配置">
    <div
      class="chart-field-shelves__row"
      @dragover.prevent
      @drop.prevent="acceptDrop($event, 'dimension')"
    >
      <strong>维度</strong>
      <div class="chart-field-shelves__content">
        <span v-for="item in dimensions" :key="item.id" class="chart-field-chip is-dimension">
          <RiArrowDownSLine aria-hidden="true" />{{ item.label }}
          <button type="button" :aria-label="`移除维度${item.label}`" @click="emit('removeDimension', item.id)">
            <RiCloseLine aria-hidden="true" />
          </button>
        </span>
        <span v-if="dimensions.length === 0" class="chart-field-shelves__placeholder">
          拖动左侧字段到此处来添加维度
        </span>
      </div>
    </div>
    <div
      class="chart-field-shelves__row"
      @dragover.prevent
      @drop.prevent="acceptDrop($event, 'metric')"
    >
      <strong>指标</strong>
      <div class="chart-field-shelves__content">
        <span v-for="item in metrics" :key="item.id" class="chart-field-chip is-metric">
          <RiArrowDownSLine aria-hidden="true" />{{ item.label }}
          <button type="button" :aria-label="`移除指标${item.label}`" @click="emit('removeMetric', item.id)">
            <RiCloseLine aria-hidden="true" />
          </button>
        </span>
        <span v-if="metrics.length === 0" class="chart-field-shelves__placeholder">
          拖动左侧字段到此处来添加指标
        </span>
      </div>
    </div>
    <div
      class="chart-field-shelves__row"
      @dragover.prevent
      @drop.prevent="acceptDrop($event, 'filter')"
    >
      <strong class="chart-field-shelves__filter-title">
        过滤条件<RiFilter3Line aria-hidden="true" />
      </strong>
      <div class="chart-field-shelves__content">
        <span class="chart-field-shelves__placeholder">拖动左侧字段到此处来添加过滤条件</span>
      </div>
    </div>
  </section>
</template>

<style scoped>
.chart-field-shelves {
  display: grid;
  gap: 10px;
  padding: 12px 16px 10px;
  background: var(--el-bg-color-page);
}

.chart-field-shelves__row {
  display: grid;
  box-sizing: border-box;
  grid-template-columns: 100px minmax(0, 1fr);
  align-items: center;
  height: 36px;
  padding: 3px 10px;
  background: var(--el-bg-color);
  border: 1px dashed var(--el-border-color);
}

.chart-field-shelves__row > strong {
  font-size: 13px;
  font-weight: 500;
}

.chart-field-shelves__content {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  min-width: 0;
}

.chart-field-shelves__placeholder {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.chart-field-chip {
  display: inline-flex;
  gap: 5px;
  align-items: center;
  max-width: 240px;
  min-height: 26px;
  padding: 0 8px;
  overflow: hidden;
  color: #fff;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  border-radius: 14px;
}

.chart-field-chip.is-dimension {
  background: #10b5a9;
}

.chart-field-chip.is-metric {
  background: #3478f6;
}

.chart-field-chip > svg {
  flex: none;
  width: 14px;
}

.chart-field-chip button {
  display: grid;
  flex: none;
  width: 17px;
  height: 17px;
  padding: 0;
  color: inherit;
  cursor: pointer;
  background: rgb(255 255 255 / 14%);
  border: 0;
  border-radius: 50%;
  opacity: 0;
  transition: opacity 120ms ease;
  place-items: center;
}

.chart-field-chip:hover button,
.chart-field-chip:focus-within button {
  opacity: 1;
}

.chart-field-chip button svg {
  width: 13px;
}

.chart-field-shelves__filter-title {
  display: flex;
  gap: 3px;
  align-items: center;
}

.chart-field-shelves__filter-title svg {
  width: 14px;
}
</style>
