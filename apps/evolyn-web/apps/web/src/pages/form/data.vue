<script setup lang="ts">
import {
  RiAddFill,
  RiCheckboxMultipleFill,
  RiDeleteBack2Fill,
  RiDeleteBin6Fill,
  RiDownload2Fill,
  RiHistoryFill,
  RiUpload2Fill,
} from '@remixicon/vue';
import { DataWorkspace, useDataWorkspace, type DataAction } from '@evolyn.do/data-workspace';
import type { QueryExpression } from '@evolyn.do/query';
import { ElMessage } from 'element-plus';
import { computed, markRaw } from 'vue';
import { useRoute } from 'vue-router';
import { useFormRecordDataSource } from '~/composables/useFormRecordDataSource';
import FormRecordFilterPanel from '~/components/form/data/FormRecordFilterPanel.vue';

defineOptions({ name: 'FormDataPage' });

const { query, updateQuery } = useDataWorkspace();
const route = useRoute();
const appCode = computed(() => String(route.params.appCode ?? ''));
const formCode = computed(() => String(route.params.formCode ?? ''));
const { columns, filterFields, records, total, status, errorMessage, reload } =
  useFormRecordDataSource({ appCode, formCode, query });
// 数据源对外只读；表格接收独立行副本，避免渲染层意外改写领域缓存。
const tableRecords = computed(() => records.value.map((record) => ({ ...record })));
// 「筛选」为工具栏工具型入口（搜索框旁的弹层面板），不在业务动作区
const actions: DataAction[] = [
  { key: 'create', label: '添加', icon: markRaw(RiAddFill), tone: 'primary' },
  { key: 'import', label: '导入', icon: markRaw(RiUpload2Fill) },
  { key: 'export', label: '导出', icon: markRaw(RiDownload2Fill) },
  { key: 'remove', label: '删除', icon: markRaw(RiDeleteBin6Fill), tone: 'danger' },
  { key: 'batch', label: '批量操作', icon: markRaw(RiCheckboxMultipleFill) },
  { key: 'operation-log', label: '操作记录', icon: markRaw(RiHistoryFill) },
  { key: 'recycle-bin', label: '数据回收站', icon: markRaw(RiDeleteBack2Fill) },
];

function handleAction(key: string) {
  const action = actions.find((item) => item.key === key);
  ElMessage.info(`${action?.label ?? '该'}功能暂未开放`);
}

function updateFilter(filter: QueryExpression | undefined) {
  // DataQuery 是 shallowRef；始终经工作台动作整体替换，才能触发一次新的
  // 服务端查询并将筛选结果从第一页开始展示。
  updateQuery({ filter, page: 1 });
}
</script>

<template>
  <section class="form-data-page" aria-label="数据管理工作台">
    <p v-if="status === 'error'" class="form-data-page__error" role="alert">
      {{ errorMessage }}
      <button type="button" @click="reload">重试</button>
    </p>
    <p v-else-if="status === 'loading'" class="form-data-page__loading" aria-live="polite">
      正在加载表单数据…
    </p>
    <DataWorkspace
      v-else
      :actions="actions"
      :columns="columns"
      :records="tableRecords"
      :query="query"
      :pagination="{ total, page: query.page, pageSize: query.pageSize }"
      @action="handleAction"
      @update-query="updateQuery"
    >
      <template #toolbar-suffix-end>
        <FormRecordFilterPanel
          :model-value="query.filter"
          :fields="filterFields"
          @update:model-value="updateFilter"
        />
      </template>
    </DataWorkspace>
  </section>
</template>

<style scoped lang="scss">
.form-data-page {
  display: flex;
  min-height: 0;
  margin: 0 var(--el-space-md) var(--el-space-md);
  overflow: hidden;
  flex: 1;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-large);
  box-shadow: var(--el-box-shadow-light);
}

.form-data-page__error {
  display: flex;
  margin: auto;
  align-items: center;
  gap: var(--el-space-sm);
  color: var(--el-color-danger);
}

.form-data-page__loading {
  margin: auto;
  color: var(--el-text-color-secondary);
}

@media (max-width: 620px) {
  .form-data-page {
    margin: 0 var(--el-space-xs) var(--el-space-xs);
    border-radius: var(--el-border-radius-large);
  }
}
</style>
