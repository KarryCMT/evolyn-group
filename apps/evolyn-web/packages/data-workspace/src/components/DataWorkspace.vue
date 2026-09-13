<script setup lang="ts">
import { EvolynTable } from '@evolyn.do/ui';
import type { EvolynTableColumn } from '@evolyn.do/ui';
import { computed, shallowRef, watch } from 'vue';
import type { DataQuery, DataRecord } from '@evolyn.do/data';
import DataColumnSettings from './DataColumnSettings.vue';
import DataToolbar from './DataToolbar.vue';
import {
  flattenDataColumns,
  isDataColumnGroup,
  type DataAction,
  type DataColumn,
  type DataPagination,
  type DataRecordId,
} from '../types.js';
import { useDataWorkspaceSelection } from '../composables/useDataWorkspaceSelection.js';

defineOptions({ name: 'DataWorkspace' });

const props = withDefaults(
  defineProps<{
    actions: DataAction[];
    columns: DataColumn[];
    records: DataRecord[];
    query: DataQuery;
    pagination: DataPagination;
    /** 递增此值可从页面级批量操作显式清空表格勾选状态。 */
    selectionResetVersion?: number;
    searchPlaceholder?: string;
  }>(),
  {
    searchPlaceholder: '搜索数据',
  },
);

const emit = defineEmits<{
  action: [key: string];
  updateQuery: [query: DataQuery];
  /** 表格首列勾选状态变化；子表单展开的明细行按父记录 ID 收敛为一次选择。 */
  selectionChange: [ids: DataRecordId[]];
}>();

const pageCount = computed(() =>
  Math.max(Math.ceil(props.pagination.total / props.query.pageSize), 1),
);
const pageSizes = computed(() => props.pagination.pageSizes ?? [20, 50, 100]);
const currentPage = computed(() => Math.min(props.query.page, pageCount.value));

// 列显隐是纯展示态：会话内由「列设置」翻转，不进入查询/路由状态。列清单
// 变化（如切换表单）时同步剔除已不存在的隐藏项，避免残留脏 field。
const hiddenFields = shallowRef<ReadonlySet<string>>(new Set());
watch(
  () => props.columns,
  (columns) => {
    const currentFields = new Set(flattenDataColumns(columns).map(({ column }) => column.field));
    const next = new Set<string>();
    for (const field of hiddenFields.value) {
      if (currentFields.has(field)) next.add(field);
    }
    hiddenFields.value = next;
  },
);

const dataRecords = computed(() => props.records);
const { selectionColumn, handleCheckboxStateChange, clearSelection } = useDataWorkspaceSelection({
  records: dataRecords,
  onChange: (ids) => emit('selectionChange', ids),
});

watch(
  () => props.selectionResetVersion,
  () => clearSelection(),
);

const columnSettings = computed(() => flattenDataColumns(props.columns));
const visibleColumns = computed(() => visibleDataColumns(props.columns, hiddenFields.value));
// 工作台可表达分组列；具体表格实现由 UI 包统一接收 VTable 的分组定义，
// 因此在边界处收敛为 UI 组件列契约。
const tableColumns = computed(
  () => [selectionColumn.value, ...visibleColumns.value] as unknown as EvolynTableColumn[],
);

/**
 * 子表单分组仅保留仍有可见叶子列的分支；这样列设置隐藏最后一个子字段时，
 * 父级合并表头会同步消失，不会留下空白表头。
 */
function visibleDataColumns(
  columns: readonly DataColumn[],
  hidden: ReadonlySet<string>,
): DataColumn[] {
  return columns.flatMap<DataColumn>((column): DataColumn | readonly DataColumn[] => {
    if (isDataColumnGroup(column)) {
      const children = visibleDataColumns(column.columns, hidden);
      return children.length === 0 ? [] : [{ ...column, columns: children } as DataColumn];
    }
    if (hidden.has(column.field)) return [];
    // icon 是列设置面板的展示元信息，剥离后再交给表格，避免透传渲染引擎。
    const { icon: _icon, ...tableColumn } = column;
    return [tableColumn as DataColumn];
  });
}

function toggleColumn(field: string) {
  const next = new Set(hiddenFields.value);
  if (next.has(field)) {
    next.delete(field);
  } else {
    // 至少保留一列：全部勾掉会让表格失去取数锚点
    if (next.size + 1 >= columnSettings.value.length) return;
    next.add(field);
  }
  hiddenFields.value = next;
}

/** 列设置「全选」行的批量翻转；「至少保留一列」约束在此统一裁决。 */
function toggleAllColumns(fields: string[], visible: boolean) {
  const next = new Set(hiddenFields.value);
  if (visible) {
    for (const field of fields) next.delete(field);
  } else {
    for (const field of fields) {
      // 隐藏至仅剩一列时停止，其余保持可见
      if (next.size + 1 >= columnSettings.value.length) break;
      next.add(field);
    }
  }
  hiddenFields.value = next;
}

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
    >
      <template #suffix>
        <DataColumnSettings
          :columns="columns"
          :hidden="hiddenFields"
          @toggle="toggleColumn"
          @toggle-all="toggleAllColumns"
        />
      </template>
      <template #suffix-end>
        <!-- 页面级工具型入口（如数据筛选）挂载位：位于搜索框之后 -->
        <slot name="toolbar-suffix-end" />
      </template>
    </DataToolbar>

    <div class="data-workspace__table">
      <EvolynTable
        :columns="tableColumns"
        :records="records"
        :options="{ frozenColCount: 1, enableHeaderCheckboxCascade: false }"
        @checkbox-state-change="handleCheckboxStateChange"
      />
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
