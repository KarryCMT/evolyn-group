<script setup lang="ts">
import type { EvolynTableColumn } from '@evolyn.do/ui';
import type { DataPushItem } from './dataPush.types';
import { EvolynTable } from '@evolyn.do/ui';
import { computed, shallowRef } from 'vue';

defineOptions({ name: 'DataPushList' });

const props = defineProps<{
  items: DataPushItem[];
}>();

const ROW_HEIGHT = 84;
const currentPage = shallowRef(1);

const columns: EvolynTableColumn[] = [
  { field: 'name', title: '数据推送名称', width: 200 },
  { field: 'serverAddress', title: '服务器地址', width: 380 },
  { field: 'formName', title: '推送表单', width: 200 },
  { field: 'events', title: '推送事件', width: 340, cellType: 'multilinetext' },
  { field: 'remark', title: '备注', width: 250 },
  { field: 'operation', title: '操作', width: 310 },
];

/** VTable 使用普通文本单元格，规避复杂画布单元格在当前版本中的挂载异常。 */
const tableRecords = computed(() =>
  props.items.map((item) => ({
    ...item,
    operation: item.enabled ? '推送日志    编辑    删除    ●' : '推送日志    编辑    删除    ○',
  })),
);
const tableOptions = { defaultHeaderRowHeight: 60, defaultRowHeight: ROW_HEIGHT };
</script>

<template>
  <section class="data-push-list" aria-label="数据推送列表">
    <EvolynTable
      class="data-push-list__table"
      :columns="columns"
      :records="tableRecords"
      :options="tableOptions"
      empty-text="暂无数据推送"
    />
    <footer class="data-push-list__footer">
      <span>共 {{ props.items.length }} 条</span>
      <el-pagination
        v-model:current-page="currentPage"
        :page-size="20"
        :total="props.items.length"
        layout="prev, pager, next"
      />
    </footer>
  </section>
</template>

<style scoped lang="scss">
.data-push-list {
  display: flex;
  min-height: 0;
  padding: var(--el-space-lg) var(--el-space-3xl) var(--el-space-2xl);
  flex: 1;
  flex-direction: column;

  &__table {
    min-height: 0;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-base);
    flex: 1;
    overflow: hidden;
  }

  &__footer {
    display: flex;
    min-height: 58px;
    padding: var(--el-space-md) var(--el-space-md) 0;
    align-items: center;
    justify-content: space-between;
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-base);
  }
}
</style>
