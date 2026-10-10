<script setup lang="ts">
import type { DashboardDataSourceField, DashboardFormDataSource } from '~/types';
import {
  RiAddLine,
  RiCalendarLine,
  RiFileList3Fill,
  RiMapPinLine,
  RiSearchLine,
  RiTableLine,
  RiText,
  RiUser3Line,
} from '@remixicon/vue';
import { computed, shallowRef } from 'vue';

const props = defineProps<{
  source: DashboardFormDataSource;
  fields: DashboardDataSourceField[];
}>();
const emit = defineEmits<{
  changeSource: [];
  addField: [field: DashboardDataSourceField, target: 'dimension' | 'metric'];
}>();

const search = shallowRef('');
const searchOpen = shallowRef(false);
const visibleFields = computed(() => {
  const keyword = search.value.trim().toLocaleLowerCase();
  if (!keyword) return props.fields;
  return props.fields.filter((field) => field.label.toLocaleLowerCase().includes(keyword));
});

function fieldIcon(field: DashboardDataSourceField) {
  if (field.type === 'member') return RiUser3Line;
  if (field.type === 'department') return RiTableLine;
  if (field.fieldCode.toLocaleLowerCase().includes('address')) return RiMapPinLine;
  if (field.type === 'date' || field.type === 'datetime') return RiCalendarLine;
  return RiText;
}

function startDrag(event: DragEvent, field: DashboardDataSourceField) {
  event.dataTransfer?.setData('application/x-dashboard-field', field.fieldId);
  if (event.dataTransfer) event.dataTransfer.effectAllowed = 'copy';
}

function addField(field: DashboardDataSourceField) {
  emit('addField', field, field.groupable ? 'dimension' : 'metric');
}
</script>

<template>
  <aside class="chart-data-sidebar" aria-label="图表数据字段">
    <section class="chart-data-sidebar__source">
      <div class="chart-data-sidebar__section-title">
        <strong>数据源</strong>
        <button type="button" @click="emit('changeSource')">
          更改数据源
        </button>
      </div>
      <div class="chart-data-sidebar__source-name">
        <RiFileList3Fill aria-hidden="true" />
        <span>{{ source.name }}</span>
      </div>
      <label class="chart-data-sidebar__permission-label">数据获取权限</label>
      <el-select model-value="all" aria-label="数据获取权限">
        <el-option label="表单中的全部数据" value="all" />
      </el-select>
      <p class="chart-data-sidebar__permission-copy">
        用户访问图表时可以查看图表中的全部数据。
      </p>
    </section>

    <section class="chart-data-sidebar__fields">
      <div class="chart-data-sidebar__section-title">
        <strong>字段</strong>
        <span class="chart-data-sidebar__field-actions">
          <button type="button" aria-label="添加字段"><RiAddLine aria-hidden="true" /></button>
          <button
            type="button"
            aria-label="搜索字段"
            @click="searchOpen = !searchOpen; search = ''"
          >
            <RiSearchLine aria-hidden="true" />
          </button>
        </span>
      </div>
      <el-input
        v-if="searchOpen"
        v-model="search"
        class="chart-data-sidebar__field-search"
        clearable
        placeholder="搜索字段"
      />
      <div class="chart-data-sidebar__field-list">
        <button
          v-for="field in visibleFields"
          :key="field.fieldId"
          class="chart-data-sidebar__field"
          type="button"
          draggable="true"
          :title="`拖拽或点击添加${field.label}`"
          @dragstart="startDrag($event, field)"
          @click="addField(field)"
        >
          <component :is="fieldIcon(field)" aria-hidden="true" />
          <span>{{ field.label }}</span>
        </button>
        <p v-if="visibleFields.length === 0" class="chart-data-sidebar__empty">
          未找到匹配字段
        </p>
      </div>
    </section>
  </aside>
</template>

<style scoped>
.chart-data-sidebar {
  display: flex;
  flex: 0 0 248px;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color-lighter);
}

.chart-data-sidebar__source {
  padding: 14px 18px 8px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.chart-data-sidebar__section-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 28px;
}

.chart-data-sidebar__section-title strong {
  font-size: 14px;
  font-weight: 600;
}

.chart-data-sidebar__section-title > button {
  padding: 0;
  font: inherit;
  color: var(--el-color-primary);
  cursor: pointer;
  background: transparent;
  border: 0;
}

.chart-data-sidebar__source-name {
  display: flex;
  gap: 10px;
  align-items: center;
  margin: 12px 0 20px;
  color: var(--el-text-color-regular);
}

.chart-data-sidebar__source-name svg {
  width: 18px;
  color: #2ba8e8;
}

.chart-data-sidebar__permission-label {
  display: block;
  margin-bottom: 8px;
  font-size: 13px;
  font-weight: 600;
}

.chart-data-sidebar__source :deep(.el-select) {
  width: 100%;
}

.chart-data-sidebar__permission-copy {
  margin: 8px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

.chart-data-sidebar__fields {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  padding: 18px 0 0;
}

.chart-data-sidebar__fields > .chart-data-sidebar__section-title {
  padding: 0 18px 8px;
}

.chart-data-sidebar__field-actions {
  display: flex;
  gap: 4px;
}

.chart-data-sidebar__field-actions button {
  display: grid;
  width: 28px;
  height: 28px;
  padding: 0;
  color: var(--el-text-color-primary);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 4px;
  place-items: center;
}

.chart-data-sidebar__field-actions button:hover {
  color: var(--el-color-primary);
  background: var(--el-fill-color-light);
}

.chart-data-sidebar__field-actions svg {
  width: 18px;
}

.chart-data-sidebar__field-search {
  width: calc(100% - 36px);
  margin: 0 18px 8px;
}

.chart-data-sidebar__field-list {
  flex: 1;
  min-height: 0;
  padding: 0 12px 18px;
  overflow: auto;
}

.chart-data-sidebar__field {
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  gap: 6px;
  align-items: center;
  width: 100%;
  min-height: 28px;
  padding: 0 6px;
  font: inherit;
  color: var(--el-text-color-regular);
  text-align: left;
  cursor: grab;
  background: transparent;
  border: 0;
  border-radius: 4px;
}

.chart-data-sidebar__field:hover,
.chart-data-sidebar__field:focus-visible {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  outline: 0;
}

.chart-data-sidebar__field svg {
  width: 16px;
  color: #3478f6;
}

.chart-data-sidebar__field span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chart-data-sidebar__empty {
  padding: 36px 10px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  text-align: center;
}
</style>
