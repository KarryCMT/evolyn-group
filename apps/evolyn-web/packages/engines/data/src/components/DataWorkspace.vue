<script setup lang="ts">
import { EvolynTable } from '@evolyn.do/ui';
import { computed } from 'vue';
import DataToolbar from './DataToolbar.vue';
import type { DataAction, DataColumn, DataPagination, DataQuery, DataRecord } from '../types.js';

defineOptions({ name: 'DataWorkspace' });

const props = withDefaults(
  defineProps<{
    actions: DataAction[];
    columns: DataColumn[];
    records: DataRecord[];
    query: DataQuery;
    pagination: DataPagination;
    searchPlaceholder?: string;
  }>(),
  {
    searchPlaceholder: '搜索数据',
  },
);

const emit = defineEmits<{
  action: [key: string];
  updateQuery: [query: DataQuery];
}>();

const pageCount = computed(() =>
  Math.max(Math.ceil(props.pagination.total / props.query.pageSize), 1),
);
const pageSizes = computed(() => props.pagination.pageSizes ?? [20, 50, 100]);
const currentPage = computed(() => Math.min(props.query.page, pageCount.value));

function updateSearch(keyword: string) {
  emit('updateQuery', { ...props.query, keyword, page: 1 });
}

function updatePage(page: number) {
  if (page < 1 || page > pageCount.value || page === currentPage.value) return;
  emit('updateQuery', { ...props.query, page });
}

function updatePageSize(event: Event) {
  const pageSize = Number((event.target as HTMLSelectElement).value);
  if (!Number.isFinite(pageSize) || pageSize < 1) return;
  emit('updateQuery', { ...props.query, page: 1, pageSize });
}
</script>

<template>
  <section class="data-workspace" aria-label="数据工作台">
    <DataToolbar
      :actions="actions"
      :placeholder="searchPlaceholder"
      :search="query.keyword"
      @action="emit('action', $event)"
      @update:search="updateSearch"
    />

    <div class="data-workspace__table">
      <EvolynTable :columns="columns" :records="records" />
    </div>

    <footer class="data-workspace__footer">
      <div class="data-workspace__pagination-summary">
        <label class="data-workspace__page-size-label">
          <select :value="query.pageSize" @change="updatePageSize">
            <option v-for="size in pageSizes" :key="size" :value="size">{{ size }} 条/页</option>
          </select>
        </label>
        <span>共 {{ pagination.total }} 条</span>
      </div>

      <div class="data-workspace__pagination-controls">
        <button
          type="button"
          aria-label="上一页"
          :disabled="currentPage <= 1"
          @click="updatePage(currentPage - 1)"
        >
          ‹
        </button>
        <span class="data-workspace__current-page">{{ currentPage }} / {{ pageCount }}</span>
        <button
          type="button"
          aria-label="下一页"
          :disabled="currentPage >= pageCount"
          @click="updatePage(currentPage + 1)"
        >
          ›
        </button>
      </div>
    </footer>
  </section>
</template>

<style scoped lang="scss">
.data-workspace {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  background: var(--el-bg-color);

  &__table {
    min-height: 0;
    padding: 0 14px;
    flex: 1;
  }

  &__footer,
  &__pagination-summary,
  &__pagination-controls {
    display: flex;
    align-items: center;
  }

  &__footer {
    min-height: 58px;
    padding: 0 18px;
    justify-content: space-between;
    gap: 16px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  &__pagination-summary {
    gap: 12px;
    color: var(--el-text-color-regular);
    font-size: 14px;
  }

  &__page-size-label select,
  &__pagination-controls button {
    height: 32px;
    color: var(--el-text-color-regular);
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color);
    border-radius: var(--el-border-radius-base);
  }

  &__page-size-label select {
    min-width: 104px;
    padding: 0 8px;
    cursor: pointer;

    &:hover {
      border-color: var(--el-color-primary);
    }
  }

  &__pagination-controls {
    gap: 8px;

    button {
      width: 32px;
      padding: 0;
      cursor: pointer;
      font-size: 22px;
      line-height: 1;

      &:hover:not(:disabled) {
        color: var(--el-color-primary);
        background: var(--el-color-primary-light-9);
        border-color: var(--el-color-primary);
      }

      &:disabled {
        color: var(--el-text-color-disabled);
        cursor: not-allowed;
      }
    }
  }

  &__current-page {
    min-width: 54px;
    color: var(--el-text-color-regular);
    text-align: center;
  }
}

@media (max-width: 620px) {
  .data-workspace {
    &__table {
      padding: 0 4px;
    }

    &__footer {
      padding: 8px 12px;
      align-items: flex-start;
      flex-direction: column;
    }
  }
}
</style>
