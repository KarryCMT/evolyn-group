<script setup lang="ts">
import {
  RiAddFill,
  RiCheckboxMultipleFill,
  RiDeleteBack2Fill,
  RiDeleteBin6Fill,
  RiDownload2Fill,
  RiFilter3Fill,
  RiHistoryFill,
  RiUpload2Fill,
} from '@remixicon/vue';
import {
  DataWorkspace,
  useDataWorkspace,
  type DataAction,
  type DataColumn,
} from '@evolyn.do/data-workspace';
import type { DataRecord } from '@evolyn.do/data';
import { ElMessage } from 'element-plus';
import { computed, markRaw } from 'vue';

defineOptions({ name: 'FormDataPage' });

const { query, updateQuery } = useDataWorkspace();
const actions: DataAction[] = [
  { key: 'create', label: '添加', icon: markRaw(RiAddFill), tone: 'primary' },
  { key: 'import', label: '导入', icon: markRaw(RiUpload2Fill) },
  { key: 'export', label: '导出', icon: markRaw(RiDownload2Fill) },
  { key: 'remove', label: '删除', icon: markRaw(RiDeleteBin6Fill), tone: 'danger' },
  { key: 'batch', label: '批量操作', icon: markRaw(RiCheckboxMultipleFill) },
  { key: 'operation-log', label: '操作记录', icon: markRaw(RiHistoryFill) },
  { key: 'recycle-bin', label: '数据回收站', icon: markRaw(RiDeleteBack2Fill) },
  { key: 'filter', label: '筛选', icon: markRaw(RiFilter3Fill) },
];

const columns: DataColumn[] = [
  { field: 'projectCode', title: '项目编号', width: 132, sortable: true },
  { field: 'projectName', title: '项目名称', width: 190, sortable: true },
  { field: 'partyAPm', title: '甲方 PM', width: 152, cellType: 'link' },
  { field: 'partyBPm', title: '乙方 PM', width: 152 },
  { field: 'department', title: '归属部门', width: 190, cellType: 'link' },
  { field: 'plannedStart', title: '项目计划开始时间', width: 178, sortable: true },
  { field: 'plannedEnd', title: '项目计划结束时间', width: 178, sortable: true },
  { field: 'status', title: '项目状态', width: 130 },
  { field: 'submitter', title: '提交人', width: 130, cellType: 'link' },
];

const sourceRecords: DataRecord[] = [
  {
    projectCode: 'A002',
    projectName: '资产管理项目',
    partyAPm: '李同学',
    partyBPm: '林一',
    department: '重庆万柯互联网科技有限公司',
    plannedStart: '2022-06-01',
    plannedEnd: '2022-07-31',
    status: '进行中',
    submitter: '李同学',
  },
  {
    projectCode: 'A001',
    projectName: 'ERP 项目一期',
    partyAPm: '李同学',
    partyBPm: '张林',
    department: '重庆万柯互联网科技有限公司',
    plannedStart: '2022-04-01',
    plannedEnd: '2022-11-01',
    status: '进行中',
    submitter: '李同学',
  },
];

const filteredRecords = computed(() => {
  const keyword = query.value.keyword.trim().toLowerCase();
  if (!keyword) return sourceRecords;

  return sourceRecords.filter((record) =>
    Object.values(record).some((value) => String(value).toLowerCase().includes(keyword)),
  );
});
const pageRecords = computed(() => {
  const start = (query.value.page - 1) * query.value.pageSize;
  return filteredRecords.value.slice(start, start + query.value.pageSize);
});

function handleAction(key: string) {
  const action = actions.find((item) => item.key === key);
  ElMessage.info(`${action?.label ?? '该'}功能将在表单数据接口接入后提供`);
}
</script>

<template>
  <section class="form-data-page" aria-label="数据管理工作台">
    <DataWorkspace
      :actions="actions"
      :columns="columns"
      :records="pageRecords"
      :query="query"
      :pagination="{ total: filteredRecords.length, page: query.page, pageSize: query.pageSize }"
      @action="handleAction"
      @update-query="updateQuery"
    />
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

@media (max-width: 620px) {
  .form-data-page {
    margin: 0 var(--el-space-xs) var(--el-space-xs);
    border-radius: var(--el-border-radius-large);
  }
}
</style>
